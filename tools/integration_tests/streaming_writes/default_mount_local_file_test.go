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
	"testing"

	local_file "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/local_file"
	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/suite"
)

type defaultMountLocalFile struct {
	defaultMountCommonTest
}

func (t *defaultMountLocalFile) SetupTest() {
	t.fileName = FileName1 + setup.GenerateRandomString(5)
	// Create a local file.
	_, t.f1 = CreateLocalFileInTestDir(ctx, storageClient, testDirPath, t.fileName, t.T())
}

func (t *defaultMountLocalFile) SetupSubTest() {
	t.fileName = FileName1 + setup.GenerateRandomString(5)
	// Create a local file.
	_, t.f1 = CreateLocalFileInTestDir(ctx, storageClient, testDirPath, t.fileName, t.T())
}

// Executes all tests that run with single streamingWrites configuration for localFiles.
func TestDefaultMountLocalFileTest(t *testing.T) {
	suite.Run(t, new(defaultMountLocalFile))
}

func TestDefaultMountLocalFileTestSuite(t *testing.T) {
	suite.Run(t, new(local_file.CommonLocalFileTestSuite))
}
