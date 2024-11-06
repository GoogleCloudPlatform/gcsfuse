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

package benchmark_setup

import (
	"reflect"
	"strings"
	"testing"
)

type Benchmark interface {
	SetupB(*testing.B)
	TeardownB(*testing.B)
}

func getBenchmarkFunc(b *testing.B, xv reflect.Value, name string) func(*testing.B) {
	if m := xv.MethodByName(name); m.IsValid() {
		if f, ok := m.Interface().(func(*testing.B)); ok {
			return f
		}
		// Method exists but has the wrong type signature.
		b.Fatalf("benchmark function %v has unexpected signature (%T)", name, m.Interface())
	}
	return func(*testing.B) {}
}

// RunBenchmarks runs all "Benchmark*" functions that are members of x as sub-benchmarks
// of the current benchmark. SetupB is run before the benchmark function and TeardownB is
// run after each benchmark.
// x must extend Benchmark interface by implementing SetupB and TearDownB methods.
func RunBenchmarks(b *testing.B, x Benchmark) {
	xt := reflect.TypeOf(x)
	xv := reflect.ValueOf(x)

	for i := 0; i < xt.NumMethod(); i++ {
		methodName := xt.Method(i).Name
		if !strings.HasPrefix(methodName, "Benchmark") {
			continue
		}
		benchmarkFunc := getBenchmarkFunc(b, xv, methodName)
		b.Run(methodName, func(b *testing.B) {
			// Execute TeardownB in b.Cleanup() to guarantee it is run even if benchmark
			// function or setup uses b.Fatal().
			b.Cleanup(func() { x.TeardownB(b) })
			x.SetupB(b)
			benchmarkFunc(b)
		})
	}
}
