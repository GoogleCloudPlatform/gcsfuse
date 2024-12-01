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
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/benchmark_setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const (
	expectedStatLatency time.Duration = 260 * time.Millisecond
)

type benchmarkStatTest struct{}

func (s *benchmarkStatTest) SetupB(b *testing.B) {
	testDirPath = setup.SetupTestDirectory(testDirName)
}

func (s *benchmarkStatTest) TeardownB(b *testing.B) {}

// createFilesToStat creates the below object in the bucket.
// benchmarking/a.txt
func createFilesToStat(b *testing.B) {
	operations.CreateFileOfSize(5, path.Join(testDirPath, "a.txt"), b)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *benchmarkStatTest) Benchmark_Stat(b *testing.B) {
	createFilesToStat(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := operations.StatFile(path.Join(testDirPath, "a.txt")); err != nil {
			b.Errorf("testing error: %v", err)
		}
	}
	averageStatLatency := time.Duration(int(b.Elapsed()) / b.N)
	if averageStatLatency > expectedStatLatency {
		b.Errorf("StatFile took more time (%d msec) than expected (%d msec)", averageStatLatency.Milliseconds(), expectedStatLatency.Milliseconds())
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func Benchmark_Stat(b *testing.B) {
	ts := &benchmarkStatTest{}
	benchmark_setup.RunBenchmarks(b, ts)
}
