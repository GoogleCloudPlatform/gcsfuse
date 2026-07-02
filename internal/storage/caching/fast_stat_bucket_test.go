// Copyright 2023 Google LLC
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

package caching_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	gostorage "cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/caching"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const primaryCacheTTL = time.Second
const negativeCacheTTL = time.Second * 5
const isTypeCacheDeprecated = true
const isImplicitDir = true
const isEnableEmptyManagedFolders = false

type TestifyMockStatCache struct {
	mock.Mock
}

var _ metadata.StatCache = (*TestifyMockStatCache)(nil)

func (m *TestifyMockStatCache) Insert(obj *gcs.MinObject, expiration time.Time) {
	m.Called(obj, expiration)
}

func (m *TestifyMockStatCache) InsertImplicitDir(objectName string, expiration time.Time) {
	m.Called(objectName, expiration)
}

func (m *TestifyMockStatCache) AddNegativeEntry(name string, expiration time.Time) {
	m.Called(name, expiration)
}

func (m *TestifyMockStatCache) Erase(name string) {
	m.Called(name)
}

func (m *TestifyMockStatCache) LookUp(name string, now time.Time) (bool, *gcs.MinObject) {
	args := m.Called(name, now)
	var obj *gcs.MinObject
	if args.Get(1) != nil {
		obj = args.Get(1).(*gcs.MinObject)
	}
	return args.Bool(0), obj
}

func (m *TestifyMockStatCache) InsertFolder(f *gcs.Folder, expiration time.Time) {
	m.Called(f, expiration)
}

func (m *TestifyMockStatCache) LookUpFolder(folderName string, now time.Time) (bool, *gcs.Folder) {
	args := m.Called(folderName, now)
	var f *gcs.Folder
	if args.Get(1) != nil {
		f = args.Get(1).(*gcs.Folder)
	}
	return args.Bool(0), f
}

func (m *TestifyMockStatCache) AddNegativeEntryForFolder(folderName string, expiration time.Time) {
	m.Called(folderName, expiration)
}

func (m *TestifyMockStatCache) EraseEntriesWithGivenPrefix(prefix string) {
	m.Called(prefix)
}

type FastStatBucketSuite struct {
	suite.Suite
	cache   *TestifyMockStatCache
	clock   timeutil.SimulatedClock
	wrapped *storage.TestifyMockBucket
	bucket  gcs.Bucket
}

func (s *FastStatBucketSuite) SetupTest() {
	// Set up a fixed, non-zero time.
	s.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))

	// Set up dependencies.
	s.cache = new(TestifyMockStatCache)
	s.wrapped = new(storage.TestifyMockBucket)

	s.bucket = caching.NewFastStatBucket(
		primaryCacheTTL,
		s.cache,
		&s.clock,
		s.wrapped,
		negativeCacheTTL,
		isTypeCacheDeprecated,
		isImplicitDir,
		isEnableEmptyManagedFolders)
}

func TestFastStatBucket(t *testing.T) {
	suite.Run(t, new(FastStatBucketSuite))
}

// //////////////////////////////////////////////////////////////////////
// CreateObject
// //////////////////////////////////////////////////////////////////////

func (s *FastStatBucketSuite) Test_CreateObject_CallsEraseAndWrapped() {
	const name = "taco"

	// Erase
	s.cache.On("Erase", name).Return().Once()

	// Wrapped
	var wrappedReq *gcs.CreateObjectRequest
	s.wrapped.On("CreateObject", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			wrappedReq = args.Get(1).(*gcs.CreateObjectRequest)
		}).
		Return(nil, errors.New("")).
		Once()

	// Call
	req := &gcs.CreateObjectRequest{
		Name: name,
	}

	_, _ = s.bucket.CreateObject(context.TODO(), req)

	s.NotNil(wrappedReq)
	s.Equal(req, wrappedReq)
}

func (s *FastStatBucketSuite) Test_CreateObject_WrappedFails() {
	// Erase
	s.cache.On("Erase", mock.Anything).Return().Once()

	// Wrapped
	s.wrapped.On("CreateObject", mock.Anything, mock.Anything).
		Return(nil, errors.New("taco")).
		Once()

	// Call
	_, err := s.bucket.CreateObject(context.TODO(), &gcs.CreateObjectRequest{})

	s.ErrorContains(err, "taco")
}

func (s *FastStatBucketSuite) Test_CreateObject_WrappedSucceeds() {
	const name = "taco"

	// Erase
	s.cache.On("Erase", mock.Anything).Return().Once()

	// Wrapped
	obj := &gcs.Object{
		Name:       name,
		Generation: 1234,
	}

	s.wrapped.On("CreateObject", mock.Anything, mock.Anything).
		Return(obj, nil).
		Once()

	// Insert
	s.cache.On("Insert", mock.Anything, s.clock.Now().Add(primaryCacheTTL)).Return().Once()

	// Call
	o, err := s.bucket.CreateObject(context.TODO(), &gcs.CreateObjectRequest{})

	s.NoError(err)
	s.Equal(obj, o)
}

// //////////////////////////////////////////////////////////////////////
// CreateObjectChunkWriter
// //////////////////////////////////////////////////////////////////////

func (s *FastStatBucketSuite) Test_CreateObjectChunkWriter_CallsWrappedWithExpectedParameters() {
	const name = "taco"
	// Wrapped
	var wrappedReq *gcs.CreateObjectRequest
	var wrappedChunkSize int
	var wrappedCallback func(_ int64)
	s.wrapped.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			wrappedReq = args.Get(1).(*gcs.CreateObjectRequest)
			wrappedChunkSize = args.Int(2)
			wrappedCallback = args.Get(3).(func(int64))
		}).
		Return(nil, errors.New("")).
		Once()
	// Call
	req := &gcs.CreateObjectRequest{
		Name: name,
	}
	chunkSize := 1024
	callback := func(_ int64) {
		fmt.Println("callback called!")
	}

	_, _ = s.bucket.CreateObjectChunkWriter(context.TODO(), req, chunkSize, callback)

	s.NotNil(wrappedReq)
	s.Equal(req, wrappedReq)
	s.Equal(chunkSize, wrappedChunkSize)
	s.NotNil(wrappedCallback)
}

func (s *FastStatBucketSuite) Test_CreateObjectChunkWriter_WrappedFails() {
	chunkSize := 1024
	progressFunc := func(_ int64) {}
	ctx := context.TODO()
	req := &gcs.CreateObjectRequest{}
	// Wrapped
	s.wrapped.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("taco")).
		Once()

	// Call
	_, err := s.bucket.CreateObjectChunkWriter(ctx, req, chunkSize, progressFunc)

	s.ErrorContains(err, "taco")
}

func (s *FastStatBucketSuite) Test_CreateObjectChunkWriter_WrappedSucceeds() {
	chunkSize := 1024
	progressFunc := func(_ int64) {}
	ctx := context.TODO()
	req := &gcs.CreateObjectRequest{}
	// Wrapped
	wr := &storage.ObjectWriter{
		Writer: &gostorage.Writer{ChunkSize: chunkSize, ProgressFunc: progressFunc},
	}
	s.wrapped.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(wr, nil).
		Once()

	// Call
	gotWr, err := s.bucket.CreateObjectChunkWriter(ctx, req, chunkSize, progressFunc)

	s.NoError(err)
	s.Equal(wr, gotWr)
}

// //////////////////////////////////////////////////////////////////////
// CreateAppendableObjectWriter
// //////////////////////////////////////////////////////////////////////

