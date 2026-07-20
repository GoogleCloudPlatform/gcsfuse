// Copyright 2023 Google LLC
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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"syscall"
	"testing"
	"unsafe"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/data"
	testutil "github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const FileName = "foo.txt"

func setupUtilTest(t *testing.T) (data.FileSpec, int, int, int) {
	tempDir := t.TempDir()
	fileSpec := data.FileSpec{
		Path:     path.Join(tempDir, "subdir", FileName), // "subdir" does not exist yet
		FilePerm: DefaultFilePerm,
		DirPerm:  DefaultDirPerm,
	}
	flag := os.O_RDWR
	uid := os.Getuid()
	gid := os.Getgid()
	return fileSpec, flag, uid, gid
}

func assertFileAndDirCreationWithGivenDirPerm(t *testing.T, file *os.File, err error, fileSpec data.FileSpec, uid, gid int, dirPerm os.FileMode) {
	require.NoError(t, err)

	dirStat, dirErr := os.Stat(path.Dir(file.Name()))
	assert.False(t, os.IsNotExist(dirErr))
	assert.Equal(t, path.Dir(fileSpec.Path), path.Dir(file.Name()))
	assert.Equal(t, dirPerm, dirStat.Mode().Perm())
	assert.Equal(t, uid, int(dirStat.Sys().(*syscall.Stat_t).Uid))
	assert.Equal(t, gid, int(dirStat.Sys().(*syscall.Stat_t).Gid))

	fileStat, fileErr := os.Stat(file.Name())
	assert.False(t, os.IsNotExist(fileErr))
	assert.Equal(t, fileSpec.Path, file.Name())
	assert.Equal(t, fileSpec.FilePerm, fileStat.Mode())
	assert.Equal(t, uid, int(fileStat.Sys().(*syscall.Stat_t).Uid))
	assert.Equal(t, gid, int(fileStat.Sys().(*syscall.Stat_t).Gid))
}

func Test_CreateFile_FileDirNotPresent(t *testing.T) {
	fileSpec, flag, uid, gid := setupUtilTest(t)

	file, err := CreateFile(fileSpec, flag)
	if err == nil {
		defer func() { assert.NoError(t, file.Close()) }()
	}

	assertFileAndDirCreationWithGivenDirPerm(t, file, err, fileSpec, uid, gid, 0700)
}

