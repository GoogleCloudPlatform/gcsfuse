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
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	gostorage "cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/caching"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/caching/mock_gcscaching"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
)

func TestFastStatBucket(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const primaryCacheTTL = time.Second
const negativeCacheTTL = time.Second * 5
const isTypeCacheDeprecated = false

type fastStatBucketTest struct {
	cache   mock_gcscaching.MockStatCache
	clock   timeutil.SimulatedClock
	wrapped storage.MockBucket

	bucket gcs.Bucket
}

func (t *fastStatBucketTest) SetUp(ti *TestInfo) {
	// Set up a fixed, non-zero time.
	t.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))

	// Set up dependencies.
	t.cache = mock_gcscaching.NewMockStatCache(ti.MockController, "cache")
	t.wrapped = storage.NewMockBucket(ti.MockController, "wrapped")

	t.bucket = caching.NewFastStatBucket(
		primaryCacheTTL,
		t.cache,
		&t.clock,
		t.wrapped,
		negativeCacheTTL,
		isTypeCacheDeprecated)
}

////////////////////////////////////////////////////////////////////////
// CreateObject
////////////////////////////////////////////////////////////////////////

type CreateObjectTest struct {
	fastStatBucketTest
}

func init() { RegisterTestSuite(&CreateObjectTest{}) }

func (t *CreateObjectTest) CallsEraseAndWrapped() {
	const name = "taco"

	// Erase
	ExpectCall(t.cache, "Erase")(name)

	// Wrapped
	var wrappedReq *gcs.CreateObjectRequest
	ExpectCall(t.wrapped, "CreateObject")(Any(), Any()).
		WillOnce(DoAll(SaveArg(1, &wrappedReq), Return(nil, errors.New(""))))

	// Call
	req := &gcs.CreateObjectRequest{
		Name: name,
	}

	_, _ = t.bucket.CreateObject(context.TODO(), req)

	AssertNe(nil, wrappedReq)
	ExpectEq(req, wrappedReq)
}

func (t *CreateObjectTest) WrappedFails() {
	var err error

	// Erase
	ExpectCall(t.cache, "Erase")(Any())

	// Wrapped
	ExpectCall(t.wrapped, "CreateObject")(Any(), Any()).
		WillOnce(Return(nil, errors.New("taco")))

	// Call
	_, err = t.bucket.CreateObject(context.TODO(), &gcs.CreateObjectRequest{})

	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *CreateObjectTest) WrappedSucceeds() {
	const name = "taco"
	var err error

	// Erase
	ExpectCall(t.cache, "Erase")(Any())

	// Wrapped
	obj := &gcs.Object{
		Name:       name,
		Generation: 1234,
	}

	ExpectCall(t.wrapped, "CreateObject")(Any(), Any()).
		WillOnce(Return(obj, nil))

	// Insert
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(primaryCacheTTL)))

	// Call
	o, err := t.bucket.CreateObject(context.TODO(), &gcs.CreateObjectRequest{})

	AssertEq(nil, err)
	ExpectEq(obj, o)
}

////////////////////////////////////////////////////////////////////////
// CreateObject
////////////////////////////////////////////////////////////////////////

type CreateObjectChunkWriterTest struct {
	fastStatBucketTest
}

func init() { RegisterTestSuite(&CreateObjectChunkWriterTest{}) }

func (t *CreateObjectChunkWriterTest) CallsWrappedWithExpectedParameters() {
	const name = "taco"
	// Wrapped
	var wrappedReq *gcs.CreateObjectRequest
	var wrappedChunkSize int
	var wrappedCallback func(_ int64)
	ExpectCall(t.wrapped, "CreateObjectChunkWriter")(Any(), Any(), Any(), Any()).
		WillOnce(DoAll(SaveArg(1, &wrappedReq), SaveArg(2, &wrappedChunkSize), SaveArg(3, &wrappedCallback), Return(nil, errors.New(""))))
	// Call
	req := &gcs.CreateObjectRequest{
		Name: name,
	}
	chunkSize := 1024
	callback := func(_ int64) {
		fmt.Println("callback called!")
	}

	_, _ = t.bucket.CreateObjectChunkWriter(context.TODO(), req, chunkSize, callback)

	AssertNe(nil, wrappedReq)
	ExpectEq(req, wrappedReq)
	ExpectEq(chunkSize, wrappedChunkSize)
	ExpectEq(callback, wrappedCallback)
}

func (t *CreateObjectChunkWriterTest) WrappedFails() {
	chunkSize := 1024
	progressFunc := func(_ int64) {}
	ctx := context.TODO()
	req := &gcs.CreateObjectRequest{}
	var err error
	// Wrapped
	ExpectCall(t.wrapped, "CreateObjectChunkWriter")(Any(), Any(), Any(), Any()).
		WillOnce(Return(nil, errors.New("taco")))

	// Call
	_, err = t.bucket.CreateObjectChunkWriter(ctx, req, chunkSize, progressFunc)

	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *CreateObjectChunkWriterTest) WrappedSucceeds() {
	chunkSize := 1024
	progressFunc := func(_ int64) {}
	ctx := context.TODO()
	req := &gcs.CreateObjectRequest{}
	var err error
	// Wrapped
	wr := &storage.ObjectWriter{
		Writer: &gostorage.Writer{ChunkSize: chunkSize, ProgressFunc: progressFunc},
	}
	ExpectCall(t.wrapped, "CreateObjectChunkWriter")(Any(), Any(), Any(), Any()).
		WillOnce(Return(wr, nil))

	// Call
	gotWr, err := t.bucket.CreateObjectChunkWriter(ctx, req, chunkSize, progressFunc)

	AssertEq(nil, err)
	ExpectEq(wr, gotWr)
}

////////////////////////////////////////////////////////////////////////
// CreateAppendableObjectWriterTest
////////////////////////////////////////////////////////////////////////

type CreateAppendableObjectWriterTest struct {
	fastStatBucketTest
}

func init() { RegisterTestSuite(&CreateAppendableObjectWriterTest{}) }

func (t *CreateAppendableObjectWriterTest) CallsWrappedWithExpectedParameters() {
	const name = "taco"
	const offset int64 = 10
	const chunkSize = 1024
	ctx := context.TODO()
	// Wrapped
	var wrappedReq *gcs.CreateObjectChunkWriterRequest
	ExpectCall(t.wrapped, "CreateAppendableObjectWriter")(Any(), Any()).
		WillOnce(DoAll(SaveArg(1, &wrappedReq), Return(nil, errors.New(""))))
	// Call
	req := &gcs.CreateObjectChunkWriterRequest{
		CreateObjectRequest: gcs.CreateObjectRequest{
			Name: name,
		},
		Offset:    offset,
		ChunkSize: chunkSize,
	}

	_, _ = t.bucket.CreateAppendableObjectWriter(ctx, req)

	AssertNe(nil, wrappedReq)
	ExpectEq(req, wrappedReq)
}