func (s *FastStatBucketSuite) Test_CreateAppendableObjectWriter_CallsWrappedWithExpectedParameters() {
	const name = "taco"
	const offset int64 = 10
	const chunkSize = 1024
	ctx := context.TODO()
	// Wrapped
	var wrappedReq *gcs.CreateObjectChunkWriterRequest
	s.wrapped.On("CreateAppendableObjectWriter", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			wrappedReq = args.Get(1).(*gcs.CreateObjectChunkWriterRequest)
		}).
		Return(nil, errors.New("")).
		Once()
	// Call
	req := &gcs.CreateObjectChunkWriterRequest{
		CreateObjectRequest: gcs.CreateObjectRequest{
			Name: name,
		},
		Offset:    offset,
		ChunkSize: chunkSize,
	}

	_, _ = s.bucket.CreateAppendableObjectWriter(ctx, req)

	s.NotNil(wrappedReq)
	s.Equal(req, wrappedReq)
}

func (s *FastStatBucketSuite) Test_CreateAppendableObjectWriter_WrappedFails() {
	ctx := context.TODO()
	req := &gcs.CreateObjectChunkWriterRequest{}
	// Wrapped
	s.wrapped.On("CreateAppendableObjectWriter", mock.Anything, mock.Anything).
		Return(nil, errors.New("taco")).
		Once()

	// Call
	_, err := s.bucket.CreateAppendableObjectWriter(ctx, req)

	s.ErrorContains(err, "taco")
}

func (s *FastStatBucketSuite) Test_CreateAppendableObjectWriter_WrappedSucceeds() {
	ctx := context.TODO()
	req := &gcs.CreateObjectChunkWriterRequest{}
	// Wrapped
	wr := &storage.ObjectWriter{
		Writer: &gostorage.Writer{},
	}
	s.wrapped.On("CreateAppendableObjectWriter", mock.Anything, mock.Anything).
		Return(wr, nil).
		Once()

	// Call
	gotWr, err := s.bucket.CreateAppendableObjectWriter(ctx, req)

	s.NoError(err)
	s.Equal(wr, gotWr)
}

func (s *FastStatBucketSuite) Test_CreateAppendableObjectWriter_WrappedReturnsPreconditionError() {
	const name = "taco"
	ctx := context.TODO()
	req := &gcs.CreateObjectChunkWriterRequest{
		CreateObjectRequest: gcs.CreateObjectRequest{
			Name: name,
		},
	}
	// Erase
	s.cache.On("Erase", name).Return().Once()
	// Wrapped
	s.wrapped.On("CreateAppendableObjectWriter", mock.Anything, mock.Anything).
		Return(nil, &gcs.PreconditionError{Err: errors.New("precondition failed")}).
		Once()

	// Call
	_, err := s.bucket.CreateAppendableObjectWriter(ctx, req)

	s.ErrorContains(err, "precondition failed")
}

// //////////////////////////////////////////////////////////////////////
// FinalizeUpload
// //////////////////////////////////////////////////////////////////////

func (s *FastStatBucketSuite) Test_FinalizeUpload_CallsEraseAndWrappedWithExpectedParameter() {
	const name = "taco"
	writer := &storage.ObjectWriter{
		Writer: &gostorage.Writer{ObjectAttrs: gostorage.ObjectAttrs{Name: name}},
	}
	// Erase
	s.cache.On("Erase", name).Return().Once()
	// Wrapped
	var wrappedWriter gcs.Writer
	s.wrapped.On("FinalizeUpload", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			wrappedWriter = args.Get(1).(gcs.Writer)
		}).
		Return(&gcs.MinObject{}, errors.New("")).
		Once()

	// Call
	_, _ = s.bucket.FinalizeUpload(context.TODO(), writer)

	s.NotNil(wrappedWriter)
	s.Equal(writer, wrappedWriter)
}

func (s *FastStatBucketSuite) Test_FinalizeUpload_WrappedFails() {
	writer := &storage.ObjectWriter{
		Writer: &gostorage.Writer{ObjectAttrs: gostorage.ObjectAttrs{Name: "name"}},
	}
	// Erase
	s.cache.On("Erase", mock.Anything).Return().Once()
	// Wrapped
	s.wrapped.On("FinalizeUpload", mock.Anything, mock.Anything).
		Return(&gcs.MinObject{}, errors.New("taco")).
		Once()

	// Call
	o, err := s.bucket.FinalizeUpload(context.TODO(), writer)

	s.ErrorContains(err, "taco")
	s.NotNil(o)
}

func (s *FastStatBucketSuite) Test_FinalizeUpload_WrappedSucceeds() {
	const name = "taco"
	writer := &storage.ObjectWriter{
		Writer: &gostorage.Writer{ObjectAttrs: gostorage.ObjectAttrs{Name: name}},
	}
	// Erase
	s.cache.On("Erase", mock.Anything).Return().Once()
	// Wrapped
	s.wrapped.On("FinalizeUpload", mock.Anything, mock.Anything).
		Return(&gcs.MinObject{}, nil).
		Once()
	// Insert
	s.cache.On("Insert", mock.Anything, s.clock.Now().Add(primaryCacheTTL)).Return().Once()

	// Call
	o, err := s.bucket.FinalizeUpload(context.TODO(), writer)

	s.NoError(err)
	s.NotNil(o)
}

// //////////////////////////////////////////////////////////////////////
// FlushPendingWrites
// //////////////////////////////////////////////////////////////////////

func (s *FastStatBucketSuite) Test_FlushPendingWrites_WrappedFails() {
	const name = "taco"
	writer := &storage.ObjectWriter{
		Writer: &gostorage.Writer{ObjectAttrs: gostorage.ObjectAttrs{Name: name}},
	}
	// Expect cache Erase.
	s.cache.On("Erase", name).Return().Once()
	// Expect call to Wrapped method.
	var wrappedWriter gcs.Writer
	mockObject := &gcs.MinObject{Size: 10}
	s.wrapped.On("FlushPendingWrites", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			wrappedWriter = args.Get(1).(gcs.Writer)
		}).
		Return(mockObject, errors.New("taco")).
		Once()

	// Call.
	gotObject, err := s.bucket.FlushPendingWrites(context.TODO(), writer)

	s.Equal(writer, wrappedWriter)
	s.Equal(mockObject, gotObject)
	s.ErrorContains(err, "taco")
}

func (s *FastStatBucketSuite) Test_FlushPendingWrites_WrappedSucceeds() {
	const name = "taco"
	writer := &storage.ObjectWriter{
		Writer: &gostorage.Writer{ObjectAttrs: gostorage.ObjectAttrs{Name: name}},
	}
	// Expect cache Erase.
	s.cache.On("Erase", name).Return().Once()
	// Wrapped.
	mockObject := &gcs.MinObject{Size: 10}
	s.wrapped.On("FlushPendingWrites", mock.Anything, mock.Anything).
		Return(mockObject, nil).
		Once()
	// Insert.
	var cachedMinObject *gcs.MinObject
	s.cache.On("Insert", mock.Anything, s.clock.Now().Add(primaryCacheTTL)).
		Run(func(args mock.Arguments) {
			cachedMinObject = args.Get(0).(*gcs.MinObject)
		}).
		Return().
		Once()

	// Call
	gotObject, err := s.bucket.FlushPendingWrites(context.TODO(), writer)

	s.NoError(err)
	s.Equal(mockObject, gotObject)
	s.Equal(mockObject.Size, cachedMinObject.Size)
}

// //////////////////////////////////////////////////////////////////////
// CopyObject
// //////////////////////////////////////////////////////////////////////

