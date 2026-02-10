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
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path"
	"path/filepath"
	"unsafe"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/data"
	"github.com/jacobsa/fuse/fsutil"
)

var (
	ErrInvalidFileHandle                   = errors.New("invalid file handle")
	ErrInvalidFileDownloadJob              = errors.New("invalid download job")
	ErrInvalidFileInfoCache                = errors.New("invalid file info cache")
	ErrInReadingFileHandle                 = errors.New("error while reading file handle")
	ErrFallbackToGCS                       = errors.New("read via gcs")
	ErrFileNotPresentInCache               = errors.New("file is not present in cache")
	ErrCacheHandleNotRequiredForRandomRead = errors.New("cacheFileForRangeRead is false, read type random read and fileInfo entry is absent")
	ErrFileExcludedFromCacheByRegex        = errors.New("file excluded from cache by regex")
	ErrShortRead                           = errors.New("short read")
)

const (
	MiB              = 1024 * 1024
	KiB              = 1024
	DefaultFilePerm  = os.FileMode(0600)
	DefaultDirPerm   = os.FileMode(0700)
	FileCache        = "gcsfuse-file-cache"
	BufferSizeForCRC = 65536
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
		err = fmt.Errorf("error in creating directory structure %s: %w", fileDir, err)
		return
	}

	// Create file if not present.
	_, err = os.Stat(fileSpec.Path)
	if err != nil {
		if os.IsNotExist(err) {
			flag = flag | os.O_CREATE
		} else {
			err = fmt.Errorf("error in stating file %s: %w", fileSpec.Path, err)
			return
		}
	}
	file, err = os.OpenFile(fileSpec.Path, flag, fileSpec.FilePerm)
	if err != nil && os.IsNotExist(err) {
		// Retry creating directory structure if not present
		err = os.MkdirAll(fileDir, fileSpec.DirPerm)
		if err == nil {
			file, err = os.OpenFile(fileSpec.Path, flag, fileSpec.FilePerm)
		}
	}
	if err != nil {
		err = fmt.Errorf("error in creating file %s: %w", fileSpec.Path, err)
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
	return errors.Is(readErr, ErrInvalidFileHandle) ||
		errors.Is(readErr, ErrInvalidFileDownloadJob) ||
		errors.Is(readErr, ErrInvalidFileInfoCache) ||
		errors.Is(readErr, ErrInReadingFileHandle)
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

// GetMemoryAlignedBuffer creates a buffer([]byte) of size bufferSize aligned to
// memory address in multiple of alignSize.
func GetMemoryAlignedBuffer(bufferSize int64, alignSize int64) (buffer []byte, err error) {
	if bufferSize == 0 {
		return make([]byte, 0), nil
	}
	if alignSize == 0 {
		return make([]byte, bufferSize), nil
	}

	// Create and align buffer
	createAndAlignBuffer := func() ([]byte, error) {
		newBuffer := make([]byte, bufferSize+alignSize)
		l := int64(uintptr(unsafe.Pointer(&newBuffer[0])) % uintptr(alignSize))
		skipOffset := alignSize - l
		newBuffer = newBuffer[skipOffset : skipOffset+bufferSize]

		// Check if buffer is aligned or not
		l = int64(uintptr(unsafe.Pointer(&newBuffer[0])) % uintptr(alignSize))
		if l != 0 {
			return nil, fmt.Errorf("failed to align buffer")
		}
		return newBuffer, nil
	}

	// Though we haven't seen any error while aligning buffer but still it is safer
	// to attempt few times in case alignment fails.
	for range 3 {
		buffer, err = createAndAlignBuffer()
		if err == nil {
			return buffer, err
		}
	}
	return buffer, err
}

// CopyUsingMemoryAlignedBuffer copies content from src reader to dst writer
// by staging content into a memory aligned buffer of size bufferSize and
// aligned to multiple of cfg.CacheUtilMinimumAlignSizeForWriting. Note: The minimum write
// size is cfg.CacheUtilMinimumAlignSizeForWriting which means the total size of content
// written to dst writer is always in multiple of cfg.CacheUtilMinimumAlignSizeForWriting.
// If contentSize is lesser than cfg.CacheUtilMinimumAlignSizeForWriting then extra null data
// is written at the last.
func CopyUsingMemoryAlignedBuffer(ctx context.Context, src io.Reader, dst io.Writer, contentSize, bufferSize int64) (n int64, err error) {
	var alignSize int64 = cfg.CacheUtilMinimumAlignSizeForWriting
	if bufferSize < alignSize || ((bufferSize % alignSize) != 0) {
		return 0, fmt.Errorf("buffer size (%v) should be a multiple of %v", bufferSize, alignSize)
	}

	calculateReqBufferSize := func(remainingContentSize int64) int64 {
		reqBufferSize := min(bufferSize, remainingContentSize)
		roundOffVal := alignSize - reqBufferSize%alignSize
		// Only add roundOffVal if reqBufferSize is not multiple of alignSize
		if roundOffVal != alignSize {
			reqBufferSize = reqBufferSize + roundOffVal
		}
		return reqBufferSize
	}

	reqBufferSize := calculateReqBufferSize(contentSize)
	buffer, err := GetMemoryAlignedBuffer(reqBufferSize, alignSize)
	if err != nil {
		return 0, fmt.Errorf("error in creating memory aligned buffer %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return n, fmt.Errorf("copying cancelled: %w", ctx.Err())
		default:
			if n < contentSize {
				remainingContentSize := contentSize - n
				reqBufferSize = calculateReqBufferSize(remainingContentSize)
				buffer = buffer[:reqBufferSize]

				readN, readErr := io.ReadFull(src, buffer)
				expectedEOFError := int64(len(buffer)) > remainingContentSize

				// Return error in case of ErrUnexpectedEOF only if it is not expected.
				if readErr != nil && !errors.Is(readErr, io.EOF) && !(errors.Is(readErr, io.ErrUnexpectedEOF) && expectedEOFError) {
					return n, fmt.Errorf("error while reading to buffer: %w", readErr)
				}

				writeN, writeErr := dst.Write(buffer)
				if writeErr != nil {
					return n, fmt.Errorf("error while writing from buffer: %w", writeErr)
				}

				// The last readN is smaller than writeN if file size is not multiple of
				// MinimumAlignSizeForWriting.
				if readN != writeN && !(expectedEOFError && (int64(readN) == remainingContentSize)) {
					return n, fmt.Errorf("size of content read (%v) mismatch with content written (%v)", readN, writeN)
				}

				n = n + int64(writeN)
			} else {
				return n, err
			}
		}
	}
}
