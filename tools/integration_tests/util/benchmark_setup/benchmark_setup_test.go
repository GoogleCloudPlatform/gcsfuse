// Copyright 2024 Google Inc. All Rights Reserved.
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
	"fmt"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/benchmark_setup"
	. "github.com/jacobsa/ogletest"
)

type benchmarkStructure struct {
	setupCtr, teardownCtr, bench1, bench2 int
}

func (b *benchmarkStructure) SetupB(*testing.B) {
	fmt.Println("tes")
	b.setupCtr++
}
func (b *benchmarkStructure) BenchmarkExample1(*testing.B) {
	b.bench1++
}

func (b *benchmarkStructure) BenchmarkExample2(*testing.B) {
	b.bench2++
}

func (b *benchmarkStructure) TeardownB(*testing.B) {
	b.teardownCtr++
	fmt.Println("test")
}

func BenchmarkRunBenchmarks(b *testing.B) {
	benchmarkStruct := &benchmarkStructure{}

	benchmark_setup.RunBenchmarks(b, benchmarkStruct)

	AssertEq(benchmarkStruct.setupCtr, 2)
	AssertEq(benchmarkStruct.bench1, 1)
	AssertEq(benchmarkStruct.bench2, 1)
	AssertEq(benchmarkStruct.teardownCtr, 2)
}
