// Copyright 2026 Google LLC
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

package symlink_handling

import (
	"os"
	"path"
	"syscall"
)

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (s *DisableSymlinksTestSuite) TestCreateSymlink_FailsWithENOTSUP() {
	linkPath := path.Join(s.base.testDirPath, s.base.linkName)

	err := os.Symlink(s.base.targetPath, linkPath)

	s.Require().Error(err)
	s.Assert().ErrorIs(err, syscall.ENOTSUP)
}

func (s *DisableSymlinksTestSuite) TestExistingStandardSymlink_TreatedAsRegularFile() {
	s.base.isStandardSymlink = true
	s.base.createGCSSymlinkObject(s.base.linkName, s.base.targetPath)
	linkPath := path.Join(s.base.testDirPath, s.base.linkName)

	fi, err := os.Lstat(linkPath)
	s.Require().NoError(err)
	s.Assert().Zero(fi.Mode() & os.ModeSymlink)

	_, err = os.Readlink(linkPath)
	s.Require().Error(err)
	s.Assert().ErrorIs(err, syscall.EINVAL)

	content, err := os.ReadFile(linkPath)
	s.Require().NoError(err)
	s.Assert().Equal(s.base.targetPath, string(content))
}

func (s *DisableSymlinksTestSuite) TestExistingLegacySymlink_TreatedAsRegularFile() {
	s.base.isStandardSymlink = false
	s.base.createGCSSymlinkObject(s.base.linkName, s.base.targetPath)
	linkPath := path.Join(s.base.testDirPath, s.base.linkName)

	fi, err := os.Lstat(linkPath)
	s.Require().NoError(err)
	s.Assert().Zero(fi.Mode() & os.ModeSymlink)

	_, err = os.Readlink(linkPath)
	s.Require().Error(err)
	s.Assert().ErrorIs(err, syscall.EINVAL)

	content, err := os.ReadFile(linkPath)
	s.Require().NoError(err)
	s.Assert().Empty(content)
}