func Test_CreateFile_ShouldThrowErrorIfFileDirNotPresentAndProvidedPermissionsAreInsufficient(t *testing.T) {
	fileSpec, flag, _, _ := setupUtilTest(t)
	fileSpec.DirPerm = 0644 // No execute permission, cannot traverse

	_, err := CreateFile(fileSpec, flag)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func Test_CreateFile_FileDirPresent(t *testing.T) {
	fileSpec, flag, uid, gid := setupUtilTest(t)
	err := os.MkdirAll(path.Dir(fileSpec.Path), 0755)
	require.NoError(t, err)

	file, err := CreateFile(fileSpec, flag)
	if err == nil {
		defer func() { assert.NoError(t, file.Close()) }()
	}

	assertFileAndDirCreationWithGivenDirPerm(t, file, err, fileSpec, uid, gid, 0755)
}

func Test_CreateFile_ReadOnlyFile(t *testing.T) {
	fileSpec, _, uid, gid := setupUtilTest(t)
	flag := os.O_RDONLY

	file, err := CreateFile(fileSpec, flag)
	if err == nil {
		defer func() { assert.NoError(t, file.Close()) }()
	}

	assertFileAndDirCreationWithGivenDirPerm(t, file, err, fileSpec, uid, gid, 0700)
	content := "foo"
	_, err = file.Write([]byte(content))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad file descriptor")

	fileContent, err := os.ReadFile(fileSpec.Path)
	assert.NoError(t, err)
	assert.Equal(t, "", string(fileContent))
}

func Test_CreateFile_ReadWriteFile(t *testing.T) {
	fileSpec, flag, uid, gid := setupUtilTest(t)

	file, err := CreateFile(fileSpec, flag)
	if err == nil {
		defer func() { assert.NoError(t, file.Close()) }()
	}

	assertFileAndDirCreationWithGivenDirPerm(t, file, err, fileSpec, uid, gid, 0700)
	content := "foo"
	n, err := file.Write([]byte(content))
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	fileContent, err := os.ReadFile(fileSpec.Path)
	assert.NoError(t, err)
	assert.Equal(t, content, string(fileContent))
}

func Test_CreateFile_FilePerm0755(t *testing.T) {
	fileSpec, flag, uid, gid := setupUtilTest(t)
	fileSpec.FilePerm = os.FileMode(0755)

	file, err := CreateFile(fileSpec, flag)
	if err == nil {
		defer func() { assert.NoError(t, file.Close()) }()
	}

	assertFileAndDirCreationWithGivenDirPerm(t, file, err, fileSpec, uid, gid, 0700)
}

func Test_CreateFile_FilePerm0544(t *testing.T) {
	fileSpec, flag, uid, gid := setupUtilTest(t)
	fileSpec.FilePerm = os.FileMode(0544)

	file, err := CreateFile(fileSpec, flag)
	if err == nil {
		defer func() { assert.NoError(t, file.Close()) }()
	}

	assertFileAndDirCreationWithGivenDirPerm(t, file, err, fileSpec, uid, gid, 0700)
}

func Test_CreateFile_FilePresent(t *testing.T) {
	fileSpec, flag, uid, gid := setupUtilTest(t)
	err := os.MkdirAll(path.Dir(fileSpec.Path), 0755)
	require.NoError(t, err)
	file, err := os.OpenFile(fileSpec.Path, os.O_CREATE|os.O_RDWR, DefaultFilePerm)
	require.NoError(t, err)
	assert.NoError(t, file.Close())

	file, err = CreateFile(fileSpec, flag)
	if err == nil {
		defer func() { assert.NoError(t, file.Close()) }()
	}

	assertFileAndDirCreationWithGivenDirPerm(t, file, err, fileSpec, uid, gid, 0755)
}

func Test_CreateFile_FilePresentWithLessAccess(t *testing.T) {
	fileSpec, flag, _, _ := setupUtilTest(t)
	err := os.MkdirAll(path.Dir(fileSpec.Path), 0755)
	require.NoError(t, err)
	file, err := os.OpenFile(fileSpec.Path, os.O_CREATE, os.FileMode(0544))
	require.NoError(t, err)
	assert.NoError(t, file.Close())

	_, err = CreateFile(fileSpec, flag)

	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "permission denied")
}

func Test_CreateFile_RelativePath(t *testing.T) {
	fileSpec := data.FileSpec{
		Path:     "./some/path/foo.txt",
		FilePerm: DefaultFilePerm,
		DirPerm:  DefaultDirPerm,
	}
	flag := os.O_RDWR
	uid := os.Getuid()
	gid := os.Getgid()

	defer func() { _ = os.RemoveAll("./some") }()

	file, err := CreateFile(fileSpec, flag)
	if err == nil {
		defer func() { assert.NoError(t, file.Close()) }()
	}

	assertFileAndDirCreationWithGivenDirPerm(t, file, err, fileSpec, uid, gid, 0700)
}

func Test_getObjectPath(t *testing.T) {
	inputs := [][]string{{"", ""}, {"a", "b"}, {"a/b/", "/c/d"}, {"", "a"}, {"a", ""}}
	expectedOutPuts := [5]string{"", "a/b", "a/b/c/d", "a", "a"}
	results := [5]string{}

	for i := range 5 {
		results[i] = GetObjectPath(inputs[i][0], inputs[i][1])
	}

	assert.Equal(t, expectedOutPuts, results)
}

func Test_getDownloadPath(t *testing.T) {
	// Arrange
	inputs := []string{"a/b", "a/b/c/d", "/a", "a/"}
	cacheDir := "/test/dir"
	expectedOutputs := [4]string{
		cacheDir + "/a/b",
		cacheDir + "/a/b/c/d",
		cacheDir + "/a",
		cacheDir + "/a",
	}
	results := [4]string{}

	// Act
	for i := range len(inputs) {
		path, err := GetDownloadPath(cacheDir, inputs[i])
		require.NoError(t, err)
		results[i] = path
	}

	// Assert
	assert.Equal(t, expectedOutputs, results)
}

func Test_getDownloadPath_EscapingPath(t *testing.T) {
	// Arrange
	cacheDir := "/test/dir"
	badInputs := []string{"/", "../etc", "a/../../etc", "../../etc/cron.d/pwn"}

	for _, input := range badInputs {
		// Act
		_, err := GetDownloadPath(cacheDir, input)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is outside cache directory")
	}
}