func (t *CreateAppendableObjectWriterTest) WrappedFails() {
	ctx := context.TODO()
	req := &gcs.CreateObjectChunkWriterRequest{}
	var err error
	// Wrapped
	ExpectCall(t.wrapped, "CreateAppendableObjectWriter")(Any(), Any()).
		WillOnce(Return(nil, errors.New("taco")))

	// Call
	_, err = t.bucket.CreateAppendableObjectWriter(ctx, req)

	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *CreateAppendableObjectWriterTest) WrappedSucceeds() {
	ctx := context.TODO()
	req := &gcs.CreateObjectChunkWriterRequest{}
	var err error
	// Wrapped
	wr := &storage.ObjectWriter{
		Writer: &gostorage.Writer{},
	}
	ExpectCall(t.wrapped, "CreateAppendableObjectWriter")(Any(), Any()).
		WillOnce(Return(wr, nil))

	// Call
	gotWr, err := t.bucket.CreateAppendableObjectWriter(ctx, req)

	AssertEq(nil, err)
	ExpectEq(wr, gotWr)
}

////////////////////////////////////////////////////////////////////////
// FinalizeUpload
////////////////////////////////////////////////////////////////////////

type FinalizeUploadTest struct {
	fastStatBucketTest
}

func init() { RegisterTestSuite(&FinalizeUploadTest{}) }

func (t *FinalizeUploadTest) CallsEraseAndWrappedWithExpectedParameter() {
	const name = "taco"
	writer := &storage.ObjectWriter{
		Writer: &gostorage.Writer{ObjectAttrs: gostorage.ObjectAttrs{Name: name}},
	}
	// Erase
	ExpectCall(t.cache, "Erase")(name)
	// Wrapped
	var wrappedWriter gcs.Writer
	ExpectCall(t.wrapped, "FinalizeUpload")(Any(), Any()).
		WillOnce(DoAll(SaveArg(1, &wrappedWriter), Return(&gcs.MinObject{}, errors.New(""))))

	// Call
	_, _ = t.bucket.FinalizeUpload(context.TODO(), writer)

	AssertNe(nil, wrappedWriter)
	ExpectEq(writer, wrappedWriter)
}

func (t *FinalizeUploadTest) WrappedFails() {
	var err error
	writer := &storage.ObjectWriter{
		Writer: &gostorage.Writer{ObjectAttrs: gostorage.ObjectAttrs{Name: "name"}},
	}
	// Erase
	ExpectCall(t.cache, "Erase")(Any())
	// Wrapped
	ExpectCall(t.wrapped, "FinalizeUpload")(Any(), Any()).
		WillOnce(Return(&gcs.MinObject{}, errors.New("taco")))

	// Call
	o, err := t.bucket.FinalizeUpload(context.TODO(), writer)

	ExpectThat(err, Error(HasSubstr("taco")))
	ExpectNe(nil, o)
}

func (t *FinalizeUploadTest) WrappedSucceeds() {
	const name = "taco"
	writer := &storage.ObjectWriter{
		Writer: &gostorage.Writer{ObjectAttrs: gostorage.ObjectAttrs{Name: name}},
	}
	var err error
	// Erase
	ExpectCall(t.cache, "Erase")(Any())
	// Wrapped
	ExpectCall(t.wrapped, "FinalizeUpload")(Any(), Any()).
		WillOnce(Return(&gcs.MinObject{}, nil))
	// Insert
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(primaryCacheTTL)))

	// Call
	o, err := t.bucket.FinalizeUpload(context.TODO(), writer)

	AssertEq(nil, err)
	ExpectNe(nil, o)
}

////////////////////////////////////////////////////////////////////////
// FinalizeUpload
////////////////////////////////////////////////////////////////////////

type FlushPendingWritesTest struct {
	fastStatBucketTest
}

func init() { RegisterTestSuite(&FlushPendingWritesTest{}) }

func (t *FlushPendingWritesTest) WrappedFails() {
	const name = "taco"
	writer := &storage.ObjectWriter{
		Writer: &gostorage.Writer{ObjectAttrs: gostorage.ObjectAttrs{Name: name}},
	}
	// Expect cache Erase.
	ExpectCall(t.cache, "Erase")(name)
	// Expect call to Wrapped method.
	var wrappedWriter gcs.Writer
	mockObject := &gcs.MinObject{Size: 10}
	ExpectCall(t.wrapped, "FlushPendingWrites")(Any(), Any()).
		WillOnce(DoAll(SaveArg(1, &wrappedWriter), Return(mockObject, errors.New("taco"))))

	// Call.
	gotObject, err := t.bucket.FlushPendingWrites(context.TODO(), writer)

	ExpectEq(writer, wrappedWriter)
	ExpectEq(mockObject, gotObject)
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *FlushPendingWritesTest) WrappedSucceeds() {
	const name = "taco"
	writer := &storage.ObjectWriter{
		Writer: &gostorage.Writer{ObjectAttrs: gostorage.ObjectAttrs{Name: name}},
	}
	var err error
	// Expect cache Erase.
	ExpectCall(t.cache, "Erase")(name)
	// Wrapped.
	mockObject := &gcs.MinObject{Size: 10}
	ExpectCall(t.wrapped, "FlushPendingWrites")(Any(), Any()).
		WillOnce(Return(mockObject, nil))
	// Insert.
	var cachedMinObject *gcs.MinObject
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(primaryCacheTTL))).
		WillOnce(DoAll(SaveArg(0, &cachedMinObject)))

	// Call
	gotObject, err := t.bucket.FlushPendingWrites(context.TODO(), writer)

	AssertEq(nil, err)
	ExpectEq(mockObject, gotObject)
	ExpectEq(mockObject.Size, cachedMinObject.Size)
}

////////////////////////////////////////////////////////////////////////
// CopyObject
////////////////////////////////////////////////////////////////////////

type CopyObjectTest struct {
	fastStatBucketTest
}

func init() { RegisterTestSuite(&CopyObjectTest{}) }

func (t *CopyObjectTest) CallsEraseAndWrapped() {
	const srcName = "taco"
	const dstName = "burrito"

	// Erase
	ExpectCall(t.cache, "Erase")(dstName)

	// Wrapped
	var wrappedReq *gcs.CopyObjectRequest
	ExpectCall(t.wrapped, "CopyObject")(Any(), Any()).
		WillOnce(DoAll(SaveArg(1, &wrappedReq), Return(nil, errors.New(""))))

	// Call
	req := &gcs.CopyObjectRequest{
		SrcName: srcName,
		DstName: dstName,
	}

	_, _ = t.bucket.CopyObject(context.TODO(), req)

	AssertNe(nil, wrappedReq)
	ExpectEq(req, wrappedReq)
}

