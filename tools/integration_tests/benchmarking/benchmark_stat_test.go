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
	"log"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/benchmark_setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const (
	expectedStatLatency time.Duration = 1100 * time.Millisecond
)

type benchmarkStatTest struct {
	flags []string
}

func (s *benchmarkStatTest) SetupB(b *testing.B) {
	mountGCSFuseAndSetupTestDir(s.flags, testEnv.ctx, testEnv.storageClient)
}

func (s *benchmarkStatTest) TeardownB(b *testing.B) {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
	setup.SaveGCSFuseLogFileInCaseOfFailure(b)
}

// createFilesToStat creates the below object in the bucket.
// benchmarking/a.txt
func createFilesToStat(b *testing.B) {
	operations.CreateFileOfSize(1, path.Join(testEnv.testDirPath, "a.txt"), b)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *benchmarkStatTest) Benchmark_Stat(b *testing.B) {
	createFilesToStat(b)
	var maxElapsedDuration time.Duration
	maxElapsedIteration := -1
	b.ResetTimer()
	// Don't start the timer yet.
	b.StopTimer()

	filePath := path.Join(testEnv.testDirPath, "a.txt")

	for i := range b.N {
		// Manually time the operation to find the maximum latency with highest accuracy.
		// This happens while the benchmark's timer is paused and will not affect the average.
		startTime := time.Now()

		// Start the benchmark timer just for the operations.StatFile call.
		b.StartTimer()
		_, err := operations.StatFile(filePath)
		b.StopTimer() // Stop the timer immediately after the operation.

		timeElapsedThisIter := time.Since(startTime)

		// The remaining checks and calculations also happen while the timer is paused.
		if err != nil {
			b.Errorf("failed to stat %q: %v", filePath, err)
		}

		if maxElapsedDuration < timeElapsedThisIter {
			maxElapsedDuration = timeElapsedThisIter
			maxElapsedIteration = i
		}
	}

	// b.Elapsed() is the sum of the time spent only on stat calls,
	// leading to a highly accurate average latency.
	averageStatLatency := b.Elapsed() / time.Duration(b.N)

	if averageStatLatency > expectedStatLatency {
		b.Errorf("StatFile took more time on average (%v) than expected (%v)", averageStatLatency, expectedStatLatency)
		b.Errorf("Maximum time taken by a single iteration = %v, in iteration # %v.", maxElapsedDuration, maxElapsedIteration)
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func Benchmark_Stat(b *testing.B) {
	setup.IgnoreTestIfPresubmitFlagIsSet(b)

	ts := &benchmarkStatTest{}
	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, b.Name())

	// Run tests.
	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		benchmark_setup.RunBenchmarks(b, ts)
	}
}
