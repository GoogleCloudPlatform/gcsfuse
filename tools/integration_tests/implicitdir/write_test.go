// Copyright 2021 Google Inc. All Rights Reserved.
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

// Provides integration tests for read flows with implicit_dir flag set.
package implicitdir_test

import (
	"os"
	"testing"
)

func TestAppendToEndOfFile(t *testing.T) {

	for i := 0; i < 1; i++ {

		fileName := tmpFile.Name()

		f, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
		    t.Errorf("Open file for append: %v", err)
		}

		if _, err = f.WriteString("line 3\n"); err != nil {
		    t.Errorf("AppendString: %v", err)
		}

		f.Close()

		// After write, data will be cached by kernel. So subsequent read will be
		// served using cached data by kernel instead of calling gcsfuse.
		// Clearing kernel cache to ensure that gcsfuse is invoked during read operation.
		compareFileContents(t, fileName, "line 1\nline 2\nline 3\n")
	}
}