func (t *CopyObjectTest) WrappedFails() {
	var err error

	// Erase
	ExpectCall(t.cache, "Erase")(Any())

	// Wrapped
	ExpectCall(t.wrapped, "CopyObject")(Any(), Any()).
		WillOnce(Return(nil, errors.New("taco")))

	// Call
	_, err = t.bucket.CopyObject(context.TODO(), &gcs.CopyObjectRequest{})

	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *CopyObjectTest) WrappedSucceeds() {
	const dstName = "burrito"
	var err error

	// Erase
	ExpectCall(t.cache, "Erase")(Any())

	// Wrapped
	obj := &gcs.Object{
		Name:       dstName,
		Generation: 1234,
	}

	ExpectCall(t.wrapped, "CopyObject")(Any(), Any()).
		WillOnce(Return(obj, nil))

	// Insert
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(primaryCacheTTL)))

	// Call
	o, err := t.bucket.CopyObject(context.TODO(), &gcs.CopyObjectRequest{})

	AssertEq(nil, err)
	ExpectEq(obj, o)
}

////////////////////////////////////////////////////////////////////////
// ComposeObjects
////////////////////////////////////////////////////////////////////////

type ComposeObjectsTest struct {
	fastStatBucketTest
}

func init() { RegisterTestSuite(&ComposeObjectsTest{}) }

func (t *ComposeObjectsTest) CallsEraseAndWrapped() {
	const srcName = "taco"
	const dstName = "burrito"

	// Erase
	ExpectCall(t.cache, "Erase")(dstName)

	// Wrapped
	var wrappedReq *gcs.ComposeObjectsRequest
	ExpectCall(t.wrapped, "ComposeObjects")(Any(), Any()).
		WillOnce(DoAll(SaveArg(1, &wrappedReq), Return(nil, errors.New(""))))

	// Call
	req := &gcs.ComposeObjectsRequest{
		DstName: dstName,
		Sources: []gcs.ComposeSource{
			{Name: srcName},
		},
	}

	_, _ = t.bucket.ComposeObjects(context.TODO(), req)

	AssertNe(nil, wrappedReq)
	ExpectEq(req, wrappedReq)
}

func (t *ComposeObjectsTest) WrappedFails() {
	var err error

	// Erase
	ExpectCall(t.cache, "Erase")(Any())

	// Wrapped
	ExpectCall(t.wrapped, "ComposeObjects")(Any(), Any()).
		WillOnce(Return(nil, errors.New("taco")))

	// Call
	_, err = t.bucket.ComposeObjects(context.TODO(), &gcs.ComposeObjectsRequest{})

	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ComposeObjectsTest) WrappedSucceeds() {
	const dstName = "burrito"
	var err error

	// Erase
	ExpectCall(t.cache, "Erase")(Any())

	// Wrapped
	obj := &gcs.Object{
		Name:       dstName,
		Generation: 1234,
	}

	ExpectCall(t.wrapped, "ComposeObjects")(Any(), Any()).
		WillOnce(Return(obj, nil))

	// Insert
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(primaryCacheTTL)))

	// Call
	o, err := t.bucket.ComposeObjects(context.TODO(), &gcs.ComposeObjectsRequest{})

	AssertEq(nil, err)
	ExpectEq(obj, o)
}

////////////////////////////////////////////////////////////////////////
// StatObject
////////////////////////////////////////////////////////////////////////

type StatObjectTest struct {
	fastStatBucketTest
}

func init() { RegisterTestSuite(&StatObjectTest{}) }

func (t *StatObjectTest) CallsCache() {
	const name = "taco"

	// LookUp
	ExpectCall(t.cache, "LookUp")(name, timeutil.TimeEq(t.clock.Now())).
		WillOnce(Return(true, &gcs.MinObject{}))

	// Call
	req := &gcs.StatObjectRequest{
		Name: name,
	}

	_, _, _ = t.bucket.StatObject(context.TODO(), req)
}

func (t *StatObjectTest) CacheHit_Positive() {
	const name = "taco"

	// LookUp
	minObj := &gcs.MinObject{
		Name: name,
	}

	ExpectCall(t.cache, "LookUp")(Any(), Any()).
		WillOnce(Return(true, minObj))

	// Call
	req := &gcs.StatObjectRequest{
		Name: name,
	}

	m, e, err := t.bucket.StatObject(context.TODO(), req)
	AssertEq(nil, err)
	AssertEq(nil, e)
	AssertNe(nil, m)
	ExpectThat(m, Pointee(DeepEquals(*minObj)))
}

func (t *StatObjectTest) CacheHit_Negative() {
	const name = "taco"

	// LookUp
	ExpectCall(t.cache, "LookUp")(Any(), Any()).
		WillOnce(Return(true, nil))

	// Call
	req := &gcs.StatObjectRequest{
		Name: name,
	}

	_, _, err := t.bucket.StatObject(context.TODO(), req)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *StatObjectTest) IgnoresCacheEntryWhenForceFetchFromGcsIsTrue() {
	const name = "taco"

	// Lookup
	ExpectCall(t.cache, "LookUp")(Any(), Any()).Times(0)

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

	ExpectCall(t.wrapped, "StatObject")(Any(), req).
		WillOnce(Return(minObjFromGcs, extObjAttrFromGcs, nil))

	// Insert
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(primaryCacheTTL)))

	m, e, err := t.bucket.StatObject(context.TODO(), req)
	AssertEq(nil, err)
	AssertNe(nil, m)
	AssertNe(nil, e)
	ExpectEq(minObjFromGcs, m)
	ExpectEq(extObjAttrFromGcs, e)
}

func (t *StatObjectTest) TestStatObject_ForceFetchFromGcsTrueAndReturnExtendedObjectAttributesFalse() {
	const name = "taco"

	// Lookup
	ExpectCall(t.cache, "LookUp")(Any(), Any()).Times(0)

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

	ExpectCall(t.wrapped, "StatObject")(Any(), req).
		WillOnce(Return(minObjFromGcs, &gcs.ExtendedObjectAttributes{}, nil))

	// Insert
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(primaryCacheTTL)))

	m, e, err := t.bucket.StatObject(context.TODO(), req)
	AssertEq(nil, err)
	AssertNe(nil, m)
	ExpectEq(minObjFromGcs, m)
	ExpectEq(nil, e)
}

