package main

import (
	"encoding/binary"
	"hash/crc32"
	"io"
	"os"
	"sync"
)


type WAL struct{
	mu sync.Mutex
	file *os.File
	path string
}



func OpenWAL(path string) (*WAL,error){
	f,err:= os.OpenFile(path,os.O_CREATE|os.O_RDWR|os.O_APPEND,0644)
	if err != nil{
		return nil,err
	}

	return &WAL{
		file : f,
		path: path,
	},nil
}


func (w *WAL) Write(data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()


	frame := EncodeRecord(data)

	if _,err := w.file.Write(frame) ; err != nil{
		return err
	}

	return w.file.Sync()

}

func (w *WAL) Close() error{
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}


func (w *WAL) Recover() ([][]byte,error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	var records [][]byte
	var lastValidOffset int64 = 0


	if _,err:= w.file.Seek(0,io.SeekStart); err != nil{
		return nil, err
	}

	for{

		header := make([]byte,8)

		_,err:= io.ReadFull(w.file,header)

		if err == io.EOF{
			break
		}

		if err == io.ErrUnexpectedEOF{
			// Power cut mid-header! Chop off garbage:

			w.file.Truncate(lastValidOffset)
			break
		}

		if err != nil{
			return nil,err
		}


		expectedCRC:= binary.BigEndian.Uint32(header[0:4])
		length:= binary.BigEndian.Uint32(header[4:8])

		payload := make([]byte,length)

		_,err = io.ReadFull(w.file,payload)

		if err != nil{
			w.file.Truncate(lastValidOffset)
			break
		}

		if crc32.ChecksumIEEE(payload) != expectedCRC {
			// Corrupted bytes! Chop off:
			w.file.Truncate(lastValidOffset)
			break
		}


		records = append(records,payload)

		lastValidOffset+= int64(8 + length)
	}

	return records,nil
}
