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

package operations

import (
	"bytes"
	"path"
	"strconv"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const OneGigaBytes = 100000000
const OneGBFile = "oneGbFile.txt"

func TestReadLargeFileSequentially(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	file := path.Join(setup.MntDir(), OneGBFile)

	setup.RunScriptForTestData("testdata/write_content_fix_size.sh", file, strconv.Itoa(OneGigaBytes))
	content, err := operations.ReadFileSequentially(file, OneGigaBytes)
	if err != nil {
		t.Errorf("Error in reading file: %v", err)
	}

	actualContent, err := operations.ReadFile(file)
	if err != nil {
		t.Errorf("Error in reading file: %v", err)
	}

	if bytes.Equal(content, actualContent) == false {
		t.Errorf("Error in reading file sequentially.")
	}
}
