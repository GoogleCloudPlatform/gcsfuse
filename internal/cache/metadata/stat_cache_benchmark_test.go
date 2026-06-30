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

package metadata_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
)

func BenchmarkStatCache_Insert(b *testing.B) {
	lc := lru.NewCache(util.MiBsToBytes(100))
	sc := metadata.NewStatCacheBucketView(lc, "")
	expiration := time.Now().Add(time.Hour)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("file-%d", i)
		m := &gcs.MinObject{
			Name: name,
			Size: 100,
		}
		sc.Insert(m, expiration)
	}
}

func BenchmarkStatCache_AddNegativeEntry(b *testing.B) {
	lc := lru.NewCache(util.MiBsToBytes(100))
	sc := metadata.NewStatCacheBucketView(lc, "")
	expiration := time.Now().Add(time.Hour)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("file-%d", i)
		sc.AddNegativeEntry(name, expiration)
	}
}

func BenchmarkStatCache_LookUp(b *testing.B) {
	lc := lru.NewCache(util.MiBsToBytes(100))
	sc := metadata.NewStatCacheBucketView(lc, "")
	expiration := time.Now().Add(time.Hour)

	// Pre-populate
	for i := 0; i < 10000; i++ {
		name := fmt.Sprintf("file-%d", i)
		m := &gcs.MinObject{
			Name: name,
			Size: 100,
		}
		sc.Insert(m, expiration)
	}

	now := time.Now()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("file-%d", i%10000)
		sc.LookUp(name, now)
	}
}
