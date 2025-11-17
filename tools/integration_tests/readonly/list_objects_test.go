// Copyright 2023 Google LLC
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

	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

func TestListObjectsInBucket(t *testing.T) {
	// ** Directory structure **
	// testBucket
	// testBucket/testDirForReadOnlyTest/Test       -- Dir
	// testBucket/testDirForReadOnlyTest/Test1.txt  -- File

	obj, err := os.ReadDir(path.Join(setup.MntDir(), TestDirForReadOnlyTest))
	if err != nil {
		log.Fatal(err)
	}

	// Comparing number of objects in the testBucket - 2
	if len(obj) != NumberOfObjectsInTestBucket {
		t.Errorf("The number of objects in the current directory doesn't match.")
	}

	// Comparing first object name and type
	// Name - testBucket/testDirForReadOnlyTest/Test/, Type - Dir
	if obj[0].Name() != DirectoryNameInTestBucket || obj[0].IsDir() != true {
		t.Errorf("Object Listed for bucket directory is incorrect.")
	}

	// Comparing second object name and type
	// Name - testBucket/testDirForReadOnlyTest/Test1.txt, Type - File
	if obj[1].Name() != FileNameInTestBucket || obj[1].IsDir() != false {
		t.Errorf("Object Listed for file in bucket is incorrect.")
	}
}

func TestListObjectsInBucketDirectory(t *testing.T) {
	// ** SubDirectory structure **
	// testBucket/testDirForReadOnlyTest/Test
	// testBucket/testDirForReadOnlyTest/Test/a.txt -- File
	// testBucket/testDirForReadOnlyTest/Test/b/    -- Dir

	Dir := path.Join(setup.MntDir(), TestDirForReadOnlyTest, DirectoryNameInTestBucket)
	obj, err := os.ReadDir(Dir)
	if err != nil {
		log.Fatal(err)
	}

	// Comparing number of objects in the directory of testBucket - 2.
	if len(obj) != NumberOfObjectsInDirectoryTestBucket {
		t.Errorf("The number of objects in the current directory doesn't match.")
	}

	// Comparing first object name and type.
	// Name - testBucket/testDirForReadOnlyTest/Test/a.txt, Type - File
	if obj[0].Name() != FileNameInDirectoryTestBucket || obj[0].IsDir() != false {
		t.Errorf("Object Listed for file in bucket directory is incorrect.")
	}

	// Comparing second object name and type.
	// Name - testBucket/testDirForReadOnlyTest/Test/b, Type - Dir
	if obj[1].Name() != SubDirectoryNameInTestBucket || obj[1].IsDir() != true {
		t.Errorf("Object Listed for bucket sub directory is incorrect.")
	}
}
