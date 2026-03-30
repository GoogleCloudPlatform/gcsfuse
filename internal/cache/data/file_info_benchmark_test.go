// Copyright 2024 Google LLC
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

package data

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkGetFileInfoKeyName_Original(b *testing.B) {
	bucketCreationTime := time.Unix(TestTimeInEpoch, 0)
	b.ResetTimer()
	for b.Loop() {
		unixTimeString := fmt.Sprintf("%d", bucketCreationTime.Unix())
		_ = TestBucketName + unixTimeString + TestObjectName
	}
}

func BenchmarkGetFileInfoKeyName_Optimized(b *testing.B) {
	bucketCreationTime := time.Unix(TestTimeInEpoch, 0)
	b.ResetTimer()
	for b.Loop() {
		_, _ = GetFileInfoKeyName(TestObjectName, bucketCreationTime, TestBucketName)
	}
}
