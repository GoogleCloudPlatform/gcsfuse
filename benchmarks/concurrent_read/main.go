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
// 	 gsutil ls 'gs://bucket/prefix*' | go run \
//      --http=1x50 --reader=official ./benchmark/concurrent_read
//

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

var fHTTP = flag.String(
	"http",
	"1x100",
	"Protocol and connections per host: 1x10, 1x50, 1x100, 2",
)
var fReader = flag.String(
	"reader",
	"vendor",
	"Reader type: vendor, official.",
)

const (
	KB = 1024
	MB = 1024 * KB
)

func testReader(
	httpVersion string,
	readerVersion string,
	bucketName string,
	objectNames []string) (stats testStats) {
	reportDuration := 10 * time.Second
	ticker := time.NewTicker(reportDuration)
	defer ticker.Stop()

	doneBytes := make(chan int64)
	doneFiles := make(chan int)
	start := time.Now()

	// run readers concurrently
	transport := getTransport(httpVersion)
	defer transport.CloseIdleConnections()
	rf := newReaderFactory(transport, readerVersion, bucketName)
	for _, objectName := range objectNames {
		name := objectName
		go func() {
			reader := rf.NewReader(name)
			defer reader.Close()
			p := make([]byte, 128*1024)
			for {
				n, err := reader.Read(p)
				doneBytes <- int64(n)
				if err == io.EOF {
					break
				} else if err != nil {
					panic(fmt.Errorf("read %v fails: %v", name, err))
				}
			}
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
	stats.report(httpVersion, readerVersion)
	return
}

func run(bucketName string, objectNames []string) {
	protocols := map[string]string{
		"1x10":  http1x10,
		"1x50":  http1x50,
		"1x100": http1x100,
		"2":     http2,
	}
	readers := map[string]string{
		"vendor":   vendorClientReader,
		"official": officialClientReader,
	}
	testReader(
		protocols[*fHTTP],
		readers[*fReader],
		bucketName,
		objectNames,
	)
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

func (s testStats) report(
	httpVersion string,
	readerVersion string,
) {
	fmt.Printf(
		"# TEST READER %s\n"+
			"Protocol: %s\n"+
			"Total bytes: %d\n"+
			"Total files: %d\n"+
			"Avg Throughput: %.1f\n MB/s",
		readerVersion,
		httpVersion,
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
			panic(fmt.Errorf("Stdin error: %v", err))
		}
		lines = append(lines, line)
	}
	return
}

func getObjectNames() (bucketName string, objectNames []string) {
	uris := getLinesFromStdin()
	for _, uri := range uris {
		path := strings.TrimLeft(uri, "gs://")
		path = strings.TrimRight(path, "\n")
		segs := strings.Split(path, "/")
		if len(segs) <= 1 {
			panic(fmt.Errorf("Not a file name: %v", uri))
		}

		if bucketName == "" {
			bucketName = segs[0]
		} else if bucketName != segs[0] {
			panic(fmt.Errorf("Multiple buckets: %v, %v", bucketName, segs[0]))
		}

		objectName := strings.Join(segs[1:], "/")
		objectNames = append(objectNames, objectName)
	}
	return
}

func main() {
	flag.Parse()
	bucketName, objectNames := getObjectNames()
	run(bucketName, objectNames)
	return
}
