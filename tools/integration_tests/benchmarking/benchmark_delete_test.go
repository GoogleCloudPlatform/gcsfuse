// Copyright 2025 Google LLC
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
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/benchmark_setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

const (
	expectedDeleteLatency time.Duration = 1800 * time.Millisecond
)

type benchmarkDeleteTest struct {
	flags []string
}

func (s *benchmarkDeleteTest) SetupB(b *testing.B) {
	mountGCSFuseAndSetupTestDir(s.flags, testDirName)
}

func (s *benchmarkDeleteTest) TeardownB(b *testing.B) {
	setup.UnmountGCSFuse(rootDir)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *benchmarkDeleteTest) Benchmark_Delete(b *testing.B) {
	createFiles(b)
	var totalTimeElapsedTillPrevIter time.Duration
	var maxElapsedDuration time.Duration
	maxElapsedIteration := -1
	b.ResetTimer()
	for i := range b.N {
		filePath := path.Join(testDirPath, fmt.Sprintf("a%d.txt", i))
		if err := os.Remove(filePath); err != nil {
			b.Errorf("error while deleting %q: %v", filePath, err)
		}

		// Update maxElapsedIteration and totalTimeElapsedTillPrevIter.
		totalTimeElapsedSoFar := b.Elapsed()
		timeElapsedThisIter := totalTimeElapsedSoFar - totalTimeElapsedTillPrevIter
		if maxElapsedDuration < timeElapsedThisIter {
			maxElapsedDuration = timeElapsedThisIter
			maxElapsedIteration = i
		}
		totalTimeElapsedTillPrevIter = totalTimeElapsedSoFar
	}
	averageDeleteLatency := time.Duration(int(b.Elapsed()) / b.N)
	if averageDeleteLatency > expectedDeleteLatency {
		b.Errorf("DeleteFile took more time on average (%v) than expected (%v).", averageDeleteLatency, expectedDeleteLatency)
		b.Errorf("Maximum time taken by a single iteration = %v, in iteration # %v.", maxElapsedDuration, maxElapsedIteration)
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func Benchmark_Delete(b *testing.B) {
	setup.IgnoreTestIfPresubmitFlagIsSet(b)

	ts := &benchmarkDeleteTest{}

	flagsSet := [][]string{
		{"--stat-cache-ttl=0"}, {"--client-protocol=grpc", "--stat-cache-ttl=0"},
	}

	// Run tests.
	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		benchmark_setup.RunBenchmarks(b, ts)
	}
}