func (s *FastStatBucketSuite) Test_CopyObject_CallsEraseAndWrapped() {
	const srcName = "taco"
	const dstName = "burrito"

	// Erase
	s.cache.On("Erase", dstName).Return().Once()

	// Wrapped
	var wrappedReq *gcs.CopyObjectRequest
	s.wrapped.On("CopyObject", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			wrappedReq = args.Get(1).(*gcs.CopyObjectRequest)
		}).
		Return(nil, errors.New("")).
		Once()

	// Call
	req := &gcs.CopyObjectRequest{
		SrcName: srcName,
		DstName: dstName,
	}

	_, _ = s.bucket.CopyObject(context.TODO(), req)

	s.NotNil(wrappedReq)
	s.Equal(req, wrappedReq)
}

func (s *FastStatBucketSuite) Test_CopyObject_WrappedFails() {
	// Erase
	s.cache.On("Erase", mock.Anything).Return().Once()

	// Wrapped
	s.wrapped.On("CopyObject", mock.Anything, mock.Anything).
		Return(nil, errors.New("taco")).
		Once()

	// Call
	_, err := s.bucket.CopyObject(context.TODO(), &gcs.CopyObjectRequest{})

	s.ErrorContains(err, "taco")
}

func (s *FastStatBucketSuite) Test_CopyObject_WrappedSucceeds() {
	const dstName = "burrito"

	// Erase
	s.cache.On("Erase", mock.Anything).Return().Once()

	// Wrapped
	obj := &gcs.Object{
		Name:       dstName,
		Generation: 1234,
	}

	s.wrapped.On("CopyObject", mock.Anything, mock.Anything).
		Return(obj, nil).
		Once()

	// Insert
	s.cache.On("Insert", mock.Anything, s.clock.Now().Add(primaryCacheTTL)).Return().Once()

	// Call
	o, err := s.bucket.CopyObject(context.TODO(), &gcs.CopyObjectRequest{})

	s.NoError(err)
	s.Equal(obj, o)
}

// //////////////////////////////////////////////////////////////////////
// ComposeObjects
// //////////////////////////////////////////////////////////////////////

func (s *FastStatBucketSuite) Test_ComposeObjects_CallsEraseAndWrapped() {
	const srcName = "taco"
	const dstName = "burrito"

	// Erase
	s.cache.On("Erase", dstName).Return().Once()

	// Wrapped
	var wrappedReq *gcs.ComposeObjectsRequest
	s.wrapped.On("ComposeObjects", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			wrappedReq = args.Get(1).(*gcs.ComposeObjectsRequest)
		}).
		Return(nil, errors.New("")).
		Once()

	// Call
	req := &gcs.ComposeObjectsRequest{
		DstName: dstName,
		Sources: []gcs.ComposeSource{
			{Name: srcName},
		},
	}

	_, _ = s.bucket.ComposeObjects(context.TODO(), req)

	s.NotNil(wrappedReq)
	s.Equal(req, wrappedReq)
}

func (s *FastStatBucketSuite) Test_ComposeObjects_WrappedFails() {
	// Erase
	s.cache.On("Erase", mock.Anything).Return().Once()

	// Wrapped
	s.wrapped.On("ComposeObjects", mock.Anything, mock.Anything).
		Return(nil, errors.New("taco")).
		Once()

	// Call
	_, err := s.bucket.ComposeObjects(context.TODO(), &gcs.ComposeObjectsRequest{})

	s.ErrorContains(err, "taco")
}

func (s *FastStatBucketSuite) Test_ComposeObjects_WrappedSucceeds() {
	const dstName = "burrito"

	// Erase
	s.cache.On("Erase", mock.Anything).Return().Once()

	// Wrapped
	obj := &gcs.Object{
		Name:       dstName,
		Generation: 1234,
	}

	s.wrapped.On("ComposeObjects", mock.Anything, mock.Anything).
		Return(obj, nil).
		Once()

	// Insert
	s.cache.On("Insert", mock.Anything, s.clock.Now().Add(primaryCacheTTL)).Return().Once()

	// Call
	o, err := s.bucket.ComposeObjects(context.TODO(), &gcs.ComposeObjectsRequest{})

	s.NoError(err)
	s.Equal(obj, o)
}

// //////////////////////////////////////////////////////////////////////
// StatObject
// //////////////////////////////////////////////////////////////////////

func (s *FastStatBucketSuite) Test_StatObject_CallsCache() {
	const name = "taco"

	// LookUp
	s.cache.On("LookUp", name, s.clock.Now()).
		Return(true, &gcs.MinObject{}).
		Once()

	// Call
	req := &gcs.StatObjectRequest{
		Name: name,
	}

	_, _, _ = s.bucket.StatObject(context.TODO(), req)
}

func (s *FastStatBucketSuite) Test_StatObject_CacheHit_Positive() {
	const name = "taco"

	// LookUp
	minObj := &gcs.MinObject{
		Name: name,
	}

	s.cache.On("LookUp", mock.Anything, mock.Anything).
		Return(true, minObj).
		Once()

	// Call
	req := &gcs.StatObjectRequest{
		Name: name,
	}

	m, e, err := s.bucket.StatObject(context.TODO(), req)
	s.NoError(err)
	s.Nil(e)
	s.NotNil(m)
	s.Equal(minObj, m)
}

func (s *FastStatBucketSuite) Test_StatObject_CacheHit_Negative() {
	const name = "taco"

	// LookUp
	s.cache.On("LookUp", mock.Anything, mock.Anything).
		Return(true, (*gcs.MinObject)(nil)).
		Once()

	// Call
	req := &gcs.StatObjectRequest{
		Name: name,
	}

	_, _, err := s.bucket.StatObject(context.TODO(), req)
	s.IsType(&gcs.NotFoundError{}, err)
}

func (s *FastStatBucketSuite) Test_StatObject_IgnoresCacheEntryWhenForceFetchFromGcsIsTrue() {
	const name = "taco"

	// Request
	req := &gcs.StatObjectRequest{
		Name:                           name,
		ForceFetchFromGcs:              true,
		ReturnExtendedObjectAttributes: true,
	}

	// Wrapped
	minObjFromGcs := &gcs.MinObject{
		Name: name,
	}
	extObjAttrFromGcs := &gcs.ExtendedObjectAttributes{
		CacheControl: "testControl",
	}

	s.wrapped.On("StatObject", mock.Anything, req).
		Return(minObjFromGcs, extObjAttrFromGcs, nil).
		Once()

	// Insert
	s.cache.On("Insert", mock.Anything, s.clock.Now().Add(primaryCacheTTL)).Return().Once()

	m, e, err := s.bucket.StatObject(context.TODO(), req)
	s.NoError(err)
	s.NotNil(m)
	s.NotNil(e)
	s.Equal(minObjFromGcs, m)
	s.Equal(extObjAttrFromGcs, e)
}

func (s *FastStatBucketSuite) Test_StatObject_ForceFetchFromGcsTrueAndReturnExtendedObjectAttributesFalse() {
	const name = "taco"

	// Request
	req := &gcs.StatObjectRequest{
		Name:                           name,
		ForceFetchFromGcs:              true,
		ReturnExtendedObjectAttributes: false,
	}

	// Wrapped
	minObjFromGcs := &gcs.MinObject{
		Name: name,
	}

	s.wrapped.On("StatObject", mock.Anything, req).
		Return(minObjFromGcs, &gcs.ExtendedObjectAttributes{}, nil).
		Once()

	// Insert
	s.cache.On("Insert", mock.Anything, s.clock.Now().Add(primaryCacheTTL)).Return().Once()

	m, e, err := s.bucket.StatObject(context.TODO(), req)
	s.NoError(err)
	s.NotNil(m)
	s.Nil(e)
}

