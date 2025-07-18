package main

import (
	"context"
	"fmt"

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
	object := "a.txt"
	obj := client.Bucket(bucket).Object(object)
	obj = obj.ReadHandle(nil)
	storageReader, err := obj.NewRangeReader(ctx, 0, 2)
	if err != nil {
		fmt.Printf("NewRangeReader err: %v\n", err)
	}
	gotRH := storageReader.ReadHandle()
	p := make([]byte, 2)
	_, err = storageReader.Read(p)
	if err != nil {
		fmt.Printf("Read 1 err: %v\n", err)
	}
	fmt.Println("Read1 = ", string(p))

	err = storageReader.Close()
	if err != nil {
		fmt.Printf("Close err: %v\n", err)
		return
	}

	obj.ReadHandle(nil)
	storageReader, err = obj.NewRangeReader(ctx, 2, 4)
	if err != nil {
		fmt.Printf("Close err: %v\n", err)
		return
	}
	p = make([]byte, 2)
	_, err = storageReader.Read(p)
	if err != nil {
		fmt.Printf("Read 2 err: %v\n", err)
	}
	fmt.Println("Read2 = ", string(p))

	err = storageReader.Close()
	if err != nil {
		fmt.Printf("Close err: %v\n", err)
		return
	}

	obj.ReadHandle(gotRH)
	storageReader, err = obj.NewRangeReader(ctx, 4, 6)
	if err != nil {
		fmt.Printf("Close err: %v\n", err)
		return
	}
	p = make([]byte, 2)
	_, err = storageReader.Read(p)
	if err != nil {
		fmt.Printf("Read 3 err: %v\n", err)
	}
	fmt.Println("Read3 = ", string(p))

	return
}
