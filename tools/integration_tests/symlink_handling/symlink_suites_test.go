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
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/suite"
)

// BaseSymlinkSuite provides the common structure and configuration-driven setup logic.
type BaseSymlinkSuite struct {
	suite.Suite
	flags             []string
	mntDir            string
	testDirPath       string
	isStandardSymlink bool
	// linkName is the name of the symlink (relative to testDirPath).
	linkName string
	// targetPath is the absolute path of the target file/dir .
	targetPath string
}

// StandardSymlinksTestSuite groups all test related to symlinks following standard representation.
type StandardSymlinksTestSuite struct{ BaseSymlinkSuite }

// StandardSymlinksTestSuite groups all test related to symlinks following legacy representation.
type LegacySymlinksTestSuite struct{ BaseSymlinkSuite }

////////////////////////////////////////////////////////////////////////
// Common Suite Logic
////////////////////////////////////////////////////////////////////////

func (s *BaseSymlinkSuite) SetupTest() {
	if testEnv.cfg.GKEMountedDirectory != "" {
		s.mntDir = testEnv.cfg.GKEMountedDirectory
		s.testDirPath = path.Join(s.mntDir, TestDirName)
	} else {
		s.mntDir = testEnv.cfg.GCSFuseMountedDirectory
		setup.SetMntDir(s.mntDir)
		err := static_mounting.MountGcsfuseWithStaticMountingWithConfigFile(testEnv.cfg, s.flags)
		s.Require().NoError(err)
		s.testDirPath = setup.SetupTestDirectory(TestDirName)
	}
	// Initialize common variables for symlink tests, ensuring they are unique for each test method.
	s.linkName = setup.GenerateRandomString(5) + "_link"
	s.targetPath = path.Join(s.testDirPath, setup.GenerateRandomString(5))
}

func (s *BaseSymlinkSuite) TearDownTest() {
	if testEnv.cfg.GKEMountedDirectory == "" {
		setup.UnmountGCSFuse(s.mntDir)
	}
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), TestDirName))
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

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
	s.Require().NoError(targetFile.Close())
	return targetFile.Name()
}

// validateBackingGCSObjectForSymlink validates the GCS object created for a symlink.
func (s *BaseSymlinkSuite) validateBackingGCSObjectForSymlink(linkName, target string, isStandardSymlink bool) {
	bucketName, objectName := setup.GetBucketAndObjectBasedOnTypeOfMount(path.Join(TestDirName, linkName))
	objHandle := testEnv.storageClient.Bucket(bucketName).Object(objectName)
	attrs, err := objHandle.Attrs(testEnv.ctx)
	s.Require().NoError(err)

	if isStandardSymlink {
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
		s.Assert().True(ok)
		val, ok := attrs.Metadata[StandardSymlinkMetadataKey]
		s.Assert().True(ok)
		s.Assert().Equal("true", val)
	} else {
		// Legacy symlink
		// Validate the GCS Object content to be nil
		s.Assert().Equal(int64(0), attrs.Size, "Legacy symlink size should be 0")
		val, ok := attrs.Metadata[SymlinkMetadataKey]
		s.Assert().True(ok, "Legacy symlink should have old metadata key (%s)", SymlinkMetadataKey)
		s.Assert().Equal(target, val, "Legacy symlink metadata value should match target")
		_, ok = attrs.Metadata[StandardSymlinkMetadataKey]
		s.Assert().False(ok, "Legacy symlink should not have new metadata key (%s)", StandardSymlinkMetadataKey)
	}
}

// createGCSSymlinkObject creates a symlink object on GCS with appropriate metadata.
// The 'target' parameter is the symlink target path.
func (s *BaseSymlinkSuite) createGCSSymlinkObject(linkName, target string) {
	fullLinkPath := path.Join(TestDirName, linkName)
	bucketName, objectName := setup.GetBucketAndObjectBasedOnTypeOfMount(fullLinkPath)
	objHandle := testEnv.storageClient.Bucket(bucketName).Object(objectName)
	w, err := client.NewWriter(testEnv.ctx, objHandle, testEnv.storageClient)
	s.Require().NoError(err)

	var content []byte
	if s.isStandardSymlink {
		w.Metadata = map[string]string{StandardSymlinkMetadataKey: "true", SymlinkMetadataKey: target}
		content = []byte(target) // Standard symlinks store target in content
	} else {
		w.Metadata = map[string]string{SymlinkMetadataKey: target}
		content = []byte("") // Legacy symlinks have empty content
	}

	_, err = w.Write(content)
	s.Require().NoError(err)
	s.Require().NoError(w.Close())
	operations.WaitForSizeUpdate(setup.IsZonalBucketRun(), operations.WaitDurationAfterCloseZB)
}

////////////////////////////////////////////////////////////////////////
// Test Runner
////////////////////////////////////////////////////////////////////////

func TestStandardSymlinks(t *testing.T) {
	RunTests(t, "TestStandardSymlinksTestSuite", func(flags []string) suite.TestingSuite {
		return &StandardSymlinksTestSuite{BaseSymlinkSuite{flags: flags, isStandardSymlink: true}}
	})
}

func TestLegacySymlinks(t *testing.T) {
	RunTests(t, "TestLegacySymlinksTestSuite", func(flags []string) suite.TestingSuite {
		return &LegacySymlinksTestSuite{BaseSymlinkSuite{flags: flags, isStandardSymlink: false}}
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
