// Copyright 2024 Google Inc. All Rights Reserved.
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
	"time"

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
		Seconds: time.Now().Unix(),              // Number of seconds since Unix epoch (1970-01-01T00:00:00Z)
		Nanos:   int32(time.Now().Nanosecond()), // Nanoseconds (0 to 999,999,999)
	}
	attrs := controlpb.Folder{
		Name:           TestFolderName,
		Metageneration: 10,
		UpdateTime:     timestamp,
	}

	gcsFolder := GCSFolder(TestBucketName, &attrs)

	assert.Equal(t, attrs.Name, gcsFolder.Name)
	assert.Equal(t, attrs.Metageneration, gcsFolder.MetaGeneration)
	assert.Equal(t, attrs.UpdateTime.AsTime(), gcsFolder.UpdateTime)
}