func (t *StatObjectTest) TestStatObjectPanics_ForceFetchFromGcsFalseAndReturnExtendedObjectAttributesTrue() {
	const name = "taco"
	const panic = "invalid StatObjectRequest: ForceFetchFromGcs: false and ReturnExtendedObjectAttributes: true"

	// Request
	req := &gcs.StatObjectRequest{
		Name:                           name,
		ForceFetchFromGcs:              false,
		ReturnExtendedObjectAttributes: true,
	}

	defer func() {
		r := recover()
		AssertEq(panic, r)
	}()
	_, _, err := t.bucket.StatObject(context.TODO(), req)
	AssertEq(nil, err)
}

func (t *StatObjectTest) CallsWrapped() {
	const name = ""
	req := &gcs.StatObjectRequest{
		Name: name,
	}

	// LookUp
	ExpectCall(t.cache, "LookUp")(Any(), Any()).
		WillOnce(Return(false, nil))

	// Wrapped
	ExpectCall(t.wrapped, "StatObject")(Any(), req).
		WillOnce(Return(nil, nil, errors.New("")))

	// Call
	_, _, _ = t.bucket.StatObject(context.TODO(), req)
}

func (t *StatObjectTest) WrappedFails() {
	const name = ""

	// LookUp
	ExpectCall(t.cache, "LookUp")(Any(), Any()).
		WillOnce(Return(false, nil))

	// Wrapped
	ExpectCall(t.wrapped, "StatObject")(Any(), Any()).
		WillOnce(Return(nil, nil, errors.New("taco")))

	// Call
	req := &gcs.StatObjectRequest{
		Name: name,
	}

	_, _, err := t.bucket.StatObject(context.TODO(), req)
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *StatObjectTest) WrappedSaysNotFound() {
	const name = "taco"

	// LookUp
	ExpectCall(t.cache, "LookUp")(Any(), Any()).
		WillOnce(Return(false, nil))

	// Wrapped
	ExpectCall(t.wrapped, "StatObject")(Any(), Any()).
		WillOnce(Return(nil, nil, &gcs.NotFoundError{Err: errors.New("burrito")}))

	// AddNegativeEntry
	ExpectCall(t.cache, "AddNegativeEntry")(
		name,
		timeutil.TimeEq(t.clock.Now().Add(negativeCacheTTL)))

	// Call
	req := &gcs.StatObjectRequest{
		Name: name,
	}

	_, _, err := t.bucket.StatObject(context.TODO(), req)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
	ExpectThat(err, Error(HasSubstr("burrito")))
}

func (t *StatObjectTest) WrappedSucceeds() {
	const name = "taco"

	// LookUp
	ExpectCall(t.cache, "LookUp")(Any(), Any()).
		WillOnce(Return(false, nil))

	// Wrapped
	minObj := &gcs.MinObject{
		Name: name,
	}

	ExpectCall(t.wrapped, "StatObject")(Any(), Any()).
		WillOnce(Return(minObj, nil, nil))

	// Insert
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(primaryCacheTTL)))

	// Call
	req := &gcs.StatObjectRequest{
		Name: name,
	}

	m, _, err := t.bucket.StatObject(context.TODO(), req)
	AssertEq(nil, err)
	ExpectEq(minObj, m)
}

////////////////////////////////////////////////////////////////////////
// ListObjects
////////////////////////////////////////////////////////////////////////

type ListObjectsTest struct {
	fastStatBucketTest
}

func init() { RegisterTestSuite(&ListObjectsTest{}) }

func (t *ListObjectsTest) WrappedFails() {
	// Wrapped
	ExpectCall(t.wrapped, "ListObjects")(Any(), Any()).
		WillOnce(Return(nil, errors.New("taco")))

	// Call
	_, err := t.bucket.ListObjects(context.TODO(), &gcs.ListObjectsRequest{})
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ListObjectsTest) EmptyListing() {
	// Wrapped
	expected := &gcs.Listing{}

	ExpectCall(t.wrapped, "BucketType")().
		WillOnce(Return(gcs.BucketType{}))

	ExpectCall(t.wrapped, "ListObjects")(Any(), Any()).
		WillOnce(Return(expected, nil))

	// Call
	listing, err := t.bucket.ListObjects(context.TODO(), &gcs.ListObjectsRequest{})

	AssertEq(nil, err)
	ExpectEq(expected, listing)
}

func (t *ListObjectsTest) EmptyListingForHNS() {
	// wrapped
	expected := &gcs.Listing{}

	ExpectCall(t.wrapped, "BucketType")().
		WillOnce(Return(gcs.BucketType{Hierarchical: true}))

	ExpectCall(t.wrapped, "ListObjects")(Any(), Any()).
		WillOnce(Return(expected, nil))

	// call
	listing, err := t.bucket.ListObjects(context.TODO(), &gcs.ListObjectsRequest{})

	AssertEq(nil, err)
	ExpectEq(expected, listing)
}

func (t *ListObjectsTest) NonEmptyListing() {
	// Wrapped
	o0 := &gcs.MinObject{Name: "taco"}
	o1 := &gcs.MinObject{Name: "burrito"}

	expected := &gcs.Listing{
		MinObjects: []*gcs.MinObject{o0, o1},
	}

	ExpectCall(t.wrapped, "BucketType")().
		WillOnce(Return(gcs.BucketType{}))

	ExpectCall(t.wrapped, "ListObjects")(Any(), Any()).
		WillOnce(Return(expected, nil))

	// Insert
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(primaryCacheTTL))).Times(2)

	// Call
	listing, err := t.bucket.ListObjects(context.TODO(), &gcs.ListObjectsRequest{})

	AssertEq(nil, err)
	ExpectEq(expected, listing)
}

func (t *ListObjectsTest) NonEmptyListingForHNS() {
	// wrapped
	o0 := &gcs.MinObject{Name: "taco"}
	o1 := &gcs.MinObject{Name: "burrito"}

	expected := &gcs.Listing{
		MinObjects:    []*gcs.MinObject{o0, o1},
		CollapsedRuns: []string{"p0", "p1/"},
	}

	ExpectCall(t.wrapped, "BucketType")().
		WillOnce(Return(gcs.BucketType{Hierarchical: true}))

	ExpectCall(t.wrapped, "ListObjects")(Any(), Any()).
		WillOnce(Return(expected, nil))

	// insert
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(primaryCacheTTL))).Times(2)

	ExpectCall(t.cache, "InsertFolder")(Any(), timeutil.TimeEq(t.clock.Now().Add(primaryCacheTTL))).Times(1)

	// call
	listing, err := t.bucket.ListObjects(context.TODO(), &gcs.ListObjectsRequest{})

	AssertEq(nil, err)
	ExpectEq(expected, listing)
}

