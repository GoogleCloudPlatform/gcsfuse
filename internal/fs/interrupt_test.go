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

// A collection of tests for a file system where we do not attempt to write to
// the file system at all. Rather we set up contents in a GCS bucket out of
// band, wait for them to be available, and then read them via the file system.

package fs

import (
	"context"
	"testing"

	"github.com/jacobsa/ogletest"
)

var ignoreInterruptsTest = []struct {
	testName         string
	ignoreInterrupts bool
	err              error
}{
	{"Ignore Interrupts", true, nil},
	{"Respect Interrupts", false, context.Canceled},
}

func TestIgnoreInterruptsIfFlagIsSet(t *testing.T) {
	for _, tt := range ignoreInterruptsTest {
		t.Run(tt.testName, func(t *testing.T) {
			fs := &fileSystem{ignoreInterrupts: tt.ignoreInterrupts}
			ctx, cancel := context.WithCancel(context.Background())

			// Call the method and cancel the context
			ctx = fs.ignoreInterruptsIfFlagIsSet(ctx)
			cancel()

			ogletest.AssertEq(tt.err, ctx.Err())
		})
	}
}
