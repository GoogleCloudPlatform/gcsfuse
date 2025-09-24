// Copyright 2025 Google LLC
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

// GENERATED CODE - DO NOT EDIT MANUALLY.

package cfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyOptimizations(t *testing.T) {
	// Tests for file-cache.cache-file-for-range-read
	t.Run("file-cache.cache-file-for-range-read", func(t *testing.T) {
		// Test case 1: User has set the flag, no optimization should be applied.
		t.Run("user_set", func(t *testing.T) {
			c := &Config{}
			c.FileCache.CacheFileForRangeRead = false
			isSet := &mockIsValueSet{setFlags: map[string]bool{"file-cache-cache-file-for-range-read": true}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Empty(t, optimizedFlags)
			assert.Equal(t, false, c.FileCache.CacheFileForRangeRead)
		})

		// Test case 2: No profile or machine-based optimization match.
		t.Run("no_optimization", func(t *testing.T) {
			c := &Config{Profile: "non_existent_profile"}
			c.FileCache.CacheFileForRangeRead = false
			isSet := &mockIsValueSet{
				setFlags:    map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "n1-standard-1"}, // A machine type not in any group
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Empty(t, optimizedFlags)
			assert.Equal(t, false, c.FileCache.CacheFileForRangeRead)
		})

		// Test cases for profile-based optimizations
		t.Run("profile_aiml-serving", func(t *testing.T) {
			c := &Config{Profile: "aiml-serving"}
			c.FileCache.CacheFileForRangeRead = false
			isSet := &mockIsValueSet{setFlags: map[string]bool{}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "file-cache.cache-file-for-range-read")
			assert.Equal(t, true, c.FileCache.CacheFileForRangeRead)
		})
		t.Run("profile_aiml-checkpointing", func(t *testing.T) {
			c := &Config{Profile: "aiml-checkpointing"}
			c.FileCache.CacheFileForRangeRead = false
			isSet := &mockIsValueSet{setFlags: map[string]bool{}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "file-cache.cache-file-for-range-read")
			assert.Equal(t, true, c.FileCache.CacheFileForRangeRead)
		})

		// Test cases for machine-based optimizations
	})
	// Tests for implicit-dirs
	t.Run("implicit-dirs", func(t *testing.T) {
		// Test case 1: User has set the flag, no optimization should be applied.
		t.Run("user_set", func(t *testing.T) {
			c := &Config{}
			c.ImplicitDirs = false
			isSet := &mockIsValueSet{setFlags: map[string]bool{"implicit-dirs": true}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Empty(t, optimizedFlags)
			assert.Equal(t, false, c.ImplicitDirs)
		})

		// Test case 2: No profile or machine-based optimization match.
		t.Run("no_optimization", func(t *testing.T) {
			c := &Config{Profile: "non_existent_profile"}
			c.ImplicitDirs = false
			isSet := &mockIsValueSet{
				setFlags:    map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "n1-standard-1"}, // A machine type not in any group
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Empty(t, optimizedFlags)
			assert.Equal(t, false, c.ImplicitDirs)
		})

		// Test cases for profile-based optimizations
		t.Run("profile_aiml-training", func(t *testing.T) {
			c := &Config{Profile: "aiml-training"}
			c.ImplicitDirs = false
			isSet := &mockIsValueSet{setFlags: map[string]bool{}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "implicit-dirs")
			assert.Equal(t, true, c.ImplicitDirs)
		})
		t.Run("profile_aiml-serving", func(t *testing.T) {
			c := &Config{Profile: "aiml-serving"}
			c.ImplicitDirs = false
			isSet := &mockIsValueSet{setFlags: map[string]bool{}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "implicit-dirs")
			assert.Equal(t, true, c.ImplicitDirs)
		})
		t.Run("profile_aiml-checkpointing", func(t *testing.T) {
			c := &Config{Profile: "aiml-checkpointing"}
			c.ImplicitDirs = false
			isSet := &mockIsValueSet{setFlags: map[string]bool{}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "implicit-dirs")
			assert.Equal(t, true, c.ImplicitDirs)
		})

		// Test cases for machine-based optimizations
		t.Run("machine_group_high-performance", func(t *testing.T) {
			// Find a machine type from the group to use in the test
			c := &Config{Profile: ""}
			c.ImplicitDirs = false
			isSet := &mockIsValueSet{
				setFlags:    map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "implicit-dirs")
			assert.Equal(t, true, c.ImplicitDirs)
		})
	})
	// Tests for file-system.kernel-list-cache-ttl-secs
	t.Run("file-system.kernel-list-cache-ttl-secs", func(t *testing.T) {
		// Test case 1: User has set the flag, no optimization should be applied.
		t.Run("user_set", func(t *testing.T) {
			c := &Config{}
			c.FileSystem.KernelListCacheTtlSecs = 0
			isSet := &mockIsValueSet{setFlags: map[string]bool{"kernel-list-cache-ttl-secs": true}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Empty(t, optimizedFlags)
			assert.Equal(t, int64(0), c.FileSystem.KernelListCacheTtlSecs)
		})

		// Test case 2: No profile or machine-based optimization match.
		t.Run("no_optimization", func(t *testing.T) {
			c := &Config{Profile: "non_existent_profile"}
			c.FileSystem.KernelListCacheTtlSecs = 0
			isSet := &mockIsValueSet{
				setFlags:    map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "n1-standard-1"}, // A machine type not in any group
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Empty(t, optimizedFlags)
			assert.Equal(t, int64(0), c.FileSystem.KernelListCacheTtlSecs)
		})

		// Test cases for profile-based optimizations
		t.Run("profile_aiml-serving", func(t *testing.T) {
			c := &Config{Profile: "aiml-serving"}
			c.FileSystem.KernelListCacheTtlSecs = 0
			isSet := &mockIsValueSet{setFlags: map[string]bool{}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "file-system.kernel-list-cache-ttl-secs")
			assert.Equal(t, int64(-1), c.FileSystem.KernelListCacheTtlSecs)
		})

		// Test cases for machine-based optimizations
	})
	// Tests for metadata-cache.negative-ttl-secs
	t.Run("metadata-cache.negative-ttl-secs", func(t *testing.T) {
		// Test case 1: User has set the flag, no optimization should be applied.
		t.Run("user_set", func(t *testing.T) {
			c := &Config{}
			c.MetadataCache.NegativeTtlSecs = 5
			isSet := &mockIsValueSet{setFlags: map[string]bool{"metadata-cache-negative-ttl-secs": true}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Empty(t, optimizedFlags)
			assert.Equal(t, int64(5), c.MetadataCache.NegativeTtlSecs)
		})

		// Test case 2: No profile or machine-based optimization match.
		t.Run("no_optimization", func(t *testing.T) {
			c := &Config{Profile: "non_existent_profile"}
			c.MetadataCache.NegativeTtlSecs = 5
			isSet := &mockIsValueSet{
				setFlags:    map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "n1-standard-1"}, // A machine type not in any group
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Empty(t, optimizedFlags)
			assert.Equal(t, int64(5), c.MetadataCache.NegativeTtlSecs)
		})

		// Test cases for profile-based optimizations
		t.Run("profile_aiml-training", func(t *testing.T) {
			c := &Config{Profile: "aiml-training"}
			c.MetadataCache.NegativeTtlSecs = 5
			isSet := &mockIsValueSet{setFlags: map[string]bool{}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "metadata-cache.negative-ttl-secs")
			assert.Equal(t, int64(0), c.MetadataCache.NegativeTtlSecs)
		})
		t.Run("profile_aiml-serving", func(t *testing.T) {
			c := &Config{Profile: "aiml-serving"}
			c.MetadataCache.NegativeTtlSecs = 5
			isSet := &mockIsValueSet{setFlags: map[string]bool{}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "metadata-cache.negative-ttl-secs")
			assert.Equal(t, int64(0), c.MetadataCache.NegativeTtlSecs)
		})
		t.Run("profile_aiml-checkpointing", func(t *testing.T) {
			c := &Config{Profile: "aiml-checkpointing"}
			c.MetadataCache.NegativeTtlSecs = 5
			isSet := &mockIsValueSet{setFlags: map[string]bool{}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "metadata-cache.negative-ttl-secs")
			assert.Equal(t, int64(0), c.MetadataCache.NegativeTtlSecs)
		})

		// Test cases for machine-based optimizations
		t.Run("machine_group_high-performance", func(t *testing.T) {
			// Find a machine type from the group to use in the test
			c := &Config{Profile: ""}
			c.MetadataCache.NegativeTtlSecs = 5
			isSet := &mockIsValueSet{
				setFlags:    map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "metadata-cache.negative-ttl-secs")
			assert.Equal(t, int64(0), c.MetadataCache.NegativeTtlSecs)
		})
	})
	// Tests for metadata-cache.ttl-secs
	t.Run("metadata-cache.ttl-secs", func(t *testing.T) {
		// Test case 1: User has set the flag, no optimization should be applied.
		t.Run("user_set", func(t *testing.T) {
			c := &Config{}
			c.MetadataCache.TtlSecs = 60
			isSet := &mockIsValueSet{setFlags: map[string]bool{"metadata-cache-ttl-secs": true}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Empty(t, optimizedFlags)
			assert.Equal(t, int64(60), c.MetadataCache.TtlSecs)
		})

		// Test case 2: No profile or machine-based optimization match.
		t.Run("no_optimization", func(t *testing.T) {
			c := &Config{Profile: "non_existent_profile"}
			c.MetadataCache.TtlSecs = 60
			isSet := &mockIsValueSet{
				setFlags:    map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "n1-standard-1"}, // A machine type not in any group
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Empty(t, optimizedFlags)
			assert.Equal(t, int64(60), c.MetadataCache.TtlSecs)
		})

		// Test cases for profile-based optimizations
		t.Run("profile_aiml-training", func(t *testing.T) {
			c := &Config{Profile: "aiml-training"}
			c.MetadataCache.TtlSecs = 60
			isSet := &mockIsValueSet{setFlags: map[string]bool{}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "metadata-cache.ttl-secs")
			assert.Equal(t, int64(-1), c.MetadataCache.TtlSecs)
		})
		t.Run("profile_aiml-serving", func(t *testing.T) {
			c := &Config{Profile: "aiml-serving"}
			c.MetadataCache.TtlSecs = 60
			isSet := &mockIsValueSet{setFlags: map[string]bool{}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "metadata-cache.ttl-secs")
			assert.Equal(t, int64(-1), c.MetadataCache.TtlSecs)
		})
		t.Run("profile_aiml-checkpointing", func(t *testing.T) {
			c := &Config{Profile: "aiml-checkpointing"}
			c.MetadataCache.TtlSecs = 60
			isSet := &mockIsValueSet{setFlags: map[string]bool{}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "metadata-cache.ttl-secs")
			assert.Equal(t, int64(-1), c.MetadataCache.TtlSecs)
		})

		// Test cases for machine-based optimizations
		t.Run("machine_group_high-performance", func(t *testing.T) {
			// Find a machine type from the group to use in the test
			c := &Config{Profile: ""}
			c.MetadataCache.TtlSecs = 60
			isSet := &mockIsValueSet{
				setFlags:    map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "metadata-cache.ttl-secs")
			assert.Equal(t, int64(-1), c.MetadataCache.TtlSecs)
		})
	})
	// Tests for file-system.rename-dir-limit
	t.Run("file-system.rename-dir-limit", func(t *testing.T) {
		// Test case 1: User has set the flag, no optimization should be applied.
		t.Run("user_set", func(t *testing.T) {
			c := &Config{}
			c.FileSystem.RenameDirLimit = 0
			isSet := &mockIsValueSet{setFlags: map[string]bool{"rename-dir-limit": true}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Empty(t, optimizedFlags)
			assert.Equal(t, int64(0), c.FileSystem.RenameDirLimit)
		})

		// Test case 2: No profile or machine-based optimization match.
		t.Run("no_optimization", func(t *testing.T) {
			c := &Config{Profile: "non_existent_profile"}
			c.FileSystem.RenameDirLimit = 0
			isSet := &mockIsValueSet{
				setFlags:    map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "n1-standard-1"}, // A machine type not in any group
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Empty(t, optimizedFlags)
			assert.Equal(t, int64(0), c.FileSystem.RenameDirLimit)
		})

		// Test cases for profile-based optimizations
		t.Run("profile_aiml-checkpointing", func(t *testing.T) {
			c := &Config{Profile: "aiml-checkpointing"}
			c.FileSystem.RenameDirLimit = 0
			isSet := &mockIsValueSet{setFlags: map[string]bool{}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "file-system.rename-dir-limit")
			assert.Equal(t, int64(200000), c.FileSystem.RenameDirLimit)
		})

		// Test cases for machine-based optimizations
		t.Run("machine_group_high-performance", func(t *testing.T) {
			// Find a machine type from the group to use in the test
			c := &Config{Profile: ""}
			c.FileSystem.RenameDirLimit = 0
			isSet := &mockIsValueSet{
				setFlags:    map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "file-system.rename-dir-limit")
			assert.Equal(t, int64(200000), c.FileSystem.RenameDirLimit)
		})
	})
	// Tests for metadata-cache.stat-cache-max-size-mb
	t.Run("metadata-cache.stat-cache-max-size-mb", func(t *testing.T) {
		// Test case 1: User has set the flag, no optimization should be applied.
		t.Run("user_set", func(t *testing.T) {
			c := &Config{}
			c.MetadataCache.StatCacheMaxSizeMb = 33
			isSet := &mockIsValueSet{setFlags: map[string]bool{"stat-cache-max-size-mb": true}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Empty(t, optimizedFlags)
			assert.Equal(t, int64(33), c.MetadataCache.StatCacheMaxSizeMb)
		})

		// Test case 2: No profile or machine-based optimization match.
		t.Run("no_optimization", func(t *testing.T) {
			c := &Config{Profile: "non_existent_profile"}
			c.MetadataCache.StatCacheMaxSizeMb = 33
			isSet := &mockIsValueSet{
				setFlags:    map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "n1-standard-1"}, // A machine type not in any group
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Empty(t, optimizedFlags)
			assert.Equal(t, int64(33), c.MetadataCache.StatCacheMaxSizeMb)
		})

		// Test cases for profile-based optimizations
		t.Run("profile_aiml-training", func(t *testing.T) {
			c := &Config{Profile: "aiml-training"}
			c.MetadataCache.StatCacheMaxSizeMb = 33
			isSet := &mockIsValueSet{setFlags: map[string]bool{}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "metadata-cache.stat-cache-max-size-mb")
			assert.Equal(t, int64(-1), c.MetadataCache.StatCacheMaxSizeMb)
		})
		t.Run("profile_aiml-serving", func(t *testing.T) {
			c := &Config{Profile: "aiml-serving"}
			c.MetadataCache.StatCacheMaxSizeMb = 33
			isSet := &mockIsValueSet{setFlags: map[string]bool{}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "metadata-cache.stat-cache-max-size-mb")
			assert.Equal(t, int64(-1), c.MetadataCache.StatCacheMaxSizeMb)
		})
		t.Run("profile_aiml-checkpointing", func(t *testing.T) {
			c := &Config{Profile: "aiml-checkpointing"}
			c.MetadataCache.StatCacheMaxSizeMb = 33
			isSet := &mockIsValueSet{setFlags: map[string]bool{}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "metadata-cache.stat-cache-max-size-mb")
			assert.Equal(t, int64(-1), c.MetadataCache.StatCacheMaxSizeMb)
		})

		// Test cases for machine-based optimizations
		t.Run("machine_group_high-performance", func(t *testing.T) {
			// Find a machine type from the group to use in the test
			c := &Config{Profile: ""}
			c.MetadataCache.StatCacheMaxSizeMb = 33
			isSet := &mockIsValueSet{
				setFlags:    map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "metadata-cache.stat-cache-max-size-mb")
			assert.Equal(t, int64(1024), c.MetadataCache.StatCacheMaxSizeMb)
		})
	})
	// Tests for metadata-cache.type-cache-max-size-mb
	t.Run("metadata-cache.type-cache-max-size-mb", func(t *testing.T) {
		// Test case 1: User has set the flag, no optimization should be applied.
		t.Run("user_set", func(t *testing.T) {
			c := &Config{}
			c.MetadataCache.TypeCacheMaxSizeMb = 4
			isSet := &mockIsValueSet{setFlags: map[string]bool{"type-cache-max-size-mb": true}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Empty(t, optimizedFlags)
			assert.Equal(t, int64(4), c.MetadataCache.TypeCacheMaxSizeMb)
		})

		// Test case 2: No profile or machine-based optimization match.
		t.Run("no_optimization", func(t *testing.T) {
			c := &Config{Profile: "non_existent_profile"}
			c.MetadataCache.TypeCacheMaxSizeMb = 4
			isSet := &mockIsValueSet{
				setFlags:    map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "n1-standard-1"}, // A machine type not in any group
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Empty(t, optimizedFlags)
			assert.Equal(t, int64(4), c.MetadataCache.TypeCacheMaxSizeMb)
		})

		// Test cases for profile-based optimizations
		t.Run("profile_aiml-training", func(t *testing.T) {
			c := &Config{Profile: "aiml-training"}
			c.MetadataCache.TypeCacheMaxSizeMb = 4
			isSet := &mockIsValueSet{setFlags: map[string]bool{}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "metadata-cache.type-cache-max-size-mb")
			assert.Equal(t, int64(-1), c.MetadataCache.TypeCacheMaxSizeMb)
		})
		t.Run("profile_aiml-serving", func(t *testing.T) {
			c := &Config{Profile: "aiml-serving"}
			c.MetadataCache.TypeCacheMaxSizeMb = 4
			isSet := &mockIsValueSet{setFlags: map[string]bool{}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "metadata-cache.type-cache-max-size-mb")
			assert.Equal(t, int64(-1), c.MetadataCache.TypeCacheMaxSizeMb)
		})
		t.Run("profile_aiml-checkpointing", func(t *testing.T) {
			c := &Config{Profile: "aiml-checkpointing"}
			c.MetadataCache.TypeCacheMaxSizeMb = 4
			isSet := &mockIsValueSet{setFlags: map[string]bool{}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "metadata-cache.type-cache-max-size-mb")
			assert.Equal(t, int64(-1), c.MetadataCache.TypeCacheMaxSizeMb)
		})

		// Test cases for machine-based optimizations
		t.Run("machine_group_high-performance", func(t *testing.T) {
			// Find a machine type from the group to use in the test
			c := &Config{Profile: ""}
			c.MetadataCache.TypeCacheMaxSizeMb = 4
			isSet := &mockIsValueSet{
				setFlags:    map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "metadata-cache.type-cache-max-size-mb")
			assert.Equal(t, int64(128), c.MetadataCache.TypeCacheMaxSizeMb)
		})
	})
	// Tests for write.global-max-blocks
	t.Run("write.global-max-blocks", func(t *testing.T) {
		// Test case 1: User has set the flag, no optimization should be applied.
		t.Run("user_set", func(t *testing.T) {
			c := &Config{}
			c.Write.GlobalMaxBlocks = 4
			isSet := &mockIsValueSet{setFlags: map[string]bool{"write-global-max-blocks": true}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Empty(t, optimizedFlags)
			assert.Equal(t, int64(4), c.Write.GlobalMaxBlocks)
		})

		// Test case 2: No profile or machine-based optimization match.
		t.Run("no_optimization", func(t *testing.T) {
			c := &Config{Profile: "non_existent_profile"}
			c.Write.GlobalMaxBlocks = 4
			isSet := &mockIsValueSet{
				setFlags:    map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "n1-standard-1"}, // A machine type not in any group
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Empty(t, optimizedFlags)
			assert.Equal(t, int64(4), c.Write.GlobalMaxBlocks)
		})

		// Test cases for profile-based optimizations

		// Test cases for machine-based optimizations
		t.Run("machine_group_high-performance", func(t *testing.T) {
			// Find a machine type from the group to use in the test
			c := &Config{Profile: ""}
			c.Write.GlobalMaxBlocks = 4
			isSet := &mockIsValueSet{
				setFlags:    map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "write.global-max-blocks")
			assert.Equal(t, int64(1600), c.Write.GlobalMaxBlocks)
		})
	})
}
