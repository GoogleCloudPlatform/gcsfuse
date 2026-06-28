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

package logical_quota

import (
	"errors"
	"log"
	"os"
	"path"
	"strings"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	oneMiB = 1024 * 1024
)

type logicalQuotaSuite struct {
	flags  []string
	prefix string
	suite.Suite
}

func (s *logicalQuotaSuite) SetupSuite() {
	setup.SetUpLogFilePath(s.T().Name(), s.flags, "", "", testEnv.cfg)
	setup.MountGCSFuseWithGivenMountWithConfigFunc(
		testEnv.cfg,
		s.flags,
		static_mounting.MountGcsfuseWithStaticMountingWithConfigFile)
	setup.SetMntDir(testEnv.cfg.GCSFuseMountedDirectory)
}

func (s *logicalQuotaSuite) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(testEnv.cfg.TestBucket, s.prefix))
}

func (s *logicalQuotaSuite) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

type sizeQuotaSuite struct {
	logicalQuotaSuite
}

func (s *sizeQuotaSuite) SetupSuite() {
	s.prefix = sizeQuotaPrefix
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(testEnv.cfg.TestBucket, s.prefix))
	require.NoError(
		s.T(),
		client.CreateObjectOnGCS(
			testEnv.ctx,
			testEnv.storageClient,
			path.Join(s.prefix, "existing.bin"),
			strings.Repeat("a", 4*oneMiB)))
	s.logicalQuotaSuite.SetupSuite()
}

func (s *sizeQuotaSuite) TestSizeQuota() {
	initialStat := statFS(s.T())
	initialBlockSize := uint64(initialStat.Frsize)
	require.NotZero(s.T(), initialBlockSize)
	assert.Equal(s.T(), ceilDiv(5*oneMiB, initialBlockSize), uint64(initialStat.Blocks))
	assert.Equal(s.T(), uint64(initialStat.Bfree), uint64(initialStat.Bavail))
	assert.Equal(s.T(), uint64(oneMiB)/initialBlockSize, uint64(initialStat.Bavail))

	withinQuotaFile := path.Join(setup.MntDir(), "within-quota.bin")
	require.NoError(s.T(), os.WriteFile(withinQuotaFile, []byte(strings.Repeat("b", oneMiB)), 0600))

	fullStat := statFS(s.T())
	assert.Zero(s.T(), uint64(fullStat.Bavail))

	beyondQuotaFile := path.Join(setup.MntDir(), "beyond-quota.bin")
	err := os.WriteFile(beyondQuotaFile, []byte("x"), 0600)
	assertNoSpaceLeft(s.T(), err)

	require.NoError(s.T(), os.Remove(withinQuotaFile))
	recoveredStat := statFS(s.T())
	assert.GreaterOrEqual(s.T(), uint64(recoveredStat.Bavail), uint64(oneMiB)/uint64(recoveredStat.Frsize))

	require.NoError(s.T(), os.WriteFile(path.Join(setup.MntDir(), "after-delete.bin"), []byte(strings.Repeat("c", oneMiB)), 0600))
}

type fileCountQuotaSuite struct {
	logicalQuotaSuite
}

func (s *fileCountQuotaSuite) SetupSuite() {
	s.prefix = fileCountQuotaPrefix
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(testEnv.cfg.TestBucket, s.prefix))
	require.NoError(
		s.T(),
		client.CreateObjectOnGCS(
			testEnv.ctx,
			testEnv.storageClient,
			path.Join(s.prefix, "existing.txt"),
			"existing"))
	s.logicalQuotaSuite.SetupSuite()
}

func (s *fileCountQuotaSuite) TestFileCountQuota() {
	initialStat := statFS(s.T())
	assert.Equal(s.T(), uint64(2), uint64(initialStat.Files))
	assert.Equal(s.T(), uint64(1), uint64(initialStat.Ffree))

	firstFile := path.Join(setup.MntDir(), "first.txt")
	require.NoError(s.T(), os.WriteFile(firstFile, []byte("first"), 0600))

	fullStat := statFS(s.T())
	assert.Zero(s.T(), uint64(fullStat.Ffree))

	err := os.WriteFile(path.Join(setup.MntDir(), "second.txt"), []byte("second"), 0600)
	assertNoSpaceLeft(s.T(), err)

	require.NoError(s.T(), os.Remove(firstFile))
	recoveredStat := statFS(s.T())
	assert.Equal(s.T(), uint64(1), uint64(recoveredStat.Ffree))

	require.NoError(s.T(), os.WriteFile(path.Join(setup.MntDir(), "after-delete.txt"), []byte("after-delete"), 0600))
}

func TestSizeQuota(t *testing.T) {
	ts := &sizeQuotaSuite{}
	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, ts.flags = range flagsSet {
		log.Printf("Running logical quota size test with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}

func TestFileCountQuota(t *testing.T) {
	ts := &fileCountQuotaSuite{}
	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, ts.flags = range flagsSet {
		log.Printf("Running logical quota file count test with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}

func statFS(t *testing.T) syscall.Statfs_t {
	t.Helper()
	var stat syscall.Statfs_t
	require.NoError(t, syscall.Statfs(setup.MntDir(), &stat))
	return stat
}

func assertNoSpaceLeft(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	if errors.Is(err, syscall.ENOSPC) {
		return
	}
	assert.Contains(t, strings.ToLower(err.Error()), "no space left")
}

func ceilDiv(n int, d uint64) uint64 {
	return (uint64(n) + d - 1) / d
}