func (s *FastStatBucketSuite) Test_StatObject_Panics_ForceFetchFromGcsFalseAndReturnExtendedObjectAttributesTrue() {
	const name = "taco"
	const panicMsg = "invalid StatObjectRequest: ForceFetchFromGcs: false and ReturnExtendedObjectAttributes: true"

	// Request
	req := &gcs.StatObjectRequest{
		Name:                           name,
		ForceFetchFromGcs:              false,
		ReturnExtendedObjectAttributes: true,
	}

	s.PanicsWithValue(panicMsg, func() {
		_, _, _ = s.bucket.StatObject(context.TODO(), req)
	})
}

func (s *FastStatBucketSuite) Test_StatObject_CallsWrapped() {
	const name = ""
	req := &gcs.StatObjectRequest{
		Name: name,
	}

	// LookUp
	s.cache.On("LookUp", mock.Anything, mock.Anything).
		Return(false, (*gcs.MinObject)(nil)).
		Once()

	// Wrapped
	s.wrapped.On("StatObject", mock.Anything, req).
		Return(nil, nil, errors.New("")).
		Once()

	// Call
	_, _, _ = s.bucket.StatObject(context.TODO(), req)
}

func (s *FastStatBucketSuite) Test_StatObject_WrappedFails() {
	const name = ""

	// LookUp
	s.cache.On("LookUp", mock.Anything, mock.Anything).
		Return(false, (*gcs.MinObject)(nil)).
		Once()

	// Wrapped
	s.wrapped.On("StatObject", mock.Anything, mock.Anything).
		Return(nil, nil, errors.New("taco")).
		Once()

	// Call
	req := &gcs.StatObjectRequest{
		Name: name,
	}

	_, _, err := s.bucket.StatObject(context.TODO(), req)
	s.ErrorContains(err, "taco")
}

func (s *FastStatBucketSuite) Test_StatObject_WrappedSaysNotFound() {
	const name = "taco"

	// LookUp
	s.cache.On("LookUp", mock.Anything, mock.Anything).
		Return(false, (*gcs.MinObject)(nil)).
		Once()

	// Wrapped
	s.wrapped.On("StatObject", mock.Anything, mock.Anything).
		Return(nil, nil, &gcs.NotFoundError{Err: errors.New("burrito")}).
		Once()

	// AddNegativeEntry
	s.cache.On("AddNegativeEntry", name, s.clock.Now().Add(negativeCacheTTL)).
		Return().
		Once()

	// Call
	req := &gcs.StatObjectRequest{
		Name: name,
	}

	_, _, err := s.bucket.StatObject(context.TODO(), req)
	s.IsType(&gcs.NotFoundError{}, err)
	s.ErrorContains(err, "burrito")
}

func (s *FastStatBucketSuite) Test_StatObject_WrappedSucceeds() {
	const name = "taco"

	// LookUp
	s.cache.On("LookUp", mock.Anything, mock.Anything).
		Return(false, (*gcs.MinObject)(nil)).
		Once()

	// Wrapped
	minObj := &gcs.MinObject{
		Name: name,
	}

	s.wrapped.On("StatObject", mock.Anything, mock.Anything).
		Return(minObj, nil, nil).
		Once()

	// Insert
	s.cache.On("Insert", mock.Anything, s.clock.Now().Add(primaryCacheTTL)).Return().Once()

	// Call
	req := &gcs.StatObjectRequest{
		Name: name,
	}

	m, _, err := s.bucket.StatObject(context.TODO(), req)
	s.NoError(err)
	s.Equal(minObj, m)
}

func (s *FastStatBucketSuite) Test_StatObject_ShouldReturnFromCacheWhenEntryIsPresent() {
	const name = "some-name"
	folder := &gcs.Folder{
		Name: name,
	}
	s.cache.On("LookUpFolder", name, mock.Anything).
		Return(true, folder).
		Once()

	result, err := s.bucket.GetFolder(context.TODO(), &gcs.GetFolderRequest{Name: name})

	s.NoError(err)
	s.Equal(folder, result)
}

func (s *FastStatBucketSuite) Test_StatObject_ShouldReturnNotFoundErrorWhenNilEntryIsReturned() {
	const name = "some-name"

	s.cache.On("LookUpFolder", name, mock.Anything).
		Return(true, (*gcs.Folder)(nil)).
		Once()

	result, err := s.bucket.GetFolder(context.TODO(), &gcs.GetFolderRequest{Name: name})

	s.IsType(&gcs.NotFoundError{}, err)
	s.Nil(result)
}

func (s *FastStatBucketSuite) Test_StatObject_ShouldCallGetFolderWhenEntryIsNotPresent() {
	const name = "some-name"
	folder := &gcs.Folder{
		Name: name,
	}
	getFolderReq := &gcs.GetFolderRequest{Name: name}

	s.cache.On("LookUpFolder", name, mock.Anything).
		Return(false, (*gcs.Folder)(nil)).
		Once()
	s.cache.On("InsertFolder", folder, mock.Anything).
		Return().
		Once()
	s.wrapped.On("GetFolder", mock.Anything, getFolderReq).
		Return(folder, nil).
		Once()

	result, err := s.bucket.GetFolder(context.TODO(), getFolderReq)

	s.NoError(err)
	s.Equal(folder, result)
}

func (s *FastStatBucketSuite) Test_StatObject_ShouldReturnNilWhenErrorIsReturnedFromGetFolder() {
	const name = "some-name"
	err := errors.New("connection error")
	getFolderReq := &gcs.GetFolderRequest{Name: name}

	s.cache.On("LookUpFolder", name, mock.Anything).
		Return(false, (*gcs.Folder)(nil)).
		Once()
	s.wrapped.On("GetFolder", mock.Anything, getFolderReq).
		Return(nil, err).
		Once()

	folder, result := s.bucket.GetFolder(context.TODO(), getFolderReq)

	s.Nil(folder)
	s.Equal(err, result)
}

func (s *FastStatBucketSuite) Test_StatObject_RenameFolder() {
	const name = "some-name"
	const newName = "new-name"
	var folder = &gcs.Folder{
		Name: newName,
	}

	s.cache.On("EraseEntriesWithGivenPrefix", name).Return().Once()
	s.cache.On("InsertFolder", folder, mock.Anything).Return().Once()
	s.wrapped.On("RenameFolder", mock.Anything, name, newName).Return(folder, nil).Once()

	result, err := s.bucket.RenameFolder(context.Background(), name, newName)

	s.NoError(err)
	s.Equal(result, folder)
}

func (s *FastStatBucketSuite) Test_StatObject_FetchOnlyFromCacheFalse() {
	const name = "taco"
	req := &gcs.StatObjectRequest{
		Name:               name,
		FetchOnlyFromCache: false,
	}
	s.cache.On("LookUp", name, mock.Anything).
		Return(false, (*gcs.MinObject)(nil)).
		Once()

	minObj := &gcs.MinObject{Name: name}
	s.wrapped.On("StatObject", mock.Anything, mock.Anything).
		Return(minObj, nil, nil).
		Once()
	s.cache.On("Insert", mock.Anything, mock.Anything).Return().Once()

	m, _, err := s.bucket.StatObject(context.TODO(), req)

	s.NoError(err)
	s.Equal(minObj, m)
}

func (s *FastStatBucketSuite) Test_StatObject_FetchOnlyFromCacheTrue_CacheHitPositive() {
	const name = "taco"
	req := &gcs.StatObjectRequest{
		Name:               name,
		FetchOnlyFromCache: true,
	}
	minObj := &gcs.MinObject{Name: name}
	s.cache.On("LookUp", name, mock.Anything).
		Return(true, minObj).
		Once()

	m, _, err := s.bucket.StatObject(context.TODO(), req)

	s.NoError(err)
	s.Equal(minObj, m)
}

