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

package gcs

import (
	"testing"

	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const TestBucketName string = "testBucket"
const TestFolderName string = "testFolder"

func TestGetFolderName(t *testing.T) {
	folderPath := "projects/_/buckets/" + TestBucketName + "/folders/" + TestFolderName

	result := getFolderName(TestBucketName, folderPath)

	assert.Equal(t, result, TestFolderName)
}

func TestGCSFolder(t *testing.T) {
	timestamp := &timestamppb.Timestamp{
		Seconds: 123456789,
		Nanos:   987654321,
	}
	attrs := controlpb.Folder{
		Name:           TestFolderName,
		Metageneration: 10,
		UpdateTime:     timestamp,
	}

	gcsFolder := GCSFolder(TestBucketName, &attrs)

	assert.Equal(t, TestFolderName, gcsFolder.Name)
	assert.Equal(t, int64(123456789987654321), gcsFolder.UpdateTime)
}

func TestGCSFolder_UninitializedTime(t *testing.T) {
	attrs := controlpb.Folder{
		Name:           TestFolderName,
		Metageneration: 10,
	}

	gcsFolder := GCSFolder(TestBucketName, &attrs)

	assert.EqualValues(t, int64(0), gcsFolder.UpdateTime)
}

func BenchmarkGCSFolder(b *testing.B) {
	timestamp := &timestamppb.Timestamp{
		Seconds: 1234567890,
	}
	attrs := controlpb.Folder{
		Name:       "projects/_/buckets/testBucket/folders/testFolder",
		UpdateTime: timestamp,
	}

	for b.Loop() {
		// this triggers 2 allocations
		// the string creation inside getFolderName()
		// and the &Folder{} heap allocation
		_ = GCSFolder("testBucket", &attrs)
	}
}
