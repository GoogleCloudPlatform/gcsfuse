// Copyright 2024 Google LLC
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
	"os"
	"strings"
	"testing"
)

func ValidateNoFileOrDirError(path string, t *testing.T) {
	_, err := os.Stat(path)
	if err == nil || !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("os.Stat(%s). Expected: %s, Got: %v", path,
			"no such file or directory", err)
	}
}

func CheckErrorForReadOnlyFileSystem(err error, t *testing.T) {
	if err == nil {
		t.Error("permission denied error expected but got nil error.")
		return
	}
	if strings.Contains(err.Error(), "read-only file system") || strings.Contains(err.Error(), "permission denied") || strings.Contains(err.Error(), "Permission denied") {
		return
	}
	t.Errorf("Incorrect error for readonly file system: %v", err.Error())
}
