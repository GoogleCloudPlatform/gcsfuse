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
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
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
	CacheHandleNotRequiredForRandomReadErrMsg = "downloadFileForRandomRead is false, read type random read and fileInfo entry is absent"
)

const (
	FileDirPerm            = os.FileMode(0755) | os.ModeDir
	MiB                    = 1024 * 1024
	KiB                    = 1024
	DefaultFilePerm        = os.FileMode(0600)
	FilePermWithAllowOther = os.FileMode(0644)
)

// CreateFile creates file with given file spec i.e. permissions and returns
// file handle for that file opened with given flag.
//
// Note: If directories in path are not present, they are created with FileDirPerm
// permission.
func CreateFile(fileSpec data.FileSpec, flag int) (file *os.File, err error) {
	// Create directory structure if not present
	fileDir := filepath.Dir(fileSpec.Path)
	err = os.MkdirAll(fileDir, FileDirPerm)
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
	file, err = os.OpenFile(fileSpec.Path, flag, fileSpec.Perm)
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
func GetDownloadPath(cacheLocation string, objectPath string) string {
	return path.Join(cacheLocation, objectPath)
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
