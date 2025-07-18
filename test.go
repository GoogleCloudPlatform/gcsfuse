package main

import (
	"context"
	"fmt"
	"io"
	"slices"

	"cloud.google.com/go/storage"
	"cloud.google.com/go/storage/experimental"
)

func main() {
	ctx := context.Background()
	client, err := storage.NewGRPCClient(ctx, experimental.WithGRPCBidiReads())
	if err != nil {
		panic(err)
	}
	defer func(client *storage.Client) {
		err := client.Close()
		if err != nil {
			panic(err)
		}
	}(client)

	bucket := "ashmeen-zb"
	object := "experiment.1.0"
	obj1 := client.Bucket(bucket).Object(object)
	fmt.Println("Creating with nil read handle")
	obj1 = obj1.ReadHandle(nil)
	storageReader, err := obj1.NewRangeReader(ctx, 0, 200*1024*1024)
	if err != nil {
		fmt.Printf("NewRangeReader err: %v\n", err)
	}
	gotRH := storageReader.ReadHandle()
	p := make([]byte, 1024*1024*200)
	n, err := io.ReadFull(storageReader, p)
	if err != nil {
		fmt.Printf("Read 1 err: %v\n", err)
	}
	fmt.Println("Read %n bytes", n)
	fmt.Println("Read1 complete")
	//fmt.Println("Read1 = ", string(p))
	err = storageReader.Close()
	if err != nil {
		fmt.Printf("Close1 err: %v\n", err)
		return
	}
	obj1 = nil

	obj2 := client.Bucket(bucket).Object(object)
	fmt.Println("Creating with reader.ReadHandle (closed)")
	fmt.Println("compare: ", slices.Compare(storageReader.ReadHandle(), gotRH))
	obj2.ReadHandle(storageReader.ReadHandle())
	storageReader, err = obj2.NewRangeReader(ctx, 200*1024*1024, 200*1024*1024)
	if err != nil {
		fmt.Printf("Close err: %v\n", err)
		return
	}
	p = make([]byte, 1024*1024*200)
	n, err = io.ReadFull(storageReader, p)
	if err != nil {
		fmt.Printf("Read 2 err: %v\n", err)
	}
	//fmt.Println("Read2 = ", string(p))
	fmt.Println("Read %n bytes", n)
	fmt.Println("Read2 complete")
	err = storageReader.Close()
	if err != nil {
		fmt.Printf("Close2 err: %v\n", err)
		return
	}
	obj2 = nil

	obj3 := client.Bucket(bucket).Object(object)
	fmt.Println("Creating with stored read handle")
	obj3.ReadHandle(gotRH)
	storageReader, err = obj3.NewRangeReader(ctx, 400*1024*1024, 200*1024*1024)
	if err != nil {
		fmt.Printf("Close err: %v\n", err)
		return
	}
	p = make([]byte, 1024*1024*200)
	n, err = io.ReadFull(storageReader, p)
	if err != nil {
		fmt.Printf("Read 3 err: %v\n", err)
	}
	fmt.Println("Read %n bytes", n)
	fmt.Println("Read3 complete")
	//fmt.Println("Read3 = ", string(p))
	err = storageReader.Close()
	if err != nil {
		fmt.Printf("Close2 err: %v\n", err)
		return
	}
	obj3 = nil

	return
}
