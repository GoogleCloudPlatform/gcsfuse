package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"cloud.google.com/go/storage"
	"cloud.google.com/go/storage/experimental"
)

const (
	Charset = "abcdefghijklmnopqrstuvwxyz0123456789"
)

func GenerateRandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = Charset[rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(Charset))]
	}
	return string(b)
}

func main() {
	fmt.Println("started execution")
	// create client
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
	object := "testObject1"
	mib := 1024 * 1024
	// create writer.
	objHandle := client.Bucket(bucket).Object(object)
	wc := objHandle.NewWriter(ctx)
	wc.Append = true
	wc.FinalizeOnClose = true

	writeOffset, err := wc.Write([]byte(GenerateRandomString(2 * mib)))
	if err != nil {
		panic(err)
	}
	if writeOffset != 2*mib {
		panic("writeOffset != 2MiB")
	}
	flushOffset, err := wc.Flush()
	if err != nil {
		panic(err)
	}
	if flushOffset != int64(2*mib) {
		panic("flushOffset != 2MiB")
	}

	fmt.Println("waitting")
	time.Sleep(10 * time.Second)

	attrs, err := client.Bucket(bucket).Object(object).Attrs(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("After writer flushed: %v\n", attrs.Finalized)

	err = wc.Close()
	if err != nil {
		panic(err)
	}

	attrs, err = client.Bucket(bucket).Object(object).Attrs(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("After writer closed: %v\n", attrs.Finalized)
}
