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

package storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v4/cfg"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

type failingObjectContents struct {
	writer *storage.Writer
	err    error
}

func (r *failingObjectContents) Read([]byte) (int, error) {
	return 0, r.err
}

func (r *failingObjectContents) WriteTo(dst io.Writer) (int64, error) {
	var ok bool
	r.writer, ok = dst.(*storage.Writer)
	if !ok {
		return 0, errors.New("expected the storage SDK writer")
	}
	n, err := dst.Write([]byte("partial checkpoint"))
	if err != nil {
		return int64(n), err
	}
	return int64(n), r.err
}

func TestCreateObjectCancelsUploadAfterSourceError(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("a failed buffered upload must not publish the partial object")
		http.Error(w, "unexpected upload", http.StatusBadRequest)
	}))
	t.Cleanup(server.Close)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	client, err := storage.NewClient(ctx, option.WithEndpoint(server.URL+"/storage/v1/"), option.WithoutAuthentication())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, client.Close()) })
	sourceErr := errors.New("source read failed")
	contents := &failingObjectContents{err: sourceErr}
	t.Cleanup(func() {
		cancel()
		if contents.writer != nil {
			assert.Error(t, contents.writer.Close())
		}
	})
	bucket := &bucketHandle{
		bucket: client.Bucket("test-bucket"), bucketType: &gcs.BucketType{}, writeConfig: &cfg.WriteConfig{},
	}

	// Act
	object, err := bucket.CreateObject(ctx, &gcs.CreateObjectRequest{
		Name: "checkpoint", Contents: contents, ContentType: "application/octet-stream",
	})

	// Assert
	require.ErrorIs(t, err, sourceErr)
	assert.Nil(t, object)
	require.NotNil(t, contents.writer)
	require.Eventually(t, func() bool {
		_, writeErr := contents.writer.Write(nil)
		return errors.Is(writeErr, context.Canceled)
	}, time.Second, time.Millisecond, "the abandoned SDK writer must be canceled")
	assert.NoError(t, ctx.Err(), "upload cleanup must not cancel the caller's context")
}