func (s *FastStatBucketSuite) Test_StatObject_FetchOnlyFromCacheTrue_CacheHitNegative() {
	const name = "taco"
	req := &gcs.StatObjectRequest{
		Name:               name,
		FetchOnlyFromCache: true,
	}
	s.cache.On("LookUp", name, mock.Anything).
		Return(true, (*gcs.MinObject)(nil)).
		Once()

	_, _, err := s.bucket.StatObject(context.TODO(), req)

	s.IsType(&gcs.NotFoundError{}, err)
}

func (s *FastStatBucketSuite) Test_StatObject_FetchOnlyFromCacheTrue_CacheMiss() {
	const name = "taco"
	req := &gcs.StatObjectRequest{
		Name:               name,
		FetchOnlyFromCache: true,
	}
	s.cache.On("LookUp", name, mock.Anything).
		Return(false, (*gcs.MinObject)(nil)).
		Once()

	_, _, err := s.bucket.StatObject(context.TODO(), req)

	s.IsType(&caching.CacheMissError{}, err)
}

// //////////////////////////////////////////////////////////////////////
// ListObjects
// //////////////////////////////////////////////////////////////////////

func (s *FastStatBucketSuite) Test_ListObjects_WrappedFails() {
	// Wrapped
	s.wrapped.On("ListObjects", mock.Anything, mock.Anything).
		Return(nil, errors.New("taco")).
		Once()

	// Call
	_, err := s.bucket.ListObjects(context.TODO(), &gcs.ListObjectsRequest{})
	s.ErrorContains(err, "taco")
}

func (s *FastStatBucketSuite) Test_ListObjects_EmptyListing() {
	// Wrapped
	expected := &gcs.Listing{}

	s.wrapped.On("BucketType").
		Return(gcs.BucketType{}).
		Once()

	s.wrapped.On("ListObjects", mock.Anything, mock.Anything).
		Return(expected, nil).
		Once()

	// Call
	listing, err := s.bucket.ListObjects(context.TODO(), &gcs.ListObjectsRequest{})

	s.NoError(err)
	s.Equal(expected, listing)
}

func (s *FastStatBucketSuite) Test_ListObjects_EmptyListingForHNS() {
	// wrapped
	expected := &gcs.Listing{}

	s.wrapped.On("BucketType").
		Return(gcs.BucketType{Hierarchical: true}).
		Once()

	s.wrapped.On("ListObjects", mock.Anything, mock.Anything).
		Return(expected, nil).
		Once()

	// call
	listing, err := s.bucket.ListObjects(context.TODO(), &gcs.ListObjectsRequest{})

	s.NoError(err)
	s.Equal(expected, listing)
}

func (s *FastStatBucketSuite) Test_ListObjects_NonEmptyListing() {
	// Wrapped
	o0 := &gcs.MinObject{Name: "taco"}
	o1 := &gcs.MinObject{Name: "burrito"}

	expected := &gcs.Listing{
		MinObjects: []*gcs.MinObject{o0, o1},
	}

	s.wrapped.On("BucketType").
		Return(gcs.BucketType{}).
		Once()

	s.wrapped.On("ListObjects", mock.Anything, mock.Anything).
		Return(expected, nil).
		Once()

	// Insert
	s.cache.On("Insert", mock.Anything, s.clock.Now().Add(primaryCacheTTL)).Return().Times(2)
	s.cache.On("InsertImplicitDir", mock.Anything, s.clock.Now().Add(primaryCacheTTL)).Return().Times(1)

	// Call
	listing, err := s.bucket.ListObjects(context.TODO(), &gcs.ListObjectsRequest{})

	s.NoError(err)
	s.Equal(expected, listing)
}

func (s *FastStatBucketSuite) Test_ListObjects_NonEmptyListingForHNS() {
	// wrapped
	o0 := &gcs.MinObject{Name: "taco"}
	o1 := &gcs.MinObject{Name: "burrito"}

	expected := &gcs.Listing{
		MinObjects:    []*gcs.MinObject{o0, o1},
		CollapsedRuns: []string{"p0", "p1/"},
	}

	s.wrapped.On("BucketType").
		Return(gcs.BucketType{Hierarchical: true}).
		Once()

	s.wrapped.On("ListObjects", mock.Anything, mock.Anything).
		Return(expected, nil).
		Once()

	// insert
	s.cache.On("Insert", mock.Anything, s.clock.Now().Add(primaryCacheTTL)).Return().Times(2)
	s.cache.On("InsertFolder", mock.Anything, s.clock.Now().Add(primaryCacheTTL)).Return().Times(1)

	// call
	listing, err := s.bucket.ListObjects(context.TODO(), &gcs.ListObjectsRequest{})

	s.NoError(err)
	s.Equal(expected, listing)
}

func (s *FastStatBucketSuite) Test_ListObjects_NonEmptyListingWithCancelledContext() {
	// Wrapped
	o0 := &gcs.MinObject{Name: "taco"}
	o1 := &gcs.MinObject{Name: "burrito"}
	expected := &gcs.Listing{
		MinObjects: []*gcs.MinObject{o0, o1},
	}
	s.wrapped.On("BucketType").
		Return(gcs.BucketType{}).
		Once()
	// Create a cancellable context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.
	s.wrapped.On("ListObjects", ctx, mock.Anything).
		Return(expected, nil).
		Once()

	// Call
	listing, err := s.bucket.ListObjects(ctx, &gcs.ListObjectsRequest{})

	s.NoError(err)
	s.Equal(expected, listing)
}

func (s *FastStatBucketSuite) Test_ListObjects_NonEmptyListingWithCancelledContextForHNS() {
	// wrapped
	o0 := &gcs.MinObject{Name: "taco"}
	o1 := &gcs.MinObject{Name: "burrito"}
	expected := &gcs.Listing{
		MinObjects:    []*gcs.MinObject{o0, o1},
		CollapsedRuns: []string{"p0", "p1/"},
	}
	s.wrapped.On("BucketType").
		Return(gcs.BucketType{Hierarchical: true}).
		Once()
	// Create a cancellable context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.
	s.wrapped.On("ListObjects", ctx, mock.Anything).
		Return(expected, nil).
		Once()

	// call
	listing, err := s.bucket.ListObjects(ctx, &gcs.ListObjectsRequest{})

	s.NoError(err)
	s.Equal(expected, listing)
}

// //////////////////////////////////////////////////////////////////////
// ListObjectsTest_InsertListing
// //////////////////////////////////////////////////////////////////////

func (s *FastStatBucketSuite) callAndVerify(ctx context.Context, isHNS bool, listing *gcs.Listing, prefix string, expectedInserts []*gcs.MinObject, expectedImplicitDirs []string) {
	expectNegativeEntry := len(listing.MinObjects) == 0 && len(listing.CollapsedRuns) == 0 && prefix != "" && listing.ContinuationToken == ""
	s.callAndVerifyWithRequest(ctx, isHNS, listing, &gcs.ListObjectsRequest{Prefix: prefix}, expectedInserts, expectedImplicitDirs, expectNegativeEntry)
}

