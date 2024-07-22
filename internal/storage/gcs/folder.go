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

package gcs

import (
	"strings"
	"time"

	"cloud.google.com/go/storage/control/apiv2/controlpb"
)

type Folder struct {
	Name           string
	MetaGeneration int64
	UpdateTime     time.Time
}

func GCSFolder(bucketName string, attrs *controlpb.Folder) *Folder {
	// Setting the parameters in Folder and doing conversions as necessary.
	return &Folder{
		Name:           getFolderName(bucketName, attrs.Name),
		MetaGeneration: attrs.Metageneration,
		UpdateTime:     attrs.GetUpdateTime().AsTime(),
	}
}

// In HNS, folder paths returned by control client APIs are in the form of:
//
// projects/_/buckets/{bucket}/folders/{folder}
// This method extracts the folder name from such a string.
func getFolderName(bucketName string, fullPath string) string {
	prefix := "projects/_/buckets/" + bucketName + "/folders/"
	folderName := strings.TrimPrefix(fullPath, prefix)

	return folderName
}

func (f *Folder) ConvertFolderToMinObject() *MinObject {

	return &MinObject{
		Name:           f.Name,
		MetaGeneration: f.MetaGeneration,
		Updated:        f.UpdateTime,
	}
}
