// Copyright 2020 Google LLC
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
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

type SyncerBucket struct {
	gcs.Bucket
	Syncer
}

// NewSyncerBucket creates a SyncerBucket, which can be used either as
// a gcs.Bucket, or as a Syncer.
func NewSyncerBucket(
	appendThreshold int64,
	chunkTransferTimeoutSecs int64,
	tmpObjectPrefix string,
	bucket gcs.Bucket,
) SyncerBucket {
	syncer := NewSyncer(appendThreshold, chunkTransferTimeoutSecs, tmpObjectPrefix, bucket)
	return SyncerBucket{bucket, syncer}
}
