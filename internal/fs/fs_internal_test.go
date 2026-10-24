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
	"sync"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util/diskutil"
	"github.com/stretchr/testify/assert"
)

func TestCacheDirVolumeBlockSize(t *testing.T) {
	cacheDir := t.TempDir()
	actualBlockSize := diskutil.GetVolumeBlockSize(cacheDir)

	for _, tc := range []struct {
		name                         string
		disableSizeCalculationFix    bool
		enableExperimentalChunkCache bool
		expectedBlockSize            uint64
	}{
		{
			name:                         "SizeCalcFixEnabled_NotSparse",
			disableSizeCalculationFix:    false,
			enableExperimentalChunkCache: false,
			expectedBlockSize:            actualBlockSize,
		},
		{
			name:                         "SizeCalcFixDisabled",
			disableSizeCalculationFix:    true,
			enableExperimentalChunkCache: false,
			expectedBlockSize:            1,
		},
		{
			name:                         "SparseModeEnabled",
			disableSizeCalculationFix:    false,
			enableExperimentalChunkCache: true,
			expectedBlockSize:            1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			serverCfg := &ServerConfig{
				NewConfig: &cfg.Config{
					FileCache: cfg.FileCacheConfig{
						ExperimentalDisableSizeCalculationFix: tc.disableSizeCalculationFix,
						ExperimentalEnableChunkCache:          tc.enableExperimentalChunkCache,
					},
				},
			}

			blockSize := cacheDirVolumeBlockSize(serverCfg, cacheDir)

			assert.Equal(t, tc.expectedBlockSize, blockSize)
		})
	}
}

type fakeBucketOwnedDirInode struct {
	inode.BucketOwnedDirInode
	name inode.Name
}

func (f *fakeBucketOwnedDirInode) Name() inode.Name {
	return f.name
}

func TestEnsureNoLocalFilesInDirectory(t *testing.T) {
	root := inode.NewRootName("bucket")
	dirName := inode.NewDirName(root, "parent")
	dir := &fakeBucketOwnedDirInode{name: dirName}

	newTestFS := func(localInodes map[inode.Name]inode.Inode) *fileSystem {
		return &fileSystem{
			mu:              &sync.Mutex{},
			localFileInodes: localInodes,
		}
	}

	t.Run("NoLocalFiles", func(t *testing.T) {
		fs := newTestFS(make(map[inode.Name]inode.Inode))
		err := fs.ensureNoLocalFilesInDirectory(dir, "parent")
		assert.NoError(t, err)
	})

	t.Run("LocalFileAtDepth1", func(t *testing.T) {
		childName := inode.NewFileName(dirName, "file.txt")
		fs := newTestFS(map[inode.Name]inode.Inode{
			childName: &inode.FileInode{},
		})
		err := fs.ensureNoLocalFilesInDirectory(dir, "parent")
		assert.ErrorIs(t, err, syscall.ENOTSUP)
	})

	t.Run("LocalFileAtDepth2", func(t *testing.T) {
		subDir := inode.NewDirName(dirName, "sub")
		nestedName := inode.NewFileName(subDir, "file.txt")
		fs := newTestFS(map[inode.Name]inode.Inode{
			nestedName: &inode.FileInode{},
		})
		err := fs.ensureNoLocalFilesInDirectory(dir, "parent")
		assert.ErrorIs(t, err, syscall.ENOTSUP)
	})

	t.Run("LocalFileAtDepth3", func(t *testing.T) {
		subDir := inode.NewDirName(dirName, "sub")
		subSubDir := inode.NewDirName(subDir, "sub2")
		nestedName := inode.NewFileName(subSubDir, "file.txt")
		fs := newTestFS(map[inode.Name]inode.Inode{
			nestedName: &inode.FileInode{},
		})
		err := fs.ensureNoLocalFilesInDirectory(dir, "parent")
		assert.ErrorIs(t, err, syscall.ENOTSUP)
	})

	t.Run("UnlinkedLocalFileAtDepth2Ignored", func(t *testing.T) {
		subDir := inode.NewDirName(dirName, "sub")
		nestedName := inode.NewFileName(subDir, "file.txt")
		unlinkedFile := &inode.FileInode{}
		unlinkedFile.Unlink()
		fs := newTestFS(map[inode.Name]inode.Inode{
			nestedName: unlinkedFile,
		})
		err := fs.ensureNoLocalFilesInDirectory(dir, "parent")
		assert.NoError(t, err)
	})

	t.Run("LocalFileInAnotherDirectoryIgnored", func(t *testing.T) {
		otherDir := inode.NewDirName(root, "other")
		otherFileName := inode.NewFileName(otherDir, "file.txt")
		fs := newTestFS(map[inode.Name]inode.Inode{
			otherFileName: &inode.FileInode{},
		})
		err := fs.ensureNoLocalFilesInDirectory(dir, "parent")
		assert.NoError(t, err)
	})
}