func (s *FastStatBucketSuite) callAndVerifyWithRequest(ctx context.Context, isHNS bool, listing *gcs.Listing, req *gcs.ListObjectsRequest, expectedInserts []*gcs.MinObject, expectedImplicitDirs []string, expectNegativeEntry bool) {
	// Wrapped
	s.wrapped.On("BucketType").
		Return(gcs.BucketType{Hierarchical: isHNS}).
		Once()
	s.wrapped.On("ListObjects", mock.Anything, mock.Anything).
		Return(listing, nil).
		Once()
	// Register expectations.
	for _, obj := range expectedInserts {
		s.cache.On("Insert", obj, mock.Anything).Return().Once()
	}
	for _, dir := range expectedImplicitDirs {
		s.cache.On("InsertImplicitDir", dir, mock.Anything).Return().Once()
	}
	if expectNegativeEntry {
		s.cache.On("AddNegativeEntry", req.Prefix, mock.Anything).Return().Once()
	}

	// Call
	gotListing, err := s.bucket.ListObjects(ctx, req)

	s.NoError(err)
	s.Equal(listing, gotListing)
}

// An empty windowed listing (StartOffset set) covers only a slice of the
// namespace and must not produce a negative entry for the directory.
func (s *FastStatBucketSuite) Test_ListObjects_InsertListing_EmptyListingWithStartOffsetDoesNotCacheNegativeEntry() {
	listing := &gcs.Listing{}
	req := &gcs.ListObjectsRequest{Prefix: "dir/", StartOffset: "dir/zzz"}

	s.callAndVerifyWithRequest(context.TODO(), false, listing, req, nil, nil, false)
}

// An empty continuation page proves nothing about the prefix as a whole and
// must not produce a negative entry for the directory.
func (s *FastStatBucketSuite) Test_ListObjects_InsertListing_EmptyListingWithRequestContinuationTokenDoesNotCacheNegativeEntry() {
	listing := &gcs.Listing{}
	req := &gcs.ListObjectsRequest{Prefix: "dir/", ContinuationToken: "token-from-previous-page"}

	s.callAndVerifyWithRequest(context.TODO(), false, listing, req, nil, nil, false)
}

// An empty page that carries a continuation token is an incomplete listing and
// must not produce a negative entry for the directory.
func (s *FastStatBucketSuite) Test_ListObjects_InsertListing_EmptyPageWithResponseContinuationTokenDoesNotCacheNegativeEntry() {
	listing := &gcs.Listing{ContinuationToken: "next-page"}
	req := &gcs.ListObjectsRequest{Prefix: "dir/"}

	s.callAndVerifyWithRequest(context.TODO(), false, listing, req, nil, nil, false)
}

// A non-empty windowed listing (StartOffset set) must insert the returned
// objects and still infer the parent implicit directory.
func (s *FastStatBucketSuite) Test_ListObjects_InsertListing_NonEmptyListingWithStartOffsetInsertsPositiveEntries() {
	listing := &gcs.Listing{
		MinObjects: []*gcs.MinObject{
			{Name: "dir/c", Size: 10},
		},
	}
	req := &gcs.ListObjectsRequest{
		Prefix:      "dir/",
		StartOffset: "dir/b",
	}
	expectedInserts := []*gcs.MinObject{
		{Name: "dir/c", Size: 10},
	}
	expectedImplicitDirs := []string{"dir/"}

	s.callAndVerifyWithRequest(
		context.TODO(),
		false, // isHNS
		listing,
		req,
		expectedInserts,
		expectedImplicitDirs,
		false, // expectNegativeEntry
	)
}

func (s *FastStatBucketSuite) Test_ListObjects_InsertListing_EmptyListing() {
	listing := &gcs.Listing{}
	expectedInserts := []*gcs.MinObject{}
	expectedImplicitDirs := []string{}

	s.callAndVerify(context.TODO(), false, listing, "dir/", expectedInserts, expectedImplicitDirs)
}

func (s *FastStatBucketSuite) Test_ListObjects_InsertListing_ObjectsOnly() {
	listing := &gcs.Listing{
		MinObjects: []*gcs.MinObject{
			{Name: "dir/a", Size: 1},
			{Name: "dir/b", Size: 2},
		},
	}
	expectedInserts := []*gcs.MinObject{
		{Name: "dir/a", Size: 1},
		{Name: "dir/b", Size: 2},
	}
	expectedImplicitDirs := []string{"dir/"}

	s.callAndVerify(context.TODO(), false, listing, "dir/", expectedInserts, expectedImplicitDirs)
}

func (s *FastStatBucketSuite) Test_ListObjects_InsertListing_CollapsedRunsOnly() {
	listing := &gcs.Listing{
		CollapsedRuns: []string{"dir/a/", "dir/b/"},
	}
	expectedImplicitDirs := []string{"dir/", "dir/a/", "dir/b/"}
	expectedInserts := []*gcs.MinObject{}

	s.callAndVerify(context.TODO(), false, listing, "dir/", expectedInserts, expectedImplicitDirs)
}

func (s *FastStatBucketSuite) Test_ListObjects_InsertListing_ObjectsAndCollapsedRuns() {
	listing := &gcs.Listing{
		MinObjects: []*gcs.MinObject{
			{Name: "dir/a", Size: 1},
		},
		CollapsedRuns: []string{"dir/b/"},
	}
	expectedInserts := []*gcs.MinObject{
		{Name: "dir/a", Size: 1},
	}
	expectedImplicitDirs := []string{"dir/", "dir/b/"}

	s.callAndVerify(context.TODO(), false, listing, "dir/", expectedInserts, expectedImplicitDirs)
}

func (s *FastStatBucketSuite) Test_ListObjects_InsertListing_ImplicitDir() {
	listing := &gcs.Listing{
		MinObjects: []*gcs.MinObject{
			{Name: "dir/a", Size: 1},
		},
	}
	expectedInserts := []*gcs.MinObject{
		{Name: "dir/a", Size: 1},
	}
	expectedImplicitDirs := []string{"dir/"}

	s.callAndVerify(context.TODO(), false, listing, "dir/", expectedInserts, expectedImplicitDirs)
}

func (s *FastStatBucketSuite) Test_ListObjects_InsertListing_ObjectSameAsCollapsedRun() {
	listing := &gcs.Listing{
		MinObjects: []*gcs.MinObject{
			{Name: "dir/a/", Size: 0},
		},
		CollapsedRuns: []string{"dir/a/"},
	}
	expectedInserts := []*gcs.MinObject{
		{Name: "dir/a/", Size: 0},
	}
	expectedImplicitDirs := []string{"dir/", "dir/a/"}

	s.callAndVerify(context.TODO(), false, listing, "dir/", expectedInserts, expectedImplicitDirs)
}

func (s *FastStatBucketSuite) cancelledContextDoesNotUpdatesCache(isHNS bool) {
	// Helper function to test for context cancelled scenarios.
	listing := &gcs.Listing{
		CollapsedRuns: []string{"dir/a/", "dir/b/"},
		MinObjects: []*gcs.MinObject{
			{Name: "dir/file.txt", Size: 123},
		},
	}
	// Create a cancellable context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.
	expectedInserts := []*gcs.MinObject{}
	expectedImplicitDirs := []string{}

	s.callAndVerify(ctx, isHNS, listing, "dir/", expectedInserts, expectedImplicitDirs)
}

func (s *FastStatBucketSuite) Test_ListObjects_InsertListing_ContextCancelledDoesNotUpdatesCache_HNSBucket() {
	s.cancelledContextDoesNotUpdatesCache(true)
}

func (s *FastStatBucketSuite) Test_ListObjects_InsertListing_ContextCancelledDoesNotUpdatesCache_FlatBucket() {
	s.cancelledContextDoesNotUpdatesCache(false)
}

