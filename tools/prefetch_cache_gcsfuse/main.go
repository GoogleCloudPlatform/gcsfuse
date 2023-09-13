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

// Prefetches files into local disk cache from specified bucket
//
// Usage:
//
//     prefetch_cache_gcsfuse cache_dir bucket_name [prefix]
//
// This will prefetch the cache files from the specified bucket
// with an optional file prefix to filter the GCS objects
// and download them into the specified cache directory
//
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func run(args []string) (err error) {
	// Extract arguments.
	if len(args) < 2 || len(args) > 3 {
		err = fmt.Errorf("Usage: %s cache_dir bucket_name [prefix]", os.Args[0])
		return
	}

	cacheDir := args[0]
	bucketName := args[1]
	var prefix string

	if len(args) > 2 {
		prefix = args[2]
	}

	log.Printf("Using settings:")
	log.Printf("  cacheDir:  %s", cacheDir)
	log.Printf("  bucketName:  %s", bucketName)
	log.Printf("  prefix:  %s", prefix)

	// Try to preload persistent cache on disk using specified parameters
	err = prefetchCache(cacheDir, bucketName, prefix)
	if err != nil {
		err = fmt.Errorf("prefetch_cache_gcsfuse: %w", err)
	}

	return
}

func main() {
	log.SetFlags(log.Lmicroseconds)
	flag.Parse()

	err := run(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
