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
	"fmt"
	"io"
	"math"
	"os"
	"path"

	"github.com/bitfield/script"
	flag "github.com/spf13/pflag"
)

var filePath *string = flag.String("mount-path", "file:///dev/shm/multireader", "Path to the mountpoint along with protocol e.g. file://dev/shm/multireader")

type multiReadConfig struct {
	fileIOConcurrency   int64
	maxInflightRequests int64
	numConfig           int64
	path                string
}

func tscliConfig(config *multiReadConfig) (string, error) {
	output, err := script.Exec(fmt.Sprintf("bazel-bin/tensorstore/tscli/tscli search -f \"%s\"", config.path)).Filter(
		func(r io.Reader, w io.Writer) error {
			scanner := newScanner(r)
			first := true
			for scanner.Scan() {
				if !first {
					fmt.Fprint(w, ", ")
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
	f, err := os.CreateTemp("", "tscli_config.json")
	if err != nil {
		return "", err
	}
	f.Write([]byte(fmt.Sprintf("[%s]", output)))
	fName := f.Name()
	f.Close()
	return fName, nil
}

func multiReadBenchmarkSetup(wd string, config *multiReadConfig) (string, error) {
	cd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if err := os.Chdir(wd); err != nil {
		defer func() { _ = os.Chdir(cd) }()
	}
	cfgPath, err := tscliConfig(config)
	if err != nil {
		return "", err
	}
	return cfgPath, nil

	/*
		bazel-bin/tensorstore/tscli/tscli search -f "file://<mount_point>" | sed '$!s/$/,/' | sed '1s/^/[\n/'  | sed -e '$a]' > output.json
		echo "echo 3 > /proc/sys/vm/drop_caches" | sudo sh && bazel-bin/tensorstore/internal/benchmark/multi_read_benchmark --read_config=output.json

	*/
}

func multiReadBenchmark(checkoutDir string, config *multiReadConfig) error {
	tscliConfig, err := multiReadBenchmarkSetup(checkoutDir, config)
	if err != nil {
		return err
	}
	/*
		cd, err := os.Getwd()
		if err != nil {
			return err
		}
		if err := os.Chdir(checkoutDir); err != nil {
			return err
		}
		defer func() { os.Chdir(cd) }()
	*/
	if _, err := script.Echo("3").AppendFile("/proc/sys/vm/drop_caches"); err != nil {
		return fmt.Errorf("unable to clear page cache: %w", err)
	}
	benchmarkOutput, err := script.Exec(fmt.Sprintf("%s --read_config=%s", path.Join(checkoutDir, "bazel-bin/tensorstore/internal/benchmark/multi_read_benchmark"), tscliConfig)).String()
	if err := os.Remove(tscliConfig); err != nil {
		return err
	}
	fmt.Println(benchmarkOutput)
	if err != nil {
		return err
	}
	return nil
}
func newScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), math.MaxInt)
	return scanner
}

func setup() (string, error) {
	/*
			gitDep := script.Exec("apt install git")
			compilerDep := script.Exec("apt install python3.10-dev g++")
			if err = gitDep.Wait(); err != nil {
				return "", fmt.Errorf("error while installing git, stderr:%s: %w", gitDep.Error(), err)
			}
		if err = compilerDep.Wait(); err != nil {
				return "", err
			}
	*/
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
	build := script.Exec("./bazelisk.py build //tensorstore/internal/benchmark:all //tensorstore/tscli")
	if err := build.Wait(); err != nil {
		return "", fmt.Errorf("error occurred while building benchmarks")
	}
	fmt.Println("Benchmark built successfully")
	return tempDir, nil
}

func main() {
	checkoutDir, err := setup()
	defer func() { os.RemoveAll(checkoutDir) }()
	if err != nil {
		panic(err)
	}
	if err = multiReadBenchmark(checkoutDir, &multiReadConfig{
		fileIOConcurrency:   -1,
		maxInflightRequests: -1,
		numConfig:           -1,
		path:                *filePath,
	}); err != nil {
		panic(err)
	}

}
