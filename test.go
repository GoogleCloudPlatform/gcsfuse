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
	storageReader1, err := obj1.NewRangeReader(ctx, 0, 200*1024*1024)
	if err != nil {
		fmt.Printf("NewRangeReader err: %v\n", err)
	}
	gotRH := storageReader1.ReadHandle()
	p := make([]byte, 1024*1024*200)
	n, err := io.ReadFull(storageReader1, p)
	if err != nil {
		fmt.Printf("Read 1 err: %v\n", err)
	}
	fmt.Printf("Read %d bytes\n", n)
	fmt.Println("Read1 complete")
	//fmt.Println("Read1 = ", string(p))
	err = storageReader1.Close()
	if err != nil {
		fmt.Printf("Close1 err: %v\n", err)
		return
	}
	obj1 = nil

	obj2 := client.Bucket(bucket).Object(object)
	fmt.Println("Creating with reader.ReadHandle (closed)")
	obj2.ReadHandle(gotRH)
	storageReader2, err := obj2.NewRangeReader(ctx, 200*1024*1024, 200*1024*1024)
	if err != nil {
		fmt.Printf("Close err: %v\n", err)
		return
	}
	fmt.Println("compare new returned rh vs prev rh: ", slices.Compare(storageReader2.ReadHandle(), gotRH))
	p = make([]byte, 1024*1024*200)
	n, err = io.ReadFull(storageReader2, p)
	if err != nil {
		fmt.Printf("Read 2 err: %v\n", err)
	}
	//fmt.Println("Read2 = ", string(p))
	fmt.Printf("Read %d bytes\n", n)
	fmt.Println("Read2 complete")
	err = storageReader2.Close()
	if err != nil {
		fmt.Printf("Close2 err: %v\n", err)
		return
	}
	obj2 = nil

	obj3 := client.Bucket(bucket).Object(object)
	fmt.Println("Creating with stored read handle")
	obj3.ReadHandle(gotRH)
	storageReader3, err := obj3.NewRangeReader(ctx, 400*1024*1024, 200*1024*1024)
	if err != nil {
		fmt.Printf("Close err: %v\n", err)
		return
	}
	p = make([]byte, 1024*1024*200)
	n, err = io.ReadFull(storageReader3, p)
	if err != nil {
		fmt.Printf("Read 3 err: %v\n", err)
	}
	fmt.Printf("Read %d bytes\n", n)
	fmt.Println("Read3 complete")
	//fmt.Println("Read3 = ", string(p))
	err = storageReader3.Close()
	if err != nil {
		fmt.Printf("Close2 err: %v\n", err)
		return
	}
	obj3 = nil

	return
}
