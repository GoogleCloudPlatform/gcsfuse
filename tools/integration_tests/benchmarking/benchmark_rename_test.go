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

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/benchmark_setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

type benchmarkRenameTest struct{}

const (
	expectedRenameLatency time.Duration = 700 * time.Millisecond
)

func (s *benchmarkRenameTest) SetupB(b *testing.B) {
	testDirPath = setup.SetupTestDirectory(testDirName)
}

func (s *benchmarkRenameTest) TeardownB(b *testing.B) {}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *benchmarkRenameTest) Benchmark_Rename(b *testing.B) {
	createFiles(b)
	b.ResetTimer()
	log.Println("N: ", b.N)
	for i := 0; i < b.N; i++ {
		if err := os.Rename(path.Join(testDirPath, fmt.Sprintf("a%d.txt", i)), path.Join(testDirPath, fmt.Sprintf("b%d.txt", i))); err != nil {
			b.Errorf("testing error: %v", err)
		}
	}
	averageRenameLatency := time.Duration(int(b.Elapsed()) / b.N)
	if averageRenameLatency > expectedRenameLatency {
		b.Errorf("RenameFile took more time (%d msec) than expected (%d msec)", averageRenameLatency.Milliseconds(), expectedRenameLatency.Milliseconds())
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func Benchmark_Rename(b *testing.B) {
	ts := &benchmarkRenameTest{}
	benchmark_setup.RunBenchmarks(b, ts)
}
