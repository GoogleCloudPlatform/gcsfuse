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

package fs

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"syscall"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

const logicalQuotaListPageSize = 1000

const logicalQuotaAllBucketsMountError = "experimental logical quotas require a single mounted bucket or --only-dir prefix; all-buckets mounts are not supported"

type logicalQuota struct {
	mu sync.Mutex

	maxFiles uint64
	maxBytes uint64

	// When the corresponding limit is enabled, these are exactly the sums of
	// the entries in byName. Disabled dimensions are intentionally not tracked.
	usedFiles uint64
	usedBytes uint64

	byName map[inode.Name]quotaEntry
}

type quotaEntry struct {
	size uint64
}

func newLogicalQuota(maxFiles int64, maxSizeMb int64) (*logicalQuota, error) {
	if maxFiles < 0 {
		return nil, fmt.Errorf("experimental-max-file-count must be non-negative")
	}
	if maxSizeMb < 0 {
		return nil, fmt.Errorf("experimental-max-size-mb must be non-negative")
	}
	if maxFiles == 0 && maxSizeMb == 0 {
		return nil, nil
	}

	var maxBytes uint64
	if maxSizeMb > 0 {
		if uint64(maxSizeMb) > math.MaxUint64/(1<<20) {
			return nil, fmt.Errorf("experimental-max-size-mb is too large")
		}
		maxBytes = uint64(maxSizeMb) * (1 << 20)
	}

	return &logicalQuota{
		maxFiles: uint64(maxFiles),
		maxBytes: maxBytes,
		byName:   make(map[inode.Name]quotaEntry),
	}, nil
}

func newLogicalQuotaForServerConfig(serverCfg *ServerConfig) (*logicalQuota, error) {
	q, err := newLogicalQuota(
		serverCfg.NewConfig.FileSystem.ExperimentalMaxFileCount,
		serverCfg.NewConfig.FileSystem.ExperimentalMaxSizeMb)
	if err != nil {
		return nil, err
	}
	if q == nil {
		return nil, nil
	}

	if serverCfg.BucketName == "" || serverCfg.BucketName == "_" {
		return nil, fmt.Errorf(logicalQuotaAllBucketsMountError)
	}

	return q, nil
}

func (q *logicalQuota) hasFileLimit() bool {
	return q != nil && q.maxFiles != 0
}

func (q *logicalQuota) hasSizeLimit() bool {
	return q != nil && q.maxBytes != 0
}

func (q *logicalQuota) initialize(ctx context.Context, bucket gcs.Bucket, root inode.Name) error {
	for token := ""; ; {
		listing, err := bucket.ListObjects(ctx, &gcs.ListObjectsRequest{
			ContinuationToken: token,
			MaxResults:        logicalQuotaListPageSize,
		})
		if err != nil {
			return fmt.Errorf("quota cold-start scan: %w", err)
		}

		q.mu.Lock()
		for _, object := range listing.MinObjects {
			if object == nil || strings.HasSuffix(object.Name, "/") {
				continue
			}
			q.addInitialEntryLocked(inode.NewDescendantName(root, object.Name), object.Size)
		}
		q.mu.Unlock()

		token = listing.ContinuationToken
		if token == "" {
			logger.Infof(
				"logical quota initialized: used_files=%d used_bytes=%d max_files=%d max_bytes=%d",
				q.usedFiles,
				q.usedBytes,
				q.maxFiles,
				q.maxBytes)
			if q.hasFileLimit() && q.usedFiles > q.maxFiles {
				logger.Warnf("logical quota starts over file limit: used_files=%d max_files=%d", q.usedFiles, q.maxFiles)
			}
			if q.hasSizeLimit() && q.usedBytes > q.maxBytes {
				logger.Warnf("logical quota starts over size limit: used_bytes=%d max_bytes=%d", q.usedBytes, q.maxBytes)
			}
			return nil
		}
	}
}

func (q *logicalQuota) addInitialEntryLocked(name inode.Name, size uint64) {
	old, existed := q.byName[name]
	if existed {
		if q.hasSizeLimit() {
			q.usedBytes -= old.size
		}
	} else if q.hasFileLimit() {
		q.usedFiles++
	}

	if q.hasSizeLimit() {
		q.usedBytes += size
	} else {
		size = 0
	}
	q.byName[name] = quotaEntry{size: size}
}

