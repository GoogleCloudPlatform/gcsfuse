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
	"time"

	"github.com/googlecloudplatform/gcsfuse/benchmarks/concurrent_read/readers"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/internal/perf"
)

type BenchmarkConfig struct {
	// The GCS bucket storing the objects to be read.
	Bucket string
	// The GCS objects as 'gs://...' to be read from the bucket above.
	Objects []string
	// Each job reads all the objects.
	Jobs []JobConfig
}

type JobConfig struct {
	// Choose from HTTP/1.1, HTTP/2, GRPC
	Protocol string
	// Max connections for this job
	Connections int
	// Choose from vendor, google.
	Implementation string
}

const (
	KB = 1024
	MB = 1024 * KB
)

func getJobs() []JobConfig {
	return []JobConfig{
		JobConfig{
			Protocol:       "HTTP/1.1",
			Connections:    50,
			Implementation: "vendor",
		},
		JobConfig{
			Protocol:       "HTTP/2",
			Connections:    50,
			Implementation: "vendor",
		},
		JobConfig{
			Protocol:       "HTTP/1.1",
			Connections:    50,
			Implementation: "google",
		},
		JobConfig{
			Protocol:       "HTTP/2",
			Connections:    50,
			Implementation: "google",
		},
	}
}

func testReader(ctx context.Context, client readers.Client, objectNames []string) (stats testStats) {
	reportDuration := 10 * time.Second
	ticker := time.NewTicker(reportDuration)
	defer ticker.Stop()

	doneBytes := make(chan int64)
	doneFiles := make(chan int)
	start := time.Now()

	// run readers concurrently
	for _, objectName := range objectNames {
		name := objectName
		go func() {
			region := trace.StartRegion(ctx, "NewReader")
			reader, err := client.NewReader(name)
			region.End()
			if err != nil {
				fmt.Printf("Skip %q: %s", name, err)
				return
			}
			defer reader.Close()

			p := make([]byte, 128*1024)
			region = trace.StartRegion(ctx, "ReadObject")
			for {
				n, err := reader.Read(p)

				doneBytes <- int64(n)
				if err == io.EOF {
					break
				} else if err != nil {
					panic(fmt.Errorf("read %q fails: %w", name, err))
				}
			}
			region.End()
			doneFiles <- 1
			return
		}()
	}

	// collect test stats
	var lastTotalBytes int64
	for stats.totalFiles < len(objectNames) {
		select {
		case b := <-doneBytes:
			stats.totalBytes += b
		case f := <-doneFiles:
			stats.totalFiles += f
		case <-ticker.C:
			readBytes := stats.totalBytes - lastTotalBytes
			lastTotalBytes = stats.totalBytes
			mbps := float32(readBytes/MB) / float32(reportDuration/time.Second)
			stats.mbps = append(stats.mbps, mbps)
		}
	}
	stats.duration = time.Since(start)
	return
}

func run(cfg BenchmarkConfig) {
	ctx := context.Background()

	ctx, traceTask := trace.NewTask(ctx, "ReadAllObjects")
	defer traceTask.End()

	for _, job := range cfg.Jobs {
		client, err := readers.NewClient(ctx, job.Protocol, job.Connections, job.Implementation, cfg.Bucket)
		if err != nil {
			fmt.Printf("Cannot create client for job: %v", job)
			continue
		}
		stats := testReader(ctx, client, cfg.Objects)
		stats.report(job)
	}
}

type testStats struct {
	totalBytes int64
	totalFiles int
	mbps       []float32
	duration   time.Duration
}

func (s testStats) throughput() float32 {
	mbs := float32(s.totalBytes) / float32(MB)
	seconds := float32(s.duration) / float32(time.Second)
	return mbs / seconds
}

func (s testStats) report(job JobConfig) {
	logger.Infof(
		"# TEST READER %s\n"+
			"Protocol: %s (%v connections per host)\n"+
			"Total bytes: %d\n"+
			"Total files: %d\n"+
			"Avg Throughput: %.1f MB/s\n\n",
		job.Protocol,
		job.Implementation,
		job.Connections,
		s.totalBytes,
		s.totalFiles,
		s.throughput(),
	)
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
