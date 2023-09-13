// Copyright 2023 Google Inc. All Rights Reserved.
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

// Provides integration tests for create local file.
package local_file_test

import (
	"path"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/tools/integration_tests/local_file/helpers"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func TestNewFileShouldNotGetSyncedToGCSTillClose(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	// Validate.
	NewFileShouldGetSyncedToGCSAtClose(FileName1, t)
}

func TestNewFileUnderExplicitDirectoryShouldNotGetSyncedToGCSTillClose(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()
	// Make explicit directory.
	CreateExplicitDirShouldNotThrowError(t)

	// Validate.
	NewFileShouldGetSyncedToGCSAtClose(path.Join(ExplicitDirName, ExplicitFileName1), t)
}
