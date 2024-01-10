// Copyright 2024 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package test_setup implements Setup and Teardown methods to be used in tests.
package test_setup

import (
	"reflect"
	"strings"
	"testing"
)

// Interface defines Tester's methods for use in this package.
type Testable interface {
	Setup(*testing.T)
	Teardown(*testing.T)
}

func getTestFunc(t *testing.T, xv reflect.Value, name string) func(*testing.T) {
	if m := xv.MethodByName(name); m.IsValid() {
		if f, ok := m.Interface().(func(*testing.T)); ok {
			return f
		}
		// Method exists but has the wrong type signature.
		t.Fatalf("test function %v has unexpected signature (%T)", name, m.Interface())
	}
	return func(*testing.T) {}
}

// RunSubTests runs all "Test*" functions that are member of x as subtests
// of the current test.  Setup is run before the test function and Teardown is
// run after each test.
// x must extend Interface by implementing Setup and TearDown methods.
func RunSubTests(t *testing.T, x Testable) {
	xt := reflect.TypeOf(x)
	xv := reflect.ValueOf(x)

	for i := 0; i < xt.NumMethod(); i++ {
		methodName := xt.Method(i).Name
		if !strings.HasPrefix(methodName, "Test") {
			continue
		}
		testFunc := getTestFunc(t, xv, methodName)
		t.Run(methodName, func(t *testing.T) {
			// Execute Teardown in t.Cleanup() to guarantee it is run even if test
			// function or setup uses t.Fatal().
			t.Cleanup(func() { x.Teardown(t) })
			x.Setup(t)
			testFunc(t)
		})
	}
}
