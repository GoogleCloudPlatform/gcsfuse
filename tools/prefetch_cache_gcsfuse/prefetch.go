// Copyright 2022 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/internal/contentcache"
	"google.golang.org/api/iterator"
)

const NUM_WORKERS = 10

func downloadFile(ctx context.Context, client *storage.Client, object *storage.ObjectAttrs, cacheDir string) (err error) {
	log.Printf(fmt.Sprintf("downloading file %v from bucket %v into dir %v", object.Name, object.Bucket, cacheDir))

	// We may want a way to verify the files are fully downloaded
	// and either resuming the download or discarding and redownloading the file
	// We may also want to do cleanup if files are created on disk but aren't populated in time

	f, err := ioutil.TempFile(cacheDir, contentcache.CacheFilePrefix)

	if err != nil {
		err = fmt.Errorf("ioutil.TempFile: %w", err)
		return
	}
	defer f.Close()

	rc, err := client.Bucket(object.Bucket).Object(object.Name).NewReader(ctx)

	if err != nil {
		return fmt.Errorf("Object(%q).NewReader: %w", object.Name, err)
	}
	defer rc.Close()

	if _, err := io.Copy(f, rc); err != nil {
		return fmt.Errorf("io.Copy: %w", err)
	}

	metadata := &contentcache.CacheFileObjectMetadata{
		CacheFileNameOnDisk: f.Name(),
		BucketName:          object.Bucket,
		ObjectName:          object.Name,
		Generation:          object.Generation,
		MetaGeneration:      object.Metageneration,
	}

	file, err := json.MarshalIndent(*metadata, "", " ")
	err = ioutil.WriteFile(fmt.Sprintf("%s.json", f.Name()), file, 0644)
	if err != nil {
		err = fmt.Errorf("downloadFile failed to write metadata: %w", err)
	}
	return
}

func prefetchCache(cacheDir, bucketName, prefix string) (err error) {
	start := time.Now()
	filesAttempted := 0
	var filesDownloaded int32
	var wg sync.WaitGroup
	var downloadTasks = make(chan *storage.ObjectAttrs)

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %w", err)
	}
	defer client.Close()

	// Should we set a higher timeout or let this be configurable by the user
	ctx, cancel := context.WithTimeout(ctx, time.Minute*10)
	defer cancel()

	it := client.Bucket(bucketName).Objects(ctx, &storage.Query{
		Prefix: prefix,
	})

	// Concurrently download files from specified gcs bucket with optional prefix

	// Producer
	go func() {
		for {
			attrs, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("Bucket(%q).Objects: %v", bucketName, err)
				break
			}
			filesAttempted++
			downloadTasks <- attrs
		}
		close(downloadTasks)
	}()

	// Consumers
	for i := 0; i < NUM_WORKERS; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for attrs := range downloadTasks {
				log.Printf("Worker %d: downloading file %v", i, attrs.Name)
				err := downloadFile(ctx, client, attrs, cacheDir)
				if err != nil {
					log.Printf("prefetchCache: %v", err)
				} else {
					atomic.AddInt32(&filesDownloaded, 1)
				}
			}
		}(i)
	}

	// Wait for all goroutines downloading files to finish
	wg.Wait()

	elapsed := time.Since(start)
	log.Printf("Prefetch cache took %s", elapsed)
	log.Printf("Number of files downloaded successfully %v", filesDownloaded)
	log.Printf("Number of files attempted to download %v", filesAttempted)

	return
}