func (t *ListObjectsTest) NonEmptyListingWithCancelledContext() {
	// Wrapped
	o0 := &gcs.MinObject{Name: "taco"}
	o1 := &gcs.MinObject{Name: "burrito"}
	expected := &gcs.Listing{
		MinObjects: []*gcs.MinObject{o0, o1},
	}
	ExpectCall(t.wrapped, "BucketType")().
		WillOnce(Return(gcs.BucketType{}))
	// Create a cancellable context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.
	ExpectCall(t.wrapped, "ListObjects")(ctx, Any()).
		WillOnce(Return(expected, nil))
	// Insert not called.
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(primaryCacheTTL))).Times(0)

	// Call
	listing, err := t.bucket.ListObjects(ctx, &gcs.ListObjectsRequest{})

	AssertEq(nil, err)
	ExpectEq(expected, listing)
}

func (t *ListObjectsTest) NonEmptyListingWithCancelledContextForHNS() {
	// wrapped
	o0 := &gcs.MinObject{Name: "taco"}
	o1 := &gcs.MinObject{Name: "burrito"}
	expected := &gcs.Listing{
		MinObjects:    []*gcs.MinObject{o0, o1},
		CollapsedRuns: []string{"p0", "p1/"},
	}
	ExpectCall(t.wrapped, "BucketType")().
		WillOnce(Return(gcs.BucketType{Hierarchical: true}))
	// Create a cancellable context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.
	ExpectCall(t.wrapped, "ListObjects")(ctx, Any()).
		WillOnce(Return(expected, nil))
	// insert not called.
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(primaryCacheTTL))).Times(0)
	ExpectCall(t.cache, "InsertFolder")(Any(), timeutil.TimeEq(t.clock.Now().Add(primaryCacheTTL))).Times(0)

	// call
	listing, err := t.bucket.ListObjects(ctx, &gcs.ListObjectsRequest{})

	AssertEq(nil, err)
	ExpectEq(expected, listing)
}

////////////////////////////////////////////////////////////////////////
// ListObjectsTest_InsertListing
////////////////////////////////////////////////////////////////////////

type ListObjectsTest_InsertListing struct {
	fastStatBucketTest
}

func init() { RegisterTestSuite(&ListObjectsTest_InsertListing{}) }

func (t *ListObjectsTest_InsertListing) SetUp(ti *TestInfo) {
	t.fastStatBucketTest.SetUp(ti)
	t.bucket = caching.NewFastStatBucket(
		primaryCacheTTL,
		t.cache,
		&t.clock,
		t.wrapped,
		negativeCacheTTL,
		true)
}

func (t *ListObjectsTest_InsertListing) callAndVerify(ctx context.Context, isHNS bool, listing *gcs.Listing, prefix string, expectedInserts []*gcs.MinObject) {
	// Wrapped
	ExpectCall(t.wrapped, "BucketType")().
		WillOnce(Return(gcs.BucketType{Hierarchical: isHNS}))
	ExpectCall(t.wrapped, "ListObjects")(Any(), Any()).
		WillOnce(Return(listing, nil))
	// Register expectations.
	for _, obj := range expectedInserts {
		ExpectCall(t.cache, "Insert")(Pointee(DeepEquals(*obj)), Any())
	}

	// Call
	gotListing, err := t.bucket.ListObjects(ctx, &gcs.ListObjectsRequest{Prefix: prefix})

	AssertEq(nil, err)
	AssertEq(listing, gotListing)
}

func (t *ListObjectsTest_InsertListing) EmptyListing() {
	listing := &gcs.Listing{}
	expectedInserts := []*gcs.MinObject{}

	t.callAndVerify(context.TODO(), false, listing, "dir/", expectedInserts)
}

func (t *ListObjectsTest_InsertListing) ObjectsOnly() {
	listing := &gcs.Listing{
		MinObjects: []*gcs.MinObject{
			{Name: "dir/a", Size: 1},
			{Name: "dir/b", Size: 2},
		},
	}
	expectedInserts := []*gcs.MinObject{
		{Name: "dir/"},
		{Name: "dir/a", Size: 1},
		{Name: "dir/b", Size: 2},
	}

	t.callAndVerify(context.TODO(), false, listing, "dir/", expectedInserts)
}

func (t *ListObjectsTest_InsertListing) CollapsedRunsOnly() {
	listing := &gcs.Listing{
		CollapsedRuns: []string{"dir/a/", "dir/b/"},
	}
	expectedInserts := []*gcs.MinObject{
		{Name: "dir/"},
		{Name: "dir/a/"},
		{Name: "dir/b/"},
	}

	t.callAndVerify(context.TODO(), false, listing, "dir/", expectedInserts)
}

func (t *ListObjectsTest_InsertListing) ObjectsAndCollapsedRuns() {
	listing := &gcs.Listing{
		MinObjects: []*gcs.MinObject{
			{Name: "dir/a", Size: 1},
		},
		CollapsedRuns: []string{"dir/b/"},
	}
	expectedInserts := []*gcs.MinObject{
		{Name: "dir/"},
		{Name: "dir/a", Size: 1},
		{Name: "dir/b/"},
	}

	t.callAndVerify(context.TODO(), false, listing, "dir/", expectedInserts)
}

func (t *ListObjectsTest_InsertListing) ImplicitDir() {
	listing := &gcs.Listing{
		MinObjects: []*gcs.MinObject{
			{Name: "dir/a", Size: 1},
		},
	}
	expectedInserts := []*gcs.MinObject{
		{Name: "dir/"},
		{Name: "dir/a", Size: 1},
	}

	t.callAndVerify(context.TODO(), false, listing, "dir/", expectedInserts)
}

func (t *ListObjectsTest_InsertListing) ObjectSameAsCollapsedRun() {
	listing := &gcs.Listing{
		MinObjects: []*gcs.MinObject{
			{Name: "dir/a/", Size: 0},
		},
		CollapsedRuns: []string{"dir/a/"},
	}
	expectedInserts := []*gcs.MinObject{
		{Name: "dir/"},
		{Name: "dir/a/", Size: 0},
	}

	t.callAndVerify(context.TODO(), false, listing, "dir/", expectedInserts)
}

func (t *ListObjectsTest_InsertListing) cancelledContextDoesNotUpdatesCache(isHNS bool) {
	// Helper function to test for context cancelled scenarios.
	// 1. Setup
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

	t.callAndVerify(ctx, isHNS, listing, "dir/", expectedInserts)
}

func (t *ListObjectsTest_InsertListing) TestInsertListing_ContextCancelledDoesNotUpdatesCache_HNSBucket() {
	t.cancelledContextDoesNotUpdatesCache(true)
}

func (t *ListObjectsTest_InsertListing) TestInsertListing_ContextCancelledDoesNotUpdatesCache_FlatBucket() {
	t.cancelledContextDoesNotUpdatesCache(false)
}

