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

func TestIsolateContextFromParentContext(t *testing.T) {
	parentCtx, parentCtxCancel := context.WithCancel(context.Background())

	// Call the method and cancel the parent context
	newCtx, newCtxCancel := isolateContextFromParentContext(parentCtx)
	parentCtxCancel()

	// validate new context is not cancelled after parent's cancellation.
	ogletest.AssertEq(nil, newCtx.Err())
	// cancel the new context and validate.
	newCtxCancel()
	ogletest.AssertEq(context.Canceled, newCtx.Err())
}
