// Copyright 2024 Google LLC
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
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"regexp"
	"strconv"

	"github.com/bitfield/script"
	flag "github.com/spf13/pflag"
)

const (
	multiread = "multiread"
	kvStore   = "kv_store"
)

var filePath = flag.String("mount-path", "file:///dev/shm/multiread", "Path to the mountpoint along with protocol e.g. file://dev/shm/multireader")
var resultsPath = flag.String("output-path", "/tmp/output.csv", "Results will be dumped here")
var benchmark = flag.String("benchmark", multiread, "Benchmark to run. Set to one of multiread or kv_store")

var multiReadThroughputRegex = regexp.MustCompile(`throughput:\s+(.+)\s+MB/second`)
var kvStoreThroughputRegex = regexp.MustCompile(`.*bytes in .*\s+(.+)\s+MB/second`)

type multiReadConfig struct {
	fileIOConcurrency   int64
	maxInflightRequests int64
	path                string
	tscliConfigPath     string
}

type kvStoreConfig struct {
	fileIOConcurrency int64
	path              string
}

func tscliConfig(checkoutDir string, pth string) (string, error) {
	cmd := fmt.Sprintf("%s search -f '%s'", path.Join(checkoutDir, "bazel-bin/tensorstore/tscli/tscli"), pth)
	fmt.Println(cmd)
	output, err := script.Exec(cmd).Filter(
		func(r io.Reader, w io.Writer) error {
			scanner := newScanner(r)
			first := true
			for scanner.Scan() {
				if !first {
					fmt.Fprint(w, ",")
				}
				line := scanner.Text()
				fmt.Fprint(w, line)
				first = false
			}
			fmt.Fprintln(w)
			return scanner.Err()
		}).String()
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp("", "tscli_config")
	if err != nil {
		return "", err
	}
	if _, err := f.Write([]byte(fmt.Sprintf("[%s]", output))); err != nil {
		return "", err
	}
	fName := f.Name()
	if err := f.Close(); err != nil {
		return "", err
	}
	return fName, nil
}

func newScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), math.MaxInt)
	return scanner
}

func setup() (string, error) {
	wd, err := os.Getwd()
	defer func() { os.Chdir(wd) }()
	if err != nil {
		return "", err
	}
	tempDir, err := os.MkdirTemp("", "tensorstore")
	if err != nil {
		return "", err
	}
	fmt.Println("Cloning tensorstore")
	clone := script.Exec(fmt.Sprintf("git clone https://github.com/google/tensorstore.git %s/", tempDir))
	if err = clone.Wait(); err != nil {
		return "", fmt.Errorf("error occurred while cloning repository:%w", err)
	}
	fmt.Println("Cloning tensorstore done")
	os.Chdir(tempDir)
	fmt.Println("Building benchmark targets")
	build := script.Exec("./bazelisk.py build -c opt --copt=-O3 //tensorstore/internal/benchmark:all //tensorstore/tscli")
	if err := build.Wait(); err != nil {
		return "", fmt.Errorf("error occurred while building benchmarks")
	}
	fmt.Println("Benchmark built successfully")
	return tempDir, nil
}

func validate() error {
	_, err := os.Stat(*resultsPath)
	if err == nil {
		return fmt.Errorf("output file already exists: %s", *resultsPath)
	}
	if *benchmark != "multiread" && *benchmark != "kv_store" {
		return fmt.Errorf("unknown benchmark: %s", *benchmark)
	}
	return nil
}

func multiReadBenchmark(checkoutDir string, config *multiReadConfig, _ int64) (string, error) {
	if _, err := script.Echo("3").AppendFile("/proc/sys/vm/drop_caches"); err != nil {
		return "", fmt.Errorf("unable to clear page cache: %w", err)
	}
	cmd := fmt.Sprintf(`%s --read_config=%s --max_in_flight=%d --context_spec='{"file_io_concurrency": {"limit": %d}, "cache_pool": {"total_bytes_limit": 0}}'`,
		path.Join(checkoutDir, "bazel-bin/tensorstore/internal/benchmark/multi_read_benchmark"), config.tscliConfigPath, config.maxInflightRequests, config.fileIOConcurrency)
	fmt.Println(cmd)
	return script.Exec(cmd).String()
}