func Test_IsCacheHandleValid_True(t *testing.T) {
	errs := []error{
		fmt.Errorf("%w: %s", ErrInvalidFileHandle, "test"),
		fmt.Errorf("%w: %s", ErrInvalidFileDownloadJob, "test"),
		fmt.Errorf("%w: %s", ErrInvalidFileInfoCache, "test"),
		fmt.Errorf("%w: %s", ErrInReadingFileHandle, "test"),
	}

	for _, err := range errs {
		assert.True(t, IsCacheHandleInvalid(err))
	}
}

func Test_IsCacheHandleValid_False(t *testing.T) {
	errs := []error{
		fmt.Errorf("%w: %s", ErrFallbackToGCS, "test"),
		fmt.Errorf("random error message"),
	}

	for _, err := range errs {
		assert.False(t, IsCacheHandleInvalid(err))
	}
}

func Test_CalculateFileCRC32_ShouldReturnCrcForValidFile(t *testing.T) {
	crc, err := CalculateFileCRC32(context.Background(), "testdata/validfile.txt")

	assert.NoError(t, err)
	assert.Equal(t, uint32(515179668), crc)
}

func Test_CalculateFileCRC32_ShouldReturnZeroForEmptyFile(t *testing.T) {
	crc, err := CalculateFileCRC32(context.Background(), "testdata/emptyfile.txt")

	assert.NoError(t, err)
	assert.Equal(t, uint32(0), crc)
}

func Test_CalculateFileCRC32_ShouldReturnErrorForFileNotExist(t *testing.T) {
	crc, err := CalculateFileCRC32(context.Background(), "testdata/nofile.txt")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
	assert.Equal(t, uint32(0), crc)
}

func Test_CalculateFileCRC32_ShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc()
	crc, err := CalculateFileCRC32(ctx, "testdata/validfile.txt")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
	assert.Contains(t, err.Error(), "CRC computation is cancelled")
	assert.Equal(t, uint32(0), crc)
}

func Test_TruncateAndRemoveFile_FileExists(t *testing.T) {
	tempDir := t.TempDir()
	fileName := path.Join(tempDir, "temp.txt")
	file, err := os.Create(fileName)
	require.NoError(t, err)
	_, err = file.WriteString("Writing some data")
	require.NoError(t, err)
	err = file.Close()
	require.NoError(t, err)

	err = TruncateAndRemoveFile(fileName)

	assert.NoError(t, err)
	_, err = os.Stat(fileName)
	assert.True(t, os.IsNotExist(err))
}

func Test_TruncateAndRemoveFile_FileDoesNotExist(t *testing.T) {
	tempDir := t.TempDir()
	fileName := path.Join(tempDir, "temp.txt")

	err := TruncateAndRemoveFile(fileName)

	assert.True(t, os.IsNotExist(err))
}

func Test_TruncateAndRemoveFile_OpenedFileDeleted(t *testing.T) {
	tempDir := t.TempDir()
	fileName := path.Join(tempDir, "temp.txt")
	file, err := os.Create(fileName)
	require.NoError(t, err)
	fileString := "Writing some data"
	_, err = file.WriteString(fileString)
	require.NoError(t, err)
	err = file.Close()
	require.NoError(t, err)

	file, err = os.Open(fileName)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = file.Close()
	})
	fileInfo, err := file.Stat()
	require.NoError(t, err)
	assert.Equal(t, int64(len(fileString)), fileInfo.Size())

	err = TruncateAndRemoveFile(fileName)

	assert.NoError(t, err)
	fileInfo, err = file.Stat()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), fileInfo.Size())
}

func Test_CreateCacheDirectoryIfNotPresentAt_ShouldNotReturnAnyErrorWhenDirectoryExists(t *testing.T) {
	base := path.Join("./", string(testutil.GenerateRandomBytes(4)))
	dirPath := path.Join(base, "/", "path/cachedir")
	dirCreationErr := os.MkdirAll(dirPath, 0700)
	defer func() { _ = os.RemoveAll(base) }()
	require.NoError(t, dirCreationErr)

	err := CreateCacheDirectoryIfNotPresentAt(dirPath, 0000)

	require.NoError(t, err)
	fileInfo, err := os.Stat(dirPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), fileInfo.Mode().Perm())
}

