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

package benchmark_setup_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/benchmark_setup"
)

type benchmarkStructure struct {
	setupCtr, teardownCtr, bench1, bench2 int
}

func (bs *benchmarkStructure) SetupB(*testing.B) {
	bs.setupCtr++
}
func (bs *benchmarkStructure) BenchmarkExample1(*testing.B) {
	// This is to ensure, right reading in the first go. Otherwise, it calls
	// the method multiple times with different value of b.N.
	time.Sleep(time.Second)
	bs.bench1++
}

func (bs *benchmarkStructure) BenchmarkExample2(*testing.B) {
	// This is to ensure, right reading in the first go. Otherwise, it calls
	// the method multiple times with different value of b.N.
	time.Sleep(time.Second)
	bs.bench2++
}

func (bs *benchmarkStructure) TeardownB(*testing.B) {
	bs.teardownCtr++
}

func BenchmarkRunBenchmarks(b *testing.B) {
	benchmarkStruct := &benchmarkStructure{}

	benchmark_setup.RunBenchmarks(b, benchmarkStruct)

	assert.Equal(b, 2, benchmarkStruct.setupCtr)
	assert.Equal(b, 1, benchmarkStruct.bench1)
	assert.Equal(b, 1, benchmarkStruct.bench2)
	assert.Equal(b, 2, benchmarkStruct.teardownCtr)
}
