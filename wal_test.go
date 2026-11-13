package main

import (
	"log"
	"os"
	"testing"
)

func BenchmarkRAM_Read(b *testing.B){
	db["user"] = "Aditya"
	b.ReportAllocs()


	for b.Loop(){
		dbMu.Lock()
		_=db["user"]
		dbMu.Unlock()
	}

}

func BenchmarkWAL_Write_WithSync(b *testing.B){
	wal,err:= OpenWAL("bench_sync.wal")

	if err != nil{
		log.Fatal(err)
	}

	defer func(){
		wal.Close()
		os.Remove("bench_sync.wal")
	}()

	payload:= []byte("SET user Aditya")
	b.ReportAllocs()

	for b.Loop(){
		if err := wal.Write(payload) ; err != nil{
			log.Fatal(err)
		}
	}

}

// 3. How fast is writing to OS Page Cache WITHOUT fsync?
func BenchmarkWAL_Write_PageCacheOnly(b *testing.B) {
	f, err := os.OpenFile("bench_nosync.wal", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		f.Close()
		os.Remove("bench_nosync.wal")
	}()
	frame := EncodeRecord([]byte("SET user Aditya"))
	
	b.ReportAllocs()
	for b.Loop() {
		if _, err := f.Write(frame); err != nil {
			b.Fatal(err)
		}
		// NO f.Sync() called! Just OS RAM.
	}
}

