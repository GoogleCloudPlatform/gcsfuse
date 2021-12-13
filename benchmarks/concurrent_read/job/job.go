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

package job

import (
	"context"
	"fmt"
	"io"
	"runtime/trace"
	"time"

	"github.com/googlecloudplatform/gcsfuse/benchmarks/concurrent_read/readers"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
)

const (
	KB = 1024
	MB = 1024 * KB
)

type Job struct {
	// Choose from HTTP/1.1, HTTP/2, GRPC
	Protocol string
	// Max connections for this job
	Connections int
	// Choose from vendor, google.
	Implementation string
}

type Stats struct {
	Job        *Job
	TotalBytes int64
	TotalFiles int
	Mbps       []float32
	Duration   time.Duration
}

func (s *Stats) Throughput() float32 {
	mbs := float32(s.TotalBytes) / float32(MB)
	seconds := float32(s.Duration) / float32(time.Second)
	return mbs / seconds
}

func (s *Stats) Report() {
	logger.Infof(
		"# TEST READER %s\n"+
			"Protocol: %s (%v connections per host)\n"+
			"Total bytes: %d\n"+
			"Total files: %d\n"+
			"Avg Throughput: %.1f MB/s\n\n",
		s.Job.Protocol,
		s.Job.Implementation,
		s.Job.Connections,
		s.TotalBytes,
		s.TotalFiles,
		s.Throughput(),
	)
}

func (s *Stats) Query(key string) string {
	switch key {
	case "Protocol":
		return s.Job.Protocol
	case "Implementation":
		return s.Job.Implementation
	case "Connections":
		return fmt.Sprintf("%d", s.Job.Connections)
	case "TotalBytes (MB)":
		return fmt.Sprintf("%d", s.TotalBytes/MB)
	case "TotalFiles":
		return fmt.Sprintf("%d", s.TotalFiles)
	case "Throughput (MB/s)":
		return fmt.Sprintf("%.1f", s.Throughput())
	default:
		return ""
	}
}

type Client interface {
	NewReader(objectName string) (io.ReadCloser, error)
}

func (job *Job) Run(ctx context.Context, bucketName string, objects []string) (*Stats, error) {
	var client Client
	var err error

	switch job.Implementation {
	case "vendor":
		client, err = readers.NewVendorClient(ctx, job.Protocol, job.Connections, bucketName)
	case "google":
		client, err = readers.NewGoogleClient(ctx, job.Protocol, job.Connections, bucketName)
	default:
		panic(fmt.Errorf("Unknown reader implementation: %q", job.Implementation))
	}

	if err != nil {
		return nil, err
	}
	stats := job.testReader(ctx, client, objects)
	return stats, nil
}

func (job *Job) testReader(ctx context.Context, client Client, objectNames []string) *Stats {
	stats := &Stats{Job: job}
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
	for stats.TotalFiles < len(objectNames) {
		select {
		case b := <-doneBytes:
			stats.TotalBytes += b
		case f := <-doneFiles:
			stats.TotalFiles += f
		case <-ticker.C:
			readBytes := stats.TotalBytes - lastTotalBytes
			lastTotalBytes = stats.TotalBytes
			mbps := float32(readBytes/MB) / float32(reportDuration/time.Second)
			stats.Mbps = append(stats.Mbps, mbps)
		}
	}
	stats.Duration = time.Since(start)
	return stats
}
