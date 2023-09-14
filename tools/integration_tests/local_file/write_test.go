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

// Provides integration tests for write on local files.
package local_file_test

import (
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/tools/integration_tests/local_file/helpers"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func assertWriteFileErrorIsNil(err error, t *testing.T) {
	if err != nil {
		t.Fatalf("Error while writing to local file. err: %v", err)
	}
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////
func TestMultipleWritesToLocalFile(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()
	// Create a local file.
	_, fh := CreateLocalFile(FileName1, t)

	// Write some contents to file sequentially.
	WritingToLocalFileSHouldNotThrowError(fh, FileContents, t)
	WritingToLocalFileSHouldNotThrowError(fh, FileContents, t)
	WritingToLocalFileSHouldNotThrowError(fh, FileContents, t)
	ValidateObjectNotFoundErrOnGCS(FileName1, t)

	// Close the file and validate if the file is created on GCS.
	CloseFileAndValidateObjectContents(fh, FileName1, "teststringteststringteststring", t)
}

func TestRandomWritesToLocalFile(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()
	// Create a local file.
	_, fh := CreateLocalFile(FileName1, t)

	// Write some contents to file randomly.
	_, err := fh.WriteAt([]byte("string1"), 0)
	assertWriteFileErrorIsNil(err, t)
	_, err = fh.WriteAt([]byte("string2"), 2)
	assertWriteFileErrorIsNil(err, t)
	_, err = fh.WriteAt([]byte("string3"), 3)
	assertWriteFileErrorIsNil(err, t)
	ValidateObjectNotFoundErrOnGCS(FileName1, t)

	// Close the file and validate if the file is created on GCS.
	CloseFileAndValidateObjectContents(fh, FileName1, "stsstring3", t)
}
