// Copyright 2023 Google LLC
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

// Provides integration tests for read flows.
package operations_test

import (
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/parallel"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/suite"
)

type ReadSuite struct {
	suite.Suite
	runCfg parallel.RunConfiguration
}

func (s *ReadSuite) TestReadAfterWrite() {
	testDir := setup.SetupTestDirectoryWithMntDir(s.runCfg.MntDir, DirForOperationTests+"-"+setup.GenerateRandomString(5))

	tmpDir, err := os.MkdirTemp(testDir, "tmpDir")
	if err != nil {
		s.T().Errorf("Mkdir at %q: %v", testDir, err)
		return
	}
	for range 10 {
		tmpFile, err := os.CreateTemp(tmpDir, tempFileName)
		if err != nil {
			s.T().Errorf("Create file at %q: %v", tmpDir, err)
			return
		}

		// Closing file at the end
		operations.CloseFileShouldNotThrowError(s.T(), tmpFile)

		fileName := tmpFile.Name()

		err = operations.WriteFileInAppendMode(fileName, "line 1\n")
		if err != nil {
			s.T().Errorf("AppendString: %v", err)
		}

		content, err := operations.ReadFile(fileName)
		if err != nil {
			s.T().Errorf("ReadAll: %v", err)
		}
		if got, want := string(content), "line 1\n"; got != want {
			s.T().Errorf("File content %q not match %q", got, want)
		}
	}
}

func TestRead(t *testing.T) {
	s := new(ReadSuite)
	s.runCfg = parallel.RunConfiguration{
		MntDir:     setup.MntDir(),
		TestBucket: setup.TestBucket(),
		OnlyDir:    setup.OnlyDirMounted(),
	}
	suite.Run(t, s)
}
