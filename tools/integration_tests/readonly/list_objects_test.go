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

// Provides integration tests for file operations with --o=ro flag set.
package readonly_test

import (
	"log"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

func TestListObjectsInBucket(t *testing.T) {
	// ** Directory structure **
	// testBucket
	// testBucket/Test       -- Dir
	// testBucket/Test1.txt  -- File

	obj, err := os.ReadDir(setup.MntDir())
	if err != nil {
		log.Fatal(err)
	}

	if len(obj) != NumberOfObjectsInTestBucket {
		t.Errorf("The number of objects in the current directory doesn't match.")
	}

	if obj[0].Name() != DirectoryNameInTestBucket && obj[0].IsDir() != true {
		t.Errorf("Listed object is incorrect.")
	}

	if obj[1].Name() != FileNameInTestBucket && obj[1].IsDir() != false {
		t.Errorf("Listed object is incorrect.")
	}
}

func TestListObjectsInBucketSubDirectory(t *testing.T) {
	// ** SubDirectory structure **
	// testBucket/Test
	// testBucket/Test/a.txt -- File
	// testBucket/Test/b/    -- Dir

	Dir := path.Join(setup.MntDir(), DirectoryNameInTestBucket)
	obj, err := os.ReadDir(Dir)
	if err != nil {
		log.Fatal(err)
	}

	if len(obj) != NumberOfObjectsInTestBucketSubDirectory {
		t.Errorf("The number of objects in the current directory doesn't match.")
	}

	if obj[0].Name() != FileInSubDirectoryNameInTestBucket && obj[0].IsDir() != false {
		t.Errorf("Listed object is incorrect.")
	}

	if obj[1].Name() != SubDirectoryNameInTestBucket && obj[1].IsDir() != true {
		t.Errorf("Listed object is incorrect.")
	}
}
