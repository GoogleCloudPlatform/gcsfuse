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

// Provide a helper function to copy a file.
package fileoperationhelper

import (
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

func CopyFile(srcFile string, newFileName string) (err error) {
	if _, err = os.Stat(newFileName); err == nil {
		err = fmt.Errorf("Copied file %s already present", newFileName)
	}

	// File copying with io.Copy() utility.
	source, err := os.OpenFile(srcFile, syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		err = fmt.Errorf("File %s opening error: %v", srcFile, err)
	}
	defer source.Close()

	destination, err := os.OpenFile(newFileName, os.O_WRONLY|os.O_CREATE|syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		err = fmt.Errorf("Copied file creation error: %v", err)
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err != nil {
		err = fmt.Errorf("Error in file copying: %v", err)
	}
	return
}
