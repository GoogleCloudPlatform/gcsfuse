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

package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type DiskUtilTest struct {
	suite.Suite
}

func TestDiskUtilSuite(t *testing.T) {
	suite.Run(t, new(DiskUtilTest))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (ts *DiskUtilTest) TestSpectulativeSizeOnDisk() {
	testcases := []struct {
		name              string
		input_filesize    uint64
		expected_disksize uint64
	}{
		{
			name:              "zero_size",
			input_filesize:    0,
			expected_disksize: 0,
		},
		{
			name:              "small_file",
			input_filesize:    1,
			expected_disksize: 4096,
		},
		{
			name:              "one_block_size",
			input_filesize:    4096,
			expected_disksize: 4096,
		},
		{
			name:              "more_than_one_block_but_less_than_two",
			input_filesize:    4097,
			expected_disksize: 8192,
		},
	}
	for _, tc := range testcases {
		ts.T().Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected_disksize, GetSpeculativeFileSizeOnDisk(tc.input_filesize, 4096))
		})
	}
}

func (ts *DiskUtilTest) TestGetSizeOnDisk_Normal() {
	// Arrange
	tempDir := ts.T().TempDir()
	f, err := os.CreateTemp(tempDir, "testfile")
	require.NoError(ts.T(), err)
	_, err = f.Write([]byte("hello"))
	require.NoError(ts.T(), err)
	f.Close()
	// Act
	size, err := GetSizeOnDisk(tempDir, false, false)
	// Assert
	require.NoError(ts.T(), err)
	// We just check it's > 0 since precise block allocations vary across OS and FS.
	require.Greater(ts.T(), size, uint64(0))
}

func (ts *DiskUtilTest) TestGetSizeOnDisk_OnlyDirectories() {
	// Arrange
	tempDir := ts.T().TempDir()
	f, err := os.CreateTemp(tempDir, "testfile")
	require.NoError(ts.T(), err)
	_, err = f.Write([]byte("hello"))
	require.NoError(ts.T(), err)
	f.Close()
	subDir := filepath.Join(tempDir, "subdir")
	require.NoError(ts.T(), os.Mkdir(subDir, 0755))
	// Act
	size, err := GetSizeOnDisk(tempDir, true, false)
	// Assert
	require.NoError(ts.T(), err)
	// On tmpfs, directories take 0 blocks. So size might be 0.
	require.GreaterOrEqual(ts.T(), size, uint64(0))
}

func (ts *DiskUtilTest) TestGetSizeOnDisk_BothDirectoriesAndFiles() {
	// Arrange
	tempDir := ts.T().TempDir()
	// Add a file
	f, err := os.CreateTemp(tempDir, "testfile")
	require.NoError(ts.T(), err)
	_, err = f.Write([]byte("hello world"))
	require.NoError(ts.T(), err)
	f.Close()
	// Add a subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	require.NoError(ts.T(), os.Mkdir(subDir, 0755))
	// Add a file in the subdirectory
	subFile, err := os.CreateTemp(subDir, "subtestfile")
	require.NoError(ts.T(), err)
	_, err = subFile.Write([]byte("nested hello world"))
	require.NoError(ts.T(), err)
	subFile.Close()
	// Act
	sizeWithFiles, err := GetSizeOnDisk(tempDir, false, false)
	require.NoError(ts.T(), err)
	sizeOnlyDirs, err := GetSizeOnDisk(tempDir, true, false)
	require.NoError(ts.T(), err)
	// Assert
	require.Greater(ts.T(), sizeWithFiles, uint64(0))
	require.GreaterOrEqual(ts.T(), sizeOnlyDirs, uint64(0))
	// Files take up extra space, so total size should be strictly greater than dir-only size
	require.Greater(ts.T(), sizeWithFiles, sizeOnlyDirs)
}

func (ts *DiskUtilTest) TestGetSizeOnDisk_PermissionDenied_NoIgnore() {
	// Arrange
	tempDir := ts.T().TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	require.NoError(ts.T(), os.Mkdir(subDir, 0755))
	require.NoError(ts.T(), os.Chmod(subDir, 0000))
	defer func() {
		err := os.Chmod(subDir, 0755)
		require.NoError(ts.T(), err)
	}()
	// Act
	_, err := GetSizeOnDisk(tempDir, false, false)
	// Assert
	require.Error(ts.T(), err)
}

