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

// Provide a helper function to read a file.
package fileoperationhelper

import (
	"fmt"
	"os"
	"syscall"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

func Read(filePath string) (content []byte, err error) {
	file, err := os.OpenFile(filePath, os.O_RDONLY|syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		err = fmt.Errorf("Error in the opening the file %v", err)
		return
	}
	defer file.Close()

	content, err = os.ReadFile(file.Name())
	if err != nil {
		err = fmt.Errorf("ReadAll: %v", err)
		return
	}
	return
}

func ReadAfterWrite(fileName string) (content []byte, err error) {
	err = WriteAtEndOfFile(fileName, "line 1\n")
	if err != nil {
		err = fmt.Errorf("AppendString: %v", err)
		return
	}

	content, err = Read(fileName)
	if err != nil {
		err = fmt.Errorf("ReadAll: %v", err)
		return
	}
	return content, err
}
