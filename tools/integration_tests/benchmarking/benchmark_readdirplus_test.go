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

package benchmarking

import (
	"fmt"
	"log"
	"os/exec"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/benchmark_setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

const (
	// The expected latency for readdir is not fixed, and it may vary based on the
	// number of files in the directory.
	expectedReadDirPlusLatency  time.Duration = 20000 * time.Millisecond
	numFilesInDirForReadDirPlus               = 20000
)

type benchmarkReadDirPlusTest struct {
	flags         []string
	numTestFiles  int
	resultLatency time.Duration
}

func (s *benchmarkReadDirPlusTest) SetupB(b *testing.B) {
	mountGCSFuseAndSetupTestDir(s.flags, testDirName)
	// Create files required for benchmark tests.
	s.createFilesForReadDirPlus(b)
}

func (s *benchmarkReadDirPlusTest) TeardownB(b *testing.B) {
	setup.UnmountGCSFuse(rootDir)
}

func (s *benchmarkReadDirPlusTest) createFilesForReadDirPlus(b *testing.B) {
	for i := 0; i < s.numTestFiles; i++ {
		operations.CreateFileOfSize(1, path.Join(testDirPath, fmt.Sprintf("file%d.txt", i)), b)
	}
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *benchmarkReadDirPlusTest) Benchmark_ReadDirPlus(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("ls", "-l", testDirPath)
		if err := cmd.Run(); err != nil {
			b.Errorf("testing error: %v", err)
		}
	}

	//logContents, err := os.ReadFile(setup.LogFile())
	//if err != nil {
	//	b.Fatalf("Failed to read log file: %v", err)
	//}
	//
	//if !strings.Contains(string(logContents), "ReadDirPlus") {
	//	b.Error("Expected to find 'ReadDirPlus' in logs, but it was not present.")
	//}
	averageReadDirPlusLatency := time.Duration(int(b.Elapsed()) / b.N)
	s.resultLatency = averageReadDirPlusLatency
	if averageReadDirPlusLatency > expectedReadDirPlusLatency {
		b.Errorf("ReadDirPlus took more time (%d msec) than expected (%d msec)", averageReadDirPlusLatency.Milliseconds(), expectedReadDirPlusLatency.Milliseconds())
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func Benchmark_ReadDirPlus(b *testing.B) {
	setup.IgnoreTestIfPresubmitFlagIsSet(b)

	ts := &benchmarkReadDirPlusTest{
		numTestFiles: numFilesInDirForReadDirPlus,
	}

	flagsSet := [][]string{
		{"--enable-readdirplus=true"}, {"--metadata-cache-ttl-secs=36000"},
	}

	var timeWithoutReaddirplus time.Duration
	var timeWithReaddirplus time.Duration
	// Run tests.
	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		benchmark_setup.RunBenchmarks(b, ts)

		// Determine which scenario just ran and store its latency.
		isReadDirPlus := false
		for _, flag := range ts.flags {
			if flag == "--enable-readdirplus=true" {
				isReadDirPlus = true
			}
		}
		if isReadDirPlus {
			timeWithReaddirplus = ts.resultLatency
		} else {
			timeWithoutReaddirplus = ts.resultLatency
		}
	}

	if timeWithReaddirplus > timeWithoutReaddirplus {
		b.Errorf("With ReadDirPlus (%s) is slower than Without Readdirplus (%s)", timeWithReaddirplus, timeWithoutReaddirplus)
	}
}
