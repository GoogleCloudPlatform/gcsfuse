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

// Provide test for deleting implicit directory.
package implicit_dir_test

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup/implicit_and_explicit_dir_setup"
)

func TestDeleteNonEmptyImplicitDir(t *testing.T) {
	// Directory Structure
	// testBucket/implicitDirectory                                                  -- Dir
	// testBucket/implicitDirectory/fileInImplicitDir1                               -- File
	// testBucket/implicitDirectory/implicitSubDirectory                             -- Dir
	// testBucket/implicitDirectory/implicitSubDirectory/fileInImplicitDir2          -- File

	implicit_and_explicit_dir_setup.CreateImplicitDirectory()

	dirPath := path.Join(setup.MntDir(), ImplicitDirectory)
	err := os.RemoveAll(dirPath)
	if err != nil {
		t.Errorf("Error in deleting non-empty implicit directory.")
	}

	dir, err := os.Stat(dirPath)
	if err == nil && dir.Name() == ImplicitDirectory && dir.IsDir() {
		t.Errorf("Directory is not deleted.")
	}
}
