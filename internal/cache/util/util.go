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
	"path/filepath"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
)

const (
	InvalidFileHandleErrMsg      = "invalid file handle"
	InvalidFileDownloadJobErrMsg = "invalid download job"
	InvalidCacheHandleErrMsg     = "invalid cache handle"
	InvalidFileInfoCacheErrMsg   = "invalid file info cache"
	ErrInSeekingFileHandleMsg    = "error while seeking file handle"
	ErrInReadingFileHandleMsg    = "error while reading file handle"
	FallbackToGCSErrMsg          = "read via gcs"
)

const DefaultFileMode = os.FileMode(0644)
const FileDirPerm = os.FileMode(0755) | os.ModeDir
const MiB = 1024 * 1024

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

func ConvertObjToMinObject(o *gcs.Object) *gcs.MinObject {
	if o == nil {
		return nil
	}

	return &gcs.MinObject{
		Name:            o.Name,
		Size:            o.Size,
		Generation:      o.Generation,
		MetaGeneration:  o.MetaGeneration,
		Updated:         o.Updated,
		Metadata:        o.Metadata,
		ContentEncoding: o.ContentEncoding,
	}
}