func (q *logicalQuota) tryReserveFile(name inode.Name) (rollback func(), err error) {
	if q == nil || !q.hasFileLimit() {
		return func() {}, nil
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.byName[name]; ok {
		return func() {}, nil
	}
	if q.usedFiles >= q.maxFiles {
		logger.Warnf("logical file quota exceeded: op=create name=%s used_files=%d max_files=%d", name.GcsObjectName(), q.usedFiles, q.maxFiles)
		return nil, syscall.ENOSPC
	}

	q.usedFiles++
	q.byName[name] = quotaEntry{}
	rolledBack := false
	return func() {
		q.mu.Lock()
		defer q.mu.Unlock()
		if rolledBack {
			return
		}
		rolledBack = true
		entry, ok := q.byName[name]
		if ok {
			q.usedFiles--
			if q.hasSizeLimit() {
				q.usedBytes -= entry.size
			}
			delete(q.byName, name)
		}
	}, nil
}

func (q *logicalQuota) tryReserveGrowth(name inode.Name, oldSize uint64, newSize uint64) (commit func(), rollback func(), err error) {
	noOp := func() {}
	if q == nil {
		return noOp, noOp, nil
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	oldEntry, existed := q.byName[name]
	newFile := !existed
	if !newFile {
		oldSize = oldEntry.size
	}

	targetSize := newSize
	if oldSize > targetSize {
		targetSize = oldSize
	}

	var fileDelta uint64
	if newFile && q.hasFileLimit() {
		if q.usedFiles >= q.maxFiles {
			logger.Warnf("logical file quota exceeded: op=write name=%s used_files=%d max_files=%d", name.GcsObjectName(), q.usedFiles, q.maxFiles)
			return nil, nil, syscall.ENOSPC
		}
		fileDelta = 1
	}

	var byteDelta uint64
	if q.hasSizeLimit() {
		if newFile {
			byteDelta = targetSize
		} else if targetSize > oldSize {
			byteDelta = targetSize - oldSize
		}
		if byteDelta > 0 && (byteDelta > q.maxBytes || q.usedBytes > q.maxBytes-byteDelta) {
			logger.Warnf("logical quota exceeded: op=write name=%s old_size=%d new_size=%d used_bytes=%d max_bytes=%d", name.GcsObjectName(), oldSize, targetSize, q.usedBytes, q.maxBytes)
			return nil, nil, syscall.ENOSPC
		}
	}

	if fileDelta == 0 && byteDelta == 0 {
		return noOp, noOp, nil
	}

	q.usedFiles += fileDelta
	q.usedBytes += byteDelta
	if q.hasSizeLimit() {
		q.byName[name] = quotaEntry{size: targetSize}
	} else {
		q.byName[name] = quotaEntry{}
	}

	committed := false
	return func() {
			q.mu.Lock()
			defer q.mu.Unlock()
			committed = true
		}, func() {
			q.mu.Lock()
			defer q.mu.Unlock()
			if committed {
				return
			}
			q.usedFiles -= fileDelta
			q.usedBytes -= byteDelta
			if newFile {
				delete(q.byName, name)
			} else {
				q.byName[name] = oldEntry
			}
		}, nil
}

func (q *logicalQuota) applyShrink(name inode.Name, newSize uint64) {
	if q == nil || !q.hasSizeLimit() {
		return
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	entry, ok := q.byName[name]
	if !ok || newSize >= entry.size {
		return
	}
	q.usedBytes -= entry.size - newSize
	q.byName[name] = quotaEntry{size: newSize}
}

func (q *logicalQuota) releaseName(name inode.Name) {
	if q == nil {
		return
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	entry, ok := q.byName[name]
	if !ok {
		return
	}
	if q.hasFileLimit() {
		q.usedFiles--
	}
	if q.hasSizeLimit() {
		q.usedBytes -= entry.size
	}
	delete(q.byName, name)
}

func (q *logicalQuota) applyRename(oldName inode.Name, newName inode.Name, fallbackSize uint64) {
	if q == nil || oldName == newName {
		return
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	src, srcOK := q.byName[oldName]
	dst, dstOK := q.byName[newName]
	if !srcOK && !dstOK {
		return
	}

	if !srcOK {
		src = quotaEntry{}
		if q.hasSizeLimit() {
			src.size = fallbackSize
			q.usedBytes += fallbackSize
		}
		if q.hasFileLimit() {
			q.usedFiles++
		}
	}

	if dstOK {
		if q.hasFileLimit() {
			q.usedFiles--
		}
		if q.hasSizeLimit() {
			q.usedBytes -= dst.size
		}
	}

	delete(q.byName, oldName)
	q.byName[newName] = src
}

func (q *logicalQuota) sizeForName(name inode.Name, fallback uint64) uint64 {
	if q == nil || !q.hasSizeLimit() {
		return fallback
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	entry, ok := q.byName[name]
	if !ok {
		return fallback
	}
	return entry.size
}

func (q *logicalQuota) statFS(blockSize uint64) (blocks uint64, free uint64, available uint64, inodes uint64, inodesFree uint64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	blocks = 1 << 33
	free = blocks
	available = blocks
	if q.hasSizeLimit() {
		blocks = (q.maxBytes + blockSize - 1) / blockSize
		if q.usedBytes < q.maxBytes {
			free = (q.maxBytes - q.usedBytes) / blockSize
		} else {
			free = 0
		}
		available = free
	}

	inodes = 1 << 50
	inodesFree = inodes
	if q.hasFileLimit() {
		inodes = q.maxFiles
		if q.usedFiles < q.maxFiles {
			inodesFree = q.maxFiles - q.usedFiles
		} else {
			inodesFree = 0
		}
	}
	return
}
