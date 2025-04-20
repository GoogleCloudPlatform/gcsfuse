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

package fs_test

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type staleFileHandleLocalFile struct {
	staleFileHandleCommon
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (t *staleFileHandleLocalFile) SetupTest() {
	// Create a local file.
	_, t.f1 = operations.CreateLocalFile(ctx, t.T(), mntDir, bucket, "foo")
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////

// Executes all stale handle tests for local files.
func TestStaleFileHandleLocalFile(t *testing.T) {
	suite.Run(t, new(staleFileHandleLocalFile))
}
