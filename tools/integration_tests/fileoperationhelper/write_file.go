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

// Provide a helper function to write a file.
package fileoperationhelper

import (
	"fmt"
	"os"
	"syscall"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

func WriteAtEndOfFile(fileName string, content string) (err error) {
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		err = fmt.Errorf("Open file for append: %v", err)
		return
	}

	_, err = f.WriteString(content)
	f.Close()

	return
}

func WriteAtStartOfFile(fileName string, content string) (err error) {
	f, err := os.OpenFile(fileName, os.O_WRONLY|syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		err = fmt.Errorf("Open file for write at start: %v", err)
		return
	}

	_, err = f.WriteAt([]byte(content), 0)
	f.Close()

	return
}

func WriteAtRandom(fileName string, content string) (err error) {
	f, err := os.OpenFile(fileName, os.O_WRONLY|syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		err = fmt.Errorf("Open file for write at start: %v", err)
		return
	}

	_, err = f.WriteAt([]byte("line 5\n"), 7)
	f.Close()

	return
}
