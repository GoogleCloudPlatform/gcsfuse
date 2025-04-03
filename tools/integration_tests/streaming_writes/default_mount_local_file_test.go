// Copyright 2025 Google LLC
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

package streaming_writes

import (
	"os"
	"path"
	"syscall"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/local_file"
	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type defaultMountLocalFile struct {
	defaultMountCommonTest
	CommonLocalFileTestSuite
	suite.Suite
}

func (t *defaultMountLocalFile) SetupTest() {
	t.createLocalFile()
}

func (t *defaultMountLocalFile) SetupSubTest() {
	t.createLocalFile()
}

func (t *defaultMountLocalFile) createLocalFile() {
	t.fileName = FileName1 + setup.GenerateRandomString(5)
	t.filePath = path.Join(testDirPath, t.fileName)
	// Create a local file.
	var err error
	t.f1, err = os.OpenFile(path.Join(testDirPath, t.fileName), os.O_RDWR|os.O_CREATE|os.O_TRUNC|syscall.O_DIRECT, operations.FilePermission_0600)
	require.NoError(t.T(), err)
}

// Executes all tests that run with single streamingWrites configuration for localFiles.
func TestDefaultMountLocalFileTest(t *testing.T) {
	s := new(defaultMountLocalFile)
	s.CommonLocalFileTestSuite.TestifySuite = &s.Suite
	s.defaultMountCommonTest.TestifySuite = &s.Suite
	suite.Run(t, s)
}
