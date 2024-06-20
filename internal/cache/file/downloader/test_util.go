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

package downloader

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
)

func getMinObject(objectName string, bucket gcs.Bucket) gcs.MinObject {
	ctx := context.Background()
	minObject, _, err := bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: objectName,
		ForceFetchFromGcs: true})
	if err != nil {
		panic(fmt.Errorf("error occured while statting the object: %w", err))
	}
	if minObject != nil {
		return *minObject
	}
	return gcs.MinObject{}
}

func verifyFileTillOffset(t *testing.T, spec data.FileSpec, offset int64, content []byte) {
	fileStat, err := os.Stat(spec.Path)
	if assert.Nil(t, err) {
		// Verify the content of file downloaded only till the size of content passed.
		fileContent, err := os.ReadFile(spec.Path)
		assert.Nil(t, err)
		assert.Equal(t, spec.FilePerm, fileStat.Mode())
		assert.GreaterOrEqual(t, len(content), int(offset))
		assert.GreaterOrEqual(t, len(fileContent), int(offset))
		assert.Equal(t, content[:offset], fileContent[:offset])
	}

}
