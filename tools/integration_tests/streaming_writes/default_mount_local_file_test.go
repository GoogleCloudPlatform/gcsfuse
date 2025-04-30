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
	"path"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/local_file"
	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_suite"
	"github.com/stretchr/testify/suite"
)

type defaultMountCommonLocalFile struct {
	defaultMountCommonTest
	suite.Suite
}

type defaultMountRegionalBucketLocalFile struct {
	CommonLocalFileTestSuite
	defaultMountCommonLocalFile
	test_suite.TestifySuite
}

func (t *defaultMountCommonLocalFile) SetupTest() {
	t.createLocalFile()
}

func (t *defaultMountCommonLocalFile) SetupSubTest() {
	t.createLocalFile()
}

func (t *defaultMountCommonLocalFile) createLocalFile() {
	t.fileName = FileName1 + setup.GenerateRandomString(5)
	t.filePath = path.Join(testDirPath, t.fileName)
	// Create a local file with O_DIRECT.
	t.f1 = operations.OpenFileWithODirect(t.T(), t.filePath)
}

// Executes all tests that run with single streamingWrites configuration for localFiles.
func TestDefaultMountLocalFileTest(t *testing.T) {
	if setup.IsZonalBucketRun() {
		s := new(defaultMountCommonLocalFile)
		s.defaultMountCommonTest.TestifySuite = &s.Suite
		suite.Run(t, s)
	} else {
		s := new(defaultMountRegionalBucketLocalFile)
		s.defaultMountCommonTest.TestifySuite = &s.defaultMountCommonLocalFile.Suite
		s.CommonLocalFileTestSuite.TestifySuite = &s.defaultMountCommonLocalFile.Suite
		s.TestifySuite = &s.defaultMountCommonLocalFile.Suite
		suite.Run(t, s)
	}
}
