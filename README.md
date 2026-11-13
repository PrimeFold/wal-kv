wal-kv

A minimalist, crash-resilient in-memory Key-Value engine backed by an append-only Write-Ahead Log (WAL) with strict CRC32 frame checksumming, tombstone deletions, and hardware-enforced durability.

Built with zero external dependencies using pure Go standard library primitives (net/http, sync, os, hash/crc32, encoding/binary).

Architecture Overview

wal-kv decouples high-throughput in-memory reads from durable disk persistence using a two-tier lock hierarchy:

[ Client Request ] │ ├──► GET ──► dbMu.RLock() ──► Read RAM (23 ns) ──► Return │ └──► SET/DEL ──► writeMu.Lock() │ ├──► Write & Fsync to WAL File (~1.6 ms) │ └──► dbMu.Lock() ──► Mutate RAM (5 ns)

### Key Design Primitives:
1. **Unidirectional Lock Hierarchy (`writeMu` → `dbMu`)**:
   Writers serialize disk I/O through `writeMu` without holding the read lock. Readers querying keys via `GET` operate concurrently at memory speeds without blocking on disk `fsync`.
2. **Binary Framing & Checksumming**:
   Every record is wrapped in a fixed 8-byte binary header containing an IEEE CRC32 checksum and a 32-bit payload length prefix.
3. **Crash Recovery & Torn-Write Repair**:
   On reboot, the recovery scanner sequentially verifies record checksums. If a torn write is detected (e.g. from an ungraceful mid-flight power cut), the log is truncated back to the last valid byte offset.
4. **Tombstone Deletions**:
   Deletions append a tombstone (`DEL <key>`) to the WAL before evicting the key from memory. Replaying the log restores deletions deterministically across process crashes.
---
## Binary Frame Specification
All records are written sequentially in big-endian wire format:

+--------------------+----------------------+--------------------------+ | CRC32 Checksum | Payload Length (N) | Payload Bytes (Command) | | (4 Bytes) | (4 Bytes) | (N Bytes) | +--------------------+----------------------+--------------------------+ | 0x00 ... 0x03 | 0x04 ... 0x07 | 0x08 ... 0x08 + N | +--------------------+----------------------+--------------------------+

* **CRC32**: Generated via `hash/crc32` (`IEEE` polynomial table). Checked during boot to detect bit rot and torn writes.
* **Length Prefix**: Fixed 32-bit big-endian unsigned integer, eliminating the need for delimiter parsing or escaping.
---
## Hardware Benchmarks
Ran on Linux (`amd64`, Intel Core Ultra 5 125H):

pkg: walkv BenchmarkRAM_Read-6 51,340,849 23.24 ns/op 0 B/op 0 allocs/op BenchmarkWAL_Write_PageCacheOnly-6 1,404,549 849.30 ns/op 0 B/op 0 allocs/op BenchmarkWAL_Write_WithSync-6 720 1,619,807.00 ns/op 35 B/op 1 allocs/op

* **RAM Reads (`GET`)**: ~43.0M ops/sec at zero allocations (`0 B/op`).
* **Page Cache Writes**: ~1.17M ops/sec when buffering through the OS page cache.
* **Physical Fsync**: ~617 ops/sec bound by physical SSD NAND flash commit latency (~1.62ms).
---
## HTTP API
Start the engine:
```bash
go run .
1. Set a Key
bash
curl "http://localhost:8080/set?key=user&val=aditya"
# Response: OK: user = aditya
2. Read a Key
bash
curl "http://localhost:8080/get?key=user"
# Response: aditya
3. Delete a Key (Tombstone)
bash
curl "http://localhost:8080/delete?key=user"
# Response: OK: deleted user
4. Crash Recovery Verification
bash
# Force kill the process (simulating sudden power loss)
killall -9 walkv
# Restart the process
go run .
# Output:
# Scanning WAL and restoring RAM state...
# Recovery complete! Restored keys into RAM.