////////////////////////////////////////////////////////////////////////
// UpdateObject
////////////////////////////////////////////////////////////////////////

type UpdateObjectTest struct {
	fastStatBucketTest
}

func init() { RegisterTestSuite(&UpdateObjectTest{}) }

func (t *UpdateObjectTest) CallsEraseAndWrapped() {
	const name = "taco"

	// Erase
	ExpectCall(t.cache, "Erase")(name)

	// Wrapped
	var wrappedReq *gcs.UpdateObjectRequest
	ExpectCall(t.wrapped, "UpdateObject")(Any(), Any()).
		WillOnce(DoAll(SaveArg(1, &wrappedReq), Return(nil, errors.New(""))))

	// Call
	req := &gcs.UpdateObjectRequest{
		Name: name,
	}

	_, _ = t.bucket.UpdateObject(context.TODO(), req)

	AssertNe(nil, wrappedReq)
	ExpectEq(req, wrappedReq)
}

func (t *UpdateObjectTest) WrappedFails() {
	var err error

	// Erase
	ExpectCall(t.cache, "Erase")(Any())

	// Wrapped
	ExpectCall(t.wrapped, "UpdateObject")(Any(), Any()).
		WillOnce(Return(nil, errors.New("taco")))

	// Call
	_, err = t.bucket.UpdateObject(context.TODO(), &gcs.UpdateObjectRequest{})

	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *UpdateObjectTest) WrappedSucceeds() {
	const name = "taco"
	var err error

	// Erase
	ExpectCall(t.cache, "Erase")(Any())

	// Wrapped
	obj := &gcs.Object{
		Name:       name,
		Generation: 1234,
	}

	ExpectCall(t.wrapped, "UpdateObject")(Any(), Any()).
		WillOnce(Return(obj, nil))

	// Insert
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(primaryCacheTTL)))

	// Call
	o, err := t.bucket.UpdateObject(context.TODO(), &gcs.UpdateObjectRequest{})

	AssertEq(nil, err)
	ExpectEq(obj, o)
}

////////////////////////////////////////////////////////////////////////
// DeleteObject
////////////////////////////////////////////////////////////////////////

type DeleteObjectTest struct {
	fastStatBucketTest
}

func init() { RegisterTestSuite(&DeleteObjectTest{}) }

func (t *DeleteObjectTest) deleteObject(name string) (err error) {
	err = t.bucket.DeleteObject(context.TODO(), &gcs.DeleteObjectRequest{Name: name})
	return
}

func (t *DeleteObjectTest) CallsWrapped() {
	const name = "taco"

	// Wrapped
	var wrappedReq *gcs.DeleteObjectRequest
	ExpectCall(t.wrapped, "DeleteObject")(Any(), Any()).
		WillOnce(DoAll(SaveArg(1, &wrappedReq), Return(errors.New(""))))

	// Call
	_ = t.deleteObject(name)

	AssertNe(nil, wrappedReq)
	ExpectEq(name, wrappedReq.Name)
}

func (t *DeleteObjectTest) WrappedFails_GenericError() {
	const name = ""
	var err error

	// Wrapped
	ExpectCall(t.wrapped, "DeleteObject")(Any(), Any()).
		WillOnce(Return(errors.New("taco")))

	// Call
	err = t.deleteObject(name)

	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *DeleteObjectTest) WrappedReturnsPreconditionError() {
	const name = "taco"
	// Erase
	ExpectCall(t.cache, "Erase")(name)
	// Wrapped
	ExpectCall(t.wrapped, "DeleteObject")(Any(), Any()).
		WillOnce(Return(&gcs.PreconditionError{Err: errors.New("precondition failed")}))

	// Call.
	err := t.deleteObject(name)

	ExpectThat(err, Error(HasSubstr("precondition failed")))
}

func (t *DeleteObjectTest) WrappedReturnsNotFoundError() {
	const name = "taco"
	// Erase
	ExpectCall(t.cache, "Erase")(name)
	// Wrapped
	ExpectCall(t.wrapped, "DeleteObject")(Any(), Any()).
		WillOnce(Return(&gcs.NotFoundError{Err: errors.New("object not found")}))

	// Call.
	err := t.deleteObject(name)

	ExpectThat(err, Error(HasSubstr("object not found")))
}

func (t *DeleteObjectTest) WrappedSucceeds_AddsNegativeEntry() {
	const name = ""
	var err error

	// AddNegativeEntry
	ExpectCall(t.cache, "AddNegativeEntry")(Any(), Any())

	// Wrapped
	ExpectCall(t.wrapped, "DeleteObject")(Any(), Any()).
		WillOnce(Return(nil))

	// Call
	err = t.deleteObject(name)
	AssertEq(nil, err)
}

func (t *DeleteObjectTest) OnlyDeleteFromCache() {
	const name = "taco"
	req := &gcs.DeleteObjectRequest{
		Name:                name,
		OnlyDeleteFromCache: true,
	}
	// Expect AddNegativeEntry call.
	ExpectCall(t.cache, "AddNegativeEntry")(
		name,
		timeutil.TimeEq(t.clock.Now().Add(negativeCacheTTL)))

	err := t.bucket.DeleteObject(context.TODO(), req)

	AssertEq(nil, err)
}

func (t *StatObjectTest) TestShouldReturnFromCacheWhenEntryIsPresent() {
	const name = "some-name"
	folder := &gcs.Folder{
		Name: name,
	}
	ExpectCall(t.cache, "LookUpFolder")(name, Any()).
		WillOnce(Return(true, folder))

	result, err := t.bucket.GetFolder(context.TODO(), &gcs.GetFolderRequest{Name: name})

	AssertEq(nil, err)
	ExpectThat(result, Pointee(DeepEquals(*folder)))
}

func (t *StatObjectTest) TestShouldReturnNotFoundErrorWhenNilEntryIsReturned() {
	const name = "some-name"

	ExpectCall(t.cache, "LookUpFolder")(name, Any()).
		WillOnce(Return(true, nil))

	result, err := t.bucket.GetFolder(context.TODO(), &gcs.GetFolderRequest{Name: name})

	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
	AssertEq(nil, result)
}

func (t *StatObjectTest) TestShouldCallGetFolderWhenEntryIsNotPresent() {
	const name = "some-name"
	folder := &gcs.Folder{
		Name: name,
	}
	getFolderReq := &gcs.GetFolderRequest{Name: name}

	ExpectCall(t.cache, "LookUpFolder")(name, Any()).
		WillOnce(Return(false, nil))
	ExpectCall(t.cache, "InsertFolder")(folder, Any()).
		WillOnce(Return())
	ExpectCall(t.wrapped, "GetFolder")(Any(), getFolderReq).
		WillOnce(Return(folder, nil))

	result, err := t.bucket.GetFolder(context.TODO(), getFolderReq)

	AssertEq(nil, err)
	ExpectThat(result, Pointee(DeepEquals(*folder)))
}

