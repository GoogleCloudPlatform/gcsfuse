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
	"os"
	"path"
	"regexp"
	"sync"

	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	"github.com/jacobsa/timeutil"
)

const CacheFilePrefix = "gcsfusecache"

// CacheObjectKey uniquely identifies GCS objects by bucket name and object name
type CacheObjectKey struct {
	BucketName string
	ObjectName string
}

// ContentCache is a directory on local disk to store the object content
// ContentCache is thread-safe
// fileMap is an in memory map to represent cache contents on disk
type ContentCache struct {
	mu         sync.Mutex
	tempDir    string
	fileMap    map[CacheObjectKey]*CacheObject
	mtimeClock timeutil.Clock
}

// Metadata store struct
type CacheFileObjectMetadata struct {
	CacheFileNameOnDisk string
	BucketName          string
	ObjectName          string
	Generation          int64
	MetaGeneration      int64
}

// CacheObject is a wrapper struct for a cache file and its associated metadata
type CacheObject struct {
	MetadataFileName        string
	CacheFileObjectMetadata *CacheFileObjectMetadata
	CacheFile               gcsx.TempFile
}

// ValidateGeneration compares fresh gcs object generation and metageneration numbers against cached objects
func (c *CacheObject) ValidateGeneration(generation int64, metaGeneration int64) bool {
	if c.CacheFileObjectMetadata == nil {
		return false
	}
	return c.CacheFileObjectMetadata.Generation == generation && c.CacheFileObjectMetadata.MetaGeneration == metaGeneration
}

// WriteMetadataCheckpointFile writes the metadata struct to a json file so cache files can be recovered on startup
func (c *ContentCache) WriteMetadataCheckpointFile(cacheFileName string, cacheFileObjectMetadata *CacheFileObjectMetadata) (metadataFileName string, err error) {
	var file []byte
	file, err = json.MarshalIndent(cacheFileObjectMetadata, "", " ")
	if err != nil {
		err = fmt.Errorf("json.MarshalIndent failed for object metadata: %w", err)
		return
	}
	metadataFileName = fmt.Sprintf("%s.json", cacheFileName)
	err = os.WriteFile(metadataFileName, file, 0644)
	if err != nil {
		err = fmt.Errorf("WriteFile for JSON metadata: %w", err)
		return
	}
	return
}

// Destroy performs disk clean up of cache files and metadata files
func (c *CacheObject) Destroy() {
	if c.CacheFile != nil {
		os.Remove(c.CacheFile.Name())
		c.CacheFile.Destroy()
	}
	os.Remove(c.MetadataFileName)
}

// recoverFileFromCache recovers a file from the cache via metadata
func (c *ContentCache) recoverFileFromCache(metadataFile fs.FileInfo) {
	// validate not a directory and matches gcsfuse pattern
	if metadataFile.IsDir() {
		return
	}
	if !matchPattern(metadataFile.Name()) {
		return
	}
	var metadata CacheFileObjectMetadata
	metadataAbsolutePath := path.Join(c.tempDir, metadataFile.Name())
	contents, err := os.ReadFile(metadataAbsolutePath)
	if err != nil {
		logger.Errorf("content cache: Skip metadata file %v due to read error: %s", metadataFile.Name(), err)
		return
	}
	err = json.Unmarshal(contents, &metadata)
	if err != nil {
		logger.Errorf("content cache: Skip metadata file %v due to file corruption: %s", metadataFile.Name(), err)
		return
	}
	cacheObjectKey := &CacheObjectKey{
		BucketName: metadata.BucketName,
		ObjectName: metadata.ObjectName,
	}
	fileName := metadata.CacheFileNameOnDisk
	// TODO (#641) linux fs limits single process to open max of 1024 file descriptors
	// so this is probably not scalable, we should figure out if this is an actual issue or not
	file, err := os.Open(fileName)
	if err != nil {
		logger.Errorf("content cache: Skip cache file %v due to error: %v", fileName, err)
		return
	}
	cacheFile, err := c.recoverCacheFile(file)
	if err != nil {
		logger.Errorf("content cache: Skip cache file %v due to error: %v", fileName, err)
	}
	cacheObject := &CacheObject{
		MetadataFileName:        metadataAbsolutePath,
		CacheFileObjectMetadata: &metadata,
		CacheFile:               cacheFile,
	}
	c.fileMap[*cacheObjectKey] = cacheObject
}

