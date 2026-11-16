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

package fs_test

import (
	"io"
	"os"
	"path"
	"strings"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v4/cfg"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v4/metrics"
	"github.com/googlecloudplatform/gcsfuse/v4/tracing"
	"github.com/jacobsa/fuse/fusetesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DisableSymlinksTest struct {
	suite.Suite
	fsTest
}

func TestDisableSymlinksTestSuite(t *testing.T) {
	suite.Run(t, new(DisableSymlinksTest))
}

func (t *DisableSymlinksTest) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.NewConfig = &cfg.Config{
		FileSystem: cfg.FileSystemConfig{
			DisableSymlinks:               true,
			ExperimentalEnableReaddirplus: true,
		},
		EnableStandardSymlinks: true,
	}
	t.serverCfg.MetricHandle = metrics.NewNoopMetrics()
	t.serverCfg.TraceHandle = tracing.NewNoopTracer()
	t.fsTest.SetUpTestSuite()
}

func (t *DisableSymlinksTest) TearDownTest() {
	t.fsTest.TearDown()
}

func (t *DisableSymlinksTest) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func (t *DisableSymlinksTest) TestCreateSymlinkReturnsENOTSUP() {
	symlinkPath := path.Join(mntDir, "new_symlink")

	err := os.Symlink("/etc/passwd", symlinkPath)

	require.Error(t.T(), err)
	assert.ErrorIs(t.T(), err, syscall.ENOTSUP)
}

func (t *DisableSymlinksTest) TestExistingLegacySymlinkTreatedAsRegularFile() {
	req := &gcs.CreateObjectRequest{
		Name: "legacy_link",
		Metadata: map[string]string{
			inode.SymlinkMetadataKey: "/etc/passwd",
		},
		Contents: io.NopCloser(strings.NewReader("")),
	}
	_, err := bucket.CreateObject(ctx, req)
	require.NoError(t.T(), err)
	linkPath := path.Join(mntDir, "legacy_link")

	// Lstat should report a regular file, not a symlink.
	fi, err := os.Lstat(linkPath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), "legacy_link", fi.Name())
	assert.Equal(t.T(), int64(0), fi.Size())
	assert.Equal(t.T(), filePerms, fi.Mode())
	assert.Zero(t.T(), fi.Mode()&os.ModeSymlink)

	// Readlink should fail with EINVAL because the kernel sees a regular file.
	_, err = os.Readlink(linkPath)
	require.Error(t.T(), err)
	assert.ErrorIs(t.T(), err, syscall.EINVAL)

	// Reading the file should return the 0B GCS object content, NOT /etc/passwd.
	content, err := os.ReadFile(linkPath)
	require.NoError(t.T(), err)
	assert.Empty(t.T(), content)

	// ReadDirPlus should report regular file mode (no ModeSymlink).
	entries, err := fusetesting.ReadDirPlusPicky(mntDir)
	require.NoError(t.T(), err)
	require.Len(t.T(), entries, 1)
	assert.Equal(t.T(), "legacy_link", entries[0].Name())
	assert.Equal(t.T(), filePerms, entries[0].Mode())
	assert.Zero(t.T(), entries[0].Mode()&os.ModeSymlink)
}

func (t *DisableSymlinksTest) TestExistingStandardSymlinkTreatedAsRegularFile() {
	target := "/etc/passwd"
	req := &gcs.CreateObjectRequest{
		Name: "standard_link",
		Metadata: map[string]string{
			inode.StandardSymlinkMetadataKey: "true",
			inode.SymlinkMetadataKey:         target,
		},
		Contents: io.NopCloser(strings.NewReader(target)),
	}
	_, err := bucket.CreateObject(ctx, req)
	require.NoError(t.T(), err)
	linkPath := path.Join(mntDir, "standard_link")

	// Lstat should report a regular file with the size of the object content.
	fi, err := os.Lstat(linkPath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), "standard_link", fi.Name())
	assert.Equal(t.T(), int64(len(target)), fi.Size())
	assert.Equal(t.T(), filePerms, fi.Mode())
	assert.Zero(t.T(), fi.Mode()&os.ModeSymlink)

	// Readlink should fail with EINVAL because the kernel sees a regular file.
	_, err = os.Readlink(linkPath)
	require.Error(t.T(), err)
	assert.ErrorIs(t.T(), err, syscall.EINVAL)

	// Reading the file should return the literal target string stored in GCS, NOT the contents of /etc/passwd.
	content, err := os.ReadFile(linkPath)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), target, string(content))

	// ReadDirPlus should report regular file mode (no ModeSymlink).
	entries, err := fusetesting.ReadDirPlusPicky(mntDir)
	require.NoError(t.T(), err)
	require.Len(t.T(), entries, 1)
	assert.Equal(t.T(), "standard_link", entries[0].Name())
	assert.Equal(t.T(), filePerms, entries[0].Mode())
	assert.Zero(t.T(), entries[0].Mode()&os.ModeSymlink)
}
