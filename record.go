package main

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"log"
	"os"
)



func EncodeRecord(payload []byte) []byte{
	length := 8+len(payload)

	crc := crc32.ChecksumIEEE(payload)

	buf := make([]byte,length)

	binary.BigEndian.PutUint32(buf[0:4], crc)
	binary.BigEndian.PutUint32(buf[4:8], uint32(len(payload)))
	copy(buf[8:],payload)

	return buf

}

func DecodeRecord() ([]byte,error){
	
	f,err:= os.OpenFile("wal.log",os.O_CREATE|os.O_RDWR,0644)

	if err != nil{
		log.Printf("Couldn't read the file : %v",err)
	}

	var lastValidOffset int64;
	
	header:= make([]byte,8)

	io.ReadFull(f,header)//The reader object reads all the bytes and puts into the buf object i.e header

	expectedCRC := binary.BigEndian.Uint32(header[0:4])
	
	length:= binary.BigEndian.Uint32(header[4:8])
		
	payload := header[8: 8+length]


	actualCRC := crc32.ChecksumIEEE(payload)

	if expectedCRC != actualCRC {
		log.Printf("Corrupted record detected! Truncating file at byte offset %d\n", lastValidOffset)
		// 1. Physically chop off the corrupted tail:
    	if err := f.Truncate(lastValidOffset); err != nil {
			log.Printf("failed to truncate: %v", err)
    	    return nil,errors.New("corrupted record: checksum mismatch")
    	}

	}
	log.Printf("SUCCESS")
	return payload,nil

}
