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
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/benchmark_setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const (
	expectedDeleteLatency time.Duration = 675 * time.Millisecond
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
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := os.Remove(path.Join(testDirPath, fmt.Sprintf("a%d.txt", i))); err != nil {
			b.Errorf("testing error: %v", err)
		}
	}
	averageDeleteLatency := time.Duration(int(b.Elapsed()) / b.N)
	if averageDeleteLatency > expectedDeleteLatency {
		b.Errorf("DeleteFile took more time (%d msec) than expected (%d msec)", averageDeleteLatency.Milliseconds(), expectedDeleteLatency.Milliseconds())
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func Benchmark_Delete(b *testing.B) {
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