func Test_CreateCacheDirectoryIfNotPresentAt_ShouldNotReturnAnyErrorWhenDirectoryCanBeCreatedWithOwnerPermissions(t *testing.T) {
	base := path.Join("./", string(testutil.GenerateRandomBytes(4)))
	dirPath := path.Join(base, "/", "path/cachedir")
	defer func() { _ = os.RemoveAll(base) }()

	err := CreateCacheDirectoryIfNotPresentAt(dirPath, 0700)

	require.NoError(t, err)
	fileInfo, err := os.Stat(dirPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), fileInfo.Mode().Perm())
}

func Test_CreateCacheDirectoryIfNotPresentAt_ShouldNotReturnAnyErrorWhenDirectoryCanBeCreatedWithOthersPermissions(t *testing.T) {
	base := path.Join("./", string(testutil.GenerateRandomBytes(4)))
	dirPath := path.Join(base, "/", "path/cachedir")
	defer func() { _ = os.RemoveAll(base) }()

	err := CreateCacheDirectoryIfNotPresentAt(dirPath, 0755)

	require.NoError(t, err)
	fileInfo, err := os.Stat(dirPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), fileInfo.Mode().Perm())
}

func Test_CreateCacheDirectoryIfNotPresentAt_ShouldReturnErrorWhenDirectoryDoesNotHavePermissions(t *testing.T) {
	dirPath := path.Join("./", string(testutil.GenerateRandomBytes(4)))
	dirCreationErr := os.MkdirAll(dirPath, 0444)
	defer func() { _ = os.RemoveAll(dirPath) }()
	require.NoError(t, dirCreationErr)

	err := CreateCacheDirectoryIfNotPresentAt(dirPath, 0755)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error creating file at directory ("+dirPath+")")
}

func Test_GetMemoryAlignedBuffer(t *testing.T) {
	tbl := []struct {
		name                string
		bufferSize          int64
		alignSize           int64
		expectedBufferSize  int64
		bufferPtrMultipleOf int64
	}{
		{
			name:                "buffer size and align size 0",
			bufferSize:          0,
			alignSize:           0,
			expectedBufferSize:  0,
			bufferPtrMultipleOf: 1,
		},
		{
			name:                "buffer size 0 and align size non 0",
			bufferSize:          0,
			alignSize:           4096,
			expectedBufferSize:  0,
			bufferPtrMultipleOf: 1,
		},
		{
			name:                "buffer size non 0 and align size 0",
			bufferSize:          4096,
			alignSize:           0,
			expectedBufferSize:  4096,
			bufferPtrMultipleOf: 1,
		},
		{
			name:                "buffer size and align size non 0",
			bufferSize:          1024 * 1024,
			alignSize:           4096,
			expectedBufferSize:  1024 * 1024,
			bufferPtrMultipleOf: 4096,
		},
		{
			name:                "buffer size and align size power of 2",
			bufferSize:          65536,
			alignSize:           2048,
			expectedBufferSize:  65536,
			bufferPtrMultipleOf: 2048,
		},
		{
			name:                "buffer size and align size odd",
			bufferSize:          7,
			alignSize:           13,
			expectedBufferSize:  7,
			bufferPtrMultipleOf: 13,
		},
		{
			name:                "buffer size smaller than align size",
			bufferSize:          200,
			alignSize:           4096,
			expectedBufferSize:  200,
			bufferPtrMultipleOf: 4096,
		},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc
			t.Parallel()

			buffer, err := GetMemoryAlignedBuffer(tc.bufferSize, tc.alignSize)

			assert.NoError(t, err)
			assert.Equal(t, int(tc.expectedBufferSize), len(buffer))
			if tc.expectedBufferSize == 0 {
				assert.Equal(t, 0, len(buffer))
			} else {
				assert.Equal(t, uint64(0), uint64(uintptr(unsafe.Pointer(&buffer[0]))%uintptr(tc.bufferPtrMultipleOf)))
			}
		})
	}
}

