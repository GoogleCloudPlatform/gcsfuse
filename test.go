package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"syscall"
	"time"
)

func wr() error {
	fmt.Println("Creating file")
	openFileFlags := os.O_TRUNC | os.O_WRONLY | syscall.O_DIRECT
	f, err := os.OpenFile("/mnt/disks/ssd/gcsmount/demo22.txt", openFileFlags, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	const chunkSize = 1 * 1023
	offset := 0

	//offset := int64(80 * 1024 * 1024) // 80 MB
	for idx := 1; offset < 10*1024*1024; idx++ {
		fmt.Println("sleeping")
		time.Sleep(1 * time.Second)
		fmt.Println("writing at offset")
		fmt.Println(offset)

		fmt.Println("seek done")
		if _, err := f.Write(bytes.Repeat([]byte{'A'}, chunkSize)); err != nil {
			fmt.Println(err)
			return err
		}
		fmt.Println("write done")

	}

	if err := f.Sync(); err != nil {
		return err
	}

	return nil
}

func main2() {
	if err := wr(); err != nil {
		log.Fatal(err)
	}
}