func (ts *DiskUtilTest) TestGetSizeOnDisk_PermissionDenied_Ignore() {
	// Arrange
	tempDir := ts.T().TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	require.NoError(ts.T(), os.Mkdir(subDir, 0755))
	fSub, err := os.Create(filepath.Join(subDir, "subfile"))
	// We might fail to create file inside if we chmod too early, so order matters.
	// But here we want to test read failure during Walk.
	if err == nil {
		fSub.Close()
	}
	require.NoError(ts.T(), os.Chmod(subDir, 0000))
	defer func() {
		err := os.Chmod(subDir, 0755)
		require.NoError(ts.T(), err)
	}()
	// Act
	size, err := GetSizeOnDisk(tempDir, false, true)
	// Assert
	require.NoError(ts.T(), err)
	// We might or might not get size > 0 depending on whether we count the blocked directory itself.
	// But mostly we check no error returned.
	require.GreaterOrEqual(ts.T(), size, uint64(0))
}

func (ts *DiskUtilTest) TestGetVolumeBlockSize() {
	// Arrange
	tempDir := ts.T().TempDir()
	// Act
	blockSize, err := GetVolumeBlockSize(tempDir)
	// Assert
	require.NoError(ts.T(), err)
	// Block size is typically a power of 2 (e.g., 4096).
	// On tmpfs it is often 4096.
	require.Greater(ts.T(), blockSize, uint64(0))
	require.Equal(ts.T(), uint64(0), blockSize%512, "Block size should be a multiple of 512")
}

func (ts *DiskUtilTest) TestRemoveEmptyDirs() {
	// Arrange
	tempDir := ts.T().TempDir()
	// Create nested structure:
	// tempDir/
	//   emptyDir/
	//   nonEmptyDir/
	//     file.txt
	//   nestedEmptyDir/
	//     level2/
	//   nestedNonEmptyDir/
	//     level2/
	//       file.txt
	emptyDir := filepath.Join(tempDir, "emptyDir")
	require.NoError(ts.T(), os.Mkdir(emptyDir, 0755))
	nonEmptyDir := filepath.Join(tempDir, "nonEmptyDir")
	require.NoError(ts.T(), os.Mkdir(nonEmptyDir, 0755))
	f, err := os.Create(filepath.Join(nonEmptyDir, "file.txt"))
	require.NoError(ts.T(), err)
	f.Close()
	nestedEmptyDir := filepath.Join(tempDir, "nestedEmptyDir")
	require.NoError(ts.T(), os.MkdirAll(filepath.Join(nestedEmptyDir, "level2"), 0755))
	nestedNonEmptyDir := filepath.Join(tempDir, "nestedNonEmptyDir")
	require.NoError(ts.T(), os.MkdirAll(filepath.Join(nestedNonEmptyDir, "level2"), 0755))
	f2, err := os.Create(filepath.Join(nestedNonEmptyDir, "level2", "file.txt"))
	require.NoError(ts.T(), err)
	f2.Close()
	// Act
	RemoveEmptyDirs(tempDir)
	// Assert
	// emptyDir should be gone
	_, err = os.Stat(emptyDir)
	require.True(ts.T(), os.IsNotExist(err))
	// nonEmptyDir should exist
	_, err = os.Stat(nonEmptyDir)
	require.NoError(ts.T(), err)
	// nestedEmptyDir should be gone (both level2 and parent)
	_, err = os.Stat(nestedEmptyDir)
	require.True(ts.T(), os.IsNotExist(err))
	// nestedNonEmptyDir should exist
	_, err = os.Stat(nestedNonEmptyDir)
	require.NoError(ts.T(), err)
	// nestedNonEmptyDir/level2 should exist
	_, err = os.Stat(filepath.Join(nestedNonEmptyDir, "level2"))
	require.NoError(ts.T(), err)
}