// RecoverCache recovers the cache with existing persisted files when gcsfuse starts
// RecoverCache should not be called concurrently
func (c *ContentCache) RecoverCache() error {
	if c.tempDir == "" {
		c.tempDir = "/tmp"
	}
	logger.Infof("Recovering cache:\n")
	dirEntries, err := os.ReadDir(c.tempDir)
	if err != nil {
		// We failed to get the list of directory entries
		// in the temp directory, log and return error.
		return fmt.Errorf("recover cache: %w", err)
	}
	files := make([]os.FileInfo, len(dirEntries))
	for i, dirEntry := range dirEntries {
		files[i], err = dirEntry.Info()
		if err != nil {
			// We failed to read a directory entry,
			// log and return error.
			return fmt.Errorf("recover cache: %w", err)
		}
	}
	for _, metadataFile := range files {
		c.recoverFileFromCache(metadataFile)
	}
	return nil
}

// matchPattern matches the filename format of a gcsfuse file via regex
func matchPattern(fileName string) bool {
	match, err := regexp.MatchString(fmt.Sprintf("%v[0-9]+[.]json", CacheFilePrefix), fileName)
	if err != nil {
		return false
	}
	return match
}

// New creates a ContentCache.
func New(tempDir string, mtimeClock timeutil.Clock) *ContentCache {
	return &ContentCache{
		tempDir:    tempDir,
		fileMap:    make(map[CacheObjectKey]*CacheObject),
		mtimeClock: mtimeClock,
	}
}

// NewTempFile returns a handle for a temporary file on the disk. The caller
// must call Destroy on the TempFile before releasing it.
func (c *ContentCache) NewTempFile(rc io.ReadCloser) (gcsx.TempFile, error) {
	return gcsx.NewTempFile(rc, c.tempDir, c.mtimeClock)
}

// AddOrReplace creates a new cache file or updates an existing cache file
// AddOrReplace is thread-safe
func (c *ContentCache) AddOrReplace(cacheObjectKey *CacheObjectKey, generation int64, metaGeneration int64, rc io.ReadCloser) (*CacheObject, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cacheObject, exists := c.fileMap[*cacheObjectKey]; exists {
		cacheObject.Destroy()
	}
	// Create a temporary cache file on disk
	f, err := os.CreateTemp(c.tempDir, CacheFilePrefix)
	if err != nil {
		return nil, fmt.Errorf("TempFile: %w", err)
	}
	file := c.NewCacheFile(rc, f)
	metadata := &CacheFileObjectMetadata{
		CacheFileNameOnDisk: file.Name(),
		BucketName:          cacheObjectKey.BucketName,
		ObjectName:          cacheObjectKey.ObjectName,
		Generation:          generation,
		MetaGeneration:      metaGeneration,
	}
	var metadataFileName string
	metadataFileName, err = c.WriteMetadataCheckpointFile(file.Name(), metadata)
	if err != nil {
		return nil, fmt.Errorf("WriteMetadataCheckpointFile: %w", err)
	}
	cacheObject := &CacheObject{
		MetadataFileName:        metadataFileName,
		CacheFileObjectMetadata: metadata,
		CacheFile:               file,
	}
	c.fileMap[*cacheObjectKey] = cacheObject
	return cacheObject, err
}

// Get retrieves a file from the cache given the GCS object name and bucket name
// Get is thread-safe
func (c *ContentCache) Get(cacheObjectKey *CacheObjectKey) (*CacheObject, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cacheObject, exists := c.fileMap[*cacheObjectKey]
	return cacheObject, exists
}

// Remove and destroys the specfied cache file and metadata on disk
// Remove is thread-safe
func (c *ContentCache) Remove(cacheObjectKey *CacheObjectKey) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cacheObject, exists := c.fileMap[*cacheObjectKey]; exists {
		cacheObject.Destroy()
		delete(c.fileMap, *cacheObjectKey)
	}
}

// NewCacheFile returns a cache tempfile wrapper around the source reader and file
func (c *ContentCache) NewCacheFile(rc io.ReadCloser, f *os.File) gcsx.TempFile {
	return gcsx.NewCacheFile(rc, f, c.tempDir, c.mtimeClock)
}

// recoverCacheFile returns a tempfile wrapper around a prepopulated cache file from disk
func (c *ContentCache) recoverCacheFile(f *os.File) (gcsx.TempFile, error) {
	return gcsx.RecoverCacheFile(f, c.tempDir, c.mtimeClock)
}

// Size returns the size of the in memory map of cache files
func (c *ContentCache) Size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.fileMap)
}
