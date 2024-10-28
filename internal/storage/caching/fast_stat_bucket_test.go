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
	"testing"
	"time"

	gostorage "cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/caching"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/caching/mock_gcscaching"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
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

const ttl = time.Second

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
		ttl,
		t.cache,
		&t.clock,
		t.wrapped)
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
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(ttl)))

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
		WillOnce(DoAll(SaveArg(1, &wrappedWriter), Return(&gcs.Object{}, errors.New(""))))

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
		WillOnce(Return(&gcs.Object{}, errors.New("taco")))

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
		WillOnce(Return(&gcs.Object{}, nil))
	// Insert
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(ttl)))

	// Call
	o, err := t.bucket.FinalizeUpload(context.TODO(), writer)

	AssertEq(nil, err)
	ExpectNe(nil, o)
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
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(ttl)))

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
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(ttl)))

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
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(ttl)))

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
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(ttl)))

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
		timeutil.TimeEq(t.clock.Now().Add(ttl)))

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
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(ttl)))

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
		WillOnce(Return(gcs.NonHierarchical))

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
		WillOnce(Return(gcs.Hierarchical))

	ExpectCall(t.wrapped, "ListObjects")(Any(), Any()).
		WillOnce(Return(expected, nil))

	// call
	listing, err := t.bucket.ListObjects(context.TODO(), &gcs.ListObjectsRequest{})

	AssertEq(nil, err)
	ExpectEq(expected, listing)
}

func (t *ListObjectsTest) NonEmptyListing() {
	// Wrapped
	o0 := &gcs.Object{Name: "taco"}
	o1 := &gcs.Object{Name: "burrito"}

	expected := &gcs.Listing{
		Objects: []*gcs.Object{o0, o1},
	}

	ExpectCall(t.wrapped, "BucketType")().
		WillOnce(Return(gcs.NonHierarchical))

	ExpectCall(t.wrapped, "ListObjects")(Any(), Any()).
		WillOnce(Return(expected, nil))

	// Insert
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(ttl))).Times(2)

	// Call
	listing, err := t.bucket.ListObjects(context.TODO(), &gcs.ListObjectsRequest{})

	AssertEq(nil, err)
	ExpectEq(expected, listing)
}

func (t *ListObjectsTest) NonEmptyListingForHNS() {
	// wrapped
	o0 := &gcs.Object{Name: "taco"}
	o1 := &gcs.Object{Name: "burrito"}

	expected := &gcs.Listing{
		Objects:       []*gcs.Object{o0, o1},
		CollapsedRuns: []string{"p0", "p1/"},
	}

	ExpectCall(t.wrapped, "BucketType")().
		WillOnce(Return(gcs.Hierarchical))

	ExpectCall(t.wrapped, "ListObjects")(Any(), Any()).
		WillOnce(Return(expected, nil))

	// insert
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(ttl))).Times(2)

	ExpectCall(t.cache, "InsertFolder")(Any(), timeutil.TimeEq(t.clock.Now().Add(ttl))).Times(1)

	// call
	listing, err := t.bucket.ListObjects(context.TODO(), &gcs.ListObjectsRequest{})

	AssertEq(nil, err)
	ExpectEq(expected, listing)
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
	ExpectCall(t.cache, "Insert")(Any(), timeutil.TimeEq(t.clock.Now().Add(ttl)))

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

func (t *DeleteObjectTest) CallsEraseAndWrapped() {
	const name = "taco"

	// Erase
	ExpectCall(t.cache, "Erase")(name)

	// Wrapped
	var wrappedReq *gcs.DeleteObjectRequest
	ExpectCall(t.wrapped, "DeleteObject")(Any(), Any()).
		WillOnce(DoAll(SaveArg(1, &wrappedReq), Return(errors.New(""))))

	// Call
	_ = t.deleteObject(name)

	AssertNe(nil, wrappedReq)
	ExpectEq(name, wrappedReq.Name)
}

func (t *DeleteObjectTest) WrappedFails() {
	const name = ""
	var err error

	// Erase
	ExpectCall(t.cache, "Erase")(Any())

	// Wrapped
	ExpectCall(t.wrapped, "DeleteObject")(Any(), Any()).
		WillOnce(Return(errors.New("taco")))

	// Call
	err = t.deleteObject(name)

	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *DeleteObjectTest) WrappedSucceeds() {
	const name = ""
	var err error

	// Erase
	ExpectCall(t.cache, "Erase")(Any())

	// Wrapped
	ExpectCall(t.wrapped, "DeleteObject")(Any(), Any()).
		WillOnce(Return(nil))

	// Call
	err = t.deleteObject(name)
	AssertEq(nil, err)
}

func (t *StatObjectTest) TestShouldReturnFromCacheWhenEntryIsPresent() {
	const name = "some-name"
	folder := &gcs.Folder{
		Name: name,
	}
	ExpectCall(t.cache, "LookUpFolder")(name, Any()).
		WillOnce(Return(true, folder))

	result, err := t.bucket.GetFolder(context.TODO(), name)

	AssertEq(nil, err)
	ExpectThat(result, Pointee(DeepEquals(*folder)))
}

func (t *StatObjectTest) TestShouldReturnNotFoundErrorWhenNilEntryIsReturned() {
	const name = "some-name"

	ExpectCall(t.cache, "LookUpFolder")(name, Any()).
		WillOnce(Return(true, nil))

	result, err := t.bucket.GetFolder(context.TODO(), name)

	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
	AssertEq(nil, result)
}

func (t *StatObjectTest) TestShouldCallGetFolderWhenEntryIsNotPresent() {
	const name = "some-name"
	folder := &gcs.Folder{
		Name: name,
	}

	ExpectCall(t.cache, "LookUpFolder")(name, Any()).
		WillOnce(Return(false, nil))
	ExpectCall(t.cache, "InsertFolder")(folder, Any()).
		WillOnce(Return())
	ExpectCall(t.wrapped, "GetFolder")(Any(), name).
		WillOnce(Return(folder, nil))

	result, err := t.bucket.GetFolder(context.TODO(), name)

	AssertEq(nil, err)
	ExpectThat(result, Pointee(DeepEquals(*folder)))
}

func (t *StatObjectTest) TestShouldReturnNilWhenErrorIsReturnedFromGetFolder() {
	const name = "some-name"
	error := errors.New("connection error")

	ExpectCall(t.cache, "LookUpFolder")(name, Any()).
		WillOnce(Return(false, nil))
	ExpectCall(t.wrapped, "GetFolder")(Any(), name).
		WillOnce(Return(nil, error))

	folder, result := t.bucket.GetFolder(context.TODO(), name)

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

type DeleteFolderTest struct {
	fastStatBucketTest
}

func init() { RegisterTestSuite(&DeleteFolderTest{}) }

func (t *DeleteFolderTest) Test_DeleteFolder_Success() {
	const name = "some-name"
	ExpectCall(t.wrapped, "DeleteFolder")(Any(), name).
		WillOnce(Return(nil))
	ExpectCall(t.cache, "Erase")(name).WillOnce(Return())

	err := t.bucket.DeleteFolder(context.TODO(), name)

	AssertEq(nil, err)
}

func (t *DeleteFolderTest) Test_DeleteFolder_Failure() {
	const name = "some-name"
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
