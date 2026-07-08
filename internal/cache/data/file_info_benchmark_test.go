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

package data

import (
	"testing"
)

var pathDepths = []struct {
	name string
	path string
}{
	{"Short", "short.txt"},
	{"Medium", "medium/path/to/object.txt"},
	{"Long", "very/long/path/with/multiple/subdirectories/to/simulate/deep/structure/and/long/object/name/in/gcs/bucket.txt"},
}

func BenchmarkFileInfoKey_Key_Current(b *testing.B) {
	for _, tc := range pathDepths {
		b.Run(tc.name, func(b *testing.B) {
			fik := FileInfoKey{
				bucketName:         "test-bucket-name",
				objectName:         tc.path,
				bucketCreationTime: TestTimeInEpoch,
			}

			b.ResetTimer()
			for b.Loop() {
				_, err := fik.Key()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkFileInfoKey_Key_Optimized(b *testing.B) {
	for _, tc := range pathDepths {
		b.Run(tc.name, func(b *testing.B) {
			bucket := "test-bucket-name"
			creationTime := TestTimeInEpoch
			fik, err := NewFileInfoKey(bucket, creationTime, tc.path)
			if err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()
			for b.Loop() {
				_, err := fik.Key()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkGetFileInfoKeyName_Optimized(b *testing.B) {
	bucketCreationTime := TestTimeInEpoch
	b.ResetTimer()
	for b.Loop() {
		_, _ = GetFileInfoKeyName(TestObjectName, bucketCreationTime, TestBucketName)
	}
}