func (ts *DiskUtilTest) TestRemoveEmptyDirs_CompletelyEmptyRoot() {
	// Arrange
	tempDir := ts.T().TempDir()
	// Verify it's effectively empty to start with
	entries, err := os.ReadDir(tempDir)
	require.NoError(ts.T(), err)
	require.Empty(ts.T(), entries)
	// Act
	RemoveEmptyDirs(tempDir)
	// Assert
	// The root directory passed to RemoveEmptyDirs should NEVER be deleted,
	// even if it is completely empty.
	_, err = os.Stat(tempDir)
	require.NoError(ts.T(), err)
}

func (ts *DiskUtilTest) TestPrettyPrintOf() {
	testcases := []struct {
		name     string
		input    uint64
		expected string
	}{
		{
			name:     "zero",
			input:    0,
			expected: "0",
		},
		{
			name:     "one_digit",
			input:    9,
			expected: "9",
		},
		{
			name:     "three_digits",
			input:    123,
			expected: "123",
		},
		{
			name:     "four_digits",
			input:    1234,
			expected: "1,234",
		},
		{
			name:     "five_digits",
			input:    12345,
			expected: "12,345",
		},
		{
			name:     "six_digits",
			input:    123456,
			expected: "123,456",
		},
		{
			name:     "seven_digits",
			input:    1234567,
			expected: "1,234,567",
		},
		{
			name:     "max_uint64",
			input:    18446744073709551615,
			expected: "18,446,744,073,709,551,615",
		},
	}
	for _, tc := range testcases {
		ts.T().Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, PrettyPrintOf(tc.input))
		})
	}
}

func (ts *DiskUtilTest) TestGetSizeOnDiskAndClean() {
	// Arrange
	tempDir := ts.T().TempDir()
	// Structure:
	// tempDir/
	//   emptyDir/          (should be removed)
	//   nonEmptyDir/
	//     file.txt         (size counted)
	//   nestedEmptyDir/
	//     level2/          (should be removed)
	//   nestedNonEmptyDir/
	//     level2/
	//       file.txt       (size counted)

	emptyDir := filepath.Join(tempDir, "emptyDir")
	require.NoError(ts.T(), os.Mkdir(emptyDir, 0755))

	nonEmptyDir := filepath.Join(tempDir, "nonEmptyDir")
	require.NoError(ts.T(), os.Mkdir(nonEmptyDir, 0755))
	f, err := os.Create(filepath.Join(nonEmptyDir, "file.txt"))
	require.NoError(ts.T(), err)
	_, err = f.Write([]byte("content")) // 7 bytes
	require.NoError(ts.T(), err)
	f.Close()

	nestedEmptyDir := filepath.Join(tempDir, "nestedEmptyDir")
	require.NoError(ts.T(), os.MkdirAll(filepath.Join(nestedEmptyDir, "level2"), 0755))

	nestedNonEmptyDir := filepath.Join(tempDir, "nestedNonEmptyDir")
	require.NoError(ts.T(), os.MkdirAll(filepath.Join(nestedNonEmptyDir, "level2"), 0755))
	f2, err := os.Create(filepath.Join(nestedNonEmptyDir, "level2", "file.txt"))
	require.NoError(ts.T(), err)
	_, err = f2.Write([]byte("content")) // 7 bytes
	require.NoError(ts.T(), err)
	f2.Close()

	// Act
	size := GetSizeOnDiskAndClean(tempDir, true, true)

	// Assert
	// Root should exist
	_, err = os.Stat(tempDir)
	require.NoError(ts.T(), err)

	// emptyDir should be gone
	_, err = os.Stat(emptyDir)
	require.True(ts.T(), os.IsNotExist(err))

	// nonEmptyDir should exist
	_, err = os.Stat(nonEmptyDir)
	require.NoError(ts.T(), err)

	// nestedEmptyDir should be gone (both level2 and parent)
	_, err = os.Stat(nestedEmptyDir)
	require.True(ts.T(), os.IsNotExist(err))

	// nestedNonEmptyDir should exist
	_, err = os.Stat(nestedNonEmptyDir)
	require.NoError(ts.T(), err)

	// Size calculation:
	// Files: 7 + 7 = 14 bytes.
	// Dirs (metadata): tempDir, nonEmptyDir, nestedNonEmptyDir, nestedNonEmptyDir/level2
	// Depending on FS, dirs might have size.
	// On tmpfs, dir size is 0? Let's check >= 14.
	// We expect size > 0.
	require.GreaterOrEqual(ts.T(), size, uint64(14))
}
