// Copyright 2020 Google Inc. All Rights Reserved.
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

// Concurrently read objects on GCS provided by stdin. The user must ensure
//    (1) all the objects come from the same bucket, and
//    (2) the script is authorized to read from the bucket.
// The stdin should contain N lines of object name, in the form of
// "gs://bucket-name/object-name".
//
// This benchmark only tests the internal reader implementation, which
// doesn't have FUSE involved.
//
// Usage Example:
// 	 gsutil ls 'gs://bucket/prefix*' | go run ./benchmarks/concurrent_read/
//

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime/trace"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/benchmarks/concurrent_read/job"
	"github.com/googlecloudplatform/gcsfuse/internal/perf"
)

type BenchmarkConfig struct {
	// The GCS bucket storing the objects to be read.
	Bucket string
	// The GCS objects as 'gs://...' to be read from the bucket above.
	Objects []string
	// Each job reads all the objects.
	Jobs []*job.Job
}

func getJobs() []*job.Job {
	return []*job.Job{
		&job.Job{
			Protocol:       "HTTP/1.1",
			Connections:    50,
			Implementation: "vendor",
		},
		&job.Job{
			Protocol:       "HTTP/2",
			Connections:    50,
			Implementation: "vendor",
		},
		&job.Job{
			Protocol:       "HTTP/1.1",
			Connections:    50,
			Implementation: "google",
		},
		&job.Job{
			Protocol:       "HTTP/2",
			Connections:    50,
			Implementation: "google",
		},
	}
}

func run(cfg BenchmarkConfig) {
	ctx := context.Background()

	ctx, traceTask := trace.NewTask(ctx, "ReadAllObjects")
	defer traceTask.End()

	for _, job := range cfg.Jobs {
		stats, err := job.Run(ctx, cfg.Bucket, cfg.Objects)
		if err != nil {
			fmt.Printf("Job failed: %v", job)
			continue
		}
		stats.Report(job)
	}
}

func getLinesFromStdin() (lines []string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			panic(fmt.Errorf("Stdin error: %w", err))
		}
		lines = append(lines, line)
	}
	return
}

func getObjectNames() (bucketName string, objectNames []string) {
	uris := getLinesFromStdin()
	for _, uri := range uris {
		path := uri[len("gs://"):]
		path = strings.TrimRight(path, "\n")
		segs := strings.Split(path, "/")
		if len(segs) <= 1 {
			panic(fmt.Errorf("Not a file name: %q", uri))
		}

		if bucketName == "" {
			bucketName = segs[0]
		} else if bucketName != segs[0] {
			panic(fmt.Errorf("Multiple buckets: %q, %q", bucketName, segs[0]))
		}

		objectName := strings.Join(segs[1:], "/")
		objectNames = append(objectNames, objectName)
	}
	return
}

func main() {
	flag.Parse()

	go perf.HandleCPUProfileSignals()

	// Enable trace
	f, err := os.Create("/tmp/concurrent_read_trace.out")
	if err != nil {
		log.Fatalf("failed to create trace output file: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatalf("failed to close trace file: %v", err)
		}
	}()
	if err := trace.Start(f); err != nil {
		log.Fatalf("failed to start trace: %v", err)
	}
	defer trace.Stop()

	bucketName, objectNames := getObjectNames()
	config := BenchmarkConfig{
		Bucket:  bucketName,
		Objects: objectNames,
		Jobs:    getJobs(),
	}
	run(config)

	return
}
