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

// Provide a helper for directory operations.
package operations

import (
	"fmt"
	"os"
	"os/exec"
)

func CopyDir(srcDirPath string, destDirPath string) (err error) {
	cmd := exec.Command("cp", "--recursive", srcDirPath, destDirPath)

	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("Copying dir operation is failed: %v", err)
	}
	return
}

func RenameDir(dirName string, newDirName string) (err error) {
	if _, err = os.Stat(newDirName); err == nil {
		err = fmt.Errorf("Renamed directory %s already present", newDirName)
		return
	}

	if err = os.Rename(dirName, newDirName); err != nil {
		err = fmt.Errorf("Rename unsuccessful: %v", err)
		return
	}

	if _, err = os.Stat(dirName); err == nil {
		err = fmt.Errorf("Original directory %s still exists", dirName)
		return
	}
	if _, err = os.Stat(newDirName); err != nil {
		err = fmt.Errorf("Renamed directory %s not found", newDirName)
		return
	}
	return
}
