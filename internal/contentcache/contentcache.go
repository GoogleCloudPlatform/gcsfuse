// Copyright 2021 Google Inc. All Rights Reserved.
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

// Package contentcache stores GCS object contents locally.
package contentcache

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"regexp"

	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/jacobsa/timeutil"
)

// CacheObjectKey uniquely identifies GCS objects by bucket name and object name
type CacheObjectKey struct {
	BucketName string
	ObjectName string
}

// ContentCache is a directory on local disk to store the object content.
type ContentCache struct {
	tempDir        string
	localFileCache bool
	fileMap        map[CacheObjectKey]gcsx.TempFile
	mtimeClock     timeutil.Clock
}

// Populates the cache with existing persisted files when gcsfuse starts
func (c *ContentCache) PopulateCache() {
	if c.tempDir == "" {
		c.tempDir = "/tmp"
	}
	files, err := ioutil.ReadDir(c.tempDir)
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		// validate not a directory and matches gcsfuse pattern
		if !file.IsDir() && matchPattern(file.Name()) {
			// TODO ezl: load the files from disk to the in memory map
			fmt.Printf(file.Name())
		}
	}
}

// Helper function that matches the format of a gcsfuse file
func matchPattern(fileName string) bool {
	// TODO ezl: replace with constant defined in gcsx.TempFile
	match, err := regexp.MatchString(fmt.Sprintf("gcsfuse[0-9]+[.]json"), fileName)
	if err != nil {
		return false
	}
	return match
}

// New creates a ContentCache.
func New(tempDir string, mtimeClock timeutil.Clock) *ContentCache {
	return &ContentCache{
		tempDir:    tempDir,
		fileMap:    make(map[CacheObjectKey]gcsx.TempFile),
		mtimeClock: mtimeClock,
	}
}

// NewTempFile returns a handle for a temporary file on the disk. The caller
// must call Destroy on the TempFile before releasing it.
func (c *ContentCache) NewTempFile(rc io.ReadCloser) (gcsx.TempFile, error) {
	return gcsx.NewTempFile(rc, c.tempDir, c.mtimeClock)
}
