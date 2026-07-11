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

package lru

import (
	"fmt"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmpiricalStructSizes(t *testing.T) {
	radixHubNodeSize := unsafe.Sizeof(radixHubNode{})
	sievePayloadSize := unsafe.Sizeof(sievePayload{})

	t.Logf("unsafe.Sizeof(radixHubNode{}) = %d (expected 48)", radixHubNodeSize)
	t.Logf("unsafe.Sizeof(sievePayload{}) = %d (claimed/expected 40)", sievePayloadSize)

	assert.Equal(t, uintptr(48), radixHubNodeSize, "radixHubNode size should be 48 bytes")
	assert.Equal(t, uintptr(40), sievePayloadSize, "sievePayload size assertion: 40 bytes")
}

func TestParentDirectoryLocality(t *testing.T) {
	cacheInterface := NewShardedRadixCache(1024 * 1024)
	cache, ok := cacheInterface.(*ShardedRadixCache)
	require.True(t, ok)
	defer cache.Close()

	testGroups := []struct {
		groupName string
		parentDir string
		keys      []string
	}{
		{
			groupName: "Nested Directory A",
			parentDir: "photos/2026/vacation/",
			keys: []string{
				"photos/2026/vacation/img001.jpg",
				"photos/2026/vacation/img002.jpg",
				"photos/2026/vacation/img003.jpg",
				"photos/2026/vacation/notes.txt",
			},
		},
		{
			groupName: "Single Subdirectory Logs",
			parentDir: "logs/",
			keys: []string{
				"logs/app.log",
				"logs/error.log",
				"logs/audit.log",
				"logs/trace.log",
			},
		},
		{
			groupName: "Deep Directory Tree",
			parentDir: "usr/local/google/home/user/project/",
			keys: []string{
				"usr/local/google/home/user/project/file1.go",
				"usr/local/google/home/user/project/file2.go",
				"usr/local/google/home/user/project/README.md",
			},
		},
		{
			groupName: "Root Level Files (with leading slash)",
			parentDir: "/",
			keys: []string{
				"/file1.txt",
				"/file2.txt",
				"/file3.txt",
			},
		},
	}

	for _, tg := range testGroups {
		t.Run(tg.groupName, func(t *testing.T) {
			require.NotEmpty(t, tg.keys)
			_, expectedIdx := cache.getShardWithIdx(tg.keys[0])
			t.Logf("Group '%s' - Key '%s' mapped to shard index %d", tg.groupName, tg.keys[0], expectedIdx)

			mismatches := 0
			for _, k := range tg.keys {
				_, idx := cache.getShardWithIdx(k)
				t.Logf("Key '%s' -> shard %d (prefix evaluated: '%s')", k, idx, ParentDirectoryPrefix(k))
				if idx != expectedIdx {
					mismatches++
					t.Errorf("FAIL Locality mismatch for key '%s': got shard %d, expected %d (parent dir prefix: '%s')",
						k, idx, expectedIdx, ParentDirectoryPrefix(k))
				}
			}

			localityPercentage := float64(len(tg.keys)-mismatches) / float64(len(tg.keys)) * 100.0
			t.Logf("Group '%s' Parent Directory Locality: %.2f%% (%d/%d land in shard %d)",
				tg.groupName, localityPercentage, len(tg.keys)-mismatches, len(tg.keys), expectedIdx)

			assert.Zero(t, mismatches, fmt.Sprintf("100%% of keys sharing parent directory '%s' must land in identical shard index", tg.parentDir))
		})
	}
}
