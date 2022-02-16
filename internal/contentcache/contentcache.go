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
// Note: The content cache is not concurrent safe and callers should ensure thread safety
package contentcache

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	"github.com/jacobsa/timeutil"
)

// CacheObjectKey uniquely identifies GCS objects by bucket name and object name
type CacheObjectKey struct {
	BucketName string
	ObjectName string
}

// ContentCache is a directory on local disk to store the object content
type ContentCache struct {
	debug      *log.Logger
	tempDir    string
	fileMap    map[CacheObjectKey]gcsx.TempFile
	mtimeClock timeutil.Clock
}

// RecoverFileFromCache recovers a file from the cache via metadata
func (c *ContentCache) RecoverFileFromCache(metadataFile fs.FileInfo) {
	// validate not a directory and matches gcsfuse pattern
	if metadataFile.IsDir() {
		return
	}
	if !matchPattern(metadataFile.Name()) {
		return
	}
	var metadata gcsx.TempFileObjectMetadata
	metadataAbsolutePath := path.Join(c.tempDir, metadataFile.Name())
	contents, err := ioutil.ReadFile(metadataAbsolutePath)
	// TODO ezl should we exec some sort of cleanup in the cache if there are errors
	if err != nil {
		c.debug.Printf("Skip metadata file %v due to read error: %s", metadataFile.Name(), err)
		return
	}
	err = json.Unmarshal(contents, &metadata)
	if err != nil {
		c.debug.Printf("Skip metadata file %v due to file corruption: %s", metadataFile.Name(), err)
		return
	}
	cacheObjectKey := &CacheObjectKey{BucketName: metadata.BucketName, ObjectName: metadata.ObjectName}
	// TODO ezl we should probably store the cached file name inside the cache file metadata instead of using the json file name
	fileName := strings.TrimSuffix(metadataAbsolutePath, filepath.Ext(metadataFile.Name()))
	// TODO ezl linux fs limits single process to open max of 1024 file descriptors
	// so this is not scalable
	file, err := os.Open(fileName)
	if err != nil {
		c.debug.Printf("Skip cache file %v due to error: %s", fileName, err)
		return
	}
	c.AddOrReplace(cacheObjectKey, metadata.Generation, file)
}

// RecoverCache recovers the cache with existing persisted files when gcsfuse starts
func (c *ContentCache) RecoverCache() error {
	if c.tempDir == "" {
		c.tempDir = "/tmp"
	}
	logger.Infof("Recovering cache:\n")
	files, err := ioutil.ReadDir(c.tempDir)
	if err != nil {
		// if we fail to read the specified directory, log and return error
		return fmt.Errorf("recover cache: %w", err)
	}
	for _, metadataFile := range files {
		c.RecoverFileFromCache(metadataFile)
	}
	return nil
}

// Helper function that matches the format of a gcsfuse file
func matchPattern(fileName string) bool {
	// TODO ezl: replace with constant defined in gcsx.TempFile
	match, err := regexp.MatchString(fmt.Sprintf("%v[0-9]+[.]json", gcsx.CACHE_FILE_PREFIX), fileName)
	if err != nil {
		return false
	}
	return match
}

// New creates a ContentCache.
func New(tempDir string, mtimeClock timeutil.Clock) *ContentCache {
	return &ContentCache{
		debug:      logger.NewDebug("content cache: "),
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

// AddOrReplace creates a new cache file or updates an existing cache file
func (c *ContentCache) AddOrReplace(cacheObjectKey *CacheObjectKey, generation int64, rc io.ReadCloser) (gcsx.TempFile, error) {
	if cacheObject, exists := c.fileMap[*cacheObjectKey]; exists {
		cacheObject.Destroy()
	}
	metadata := &gcsx.TempFileObjectMetadata{
		BucketName: cacheObjectKey.BucketName,
		ObjectName: cacheObjectKey.ObjectName,
		Generation: generation,
	}
	file, err := c.NewCacheFile(rc, metadata)
	if err != nil {
		return nil, fmt.Errorf("Could not AddOrReplace cache file: %w", err)
	}
	c.fileMap[*cacheObjectKey] = file
	return file, err
}

// Get retrieves a file from the cache given the GCS object name and bucket name
func (c *ContentCache) Get(cacheObjectKey *CacheObjectKey) (gcsx.TempFile, bool) {
	file, exists := c.fileMap[*cacheObjectKey]
	return file, exists
}

// Remove removes and destroys the specfied cache file and metadata on disk
func (c *ContentCache) Remove(cacheObjectKey *CacheObjectKey) {
	if cacheObject, exists := c.fileMap[*cacheObjectKey]; exists {
		cacheObject.Destroy()
		delete(c.fileMap, *cacheObjectKey)
	}
}

// NewCacheFile creates a cache file on the disk storing the object content
// TODO ezl we should refactor reading/writing cache files and metadata to a different package
func (c *ContentCache) NewCacheFile(rc io.ReadCloser, metadata *gcsx.TempFileObjectMetadata) (gcsx.TempFile, error) {
	return gcsx.NewCacheFile(rc, metadata, c.tempDir, c.mtimeClock)
}
