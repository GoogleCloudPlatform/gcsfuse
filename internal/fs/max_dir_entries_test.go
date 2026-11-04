// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Tests for the file-system.max-dir-entries write guard.

package fs_test

import (
	"fmt"
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const maxDirEntriesLimit = 5

type MaxDirEntriesTest struct {
	suite.Suite
	fsTest
}

func TestMaxDirEntriesSuite(t *testing.T) { suite.Run(t, new(MaxDirEntriesTest)) }

func (t *MaxDirEntriesTest) SetupSuite() {
	t.serverCfg.MaxDirEntries = maxDirEntriesLimit
	t.serverCfg.ImplicitDirectories = true
	t.SetUpTestSuite()
}

func (t *MaxDirEntriesTest) TearDownSuite() {
	t.TearDownTestSuite()
}

func (t *MaxDirEntriesTest) TearDownTest() {
	t.TearDown()
}

// fillUntilRejected creates files in dir until a create is rejected, and
// asserts the rejection is ENOSPC. It returns the number of files successfully
// created, which must be between 1 and the limit.
func (t *MaxDirEntriesTest) fillUntilRejected(dir string) int {
	created := 0
	var lastErr error
	for i := 0; i < maxDirEntriesLimit*2; i++ {
		f := path.Join(dir, fmt.Sprintf("f%d.txt", i))
		if err := os.WriteFile(f, []byte("x"), 0600); err != nil {
			lastErr = err
			break
		}
		created++
	}
	require.Error(t.T(), lastErr, "expected a create to be rejected once the directory filled up")
	assert.ErrorIs(t.T(), lastErr, syscall.ENOSPC)
	assert.GreaterOrEqual(t.T(), created, 1)
	assert.LessOrEqual(t.T(), created, maxDirEntriesLimit)
	return created
}

// Once a directory reaches the configured limit, further creates are rejected
// with ENOSPC. The exact boundary index is intentionally not asserted
// (implicit-directory placeholders can shift the count by one); the contract is
// that creation eventually fails with ENOSPC after no more than `limit`
// successful creates.
func (t *MaxDirEntriesTest) TestCreatesBlockedWithENOSPCOnceDirIsFull() {
	dir := path.Join(mntDir, "bigdir")
	require.NoError(t.T(), os.Mkdir(dir, 0700))

	t.fillUntilRejected(dir)
}

// Creating fewer entries than the limit always succeeds.
func (t *MaxDirEntriesTest) TestCreatesAllowedBelowLimit() {
	dir := path.Join(mntDir, "smalldir")
	require.NoError(t.T(), os.Mkdir(dir, 0700))

	for i := 0; i < maxDirEntriesLimit-1; i++ {
		f := path.Join(dir, fmt.Sprintf("f%d.txt", i))
		require.NoError(t.T(), os.WriteFile(f, []byte("x"), 0600))
	}
}

// The guard counts direct entries only, not recursive descendants: a parent
// with a small number of direct entries must still accept creates even when its
// subdirectories collectively hold far more than the limit.
func (t *MaxDirEntriesTest) TestDirectEntryCountIsNotRecursive() {
	parent := path.Join(mntDir, "parent")
	require.NoError(t.T(), os.Mkdir(parent, 0700))

	// A few subdirectories, each filled just under the limit. The parent's
	// direct entry count stays small (only the subdirs), while the recursive
	// descendant count is far above the limit.
	const subDirs = 3
	for d := 0; d < subDirs; d++ {
		sub := path.Join(parent, fmt.Sprintf("sub%d", d))
		require.NoError(t.T(), os.Mkdir(sub, 0700))
		for i := 0; i < maxDirEntriesLimit-1; i++ {
			f := path.Join(sub, fmt.Sprintf("f%d.txt", i))
			require.NoError(t.T(), os.WriteFile(f, []byte("x"), 0600))
		}
	}

	// The parent has only `subDirs` direct entries (< limit), so a create in the
	// parent must succeed even though its recursive descendant count is >> limit.
	// This would fail if the guard counted descendants recursively.
	require.NoError(t.T(), os.WriteFile(path.Join(parent, "direct.txt"), []byte("x"), 0600))
}

// The guard applies to all entry-creating operations, not just plain file
// creates: once a directory is full, MkDir and CreateSymlink into it are also
// rejected with ENOSPC.
func (t *MaxDirEntriesTest) TestMkDirAndSymlinkBlockedWhenDirIsFull() {
	dir := path.Join(mntDir, "fulldir")
	require.NoError(t.T(), os.Mkdir(dir, 0700))
	t.fillUntilRejected(dir)

	// These must be rejected. They are deliberately not cleaned up: if either
	// succeeded (the bug), the leftover entry is the evidence of the failure.
	assert.ErrorIs(t.T(), os.Mkdir(path.Join(dir, "newsub"), 0700), syscall.ENOSPC)
	assert.ErrorIs(t.T(), os.Symlink("target", path.Join(dir, "newlink")), syscall.ENOSPC)
}
