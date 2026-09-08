// Copyright 2015 Google LLC
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
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const prefix = ".gcsfuse_tmp/"

type composeObjectCreatorHelper struct {
	assert      *assert.Assertions
	require     *require.Assertions
	ctx         context.Context
	bucket      *storage.TestifyMockBucket
	creator     objectCreator
	srcObject   gcs.Object
	srcContents string
	mtime       time.Time
}

func newComposeObjectCreatorHelper(t *testing.T) *composeObjectCreatorHelper {
	h := &composeObjectCreatorHelper{
		assert:  assert.New(t),
		require: require.New(t),
		ctx:     context.Background(),
	}

	h.bucket = new(storage.TestifyMockBucket)
	h.creator = newComposeObjectCreator(prefix, h.bucket)
	return h
}

func (h *composeObjectCreatorHelper) call() (*gcs.Object, error) {
	return h.creator.Create(
		h.ctx,
		h.srcObject.Name,
		&h.srcObject,
		&h.mtime,
		chunkRetryDeadlineSecs,
		chunkTransferTimeoutSecs,
		strings.NewReader(h.srcContents))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func TestComposeObjectCreator_CallsCreateObject(t *testing.T) {
	h := newComposeObjectCreatorHelper(t)
	h.srcContents = "taco"

	// CreateObject
	var req *gcs.CreateObjectRequest
	h.bucket.On("CreateObject", mock.Anything, mock.Anything).
		Return((*gcs.Object)(nil), errors.New("")).
		Run(func(args mock.Arguments) {
			req = args.Get(1).(*gcs.CreateObjectRequest)
		})

	// Call
	_, err := h.call()
	h.assert.Error(err)

	h.require.NotNil(req)
	h.assert.True(strings.HasPrefix(req.Name, prefix), "Name: %s", req.Name)
	if h.assert.NotNil(req.GenerationPrecondition) {
		h.assert.Equal(int64(0), *req.GenerationPrecondition)
	}

	b, err := io.ReadAll(req.Contents)
	h.require.NoError(err)
	h.assert.Equal(h.srcContents, string(b))
}

func TestComposeObjectCreator_CreateObjectFails(t *testing.T) {
	h := newComposeObjectCreatorHelper(t)

	// CreateObject
	h.bucket.On("CreateObject", mock.Anything, mock.Anything).
		Return((*gcs.Object)(nil), errors.New("taco"))

	// Call
	_, err := h.call()

	h.assert.ErrorContains(err, "CreateObject")
	h.assert.ErrorContains(err, "taco")
}

func TestComposeObjectCreator_CreateObjectReturnsPreconditionError(t *testing.T) {
	h := newComposeObjectCreatorHelper(t)

	// CreateObject
	h.bucket.On("CreateObject", mock.Anything, mock.Anything).
		Return((*gcs.Object)(nil), &gcs.PreconditionError{Err: errors.New("taco")})

	// Call
	_, err := h.call()

	var preconditionErr *gcs.PreconditionError
	h.assert.ErrorAs(err, &preconditionErr)
	h.assert.ErrorContains(err, "CreateObject")
	h.assert.ErrorContains(err, "taco")
}

func TestComposeObjectCreator_CallsComposeObjects(t *testing.T) {
	h := newComposeObjectCreatorHelper(t)
	h.srcObject.Name = "foo"
	h.srcObject.Generation = 17
	h.srcObject.MetaGeneration = 23
	h.mtime = time.Now().Add(123 * time.Second)

	// CreateObject
	tmpObject := &gcs.Object{
		Name:       "bar",
		Generation: 19,
	}

	h.bucket.On("CreateObject", mock.Anything, mock.Anything).
		Return(tmpObject, nil)

	// ComposeObjects
	var req *gcs.ComposeObjectsRequest
	h.bucket.On("ComposeObjects", mock.Anything, mock.Anything).
		Return((*gcs.Object)(nil), errors.New("")).
		Run(func(args mock.Arguments) {
			req = args.Get(1).(*gcs.ComposeObjectsRequest)
		})

	// DeleteObject
	h.bucket.On("DeleteObject", mock.Anything, mock.MatchedBy(func(r *gcs.DeleteObjectRequest) bool {
		return r.Name == tmpObject.Name
	})).Return(nil)

	// Call
	_, err := h.call()
	h.assert.Error(err)

	h.require.NotNil(req)
	h.assert.Equal(h.srcObject.Name, req.DstName)
	if h.assert.NotNil(req.DstGenerationPrecondition) {
		h.assert.Equal(h.srcObject.Generation, *req.DstGenerationPrecondition)
	}
	if h.assert.NotNil(req.DstMetaGenerationPrecondition) {
		h.assert.Equal(h.srcObject.MetaGeneration, *req.DstMetaGenerationPrecondition)
	}

	h.assert.Equal(1, len(req.Metadata))
	h.assert.Equal(h.mtime.UTC().Format(time.RFC3339Nano), req.Metadata["gcsfuse_mtime"])

	h.require.Equal(2, len(req.Sources))
	var src gcs.ComposeSource

	src = req.Sources[0]
	h.assert.Equal(h.srcObject.Name, src.Name)
	h.assert.Equal(h.srcObject.Generation, src.Generation)

	src = req.Sources[1]
	h.assert.Equal(tmpObject.Name, src.Name)
	h.assert.Equal(tmpObject.Generation, src.Generation)
	h.assert.True(req.DeleteSourceObjects)
}

func TestComposeObjectCreator_CallsComposeObjectsWithObjectProperties(t *testing.T) {
	h := newComposeObjectCreatorHelper(t)
	h.srcObject.Name = "foo"
	h.srcObject.Generation = 17
	h.srcObject.MetaGeneration = 23
	h.srcObject.CacheControl = "testCacheControl"
	h.srcObject.ContentDisposition = "inline"
	h.srcObject.ContentEncoding = "gzip"
	h.srcObject.ContentType = "text/plain"
	h.srcObject.CustomTime = "2022-04-02T00:30:00Z"
	h.srcObject.EventBasedHold = true
	h.srcObject.StorageClass = "STANDARD"
	h.srcObject.Metadata = map[string]string{
		"test_key": "test_value",
	}
	h.mtime = time.Now().Add(123 * time.Second)

	// CreateObject
	tmpObject := &gcs.Object{
		Name:       "bar",
		Generation: 19,
	}

	h.bucket.On("CreateObject", mock.Anything, mock.Anything).
		Return(tmpObject, nil)

	// ComposeObjects
	var req *gcs.ComposeObjectsRequest
	h.bucket.On("ComposeObjects", mock.Anything, mock.Anything).
		Return((*gcs.Object)(nil), errors.New("")).
		Run(func(args mock.Arguments) {
			req = args.Get(1).(*gcs.ComposeObjectsRequest)
		})

	// DeleteObject
	h.bucket.On("DeleteObject", mock.Anything, mock.MatchedBy(func(r *gcs.DeleteObjectRequest) bool {
		return r.Name == tmpObject.Name
	})).Return(nil)

	// Call
	_, _ = h.call()

	h.require.NotNil(req)
	h.assert.Equal(h.srcObject.Name, req.DstName)
	if h.assert.NotNil(req.DstGenerationPrecondition) {
		h.assert.Equal(h.srcObject.Generation, *req.DstGenerationPrecondition)
	}
	if h.assert.NotNil(req.DstMetaGenerationPrecondition) {
		h.assert.Equal(h.srcObject.MetaGeneration, *req.DstMetaGenerationPrecondition)
	}
	h.assert.Equal(h.srcObject.CacheControl, req.CacheControl)
	h.assert.Equal(h.srcObject.ContentDisposition, req.ContentDisposition)
	h.assert.Equal(h.srcObject.ContentEncoding, req.ContentEncoding)
	h.assert.Equal(h.srcObject.ContentType, req.ContentType)
	h.assert.Equal(h.srcObject.CustomTime, req.CustomTime)
	h.assert.Equal(h.srcObject.EventBasedHold, req.EventBasedHold)

	h.assert.Equal(2, len(req.Metadata))
	h.assert.Equal(h.mtime.UTC().Format(time.RFC3339Nano), req.Metadata["gcsfuse_mtime"])
	h.assert.Equal("test_value", req.Metadata["test_key"])

	h.require.Equal(2, len(req.Sources))
	var src gcs.ComposeSource

	src = req.Sources[0]
	h.assert.Equal(h.srcObject.Name, src.Name)
	h.assert.Equal(h.srcObject.Generation, src.Generation)

	src = req.Sources[1]
	h.assert.Equal(tmpObject.Name, src.Name)
	h.assert.Equal(tmpObject.Generation, src.Generation)
	h.assert.True(req.DeleteSourceObjects)
}

func TestComposeObjectCreator_ComposeObjectsFails(t *testing.T) {
	h := newComposeObjectCreatorHelper(t)

	// CreateObject
	tmpObject := &gcs.Object{
		Name: "bar",
	}

	h.bucket.On("CreateObject", mock.Anything, mock.Anything).
		Return(tmpObject, nil)

	// ComposeObjects
	h.bucket.On("ComposeObjects", mock.Anything, mock.Anything).
		Return((*gcs.Object)(nil), errors.New("taco"))

	// DeleteObject
	h.bucket.On("DeleteObject", mock.Anything, mock.MatchedBy(func(r *gcs.DeleteObjectRequest) bool {
		return r.Name == tmpObject.Name
	})).Return(nil)

	// Call
	_, err := h.call()

	h.assert.ErrorContains(err, "ComposeObjects")
	h.assert.ErrorContains(err, "taco")
}

func TestComposeObjectCreator_ComposeObjectsReturnsPreconditionError(t *testing.T) {
	h := newComposeObjectCreatorHelper(t)

	// CreateObject
	tmpObject := &gcs.Object{
		Name: "bar",
	}

	h.bucket.On("CreateObject", mock.Anything, mock.Anything).
		Return(tmpObject, nil)

	// ComposeObjects
	h.bucket.On("ComposeObjects", mock.Anything, mock.Anything).
		Return((*gcs.Object)(nil), &gcs.PreconditionError{Err: errors.New("taco")})

	// DeleteObject
	h.bucket.On("DeleteObject", mock.Anything, mock.MatchedBy(func(r *gcs.DeleteObjectRequest) bool {
		return r.Name == tmpObject.Name
	})).Return(nil)

	// Call
	_, err := h.call()

	var preconditionErr *gcs.PreconditionError
	h.assert.ErrorAs(err, &preconditionErr)
	h.assert.ErrorContains(err, "ComposeObjects")
	h.assert.ErrorContains(err, "taco")
}

func TestComposeObjectCreator_ComposeObjectsReturnsNotFoundError(t *testing.T) {
	h := newComposeObjectCreatorHelper(t)

	// CreateObject
	tmpObject := &gcs.Object{
		Name: "bar",
	}

	h.bucket.On("CreateObject", mock.Anything, mock.Anything).
		Return(tmpObject, nil)

	// ComposeObjects
	h.bucket.On("ComposeObjects", mock.Anything, mock.Anything).
		Return((*gcs.Object)(nil), &gcs.NotFoundError{Err: errors.New("taco")})

	// DeleteObject
	h.bucket.On("DeleteObject", mock.Anything, mock.MatchedBy(func(r *gcs.DeleteObjectRequest) bool {
		return r.Name == tmpObject.Name
	})).Return(nil)

	// Call
	_, err := h.call()

	var preconditionErr *gcs.PreconditionError
	h.assert.ErrorAs(err, &preconditionErr)
	h.assert.ErrorContains(err, "ComposeObjects")
	h.assert.ErrorContains(err, "taco")
}

func TestComposeObjectCreator_ComposeObjectsFails_DeleteObjectFails(t *testing.T) {
	h := newComposeObjectCreatorHelper(t)

	// CreateObject
	tmpObject := &gcs.Object{
		Name: "bar",
	}

	h.bucket.On("CreateObject", mock.Anything, mock.Anything).
		Return(tmpObject, nil)

	// ComposeObjects
	h.bucket.On("ComposeObjects", mock.Anything, mock.Anything).
		Return((*gcs.Object)(nil), errors.New("compose failed"))

	// DeleteObject fails
	h.bucket.On("DeleteObject", mock.Anything, mock.MatchedBy(func(r *gcs.DeleteObjectRequest) bool {
		return r.Name == tmpObject.Name
	})).Return(errors.New("delete failed"))

	// Call
	_, err := h.call()

	h.assert.ErrorContains(err, "ComposeObjects")
	h.assert.ErrorContains(err, "compose failed")
	h.assert.ErrorContains(err, "DeleteObject")
	h.assert.ErrorContains(err, "delete failed")
}

func TestComposeObjectCreator_ComposeSucceeds(t *testing.T) {
	h := newComposeObjectCreatorHelper(t)

	// CreateObject
	tmpObject := &gcs.Object{
		Name: "bar",
	}

	h.bucket.On("CreateObject", mock.Anything, mock.Anything).
		Return(tmpObject, nil)

	// ComposeObjects
	composed := &gcs.Object{}
	h.bucket.On("ComposeObjects", mock.Anything, mock.Anything).
		Return(composed, nil)

	// Call
	o, err := h.call()

	h.require.NoError(err)
	h.assert.Equal(composed, o)
}
