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

package storageutil

import (
	"testing"
	"time"

	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestFindObjectName(t *testing.T) {
	folderPath := "projects/_/buckets/" + TestBucketName + "/folders/" + TestObjectName

	folderName := findFolderName(TestBucketName, folderPath)

	assert.Equal(t, folderName, TestObjectName)
}

func TestControlFolderAttrsToGCSFolder(t *testing.T) {
	timeAttr := time.Now()
	protoTimestamp := &timestamppb.Timestamp{
		Seconds: timeAttr.Unix(),              // Number of seconds since Unix epoch (1970-01-01T00:00:00Z)
		Nanos:   int32(timeAttr.Nanosecond()), // Nanoseconds (0 to 999,999,999)
	}
	attrs := controlpb.Folder{
		Name:           TestObjectName,
		Metageneration: 10,
		UpdateTime:     protoTimestamp,
	}

	gcsFolder := ControlFolderAttrsToGCSFolder(TestBucketName, &attrs)

	assert.Equal(t, gcsFolder.Name, attrs.Name)
	assert.Equal(t, gcsFolder.Metageneration, attrs.Metageneration)
	assert.Equal(t, gcsFolder.UpdateTime, attrs.UpdateTime.AsTime())
}
