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

package gcsx

import (
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

func CreateObjectRequest(srcObject *gcs.Object, objectName string, mtime *time.Time, chunkTransferTimeoutSecs int64) *gcs.CreateObjectRequest {
	metadataMap := make(map[string]string)
	var req *gcs.CreateObjectRequest
	if srcObject == nil {
		var preCond int64
		req = &gcs.CreateObjectRequest{
			Name:                     objectName,
			GenerationPrecondition:   &preCond,
			Metadata:                 metadataMap,
			ChunkTransferTimeoutSecs: chunkTransferTimeoutSecs,
		}
	} else {
		for key, value := range srcObject.Metadata {
			metadataMap[key] = value
		}

		req = &gcs.CreateObjectRequest{
			Name:                       srcObject.Name,
			GenerationPrecondition:     &srcObject.Generation,
			MetaGenerationPrecondition: &srcObject.MetaGeneration,
			Metadata:                   metadataMap,
			CacheControl:               srcObject.CacheControl,
			ContentDisposition:         srcObject.ContentDisposition,
			ContentEncoding:            srcObject.ContentEncoding,
			ContentType:                srcObject.ContentType,
			CustomTime:                 srcObject.CustomTime,
			EventBasedHold:             srcObject.EventBasedHold,
			StorageClass:               srcObject.StorageClass,
			ChunkTransferTimeoutSecs:   chunkTransferTimeoutSecs,
		}
	}

	// Any existing mtime value will be overwritten with new value.
	if mtime != nil {
		metadataMap[MtimeMetadataKey] = mtime.UTC().Format(time.RFC3339Nano)
	}

	return req
}