func (s *FastStatBucketSuite) Test_ListObjects_InsertListing_ImplicitDirFalse_CollapsedRunsNotCached() {
	// Re-initialize bucket with implicitDir = false
	s.bucket = caching.NewFastStatBucket(
		primaryCacheTTL,
		s.cache,
		&s.clock,
		s.wrapped,
		negativeCacheTTL,
		true,
		false,
		isEnableEmptyManagedFolders)
	listing := &gcs.Listing{
		MinObjects: []*gcs.MinObject{
			{Name: "dir/a", Size: 1},
		},
		CollapsedRuns: []string{"dir/b/"},
	}
	expectedInserts := []*gcs.MinObject{
		{Name: "dir/a", Size: 1},
	}
	expectedImplicitDirs := []string{}

	s.callAndVerify(context.TODO(), false, listing, "dir/", expectedInserts, expectedImplicitDirs)
}

// //////////////////////////////////////////////////////////////////////
// UpdateObject
// //////////////////////////////////////////////////////////////////////

func (s *FastStatBucketSuite) Test_UpdateObject_CallsEraseAndWrapped() {
	const name = "taco"

	// Erase
	s.cache.On("Erase", name).Return().Once()

	// Wrapped
	var wrappedReq *gcs.UpdateObjectRequest
	s.wrapped.On("UpdateObject", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			wrappedReq = args.Get(1).(*gcs.UpdateObjectRequest)
		}).
		Return(nil, errors.New("")).
		Once()

	// Call
	req := &gcs.UpdateObjectRequest{
		Name: name,
	}

	_, _ = s.bucket.UpdateObject(context.TODO(), req)

	s.NotNil(wrappedReq)
	s.Equal(req, wrappedReq)
}

func (s *FastStatBucketSuite) Test_UpdateObject_WrappedFails() {
	// Erase
	s.cache.On("Erase", mock.Anything).Return().Once()

	// Wrapped
	s.wrapped.On("UpdateObject", mock.Anything, mock.Anything).
		Return(nil, errors.New("taco")).
		Once()

	// Call
	_, err := s.bucket.UpdateObject(context.TODO(), &gcs.UpdateObjectRequest{})

	s.ErrorContains(err, "taco")
}

func (s *FastStatBucketSuite) Test_UpdateObject_WrappedSucceeds() {
	const name = "taco"

	// Erase
	s.cache.On("Erase", mock.Anything).Return().Once()

	// Wrapped
	obj := &gcs.Object{
		Name:       name,
		Generation: 1234,
	}

	s.wrapped.On("UpdateObject", mock.Anything, mock.Anything).
		Return(obj, nil).
		Once()

	// Insert
	s.cache.On("Insert", mock.Anything, s.clock.Now().Add(primaryCacheTTL)).Return().Once()

	// Call
	o, err := s.bucket.UpdateObject(context.TODO(), &gcs.UpdateObjectRequest{})

	s.NoError(err)
	s.Equal(obj, o)
}

// //////////////////////////////////////////////////////////////////////
// DeleteObject
// //////////////////////////////////////////////////////////////////////

func (s *FastStatBucketSuite) deleteObject(name string) (err error) {
	err = s.bucket.DeleteObject(context.TODO(), &gcs.DeleteObjectRequest{Name: name})
	return
}

func (s *FastStatBucketSuite) Test_DeleteObject_CallsWrapped() {
	const name = "taco"

	// Wrapped
	var wrappedReq *gcs.DeleteObjectRequest
	s.wrapped.On("DeleteObject", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			wrappedReq = args.Get(1).(*gcs.DeleteObjectRequest)
		}).
		Return(errors.New("")).
		Once()

	// Call
	_ = s.deleteObject(name)

	s.NotNil(wrappedReq)
	s.Equal(name, wrappedReq.Name)
}

func (s *FastStatBucketSuite) Test_DeleteObject_WrappedFails_GenericError() {
	const name = ""

	// Wrapped
	s.wrapped.On("DeleteObject", mock.Anything, mock.Anything).
		Return(errors.New("taco")).
		Once()

	// Call
	err := s.deleteObject(name)

	s.ErrorContains(err, "taco")
}

func (s *FastStatBucketSuite) Test_DeleteObject_WrappedReturnsPreconditionError() {
	const name = "taco"
	// Erase
	s.cache.On("Erase", name).Return().Once()
	// Wrapped
	s.wrapped.On("DeleteObject", mock.Anything, mock.Anything).
		Return(&gcs.PreconditionError{Err: errors.New("precondition failed")}).
		Once()

	// Call.
	err := s.deleteObject(name)

	s.ErrorContains(err, "precondition failed")
}

func (s *FastStatBucketSuite) Test_DeleteObject_WrappedReturnsNotFoundError() {
	const name = "taco"
	// Erase
	s.cache.On("Erase", name).Return().Once()
	// Wrapped
	s.wrapped.On("DeleteObject", mock.Anything, mock.Anything).
		Return(&gcs.NotFoundError{Err: errors.New("object not found")}).
		Once()

	// Call.
	err := s.deleteObject(name)

	s.ErrorContains(err, "object not found")
}

func (s *FastStatBucketSuite) Test_DeleteObject_WrappedSucceeds_AddsNegativeEntry() {
	const name = ""

	// AddNegativeEntry
	s.cache.On("AddNegativeEntry", mock.Anything, mock.Anything).Return().Once()

	// Wrapped
	s.wrapped.On("DeleteObject", mock.Anything, mock.Anything).
		Return(nil).
		Once()

	// Call
	err := s.deleteObject(name)
	s.NoError(err)
}

func (s *FastStatBucketSuite) Test_DeleteObject_OnlyDeleteFromCache() {
	const name = "taco"
	req := &gcs.DeleteObjectRequest{
		Name:                name,
		OnlyDeleteFromCache: true,
	}
	// Expect AddNegativeEntry call.
	s.cache.On("AddNegativeEntry", name, s.clock.Now().Add(negativeCacheTTL)).
		Return().
		Once()

	err := s.bucket.DeleteObject(context.TODO(), req)

	s.NoError(err)
}

// //////////////////////////////////////////////////////////////////////
// DeleteFolder
// //////////////////////////////////////////////////////////////////////

func (s *FastStatBucketSuite) Test_DeleteFolder_Success() {
	const name = "some-name"
	s.cache.On("AddNegativeEntryForFolder", name, mock.Anything).
		Return().
		Once()
	s.wrapped.On("DeleteFolder", mock.Anything, name).
		Return(nil).
		Once()

	err := s.bucket.DeleteFolder(context.TODO(), name)

	s.NoError(err)
}

func (s *FastStatBucketSuite) Test_DeleteFolder_Failure() {
	const name = "some-name"
	// Erase
	s.cache.On("Erase", mock.Anything).Return().Once()
	s.wrapped.On("DeleteFolder", mock.Anything, name).
		Return(fmt.Errorf("mock error")).
		Once()

	err := s.bucket.DeleteFolder(context.TODO(), name)

	s.NotNil(err)
}

// //////////////////////////////////////////////////////////////////////
// CreateFolder
// //////////////////////////////////////////////////////////////////////

func (s *FastStatBucketSuite) Test_CreateFolder_WhenGCSCallSucceeds() {
	const name = "some-name"
	folder := &gcs.Folder{
		Name: name,
	}
	s.cache.On("Erase", name).
		Return().
		Once()
	s.cache.On("InsertFolder", folder, mock.Anything).
		Return().
		Once()
	s.wrapped.On("CreateFolder", mock.Anything, name).
		Return(folder, nil).
		Once()

	result, err := s.bucket.CreateFolder(context.TODO(), name)

	s.NoError(err)
	s.Equal(folder, result)
}

