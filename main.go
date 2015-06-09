// Copyright 2015 Google Inc. All Rights Reserved.
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
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	"github.com/jacobsa/gcloud/syncutil"
)

const tmpObjectPrefix = ".gcsfuse_tmp/"

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func registerSIGINTHandler(mountPoint string) {
	// Register for SIGINT.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	// Start a goroutine that will unmount when the signal is received.
	go func() {
		for {
			<-signalChan
			log.Println("Received SIGINT, attempting to unmount...")

			err := fuse.Unmount(mountPoint)
			if err != nil {
				log.Printf("Failed to unmount in response to SIGINT: %v", err)
			} else {
				log.Printf("Successfully unmounted in response to SIGINT.")
				return
			}
		}
	}()
}

func getConn() (c gcs.Conn, err error) {
	// Create the authenticated HTTP client.
	const scope = gcs.Scope_FullControl
	httpClient, err := google.DefaultClient(context.Background(), scope)
	if err != nil {
		return nil, err
	}

	// Create the connection.
	const userAgent = "gcsfuse/0.0"
	cfg := &gcs.ConnConfig{
		HTTPClient: httpClient,
		UserAgent:  userAgent,
	}

	return gcs.NewConn(cfg)
}

func garbageCollectOnce(
	ctx context.Context,
	bucket gcs.Bucket) (objectsDeleted uint64, err error) {
	const stalenessThreshold = 30 * time.Minute
	b := syncutil.NewBundle(ctx)

	// List all objects with the temporary prefix.
	objects := make(chan *gcs.Object, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(objects)
		err = gcsutil.ListPrefix(ctx, bucket, tmpObjectPrefix, objects)
		if err != nil {
			err = fmt.Errorf("ListPrefix: %v", err)
			return
		}

		return
	})

	// Filter to the names of objects that are stale.
	now := time.Now()
	staleNames := make(chan string, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(staleNames)
		for o := range objects {
			if now.Sub(o.Updated) < stalenessThreshold {
				continue
			}

			select {
			case <-ctx.Done():
				err = ctx.Err()
				return

			case staleNames <- o.Name:
			}
		}

		return
	})

	// Delete those objects.
	b.Add(func(ctx context.Context) (err error) {
		for name := range staleNames {
			err = bucket.DeleteObject(ctx, name)
			if err != nil {
				err = fmt.Errorf("DeleteObject(%q): %v", name, err)
				return
			}

			atomic.AddUint64(&objectsDeleted, 1)
		}

		return
	})

	err = b.Join()
	return
}

// Periodically delete stale temporary objects from the supplied bucket.
func garbageCollect(
	ctx context.Context,
	bucket gcs.Bucket) {
	const period = 10 * time.Minute
	for _ = range time.Tick(period) {
		log.Println("Starting a garbage collection run.")

		startTime := time.Now()
		objectsDeleted, err := garbageCollectOnce(ctx, bucket)

		if err != nil {
			log.Printf(
				"Garbage collection failed after deleting %d objects in %v, "+
					"with error: %v",
				objectsDeleted,
				time.Since(startTime),
				err)
		} else {
			log.Printf(
				"Garbage collection succeeded after deleted %d objects in %v.",
				objectsDeleted,
				time.Since(startTime))
		}
	}
}

////////////////////////////////////////////////////////////////////////
// main function
////////////////////////////////////////////////////////////////////////

func main() {
	// Make logging output better.
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Grab the connection.
	conn, err := getConn()
	if err != nil {
		log.Fatalf("getConn: %v", err)
	}

	// Mount the file system.
	mfs, err := mount(
		os.Args[1:],
		flag.CommandLine,
		conn)

	if err != nil {
		log.Fatalf("mount: %v", err)
	}

	log.Println("File system has been successfully mounted.")

	// Let the user unmount with Ctrl-C (SIGINT).
	registerSIGINTHandler(mfs.Dir())

	// Wait for the file system to be unmounted.
	err = mfs.Join(context.Background())
	if err != nil {
		err = fmt.Errorf("MountedFileSystem.Join: %v", err)
		return
	}

	log.Println("Successfully exiting.")
}
