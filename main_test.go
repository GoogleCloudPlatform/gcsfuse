// Copyright 2022 Google Inc. All Rights Reserved.
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

package main

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	. "github.com/jacobsa/ogletest"
)

func TestMains(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type MainTest struct {
}

func init() { RegisterTestSuite(&MainTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *MainTest) ResolvePathTest() {
	currDir, err := os.Getwd()
	if err != nil {
		fmt.Errorf("current working directory: %w", err)
	}

	inputArgs0 := []string{"gcsfuse", "--log-file", "test.json", "--ht", "-x"}
	resolvePathInCLIArguments([]string{"--log-file", "--key-file"}, inputArgs0, true)
	expected0 := []string{"gcsfuse",
		"--log-file", fmt.Sprintf("%s/test.json", currDir), "--ht", "-x"}
	ExpectTrue(reflect.DeepEqual(inputArgs0, expected0))

	inputArgs1 := []string{"gcsfuse", "--log-file=test.json", "--foreground"}
	resolvePathInCLIArguments([]string{"--log-file", "--key-file"}, inputArgs1, true)
	expected1 := []string{"gcsfuse",
		fmt.Sprintf("--log-file=%s/test.json", currDir), "--foreground"}
	ExpectTrue(reflect.DeepEqual(inputArgs1, expected1))

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Errorf("current working directory: %w", err)
	}

	inputArgs2 := []string{"gcsfuse", "--log-file=test.txt", "--key-file=~/test.json", "--foreground"}
	resolvePathInCLIArguments([]string{"--log-file", "--key-file"}, inputArgs2, true)
	expected2 := []string{"gcsfuse",
		fmt.Sprintf("--log-file=%s/test.txt", currDir),
		fmt.Sprintf("--key-file=%s/test.json", homeDir), "--foreground"}
	ExpectTrue(reflect.DeepEqual(inputArgs2, expected2))
}
