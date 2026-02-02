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
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateObjectRequest(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name                     string
		srcObject                *Object
		objectName               string
		mtime                    *time.Time
		chunkTransferTimeoutSecs int64
		expectedRequest          *CreateObjectRequest
	}{
		{
			name:                     "nil_srcObject",
			objectName:               "new-object.txt",
			mtime:                    &now,
			chunkTransferTimeoutSecs: 30,
			expectedRequest: &CreateObjectRequest{
				Name:                   "new-object.txt",
				GenerationPrecondition: &[]int64{0}[0], // Default precondition
				Metadata: map[string]string{
					MtimeMetadataKey: now.UTC().Format(time.RFC3339Nano),
				},
				ChunkTransferTimeoutSecs: 30,
			},
		},
		{
			name: "existing_srcObject",
			srcObject: &Object{
				Name:               "existing-object.txt",
				Generation:         12345,
				MetaGeneration:     67890,
				Metadata:           map[string]string{"key1": "value1", "key2": "value2"},
				CacheControl:       "public, max-age=3600",
				ContentDisposition: "attachment; filename=\"myfile.txt\"",
				ContentEncoding:    "gzip",
				ContentType:        "text/plain",
				CustomTime:         now.Add(-24 * time.Hour).String(),
				EventBasedHold:     true,
				StorageClass:       "STANDARD",
			},
			mtime:                    &now,
			chunkTransferTimeoutSecs: 60,
			expectedRequest: &CreateObjectRequest{
				Name:                       "existing-object.txt",
				GenerationPrecondition:     &[]int64{12345}[0],
				MetaGenerationPrecondition: &[]int64{67890}[0],
				Metadata: map[string]string{
					"key1":           "value1",
					"key2":           "value2",
					MtimeMetadataKey: now.UTC().Format(time.RFC3339Nano),
				},
				CacheControl:             "public, max-age=3600",
				ContentDisposition:       "attachment; filename=\"myfile.txt\"",
				ContentEncoding:          "gzip",
				ContentType:              "text/plain",
				CustomTime:               now.Add(-24 * time.Hour).String(),
				EventBasedHold:           true,
				StorageClass:             "STANDARD",
				ChunkTransferTimeoutSecs: 60,
			},
		},
		{
			name:                     "nil_mtime_nil_srcObject",
			objectName:               "no-mtime.txt",
			chunkTransferTimeoutSecs: 30,
			expectedRequest: &CreateObjectRequest{
				Name:                     "no-mtime.txt",
				GenerationPrecondition:   &[]int64{0}[0],
				Metadata:                 map[string]string{},
				ChunkTransferTimeoutSecs: 30,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NewCreateObjectRequest(tt.srcObject, tt.objectName, tt.mtime, tt.chunkTransferTimeoutSecs, 0)

			assert.Equal(t, tt.expectedRequest, req)
		})
	}
}