func (t *StatObjectTest) TestShouldReturnNilWhenErrorIsReturnedFromGetFolder() {
	const name = "some-name"
	error := errors.New("connection error")
	getFolderReq := &gcs.GetFolderRequest{Name: name}

	ExpectCall(t.cache, "LookUpFolder")(name, Any()).
		WillOnce(Return(false, nil))
	ExpectCall(t.wrapped, "GetFolder")(Any(), getFolderReq).
		WillOnce(Return(nil, error))

	folder, result := t.bucket.GetFolder(context.TODO(), getFolderReq)

	AssertEq(nil, folder)
	AssertEq(error, result)
}

func (t *StatObjectTest) TestRenameFolder() {
	const name = "some-name"
	const newName = "new-name"
	var folder = &gcs.Folder{
		Name: newName,
	}

	ExpectCall(t.cache, "EraseEntriesWithGivenPrefix")(name).WillOnce(Return())
	ExpectCall(t.cache, "InsertFolder")(folder, Any()).WillOnce(Return())
	ExpectCall(t.wrapped, "RenameFolder")(Any(), name, newName).WillOnce(Return(folder, nil))

	result, err := t.bucket.RenameFolder(context.Background(), name, newName)

	AssertEq(nil, err)
	ExpectEq(result, folder)
}

func (t *StatObjectTest) FetchOnlyFromCacheFalse() {
	const name = "taco"
	req := &gcs.StatObjectRequest{
		Name:               name,
		FetchOnlyFromCache: false,
	}
	// We expect a call to GCS, so we mock the wrapped bucket.
	ExpectCall(t.cache, "LookUp")(name, Any()).
		WillOnce(Return(false, nil))

	minObj := &gcs.MinObject{Name: name}
	ExpectCall(t.wrapped, "StatObject")(Any(), Any()).
		WillOnce(Return(minObj, nil, nil))
	ExpectCall(t.cache, "Insert")(Any(), Any())

	m, _, err := t.bucket.StatObject(context.TODO(), req)

	AssertEq(nil, err)
	ExpectEq(minObj, m)
}

func (t *StatObjectTest) FetchOnlyFromCacheTrue_CacheHitPositive() {
	const name = "taco"
	req := &gcs.StatObjectRequest{
		Name:               name,
		FetchOnlyFromCache: true,
	}
	minObj := &gcs.MinObject{Name: name}
	ExpectCall(t.cache, "LookUp")(name, Any()).
		WillOnce(Return(true, minObj))

	m, _, err := t.bucket.StatObject(context.TODO(), req)

	AssertEq(nil, err)
	ExpectEq(minObj, m)
}

func (t *StatObjectTest) FetchOnlyFromCacheTrue_CacheHitNegative() {
	const name = "taco"
	req := &gcs.StatObjectRequest{
		Name:               name,
		FetchOnlyFromCache: true,
	}
	ExpectCall(t.cache, "LookUp")(name, Any()).
		WillOnce(Return(true, nil))

	_, _, err := t.bucket.StatObject(context.TODO(), req)

	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *StatObjectTest) FetchOnlyFromCacheTrue_CacheMiss() {
	const name = "taco"
	req := &gcs.StatObjectRequest{
		Name:               name,
		FetchOnlyFromCache: true,
	}
	ExpectCall(t.cache, "LookUp")(name, Any()).
		WillOnce(Return(false, nil))

	_, _, err := t.bucket.StatObject(context.TODO(), req)

	ExpectThat(err, HasSameTypeAs(&caching.CacheMissError{}))
}

type DeleteFolderTest struct {
	fastStatBucketTest
}

func init() { RegisterTestSuite(&DeleteFolderTest{}) }

func (t *DeleteFolderTest) Test_DeleteFolder_Success() {
	const name = "some-name"
	ExpectCall(t.cache, "AddNegativeEntryForFolder")(name, Any()).
		WillOnce(Return())
	ExpectCall(t.wrapped, "DeleteFolder")(Any(), name).
		WillOnce(Return(nil))

	err := t.bucket.DeleteFolder(context.TODO(), name)

	AssertEq(nil, err)
}

func (t *DeleteFolderTest) Test_DeleteFolder_Failure() {
	const name = "some-name"
	// Erase
	ExpectCall(t.cache, "Erase")(Any())
	ExpectCall(t.wrapped, "DeleteFolder")(Any(), name).
		WillOnce(Return(fmt.Errorf("mock error")))

	err := t.bucket.DeleteFolder(context.TODO(), name)

	AssertNe(nil, err)
}

type CreateFolderTest struct {
	fastStatBucketTest
}

func init() { RegisterTestSuite(&CreateFolderTest{}) }

func (t *CreateFolderTest) TestCreateFolderWhenGCSCallSucceeds() {
	const name = "some-name"
	folder := &gcs.Folder{
		Name: name,
	}
	ExpectCall(t.cache, "Erase")(name).
		WillOnce(Return())
	ExpectCall(t.cache, "InsertFolder")(folder, Any()).
		WillOnce(Return())
	ExpectCall(t.wrapped, "CreateFolder")(Any(), name).
		WillOnce(Return(folder, nil))

	result, err := t.bucket.CreateFolder(context.TODO(), name)

	AssertEq(nil, err)
	ExpectThat(result, Pointee(DeepEquals(*folder)))
}

func (t *CreateFolderTest) TestCreateFolderWhenGCSCallFails() {
	const name = "some-name"
	ExpectCall(t.cache, "Erase")(name).
		WillOnce(Return())
	ExpectCall(t.wrapped, "CreateFolder")(Any(), name).
		WillOnce(Return(nil, fmt.Errorf("mock error")))

	result, err := t.bucket.CreateFolder(context.TODO(), name)

	AssertNe(nil, err)
	AssertEq(nil, result)
}

type MoveObjectTest struct {
	fastStatBucketTest
}

func init() { RegisterTestSuite(&MoveObjectTest{}) }

func (t *MoveObjectTest) MoveObjectFails() {
	const srcName = "taco"
	const dstName = "burrito"

	// Erase
	ExpectCall(t.cache, "Erase")(dstName)
	ExpectCall(t.cache, "Erase")(srcName)

	// Wrapped
	ExpectCall(t.wrapped, "MoveObject")(Any(), Any()).WillOnce(Return(nil, errors.New("taco")))

	// Call
	_, err := t.bucket.MoveObject(context.TODO(), &gcs.MoveObjectRequest{SrcName: srcName, DstName: dstName})

	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *MoveObjectTest) MoveObjectSucceeds() {
	const dstName = "burrito"
	// Erase
	ExpectCall(t.cache, "Erase")(Any()).Times(2)

	// Wrap object
	obj := &gcs.Object{
		Name:       dstName,
		Generation: 1234,
	}
	ExpectCall(t.wrapped, "MoveObject")(Any(), Any()).WillOnce(Return(obj, nil))

	// Insert in cache
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(primaryCacheTTL)))

	// Call
	o, err := t.bucket.MoveObject(context.TODO(), &gcs.MoveObjectRequest{})

	AssertEq(nil, err)
	ExpectEq(obj, o)
}

