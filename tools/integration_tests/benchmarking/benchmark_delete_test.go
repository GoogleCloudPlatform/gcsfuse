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
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/benchmark_setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const (
	deleteThreshold int = 450
)

type benchmarkDeleteTest struct{}

func (s *benchmarkDeleteTest) SetupB(b *testing.B) {
	testDirPath = setup.SetupTestDirectory(testDirName)
}

func (s *benchmarkDeleteTest) TeardownB(b *testing.B) {}

// createFilesToDelete creates the below objects in the bucket.
// benchmarking/a{i}.txt where i is a counter based on the benchtime value.
func createFilesToDelete(b *testing.B) {
	for i := 0; i < b.N; i++ {
		operations.CreateFileOfSize(5, path.Join(testDirPath, fmt.Sprintf("a%d.txt", i)), b)
	}
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *benchmarkDeleteTest) Benchmark_Delete(b *testing.B) {
	createFilesToDelete(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := os.Remove(path.Join(testDirPath, fmt.Sprintf("a%d.txt", i))); err != nil {
			b.Errorf("testing error: %v", err)
		}
	}
	if timeTaken := b.Elapsed().Milliseconds(); timeTaken > int64(deleteThreshold*b.N) {
		b.Errorf("Test failed due to timeout, time taken:%d msec", timeTaken)
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func Benchmark_Delete(b *testing.B) {
	ts := &benchmarkDeleteTest{}
	benchmark_setup.RunBenchmarks(b, ts)
}
