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
	"regexp"

	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	"github.com/jacobsa/timeutil"
)

const CACHE_FILE_PREFIX = "gcsfusecache"

// CacheObjectKey uniquely identifies GCS objects by bucket name and object name
type CacheObjectKey struct {
	BucketName string
	ObjectName string
}

// ContentCache is a directory on local disk to store the object content
type ContentCache struct {
	debug      *log.Logger
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

type CacheObject struct {
	MetadataFileName        string
	CacheFileObjectMetadata *CacheFileObjectMetadata
	CacheFile               gcsx.TempFile
}

func (c *CacheObject) ValidateGeneration(generation int64) bool {
	if c.CacheFileObjectMetadata == nil {
		return false
	}
	return c.CacheFileObjectMetadata.Generation == generation
}

func (c *ContentCache) WriteMetadataCheckpointFile(cacheFileName string, cacheFileObjectMetadata *CacheFileObjectMetadata) (metadataFileName string, err error) {
	var file []byte
	file, err = json.MarshalIndent(cacheFileObjectMetadata, "", " ")
	if err != nil {
		err = fmt.Errorf("json.MarshalIndent failed for object metadata: %w", err)
		return
	}
	metadataFileName = fmt.Sprintf("%s.json", cacheFileName)
	err = ioutil.WriteFile(metadataFileName, file, 0644)
	if err != nil {
		err = fmt.Errorf("WriteFile for JSON metadata: %w", err)
		return
	}
	return
}

func (c *CacheObject) Destroy() {
	if c.CacheFile != nil {
		os.Remove(c.CacheFile.Name())
		c.CacheFile.Destroy()
	}
	os.Remove(c.MetadataFileName)
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
	var metadata CacheFileObjectMetadata
	metadataAbsolutePath := path.Join(c.tempDir, metadataFile.Name())
	contents, err := ioutil.ReadFile(metadataAbsolutePath)
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
	fileName := metadata.CacheFileNameOnDisk
	// TODO ezl linux fs limits single process to open max of 1024 file descriptors
	// so this is not scalable
	file, err := os.Open(fileName)
	if err != nil {
		c.debug.Printf("Skip cache file %v due to error: %v", fileName, err)
		return
	}
	cacheFile, err := c.RecoverCacheFile(file)
	if err != nil {
		c.debug.Printf("Skip cache file %v due to error: %v", fileName, err)
	}
	cacheObject := &CacheObject{
		MetadataFileName:        metadataAbsolutePath,
		CacheFileObjectMetadata: &metadata,
		CacheFile:               cacheFile,
	}
	c.fileMap[*cacheObjectKey] = cacheObject
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
	match, err := regexp.MatchString(fmt.Sprintf("%v[0-9]+[.]json", CACHE_FILE_PREFIX), fileName)
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
func (c *ContentCache) AddOrReplace(cacheObjectKey *CacheObjectKey, generation int64, rc io.ReadCloser) (*CacheObject, error) {
	if cacheObject, exists := c.fileMap[*cacheObjectKey]; exists {
		cacheObject.Destroy()
	}
	// Create a temporary cache file on disk
	f, err := ioutil.TempFile(c.tempDir, CACHE_FILE_PREFIX)
	if err != nil {
		err = fmt.Errorf("TempFile: %w", err)
	}
	file, err := c.NewCacheFile(rc, f)
	if err != nil {
		return nil, fmt.Errorf("NewCacheFile: %w", err)
	}
	metadata := &CacheFileObjectMetadata{
		CacheFileNameOnDisk: file.Name(),
		BucketName:          cacheObjectKey.BucketName,
		ObjectName:          cacheObjectKey.ObjectName,
		Generation:          generation,
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
func (c *ContentCache) Get(cacheObjectKey *CacheObjectKey) (*CacheObject, bool) {
	cacheObject, exists := c.fileMap[*cacheObjectKey]
	return cacheObject, exists
}

// Remove removes and destroys the specfied cache file and metadata on disk
func (c *ContentCache) Remove(cacheObjectKey *CacheObjectKey) {
	if cacheObject, exists := c.fileMap[*cacheObjectKey]; exists {
		cacheObject.Destroy()
		delete(c.fileMap, *cacheObjectKey)
	}
}

// NewCacheFile creates a cache file on the disk storing the object content
func (c *ContentCache) NewCacheFile(rc io.ReadCloser, f *os.File) (gcsx.TempFile, error) {
	return gcsx.NewCacheFile(rc, f, c.tempDir, c.mtimeClock)
}

func (c *ContentCache) RecoverCacheFile(f *os.File) (gcsx.TempFile, error) {
	return gcsx.RecoverCacheFile(f, c.tempDir, c.mtimeClock)
}
