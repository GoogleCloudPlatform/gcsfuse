package main

import (
	"bytes"
	"context"
	"log"

	"cloud.google.com/go/storage"
	"cloud.google.com/go/storage/experimental"
)

func Generate10MBString() string {
	// 800 max
	// 500 min
	const sizeBytes = 500

	var buffer bytes.Buffer
	pattern := "abcdefghijklmnopqrstuvwxyz0123456789" // Example pattern

	for i := 0; i < sizeBytes; i++ {
		buffer.WriteByte(pattern[i%len(pattern)]) // Repeat the pattern
	}

	return buffer.String()
}

func main() {
	// create client
	ctx := context.Background()
	client, err := storage.NewGRPCClient(ctx, experimental.WithGRPCBidiReads())
	if err != nil {
		log.Printf("Err on NewGRPCClient: %v", err)
	}
	defer func() {
		err := client.Close()
		if err != nil {
			log.Printf("Err on client.Close(): %v", err)
		}
	}()

	bucket := "ashmeen-zb"
	object := "testObject"
	// create writer.
	objHandle := client.Bucket(bucket).Object(object)
	wc := objHandle.NewWriter(ctx)
	wc.Append = true
	attrs, err := client.Bucket(bucket).Attrs(ctx)
	if err != nil {
		//panic(err)
	}
	log.Printf("Storage Class: %v", attrs.StorageClass)
	data := Generate10MBString()
	writeOffset, err := wc.Write(bytes.NewBufferString(data).Bytes())
	if err != nil {
		log.Printf("Err on Write: %v", err)
		//panic(err)
	}
	log.Println("A")
	if writeOffset != len(data) {
		log.Println("Err on offSetMismatch")
		//panic("writeOffset != len(hello world)")
	}
	flushOffset, err := wc.Flush()
	log.Println("B")
	if err != nil {
		log.Printf("Err on Flush %v", err)
		//panic(err)
	}
	log.Println("C")
	if flushOffset != int64(len(data)) {
		//panic("flushOffset != len(hello world)")
	}
	err = wc.Close()
	log.Println("D")
	if err != nil {
		log.Printf("Err on Close %v", err)
		//panic(err)
	}
}