func Test_CopyUsingMemoryAlignedBuffer(t *testing.T) {
	tbl := []struct {
		name              string
		bufferSize        int64
		contentSize       int64
		useODIRECT        bool
		cancelCtx         bool
		writeOffset       int64
		expectedErr       bool
		expectedWriteSize int64
	}{
		{
			name:              "buffer size less than 4096",
			bufferSize:        2048,
			contentSize:       0,
			useODIRECT:        true,
			expectedErr:       true,
			expectedWriteSize: 0,
		},
		{
			name:              "buffer size not multiple of 4096",
			bufferSize:        2048 * 3,
			contentSize:       0,
			useODIRECT:        true,
			expectedErr:       true,
			expectedWriteSize: 0,
		},
		{
			name:              "buffer size and content size are equal",
			bufferSize:        8192,
			contentSize:       8192,
			useODIRECT:        true,
			expectedErr:       false,
			expectedWriteSize: 8192,
		},
		{
			name:              "content size multiple of 4096",
			bufferSize:        4096,
			contentSize:       1024 * 1024,
			useODIRECT:        true,
			writeOffset:       4096,
			expectedErr:       false,
			expectedWriteSize: 1024 * 1024,
		},
		{
			name:              "content size not multiple of 4096",
			bufferSize:        4096,
			contentSize:       1024*1024 + 1,
			useODIRECT:        true,
			expectedErr:       false,
			expectedWriteSize: 1024*1024 + 4096,
		},
		{
			name:              "buffer size and content size are not equal",
			bufferSize:        1024 * 1024,
			contentSize:       1024*1024 - 10,
			useODIRECT:        true,
			expectedErr:       false,
			expectedWriteSize: 1024 * 1024,
		},
		{
			name:              "writer offset multiple of 4096",
			bufferSize:        1024 * 1024,
			contentSize:       2*1024*1024 - 10,
			useODIRECT:        true,
			writeOffset:       1024 * 1024,
			expectedErr:       false,
			expectedWriteSize: 2 * 1024 * 1024,
		},
		{
			name:              "writer offset not multiple of 4096",
			bufferSize:        1024 * 1024,
			contentSize:       1024*1024 - 10,
			useODIRECT:        true,
			writeOffset:       1024*1024 - 1,
			expectedErr:       true,
			expectedWriteSize: 0,
		},
		{
			name:              "not use O_DIRECT",
			bufferSize:        1024 * 1024,
			contentSize:       2*1024*1024 - 10,
			useODIRECT:        false,
			expectedErr:       false,
			expectedWriteSize: 2 * 1024 * 1024,
		},
		{
			name:              "context canceled",
			bufferSize:        1024 * 1024,
			contentSize:       2*1024*1024 - 10,
			useODIRECT:        false,
			cancelCtx:         true,
			expectedErr:       true,
			expectedWriteSize: 0,
		},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc
			t.Parallel()
			randName := string(testutil.GenerateRandomBytes(10))
			flags := os.O_CREATE | os.O_TRUNC | os.O_RDWR
			if tc.useODIRECT {
				flags = flags | syscall.O_DIRECT
			}
			file, err := os.OpenFile(randName, flags, 0600)
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = file.Close()
				_ = os.Remove(randName)
			})
			content := testutil.GenerateRandomBytes(int(tc.contentSize))
			src := bytes.NewReader(content)
			ctx, cancelCtx := context.WithCancel(context.Background())
			defer cancelCtx()
			if tc.cancelCtx {
				cancelCtx()
			}
			dst := io.NewOffsetWriter(file, int64(tc.writeOffset))

			writeN, err := CopyUsingMemoryAlignedBuffer(ctx, src, dst, tc.contentSize, tc.bufferSize)

			if tc.expectedErr {
				assert.NotNil(t, err)
				if tc.cancelCtx {
					assert.ErrorIs(t, err, context.Canceled)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedWriteSize, writeN)
				fileStat, err := file.Stat()
				require.NoError(t, err)
				assert.Equal(t, tc.writeOffset+writeN, fileStat.Size())
				// Match only the content written.
				sizeToMatch := min(tc.contentSize, writeN, tc.expectedWriteSize)
				buf := make([]byte, sizeToMatch)
				// Open file again without O_DIRECT
				readFile, err := os.OpenFile(randName, os.O_RDWR, 0600)
				require.NoError(t, err)
				t.Cleanup(func() {
					_ = readFile.Close()
				})
				_, err = readFile.ReadAt(buf, tc.writeOffset)
				if err != nil && err != io.EOF {
					t.Errorf("error (%v) while reading contents at the time of assertion for: %v", err, tc.name)
				}
				assert.Equal(t, content[:sizeToMatch], buf)
			}
		})
	}
}
