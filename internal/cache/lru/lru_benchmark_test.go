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

package lru_test

import (
	"fmt"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
)

func BenchmarkEraseEntriesWithGivenPrefix(b *testing.B) {
	c := lru.NewCache(uint64(10000000))
	for i := range 100 {
		_, _ = c.Insert(fmt.Sprintf("dir1/file%d", i), testData{10, 10})
		_, _ = c.Insert(fmt.Sprintf("dir2/file%d", i), testData{10, 10})
	}

	for b.Loop() {
		c.EraseEntriesWithGivenPrefix("dir1/")

		// To sustain the benchmark we need to reinsert what we erased
		// without timing the insertion part
		b.StopTimer()
		for j := range 100 {
			_, _ = c.Insert(fmt.Sprintf("dir1/file%d", j), testData{10, 10})
		}
		b.StartTimer()
	}
}

func BenchmarkEraseEntriesWithGivenPrefix_Concurrent(b *testing.B) {
	c := lru.NewCache(uint64(10000000))

	// Pre-fill so it doesn't do empty lookups
	for i := range 20000 {
		_, _ = c.Insert(fmt.Sprintf("dir1/file%d", i), testData{10, 10})
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			i++
			if i%10 == 0 {
				c.EraseEntriesWithGivenPrefix("dir1/")
			} else {
				_, _ = c.Insert(fmt.Sprintf("dir1/file%d", i), testData{10, 10})
			}
		}
	})
}
