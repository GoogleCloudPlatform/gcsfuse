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

package operations

import (
	"context"
	"errors"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ValidateNoFileOrDirError(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	if err == nil || !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("os.Stat(%s). Expected: %s, Got: %v", path,
			"no such file or directory", err)
	}
}

func ValidateObjectNotFoundErr(ctx context.Context, t *testing.T, bucket gcs.Bucket, fileName string) {
	t.Helper()
	var notFoundErr *gcs.NotFoundError
	_, err := storageutil.ReadObject(ctx, bucket, fileName)

	assert.Error(t, err)
	assert.True(t, errors.As(err, &notFoundErr))
}

func ValidateESTALEError(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err)
	assert.Regexp(t, syscall.ESTALE.Error(), err.Error())
}

func ValidateEIOError(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err)
	assert.Regexp(t, syscall.EIO.Error(), err.Error())
}

func CheckErrorForReadOnlyFileSystem(t *testing.T, err error) {
	if err == nil {
		t.Error("permission denied error expected but got nil error.")
		return
	}
	if strings.Contains(err.Error(), "read-only file system") || strings.Contains(err.Error(), "permission denied") || strings.Contains(err.Error(), "Permission denied") {
		return
	}
	t.Errorf("Incorrect error for readonly file system: %v", err.Error())
}

func SkipKLCTestForUnsupportedKernelVersion(t *testing.T) {
	t.Helper()
	unsupported, err := common.IsKLCacheEvictionUnSupported()
	assert.NoError(t, err)
	if unsupported {
		t.SkipNow()
	}
}
