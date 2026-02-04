// Copyright 2026 Google LLC
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

package file

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// SharedChunkCacheManager manages a file cache that can be safely shared across
// multiple gcsfuse mount instances using lock-free atomic operations.
type SharedChunkCacheManager struct {
	// cacheDir is the local path which contains the cache data
	cacheDir string

	// chunkSize is the size of each chunk for chunk-based caching
	chunkSize int64

	// filePerm parameter specifies the permission of file in cache
	filePerm os.FileMode

	// dirPerm parameter specifies the permission of cache directory
	dirPerm os.FileMode

	// excludeRegex is the compiled regex for excluding files from cache
	excludeRegex *regexp.Regexp

	// includeRegex is the compiled regex for including files from cache
	includeRegex *regexp.Regexp

	// config contains file cache configuration
	config *cfg.FileCacheConfig
}

// NewSharedChunkCacheManager creates a new shared chunk cache handler.
func NewSharedChunkCacheManager(
	cacheDir string,
	filePerm os.FileMode,
	dirPerm os.FileMode,
	config *cfg.FileCacheConfig,
) (*SharedChunkCacheManager, error) {
	// Determine chunk size
	chunkSize := config.SharedCacheChunkSizeMb * 1024 * 1024

	// Compile regex patterns
	var err error
	var excludeRegex, includeRegex *regexp.Regexp
	if config.ExcludeRegex != "" {
		excludeRegex, err = regexp.Compile(config.ExcludeRegex)
		if err != nil {
			logger.Warnf("Failed to compile exclude regex %q: %v", config.ExcludeRegex, err)
		}
	}
	if config.IncludeRegex != "" {
		includeRegex, err = regexp.Compile(config.IncludeRegex)
		if err != nil {
			logger.Warnf("Failed to compile include regex %q: %v", config.IncludeRegex, err)
		}
	}

	handler := &SharedChunkCacheManager{
		cacheDir:     cacheDir,
		chunkSize:    chunkSize,
		filePerm:     filePerm,
		dirPerm:      dirPerm,
		excludeRegex: excludeRegex,
		includeRegex: includeRegex,
		config:       config,
	}

	return handler, nil
}

// ShouldExcludeFromCache checks if the file should be excluded from caching.
func (sccm *SharedChunkCacheManager) ShouldExcludeFromCache(bucket gcs.Bucket, object *gcs.MinObject) bool {
	objectPath := filepath.Join(bucket.Name(), object.Name)

	// If include regex is set, only include matching files
	if sccm.includeRegex != nil {
		if !sccm.includeRegex.MatchString(objectPath) {
			return true
		}
	}

	// Exclude files matching exclude regex
	if sccm.excludeRegex != nil {
		if sccm.excludeRegex.MatchString(objectPath) {
			return true
		}
	}

	return false
}

// GenerateTmpPath generates a unique temporary file path in the object directory.
// The temporary file name includes a random prefix to avoid conflicts.
func (sccm *SharedChunkCacheManager) GenerateTmpPath(bucketName, objectName string, generation int64, chunkIndex int64) string {
	// Generate random 8-character hex prefix
	randomPrefix := fmt.Sprintf("%016x", rand.Uint64())
	chunkPath := sccm.GetChunkPath(bucketName, objectName, generation, chunkIndex)
	return chunkPath + "." + randomPrefix + ".tmp"
}

// GetFilePerm returns the file permission used by this handler.
func (sccm *SharedChunkCacheManager) GetFilePerm() os.FileMode {
	return sccm.filePerm
}

// GetDirPerm returns the directory permission used by this handler.
func (sccm *SharedChunkCacheManager) GetDirPerm() os.FileMode {
	return sccm.dirPerm
}

// GetChunkIndex calculates which chunk contains the given offset.
func (sccm *SharedChunkCacheManager) GetChunkIndex(offset int64) int64 {
	return offset / sccm.chunkSize
}

// GetChunkSize returns the chunk size used by this handler.
func (sccm *SharedChunkCacheManager) GetChunkSize() int64 {
	return sccm.chunkSize
}

// computeObjectHash computes SHA256 hash of bucketName, objectName, and generation.
func computeObjectHash(bucketName, objectName string, generation int64) string {
	h := sha256.New()

	// Use length prefixes to avoid hash collision, in case bucketName contains '/'.
	// Format: <bucketNameLength>:<bucketName><objectNameLength>:<objectName>:<generation>
	fmt.Fprintf(h, "%d:%s", len(bucketName), bucketName)
	fmt.Fprintf(h, "%d:%s", len(objectName), objectName)
	fmt.Fprintf(h, ":%d", generation)

	return hex.EncodeToString(h.Sum(nil))
}

// GetObjectDir returns the directory path for an object with generation encoded.
// Format: /cache/<prefix1>/<prefix2>/<full-sha256-hash>/
// where prefix1 and prefix2 are the first four hex digits of the SHA256 hash (2 chars each)
func (sccm *SharedChunkCacheManager) GetObjectDir(bucketName, objectName string, generation int64) string {
	hash := computeObjectHash(bucketName, objectName, generation)
	prefix1 := hash[0:2] // First 2 characters
	prefix2 := hash[2:4] // Next 2 characters
	return filepath.Join(sccm.cacheDir, prefix1, prefix2, hash)
}

// GetChunkPath returns the path to a specific chunk file.
// Format: /cache/<prefix1>/<prefix2>/<hash>/<start-offset>_<end-offset>.bin
func (sccm *SharedChunkCacheManager) GetChunkPath(bucketName, objectName string, generation int64, chunkIndex int64) string {
	objDir := sccm.GetObjectDir(bucketName, objectName, generation)
	startOffset := chunkIndex * sccm.chunkSize
	endOffset := startOffset + sccm.chunkSize
	return filepath.Join(objDir, fmt.Sprintf("%d_%d.bin", startOffset, endOffset))
}
