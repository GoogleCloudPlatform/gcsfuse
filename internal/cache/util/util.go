// Copyright 2023 Google Inc. All Rights Reserved.
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
	"context"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jacobsa/fuse/fsutil"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/data"
)

const (
	InvalidFileHandleErrMsg                   = "invalid file handle"
	InvalidFileDownloadJobErrMsg              = "invalid download job"
	InvalidCacheHandleErrMsg                  = "invalid cache handle"
	InvalidFileInfoCacheErrMsg                = "invalid file info cache"
	ErrInSeekingFileHandleMsg                 = "error while seeking file handle"
	ErrInReadingFileHandleMsg                 = "error while reading file handle"
	FallbackToGCSErrMsg                       = "read via gcs"
	FileNotPresentInCacheErrMsg               = "file is not present in cache"
	CacheHandleNotRequiredForRandomReadErrMsg = "cacheFileForRangeRead is false, read type random read and fileInfo entry is absent"
	BufferSizeForCRC                          = 65536
)

const (
	MiB             = 1024 * 1024
	KiB             = 1024
	DefaultFilePerm = os.FileMode(0600)
	DefaultDirPerm  = os.FileMode(0700)
	FileCache       = "gcsfuse-file-cache"
)

// CreateFile creates file with given file spec i.e. permissions and returns
// file handle for that file opened with given flag.
//
// Note: If directories in path are not present, they are created with directory permissions provided in fileSpec
// permission.
func CreateFile(fileSpec data.FileSpec, flag int) (file *os.File, err error) {
	// Create directory structure if not present
	fileDir := filepath.Dir(fileSpec.Path)
	err = os.MkdirAll(fileDir, fileSpec.DirPerm)
	if err != nil {
		err = fmt.Errorf(fmt.Sprintf("error in creating directory structure %s: %v", fileDir, err))
		return
	}

	// Create file if not present.
	_, err = os.Stat(fileSpec.Path)
	if err != nil {
		if os.IsNotExist(err) {
			flag = flag | os.O_CREATE
		} else {
			err = fmt.Errorf(fmt.Sprintf("error in stating file %s: %v", fileSpec.Path, err))
			return
		}
	}
	file, err = os.OpenFile(fileSpec.Path, flag, fileSpec.FilePerm)
	if err != nil {
		err = fmt.Errorf(fmt.Sprintf("error in creating file %s: %v", fileSpec.Path, err))
		return
	}
	return
}

// GetObjectPath gives object path which is concatenation of bucket and object
// name separated by "/".
func GetObjectPath(bucketName string, objectName string) string {
	return path.Join(bucketName, objectName)
}

// GetDownloadPath gives file path to file in cache for given object path.
func GetDownloadPath(cacheDir string, objectPath string) string {
	return path.Join(cacheDir, objectPath)
}

// IsCacheHandleInvalid says either the current cacheHandle is invalid or not, based
// on the error we got while reading with the cacheHandle.
// If it's invalid then we should close that cacheHandle and create new cacheHandle
// for next call onwards.
func IsCacheHandleInvalid(readErr error) bool {
	return strings.Contains(readErr.Error(), InvalidFileHandleErrMsg) ||
		strings.Contains(readErr.Error(), InvalidFileDownloadJobErrMsg) ||
		strings.Contains(readErr.Error(), InvalidFileInfoCacheErrMsg) ||
		strings.Contains(readErr.Error(), ErrInSeekingFileHandleMsg) ||
		strings.Contains(readErr.Error(), ErrInReadingFileHandleMsg)
}

// CreateCacheDirectoryIfNotPresentAt Creates directory at given path with
// provided permissions in case not already present, returns error in case
// unable to create directory or directory is not writable.
func CreateCacheDirectoryIfNotPresentAt(dirPath string, dirPerm os.FileMode) error {
	_, statErr := os.Stat(dirPath)

	if os.IsNotExist(statErr) {
		err := os.MkdirAll(dirPath, dirPerm)
		if err != nil {
			return fmt.Errorf("error in creating directory structure %s: %v", dirPath, err)
		}
	}

	f, err := fsutil.AnonymousFile(dirPath)
	if err != nil {
		return fmt.Errorf(
			"error creating file at directory (%s), error : (%v)", dirPath, err.Error())
	}

	tempFileErr := f.Close()
	if tempFileErr != nil {
		return fmt.Errorf(
			"error closing annonymous temp file, error : (%v)", tempFileErr.Error())
	}

	return nil
}

func calculateCRC32(ctx context.Context, reader io.Reader) (uint32, error) {
	table := crc32.MakeTable(crc32.Castagnoli)
	checksum := crc32.Checksum([]byte(""), table)
	buf := make([]byte, BufferSizeForCRC)
	for {
		select {
		case <-ctx.Done():
			return 0, fmt.Errorf("CRC computation is cancelled: %w", ctx.Err())
		default:
			switch n, err := reader.Read(buf); err {
			case nil:
				checksum = crc32.Update(checksum, table, buf[:n])
			case io.EOF:
				return checksum, nil
			default:
				return 0, err
			}
		}
	}
}

// CalculateFileCRC32 calculates and returns the CRC-32 checksum of a file.
func CalculateFileCRC32(ctx context.Context, filePath string) (uint32, error) {
	// Open file with simplified flags and permissions
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close() // Ensure file closure

	return calculateCRC32(ctx, file)
}

// TruncateAndRemoveFile first truncates the file to 0 and then remove (delete)
// the file at given path.
func TruncateAndRemoveFile(filePath string) error {
	// Truncate the file to 0 size, so that even if there are open file handles
	// and linux doesn't delete the file, the file will not take space.
	err := os.Truncate(filePath, 0)
	if err != nil {
		return err
	}
	err = os.Remove(filePath)
	if err != nil {
		return err
	}
	return nil
}
