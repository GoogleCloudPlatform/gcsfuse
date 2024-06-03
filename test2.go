package main2

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/storage"
)

func main(){
	var ctx context.Context
	var err error
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 2 * time.Minute)
	client, err := storage.NewClient(ctx)


	attrs, err := client.Bucket("tulsishah_test").Object("a.txt").Attrs(ctx)
	if err != nil {
		fmt.Println("Error: ", err)
	}

	fmt.Println(attrs.Name)
  defer cancel()
}

