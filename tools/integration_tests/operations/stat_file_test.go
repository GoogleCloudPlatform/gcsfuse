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

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/all_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type OperationSuite struct {
	mountConfiguration *TestMountConfiguration
	suite.Suite
}

func (s *OperationSuite) SetupSuite() {
	err := s.mountConfiguration.Mount(s.T(), storageClient)
	require.NoError(s.T(), err)
}

func (s *OperationSuite) TestStatWithTrailingNewline() {
	testDir := setup.SetupTestDirectoryOnMntDir(s.mountConfiguration.MntDir(), TestDirName(s.T()))

	_, err := os.Stat(testDir + "/\n")

	require.Error(s.T(), err)
	assert.Equal(s.T(), err.(*os.PathError).Err, syscall.ENOENT)
}

func TestOperationsSuite(t *testing.T) {
	t.Parallel()
	for i := range configs {
		t.Run(configs[i].MountType().String()+"_"+setup.GenerateRandomString(5), func(t *testing.T) {
			t.Parallel()
			suite.Run(t, &OperationSuite{mountConfiguration: &configs[i]})
		})
	}
}
