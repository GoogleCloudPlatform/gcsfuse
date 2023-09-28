// Copyright 2020 Google Inc. All Rights Reserved.
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

package inode

import (
	"fmt"
	"strings"
)

// Name is the inode's name that can be interpreted in 2 ways:
//
//	(1) LocalName: the name of the inode in the local file system.
//	(2) GcsObjectName: the name of its gcs object backed by the inode.
type Name struct {
	// The value of bucketName can be:
	// - "", when single gcs bucket is explicitly mounted for the file system.
	// - the name of the gcs bucket, when potentially multiple buckets are
	//   mounted as subdirectories of the root of the file system.
	bucketName string
	// The gcs object's name in its bucket.
	objectName string
}

// NewRootName creates a Name for the root directory of a gcs bucket
func NewRootName(bucketName string) Name {
	return Name{bucketName, ""}
}

// NewDirName creates a new inode name for a directory.
func NewDirName(parentName Name, dirName string) Name {
	if parentName.IsFile() || dirName == "" {
		panic(fmt.Sprintf(
			"Inode '%s' cannot have child subdirectory '%s'",
			parentName,
			dirName))
	}
	if dirName[len(dirName)-1] != '/' {
		dirName = dirName + "/"
	}
	return Name{parentName.bucketName, parentName.objectName + dirName}
}

// NewFileName creates a new inode name for a file.
func NewFileName(parentName Name, fileName string) Name {
	if parentName.IsFile() || fileName == "" {
		panic(fmt.Sprintf(
			"Inode '%s' cannot have child file '%s'",
			parentName,
			fileName))
	}
	return Name{parentName.bucketName, parentName.objectName + fileName}
}

// NewDescendant creates a new inode name for an object as a descendant of
// another inode.
func NewDescendantName(ancestor Name, descendantObjectName string) Name {
	return Name{ancestor.bucketName, descendantObjectName}
}

// IsBucketRoot returns true if the name represents of a root directory
// of a GCS bucket.
func (name Name) IsBucketRoot() bool {
	return name.objectName == ""
}

// IsDir returns true if the name represents a directory.
func (name Name) IsDir() bool {
	return name.IsBucketRoot() ||
		name.objectName[len(name.objectName)-1] == '/'
}

// IsFile returns true if the name represents a file.
func (name Name) IsFile() bool {
	return !name.IsDir()
}

// GcsObjectName returns the name of the gcs object backed by the inode.
func (name Name) GcsObjectName() string {
	return name.objectName
}

// LocalName returns the name of the directory or file in the local file system.
func (name Name) LocalName() string {
	if name.bucketName == "" {
		return name.objectName
	}
	return name.bucketName + "/" + name.objectName
}

// String returns LocalName.
func (name Name) String() string {
	return name.LocalName()
}

// IsDirectChildOf returns true if the name is a direct child file or directory
// of another directory.
func (name Name) IsDirectChildOf(parent Name) bool {
	if !parent.IsDir() && name.IsBucketRoot() {
		return false
	}
	if name.bucketName != parent.bucketName {
		return false
	}
	if !strings.HasPrefix(name.objectName, parent.objectName) {
		return false
	}
	diff := strings.TrimPrefix(name.objectName, parent.objectName)
	if diff == "" {
		return false
	}
	cleanDiff := strings.TrimSuffix(diff, "/")
	return !strings.Contains(cleanDiff, "/")
}
