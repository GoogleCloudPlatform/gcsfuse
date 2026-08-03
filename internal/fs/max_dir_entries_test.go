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
	t.fsTest.SetUpTestSuite()
}

func (t *MaxDirEntriesTest) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func (t *MaxDirEntriesTest) TearDownTest() {
	t.fsTest.TearDown()
}

// Once a directory reaches the configured limit, further creates must be
// rejected with ENOSPC. The exact boundary index is intentionally not asserted
// (implicit-directory placeholders can shift the count by one); the contract is
// that creation eventually fails with ENOSPC after no more than `limit`
// successful creates.
func (t *MaxDirEntriesTest) TestCreatesBlockedWithENOSPCOnceDirIsFull() {
	dir := path.Join(mntDir, "bigdir")
	require.NoError(t.T(), os.Mkdir(dir, 0700))

	var lastErr error
	created := 0
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
}

// Creating fewer entries than the limit must always succeed.
func (t *MaxDirEntriesTest) TestCreatesAllowedBelowLimit() {
	dir := path.Join(mntDir, "smalldir")
	require.NoError(t.T(), os.Mkdir(dir, 0700))

	for i := 0; i < maxDirEntriesLimit-1; i++ {
		f := path.Join(dir, fmt.Sprintf("f%d.txt", i))
		require.NoError(t.T(), os.WriteFile(f, []byte("x"), 0600))
	}
}
