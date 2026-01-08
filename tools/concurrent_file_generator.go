// Copyright 2026 Google LLC
// Simple concurrent file generator

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func generateFiles(threadID, filesPerThread int, dir string, fileSize int, wg *sync.WaitGroup) {
	defer wg.Done()

	data := make([]byte, fileSize)
	for i := 0; i < len(data); i++ {
		data[i] = byte('A' + (i % 26))
	}

	for i := 0; i < filesPerThread; i++ {
		filename := filepath.Join(dir, fmt.Sprintf("test.%d.%d", threadID, i))
		if err := os.WriteFile(filename, data, 0644); err != nil {
			log.Printf("Error: %v", err)
			continue
		}
	}
	log.Printf("Thread %d done", threadID)
}

func main() {
	threads := flag.Int("threads", 4, "Number of threads")
	files := flag.Int("files", 1000, "Files per thread")
	dir := flag.String("dir", "/tmp/test_files", "Target directory")
	sizeKB := flag.Int("size", 1, "File size in KB")
	flag.Parse()

	os.MkdirAll(*dir, 0755)

	sizeBytes := *sizeKB * 1024

	log.Printf("Creating %d files with %d threads in %s", *threads**files, *threads, *dir)
	start := time.Now()

	var wg sync.WaitGroup
	for i := 0; i < *threads; i++ {
		wg.Add(1)
		go generateFiles(i, *files, *dir, sizeBytes, &wg)
	}
	wg.Wait()

	log.Printf("Done in %v", time.Since(start))
}
