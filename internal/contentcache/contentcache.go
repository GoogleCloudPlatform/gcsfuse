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
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"regexp"

	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/jacobsa/timeutil"
)

type CacheObjectKey struct {
	BucketName string
	ObjectName string
}

// ContentCache is a directory on local disk to store the object content.
type ContentCache struct {
	tempDir        string
	localFileCache bool
	// Filemap maps canononical file prefixes to gcsx.TempFile, wrapper for
	// temp files on disk cache
	fileMap    map[CacheObjectKey]gcsx.TempFile
	mtimeClock timeutil.Clock
}

// New creates a ContentCache.
func New(tempDir string, mtimeClock timeutil.Clock) *ContentCache {
	return &ContentCache{
		tempDir:    tempDir,
		fileMap:    make(map[CacheObjectKey]gcsx.TempFile),
		mtimeClock: mtimeClock,
	}
}

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
			var contents []byte
			var metadata *gcsx.Metadata
			contents, err = ioutil.ReadFile(file.Name())
			if err != nil {
				panic("")
			}
			err = json.Unmarshal(contents, metadata)
			if err != nil {
				panic("")
			}
			cacheObjectKey := &CacheObjectKey{BucketName: metadata.BucketName, ObjectName: metadata.ObjectName}
			_, err := c.Get(cacheObjectKey)
			panic(err)
		}
	}
}

func matchPattern(fileName string) bool {
	match, err := regexp.MatchString("gcsfusecache[0-9]+[.]json", fileName)
	if err != nil {
		return false
	}
	return match
}

// Function to add or update existing cache file
func (c *ContentCache) AddOrReplace(cacheObjectKey *CacheObjectKey, generation int64, rc io.ReadCloser) (gcsx.TempFile, error) {
	if _, exists := c.fileMap[*cacheObjectKey]; exists {
		c.fileMap[*cacheObjectKey].Destroy()
	}
	metadata := &gcsx.Metadata{
		BucketName: cacheObjectKey.BucketName,
		ObjectName: cacheObjectKey.ObjectName,
		Generation: generation,
	}
	file, err := c.NewCacheFile(rc, metadata)
	c.fileMap[*cacheObjectKey] = file
	return file, err
}

// Retrieve temp file from the cache
func (c *ContentCache) Get(cacheObjectKey *CacheObjectKey) (gcsx.TempFile, bool) {
	file, exists := c.fileMap[*cacheObjectKey]
	return file, exists
}

// Function to remove and destroy cache file and metadata on disk
func (c *ContentCache) Remove(cacheObjectKey *CacheObjectKey) {
	if _, exists := c.fileMap[*cacheObjectKey]; exists {
		c.fileMap[*cacheObjectKey].Destroy()
	}
	delete(c.fileMap, *cacheObjectKey)
}

// NewTempFile returns a handle for a temporary file on the disk. The caller
// must call Destroy on the TempFile before releasing it.
func (c *ContentCache) NewTempFile(rc io.ReadCloser) (gcsx.TempFile, error) {
	return gcsx.NewTempFile(rc, c.tempDir, c.mtimeClock)
}

// NewTempFile returns a handle for a temporary file on the disk. The caller
// must call Destroy on the TempFile before releasing it.
func (c *ContentCache) NewCacheFile(rc io.ReadCloser, metadata *gcsx.Metadata) (gcsx.TempFile, error) {
	return gcsx.NewCacheFile(rc, metadata, c.tempDir, c.mtimeClock)
}
