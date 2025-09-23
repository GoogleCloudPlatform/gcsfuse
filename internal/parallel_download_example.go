// parallel_download_example.go
// Example: Parallel download of a large object using static worker pool and DownloadTask.
// Place: internal/parallel_download_example.go

package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/bufferedread"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"golang.org/x/sync/semaphore"
)

// main demonstrates parallel download of a large object from a real GCS bucket using the internal client.
func main() {

	ctx := context.Background()

	// Parallel download using PrefetchBlockPool only

	bucketName := "princer-working-dirs"
	objectName := "100gb_file.bin"
	fmt.Printf("Using bucket: %s, object: %s\n", bucketName, objectName)

	// Minimal config for StorageHandle. Adjust as needed for your environment.
	config := storageutil.StorageClientConfig{
		UserAgent: "gcsfuse-parallel-download-example",
		// Add more config as needed (auth, endpoint, etc)
		ClientProtocol: cfg.HTTP1,
	}
	storageHandle, err := storage.NewStorageHandle(ctx, config, "")
	if err != nil {
		panic(fmt.Sprintf("Failed to create storage handle: %v", err))
	}

	bucketHandle, err := storageHandle.BucketHandle(ctx, bucketName, "")
	if err != nil {
		panic(fmt.Sprintf("Failed to get bucket handle: %v", err))
	}

	// Get object metadata
	minObj, _, err := bucketHandle.StatObject(ctx, &gcs.StatObjectRequest{Name: objectName})
	if err != nil {
		panic(fmt.Sprintf("Failed to stat object: %v", err))
	}

	objectSize := int(minObj.Size)
	blockSize := 16 * 1024 * 1024 // 1 MiB blocks
	numBlocks := (objectSize + blockSize - 1) / blockSize
	metricHandle := metrics.NewNoopMetrics()

	pool, err := workerpool.NewStaticWorkerPool(100, 100, int64(numBlocks))
	if err != nil {
		panic(err)
	}
	pool.Start()
	defer pool.Stop()

	blockPool, err := block.NewPrefetchBlockPool(int64(blockSize), int64(numBlocks), 0, semaphore.NewWeighted(int64(numBlocks)))
	if err != nil {
		panic(fmt.Sprintf("Failed to create block pool: %v", err))
	}
	blockDoneCh := make(chan block.PrefetchBlock, numBlocks)

	// Goroutine to release blocks back to the pool
	go func() {
		for b := range blockDoneCh {
			blockPool.Release(b)
		}
	}()

	// Launch download tasks
	var wg sync.WaitGroup
	for i := 0; i < numBlocks; i++ {
		wg.Add(1)
		b, err := blockPool.Get()
		if err != nil {
			panic(fmt.Sprintf("Failed to get block from PrefetchBlockPool: %v", err))
		}
		b.SetAbsStartOff(int64(i * blockSize))
		blockStart := uint64(i * blockSize)
		blockEnd := blockStart + uint64(blockSize)
		if blockEnd > uint64(minObj.Size) {
			blockEnd = uint64(minObj.Size)
		}
		blockObj := *minObj
		blockObj.Size = blockEnd
		task := bufferedread.NewDownloadTask(ctx, &blockObj, bucketHandle, b, nil, metricHandle)
		go func(b block.PrefetchBlock) {
			defer wg.Done()
			pool.Schedule(false, task)
			status, errcopy := b.AwaitReady(ctx)
			if errcopy != nil || status.Err != nil {
				fmt.Printf("Block download failed: %v\n", status.Err)
			} else {
				// fmt.Printf("Block downloaded: offset %d, size %d\n", b.AbsStartOff(), b.Cap())
			}
			blockDoneCh <- b
		}(b)
	}
	wg.Wait()
	close(blockDoneCh)

	// elapsed := time.Since(start) // The start variable is defined inside the if/else block, so it's not accessible here.
	fmt.Printf("All blocks downloaded and released!\n")
}
