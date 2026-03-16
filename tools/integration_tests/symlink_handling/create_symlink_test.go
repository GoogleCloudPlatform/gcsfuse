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

package symlink_handling_test

import (
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/suite"
)

type BaseSymlinkSuite struct {
	suite.Suite
	flags       []string
	mntDir      string
	testDirPath string
}

func (s *BaseSymlinkSuite) SetupTest() {
	if testEnv.cfg.GKEMountedDirectory != "" {
		s.mntDir = testEnv.cfg.GKEMountedDirectory
		s.testDirPath = path.Join(s.mntDir, TestDirName)
	} else {
		s.mntDir = testEnv.cfg.GCSFuseMountedDirectory
		setup.SetMntDir(s.mntDir)
		err := static_mounting.MountGcsfuseWithStaticMounting(s.flags)
		s.Require().NoError(err)
		s.testDirPath = setup.SetupTestDirectory(TestDirName)
	}
}

func (s *BaseSymlinkSuite) TearDownTest() {
	if testEnv.cfg.GKEMountedDirectory == "" {
		setup.UnmountGCSFuse(s.mntDir)
	}
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), TestDirName))
}

func (s *BaseSymlinkSuite) createSymlink(linkName, target string) string {
	linkPath := path.Join(s.testDirPath, linkName)
	err := os.Symlink(target, linkPath)
	s.Require().NoError(err)
	return linkPath
}

func (s *BaseSymlinkSuite) createTempFile() string {
	targetFile, err := os.CreateTemp("", "symlink-target")
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := os.Remove(targetFile.Name()); err != nil {
			s.T().Logf("Error removing temporary file %s: %v", targetFile.Name(), err)
		}
	})
	err = targetFile.Close()
	if err != nil {
		s.T().Logf("Error closing temporary file %s: %v", targetFile.Name(), err)
	}
	return targetFile.Name()
}

type StandardSymlinksTestSuite struct {
	BaseSymlinkSuite
}

func (s *StandardSymlinksTestSuite) TestCreateSymlink() {
	target := s.createTempFile()
	linkName := "standard_symlink"

	// Create the symlink
	_ = s.createSymlink(linkName, target)

	// Validate the underlying GCS Object
	bucketName, objectName := setup.GetBucketAndObjectBasedOnTypeOfMount(path.Join(TestDirName, linkName))
	objHandle := testEnv.storageClient.Bucket(bucketName).Object(objectName)
	attrs, err := objHandle.Attrs(testEnv.ctx)
	s.Require().NoError(err)
	// Validate the GCS Object content to be the symlink target
	rc, err := objHandle.NewReader(testEnv.ctx)
	s.Require().NoError(err)
	defer func() {
		s.Assert().NoError(rc.Close())
	}()
	content, err := io.ReadAll(rc)
	s.Require().NoError(err)
	s.Assert().Equal(target, string(content), "Standard symlink content should match target")
	s.Assert().Equal(int64(len(target)), attrs.Size, "Standard symlink size should match target length")
	_, ok := attrs.Metadata[SymlinkMetadataKey]
	s.Assert().False(ok, "Standard symlink should not have old metadata key (%s)", SymlinkMetadataKey)
	val, ok := attrs.Metadata[StandardSymlinkMetadataKey]
	s.Assert().True(ok)
	s.Assert().Equal("true", val)
}

type LegacySymlinksTestSuite struct {
	BaseSymlinkSuite
}

func (s *LegacySymlinksTestSuite) TestCreateSymlink() {
	target := s.createTempFile()
	linkName := "legacy_symlink"

	// Create the symlink
	_ = s.createSymlink(linkName, target)

	// Validate the underlying GCS Object
	bucketName, objectName := setup.GetBucketAndObjectBasedOnTypeOfMount(path.Join(TestDirName, linkName))
	objHandle := testEnv.storageClient.Bucket(bucketName).Object(objectName)
	attrs, err := objHandle.Attrs(testEnv.ctx)
	s.Require().NoError(err)
	// Validate the GCS Object content to be nil
	s.Assert().Equal(int64(0), attrs.Size, "Legacy symlink size should be 0")
	val, ok := attrs.Metadata[SymlinkMetadataKey]
	s.Assert().True(ok, "Legacy symlink should have old metadata key (%s)", SymlinkMetadataKey)
	s.Assert().Equal(target, val, "Legacy symlink metadata value should match target")
	_, ok = attrs.Metadata[StandardSymlinkMetadataKey]
	s.Assert().False(ok, "Legacy symlink should not have new metadata key (%s)", StandardSymlinkMetadataKey)
}

func TestStandardSymlinks(t *testing.T) {
	RunTests(t, "TestStandardSymlinksTestSuite", func(flags []string) suite.TestingSuite {
		return &StandardSymlinksTestSuite{BaseSymlinkSuite{flags: flags}}
	})
}

func TestLegacySymlinks(t *testing.T) {
	RunTests(t, "TestLegacySymlinksTestSuite", func(flags []string) suite.TestingSuite {
		return &LegacySymlinksTestSuite{BaseSymlinkSuite{flags: flags}}
	})
}

func RunTests(t *testing.T, runName string, factory func(flags []string) suite.TestingSuite) {
	for _, cfg := range testEnv.cfg.Configs {
		if cfg.Run == runName {
			for _, flagStr := range cfg.Flags {
				flags := strings.Fields(flagStr)
				suite.Run(t, factory(flags))
			}
		}
	}
}