////////////////////////////////////////////////////////////////////////
// NewReaderWithReadHandleTest
////////////////////////////////////////////////////////////////////////

type NewReaderWithReadHandleTest struct {
	fastStatBucketTest
}

func init() { RegisterTestSuite(&NewReaderWithReadHandleTest{}) }

func (t *NewReaderWithReadHandleTest) CallsWrappedAndInvalidatesOnNotFound() {
	const name = "some-name"
	// Expect: wrapped bucket returns NotFoundError
	var wrappedReq *gcs.ReadObjectRequest
	ExpectCall(t.wrapped, "NewReaderWithReadHandle")(Any(), Any()).
		WillOnce(DoAll(SaveArg(1, &wrappedReq), Return(nil, &gcs.NotFoundError{Err: errors.New("not found")})))
	// Expect: cache invalidate is called
	ExpectCall(t.cache, "Erase")(name)

	// Call
	req := &gcs.ReadObjectRequest{Name: name}
	rd, err := t.bucket.NewReaderWithReadHandle(context.TODO(), req)

	AssertEq(nil, rd)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
	AssertEq(name, wrappedReq.Name)
}

func (t *NewReaderWithReadHandleTest) CallsWrappedAndDoesNotInvalidateOnSuccess() {
	const name = "some-name"
	expectedReader := &fake.FakeReader{ReadCloser: io.NopCloser(strings.NewReader("abc")), Handle: []byte("fake")}
	// Expect: wrapped returns reader, no error
	ExpectCall(t.wrapped, "NewReaderWithReadHandle")(Any(), Any()).
		WillOnce(Return(expectedReader, nil))

	// Call
	req := &gcs.ReadObjectRequest{Name: name}
	rd, err := t.bucket.NewReaderWithReadHandle(context.TODO(), req)

	AssertEq(nil, err)
	ExpectEq(expectedReader, rd)
}

////////////////////////////////////////////////////////////////////////
// NewMultiRangeDownloader
////////////////////////////////////////////////////////////////////////

type NewMultiRangeDownloaderTest struct {
	fastStatBucketTest
}

func init() { RegisterTestSuite(&NewMultiRangeDownloaderTest{}) }

func (t *NewMultiRangeDownloaderTest) CallsWrappedAndInvalidatesOnNotFound() {
	const name = "some-name"
	// Expect: wrapped bucket returns NotFoundError
	var wrappedReq *gcs.MultiRangeDownloaderRequest
	ExpectCall(t.wrapped, "NewMultiRangeDownloader")(Any(), Any()).
		WillOnce(DoAll(SaveArg(1, &wrappedReq), Return(nil, &gcs.NotFoundError{Err: errors.New("not found")})))
	// Expect: cache invalidate is called
	ExpectCall(t.cache, "Erase")(name)

	// Call
	req := &gcs.MultiRangeDownloaderRequest{Name: name}
	mrd, err := t.bucket.NewMultiRangeDownloader(context.TODO(), req)

	AssertEq(nil, mrd)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
	AssertEq(name, wrappedReq.Name)
}

func (t *NewMultiRangeDownloaderTest) CallsWrappedAndDoesNotInvalidateOnSuccess() {
	const name = "some-name"
	expectedMrd := fake.NewFakeMultiRangeDownloader(&gcs.MinObject{Name: name}, nil)
	// Expect: wrapped returns mrd, no error
	ExpectCall(t.wrapped, "NewMultiRangeDownloader")(Any(), Any()).
		WillOnce(Return(expectedMrd, nil))

	// Call
	req := &gcs.MultiRangeDownloaderRequest{Name: name}
	mrd, err := t.bucket.NewMultiRangeDownloader(context.TODO(), req)

	AssertEq(nil, err)
	ExpectEq(expectedMrd, mrd)
}

////////////////////////////////////////////////////////////////////////
// GetFolder
////////////////////////////////////////////////////////////////////////

type GetFolderTest struct {
	fastStatBucketTest
}

func init() { RegisterTestSuite(&GetFolderTest{}) }

func (t *GetFolderTest) FetchOnlyFromCacheFalse() {
	const name = "taco/"
	req := &gcs.GetFolderRequest{
		Name:               name,
		FetchOnlyFromCache: false,
	}
	folder := &gcs.Folder{Name: name}
	ExpectCall(t.cache, "LookUpFolder")(name, Any()).
		WillOnce(Return(false, nil))
	ExpectCall(t.wrapped, "GetFolder")(Any(), Any()).
		WillOnce(Return(folder, nil))
	ExpectCall(t.cache, "InsertFolder")(Any(), Any())

	f, err := t.bucket.GetFolder(context.TODO(), req)

	AssertEq(nil, err)
	ExpectEq(folder, f)
}

func (t *GetFolderTest) FetchOnlyFromCacheTrue_CacheHitPositive() {
	const name = "taco/"
	req := &gcs.GetFolderRequest{
		Name:               name,
		FetchOnlyFromCache: true,
	}
	folder := &gcs.Folder{Name: name}
	ExpectCall(t.cache, "LookUpFolder")(name, Any()).
		WillOnce(Return(true, folder))

	f, err := t.bucket.GetFolder(context.TODO(), req)

	AssertEq(nil, err)
	ExpectEq(folder, f)
}

func (t *GetFolderTest) FetchOnlyFromCacheTrue_CacheHitNegative() {
	const name = "taco/"
	req := &gcs.GetFolderRequest{
		Name:               name,
		FetchOnlyFromCache: true,
	}
	ExpectCall(t.cache, "LookUpFolder")(name, Any()).
		WillOnce(Return(true, nil))

	_, err := t.bucket.GetFolder(context.TODO(), req)

	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *GetFolderTest) FetchOnlyFromCacheTrue_CacheMiss() {
	const name = "taco/"
	req := &gcs.GetFolderRequest{
		Name:               name,
		FetchOnlyFromCache: true,
	}
	ExpectCall(t.cache, "LookUpFolder")(name, Any()).
		WillOnce(Return(false, nil))

	_, err := t.bucket.GetFolder(context.TODO(), req)

	ExpectThat(err, HasSameTypeAs(&caching.CacheMissError{}))
}
