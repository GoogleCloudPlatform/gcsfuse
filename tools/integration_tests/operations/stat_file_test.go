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

package operations_test

import (
	"os"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/parallel"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type StatFileSuite struct {
	suite.Suite
	runCfg parallel.RunConfiguration
}

func (s *StatFileSuite) TestStatWithTrailingNewline() {
	testDir := setup.SetupTestDirectoryWithMntDir(s.runCfg.MntDir, DirForOperationTests+"-"+setup.GenerateRandomString(5))

	_, err := os.Stat(testDir + "/\n")

	require.Error(s.T(), err)
	assert.Equal(s.T(), err.(*os.PathError).Err, syscall.ENOENT)
}

func TestStatFile(t *testing.T) {
	s := new(StatFileSuite)
	s.runCfg = parallel.RunConfiguration{
		MntDir:     setup.MntDir(),
		TestBucket: setup.TestBucket(),
		OnlyDir:    setup.OnlyDirMounted(),
	}
	suite.Run(t, s)
}