func (s *FastStatBucketSuite) Test_CreateFolder_WhenGCSCallFails() {
	const name = "some-name"
	s.cache.On("Erase", name).
		Return().
		Once()
	s.wrapped.On("CreateFolder", mock.Anything, name).
		Return(nil, fmt.Errorf("mock error")).
		Once()

	result, err := s.bucket.CreateFolder(context.TODO(), name)

	s.NotNil(err)
	s.Nil(result)
}

// //////////////////////////////////////////////////////////////////////
// MoveObject
// //////////////////////////////////////////////////////////////////////

func (s *FastStatBucketSuite) Test_MoveObject_MoveObjectFails() {
	const srcName = "taco"
	const dstName = "burrito"

	// Erase
	s.cache.On("Erase", dstName).Return().Once()
	s.cache.On("Erase", srcName).Return().Once()

	// Wrapped
	s.wrapped.On("MoveObject", mock.Anything, mock.Anything).
		Return(nil, errors.New("taco")).
		Once()

	// Call
	_, err := s.bucket.MoveObject(context.TODO(), &gcs.MoveObjectRequest{SrcName: srcName, DstName: dstName})

	s.ErrorContains(err, "taco")
}

func (s *FastStatBucketSuite) Test_MoveObject_MoveObjectSucceeds() {
	const dstName = "burrito"
	// Erase
	s.cache.On("Erase", mock.Anything).Return().Times(2)

	// Wrap object
	obj := &gcs.Object{
		Name:       dstName,
		Generation: 1234,
	}
	s.wrapped.On("MoveObject", mock.Anything, mock.Anything).
		Return(obj, nil).
		Once()

	// Insert in cache
	s.cache.On("Insert", mock.Anything, s.clock.Now().Add(primaryCacheTTL)).Return().Once()

	// Call
	o, err := s.bucket.MoveObject(context.TODO(), &gcs.MoveObjectRequest{})

	s.NoError(err)
	s.Equal(obj, o)
}

// //////////////////////////////////////////////////////////////////////
// NewReaderWithReadHandleTest
// //////////////////////////////////////////////////////////////////////

func (s *FastStatBucketSuite) Test_NewReaderWithReadHandle_CallsWrappedAndInvalidatesOnNotFound() {
	const name = "some-name"
	// Expect: wrapped bucket returns NotFoundError
	var wrappedReq *gcs.ReadObjectRequest
	s.wrapped.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			wrappedReq = args.Get(1).(*gcs.ReadObjectRequest)
		}).
		Return(nil, &gcs.NotFoundError{Err: errors.New("not found")}).
		Once()
	// Expect: cache invalidate is called
	s.cache.On("Erase", name).Return().Once()

	// Call
	req := &gcs.ReadObjectRequest{Name: name}
	rd, err := s.bucket.NewReaderWithReadHandle(context.TODO(), req)

	s.Nil(rd)
	s.IsType(&gcs.NotFoundError{}, err)
	s.Equal(name, wrappedReq.Name)
}

func (s *FastStatBucketSuite) Test_NewReaderWithReadHandle_CallsWrappedAndDoesNotInvalidateOnSuccess() {
	const name = "some-name"
	expectedReader := &fake.FakeReader{ReadCloser: io.NopCloser(strings.NewReader("abc")), Handle: []byte("fake")}
	// Expect: wrapped returns reader, no error
	s.wrapped.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).
		Return(expectedReader, nil).
		Once()

	// Call
	req := &gcs.ReadObjectRequest{Name: name}
	rd, err := s.bucket.NewReaderWithReadHandle(context.TODO(), req)

	s.NoError(err)
	s.Equal(expectedReader, rd)
}

// //////////////////////////////////////////////////////////////////////
// NewMultiRangeDownloader
// //////////////////////////////////////////////////////////////////////

func (s *FastStatBucketSuite) Test_NewMultiRangeDownloader_CallsWrappedAndInvalidatesOnNotFound() {
	const name = "some-name"
	// Expect: wrapped bucket returns NotFoundError
	var wrappedReq *gcs.MultiRangeDownloaderRequest
	s.wrapped.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			wrappedReq = args.Get(1).(*gcs.MultiRangeDownloaderRequest)
		}).
		Return(nil, &gcs.NotFoundError{Err: errors.New("not found")}).
		Once()
	// Expect: cache invalidate is called
	s.cache.On("Erase", name).Return().Once()

	// Call
	req := &gcs.MultiRangeDownloaderRequest{Name: name}
	mrd, err := s.bucket.NewMultiRangeDownloader(context.TODO(), req)

	s.Nil(mrd)
	s.IsType(&gcs.NotFoundError{}, err)
	s.Equal(name, wrappedReq.Name)
}

func (s *FastStatBucketSuite) Test_NewMultiRangeDownloader_CallsWrappedAndDoesNotInvalidateOnSuccess() {
	const name = "some-name"
	expectedMrd := fake.NewFakeMultiRangeDownloader(&gcs.MinObject{Name: name}, nil)
	// Expect: wrapped returns mrd, no error
	s.wrapped.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).
		Return(expectedMrd, nil).
		Once()

	// Call
	req := &gcs.MultiRangeDownloaderRequest{Name: name}
	mrd, err := s.bucket.NewMultiRangeDownloader(context.TODO(), req)

	s.NoError(err)
	s.Equal(expectedMrd, mrd)
}

// //////////////////////////////////////////////////////////////////////
// GetFolder
// //////////////////////////////////////////////////////////////////////

func (s *FastStatBucketSuite) Test_GetFolder_FetchOnlyFromCacheFalse() {
	const name = "taco/"
	req := &gcs.GetFolderRequest{
		Name:               name,
		FetchOnlyFromCache: false,
	}
	folder := &gcs.Folder{Name: name}
	s.cache.On("LookUpFolder", name, mock.Anything).
		Return(false, (*gcs.Folder)(nil)).
		Once()
	s.wrapped.On("GetFolder", mock.Anything, mock.Anything).
		Return(folder, nil).
		Once()
	s.cache.On("InsertFolder", mock.Anything, mock.Anything).
		Return().
		Once()

	f, err := s.bucket.GetFolder(context.TODO(), req)

	s.NoError(err)
	s.Equal(folder, f)
}

func (s *FastStatBucketSuite) Test_GetFolder_FetchOnlyFromCacheTrue_CacheHitPositive() {
	const name = "taco/"
	req := &gcs.GetFolderRequest{
		Name:               name,
		FetchOnlyFromCache: true,
	}
	folder := &gcs.Folder{Name: name}
	s.cache.On("LookUpFolder", name, mock.Anything).
		Return(true, folder).
		Once()

	f, err := s.bucket.GetFolder(context.TODO(), req)

	s.NoError(err)
	s.Equal(folder, f)
}

func (s *FastStatBucketSuite) Test_GetFolder_FetchOnlyFromCacheTrue_CacheHitNegative() {
	const name = "taco/"
	req := &gcs.GetFolderRequest{
		Name:               name,
		FetchOnlyFromCache: true,
	}
	s.cache.On("LookUpFolder", name, mock.Anything).
		Return(true, (*gcs.Folder)(nil)).
		Once()

	_, err := s.bucket.GetFolder(context.TODO(), req)

	s.IsType(&gcs.NotFoundError{}, err)
}

func (s *FastStatBucketSuite) Test_GetFolder_FetchOnlyFromCacheTrue_CacheMiss() {
	const name = "taco/"
	req := &gcs.GetFolderRequest{
		Name:               name,
		FetchOnlyFromCache: true,
	}
	s.cache.On("LookUpFolder", name, mock.Anything).
		Return(false, (*gcs.Folder)(nil)).
		Once()

	_, err := s.bucket.GetFolder(context.TODO(), req)

	s.IsType(&caching.CacheMissError{}, err)
}