func kvStoreBenchmark(checkoutDir string, config *kvStoreConfig) (string, error) {
	if _, err := script.Echo("3").AppendFile("/proc/sys/vm/drop_caches"); err != nil {
		return "", fmt.Errorf("unable to clear page cache: %w", err)
	}
	cmd := fmt.Sprintf(`%s --kvstore_spec=\"%s\" --context_spec='{"file_io_concurrency": {"limit": %d}, "cache_pool": {"total_bytes_limit": 0}}'`,
		path.Join(checkoutDir, "bazel-bin/tensorstore/internal/benchmark/kvstore_benchmark"), config.path, config.fileIOConcurrency)
	fmt.Println(cmd)
	return script.Exec(cmd).String()

}

func invokeKVStoreBenchmark(checkoutDir string) {
	f, err := os.Create(*resultsPath)
	if err != nil {
		panic(err)
	}
	writer := csv.NewWriter(f)
	defer func() {
		writer.Flush()
		f.Close()
	}()
	record := []string{"file_io_concurrency", "Round", "Throughput MB/s"}
	writer.Write(record)
	idx := int64(0)
	fileIOConcurrencyRange := []int64{256, 128, 64, 32}

	for _, ioConc := range fileIOConcurrencyRange {
		for _, round := range []int64{1} {
			output, err := kvStoreBenchmark(checkoutDir, &kvStoreConfig{
				fileIOConcurrency: ioConc,
				path:              *filePath,
			})
			idx++

			if err != nil {
				panic(err)
			}
			fmt.Println(output)
			bw, err := extractBWFromKVStoreBenchmark(output)
			if err != nil {
				panic(err)
			}
			writer.Write([]string{strconv.FormatInt(ioConc, 10), strconv.FormatInt(round, 10), strconv.FormatInt(int64(bw), 10)})
			fmt.Printf("Bandwidth obtained: %d MB/s\n", int64(bw))
		}
	}
}

func invokeMultiReadBenchmark(checkoutDir string) {
	f, err := os.Create(*resultsPath)
	if err != nil {
		panic(err)
	}
	writer := csv.NewWriter(f)
	defer func() {
		writer.Flush()
		f.Close()
	}()
	record := []string{"file_io_concurrency", "max_inflight_requests", "Round", "Throughput MB/s"}
	writer.Write(record)
	tscliConfigPath, err := tscliConfig(checkoutDir, *filePath)
	if err != nil {
		panic(err)
	}
	defer func() { os.RemoveAll(tscliConfigPath) }()
	idx := int64(0)
	fileIOConcurrencyRange := []int64{256, 128, 64, 32}
	maxInflightRequestMultiplicand := []int64{16, 12, 10, 8, 4, 2, 1}

	for _, ioConc := range fileIOConcurrencyRange {
		for _, inflightMaxMulti := range maxInflightRequestMultiplicand {
			for _, round := range []int64{1} {
				output, err := multiReadBenchmark(checkoutDir, &multiReadConfig{
					fileIOConcurrency:   ioConc,
					maxInflightRequests: int64(64) * 1024 * 1024 * 1024 * inflightMaxMulti,
					path:                *filePath,
					tscliConfigPath:     tscliConfigPath,
				},
					idx)
				idx++

				if err != nil {
					panic(err)
				}
				fmt.Println(output)
				bw, err := extractBWFromMutiReadBenchmark(output)
				if err != nil {
					panic(err)
				}
				writer.Write([]string{strconv.FormatInt(ioConc, 10), strconv.FormatInt(inflightMaxMulti*64, 10), strconv.FormatInt(round, 10), strconv.FormatInt(int64(bw), 10)})
				fmt.Printf("Bandwidth obtained: %d MB/s\n", int64(bw))
			}
		}
	}
}

func main() {
	flag.Parse()
	if err := validate(); err != nil {
		panic(err)
	}
	checkoutDir, err := setup()
	defer func() { os.RemoveAll(checkoutDir) }()
	if err != nil {
		panic(err)
	}

	if *benchmark == multiread {
		invokeMultiReadBenchmark(checkoutDir)
	} else {
		invokeKVStoreBenchmark(checkoutDir)
	}
}

func extractBWFromKVStoreBenchmark(output string) (float64, error) {
	subMatches := kvStoreThroughputRegex.FindSubmatch([]byte(output))
	if len(subMatches) != 2 {
		return 0, fmt.Errorf("unable to parse KVStore output")
	}
	return strconv.ParseFloat(string(subMatches[1]), 64)
}

func extractBWFromMutiReadBenchmark(output string) (float64, error) {
	subMatches := multiReadThroughputRegex.FindSubmatch([]byte(output))
	if len(subMatches) != 2 {
		return 0, fmt.Errorf("unable to parse multi-read-benchmark output")
	}
	return strconv.ParseFloat(string(subMatches[1]), 64)
}
