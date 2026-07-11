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

package caching_test

import (
	"errors"
	"testing"
	"time"

	gostorage "cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/caching"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/caching/mock_gcscaching"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
)

func TestFastStatBucketAdv(t *testing.T) { RunTests(t) }

type FastStatBucketAdvTest struct {
	cache   mock_gcscaching.MockStatCache
	clock   timeutil.SimulatedClock
	wrapped storage.MockBucket

	bucket gcs.Bucket
}

func init() { RegisterTestSuite(&FastStatBucketAdvTest{}) }

func (t *FastStatBucketAdvTest) SetUp(ti *TestInfo) {
	t.clock.SetTime(time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC))
	t.cache = mock_gcscaching.NewMockStatCache(ti.MockController, "cache")
	t.wrapped = storage.NewMockBucket(ti.MockController, "wrapped")

	t.bucket = caching.NewFastStatBucket(
		time.Second,
		t.cache,
		&t.clock,
		t.wrapped,
		time.Second*5,
		true,
		true,
	)
}

func (t *FastStatBucketAdvTest) TestAdv_StatObject_ContextCancelledDoesNotUpdateCache() {
	const name = "taco"
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	req := &gcs.StatObjectRequest{
		Name:              name,
		ForceFetchFromGcs: true,
	}
	minObj := &gcs.MinObject{Name: name, Size: 123}

	ExpectCall(t.wrapped, "StatObject")(ctx, req).WillOnce(Return(minObj, nil, nil))
	// Because context is cancelled, insertMinObject should return early without updating cache.
	ExpectCall(t.cache, "Insert")(Any(), Any()).Times(0)

	m, _, err := t.bucket.StatObject(ctx, req)
	AssertEq(nil, err)
	ExpectEq(minObj, m)
}

func (t *FastStatBucketAdvTest) TestAdv_GetFolder_ContextCancelledDoesNotUpdateCache() {
	const name = "folder/"
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	req := &gcs.GetFolderRequest{Name: name}
	folder := &gcs.Folder{Name: name}

	ExpectCall(t.cache, "LookUpFolder")(name, Any()).WillOnce(Return(false, nil))
	ExpectCall(t.wrapped, "GetFolder")(ctx, req).WillOnce(Return(folder, nil))
	// In an adversarial test for context cancellation consistency, if ctx is cancelled,
	// GetFolder should NOT insert into cache, consistent with StatObject and ListObjects.
	ExpectCall(t.cache, "InsertFolder")(Any(), Any()).Times(0)

	f, err := t.bucket.GetFolder(ctx, req)
	AssertEq(nil, err)
	ExpectEq(folder, f)
}

func (t *FastStatBucketAdvTest) TestAdv_CreateFolder_ContextCancelledDoesNotUpdateCache() {
	const name = "folder/"
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	folder := &gcs.Folder{Name: name}
	ExpectCall(t.wrapped, "CreateFolder")(ctx, name).WillOnce(Return(folder, nil))
	ExpectCall(t.cache, "Erase")(name).WillOnce(Return())
	// In an adversarial test for context cancellation consistency, if ctx is cancelled,
	// CreateFolder should NOT insert into cache.
	ExpectCall(t.cache, "InsertFolder")(Any(), Any()).Times(0)

	f, err := t.bucket.CreateFolder(ctx, name)
	AssertEq(nil, err)
	ExpectEq(folder, f)
}

func (t *FastStatBucketAdvTest) TestAdv_CreateObject_ContextCancelledDoesNotUpdateCache() {
	const name = "taco"
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	req := &gcs.CreateObjectRequest{Name: name}
	obj := &gcs.Object{Name: name, Size: 100}
	ExpectCall(t.wrapped, "CreateObject")(ctx, req).WillOnce(Return(obj, nil))
	ExpectCall(t.cache, "Erase")(name).WillOnce(Return())
	// In an adversarial test for context cancellation consistency, if ctx is cancelled,
	// CreateObject should NOT insert into cache.
	ExpectCall(t.cache, "Insert")(Any(), Any()).Times(0)

	o, err := t.bucket.CreateObject(ctx, req)
	AssertEq(nil, err)
	ExpectEq(obj, o)
}

func (t *FastStatBucketAdvTest) TestAdv_CopyObject_ContextCancelledDoesNotUpdateCache() {
	const src = "src"
	const dst = "dst"
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	req := &gcs.CopyObjectRequest{SrcName: src, DstName: dst}
	obj := &gcs.Object{Name: dst, Size: 200}
	ExpectCall(t.wrapped, "CopyObject")(ctx, req).WillOnce(Return(obj, nil))
	ExpectCall(t.cache, "Erase")(dst).WillOnce(Return())
	// In an adversarial test for context cancellation consistency, if ctx is cancelled,
	// CopyObject should NOT insert into cache.
	ExpectCall(t.cache, "Insert")(Any(), Any()).Times(0)

	o, err := t.bucket.CopyObject(ctx, req)
	AssertEq(nil, err)
	ExpectEq(obj, o)
}

func (t *FastStatBucketAdvTest) TestAdv_ComposeObjects_ContextCancelledDoesNotUpdateCache() {
	const dst = "dst"
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	req := &gcs.ComposeObjectsRequest{DstName: dst}
	obj := &gcs.Object{Name: dst, Size: 300}
	ExpectCall(t.wrapped, "ComposeObjects")(ctx, req).WillOnce(Return(obj, nil))
	ExpectCall(t.cache, "Erase")(dst).WillOnce(Return())
	// In an adversarial test for context cancellation consistency, if ctx is cancelled,
	// ComposeObjects should NOT insert into cache.
	ExpectCall(t.cache, "Insert")(Any(), Any()).Times(0)

	o, err := t.bucket.ComposeObjects(ctx, req)
	AssertEq(nil, err)
	ExpectEq(obj, o)
}

func (t *FastStatBucketAdvTest) TestAdv_MoveObject_ContextCancelledDoesNotUpdateCache() {
	const src = "src"
	const dst = "dst"
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	req := &gcs.MoveObjectRequest{SrcName: src, DstName: dst}
	obj := &gcs.Object{Name: dst, Size: 400}
	ExpectCall(t.wrapped, "MoveObject")(ctx, req).WillOnce(Return(obj, nil))
	ExpectCall(t.cache, "Erase")(src).WillOnce(Return())
	ExpectCall(t.cache, "Erase")(dst).WillOnce(Return())
	// In an adversarial test for context cancellation consistency, if ctx is cancelled,
	// MoveObject should NOT insert into cache.
	ExpectCall(t.cache, "Insert")(Any(), Any()).Times(0)

	o, err := t.bucket.MoveObject(ctx, req)
	AssertEq(nil, err)
	ExpectEq(obj, o)
}

func (t *FastStatBucketAdvTest) TestAdv_FinalizeUpload_ContextCancelledDoesNotUpdateCache() {
	const name = "taco"
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	writer := &storage.ObjectWriter{
		Writer: &gostorage.Writer{ObjectAttrs: gostorage.ObjectAttrs{Name: name}},
	}
	minObj := &gcs.MinObject{Name: name, Size: 456}

	ExpectCall(t.wrapped, "FinalizeUpload")(ctx, writer).WillOnce(Return(minObj, nil))
	ExpectCall(t.cache, "Erase")(name).WillOnce(Return())
	// Because context is cancelled, insertMinObject should not insert into cache.
	ExpectCall(t.cache, "Insert")(Any(), Any()).Times(0)

	m, err := t.bucket.FinalizeUpload(ctx, writer)
	AssertEq(nil, err)
	ExpectEq(minObj, m)
}

func (t *FastStatBucketAdvTest) TestAdv_FlushPendingWrites_ContextCancelledDoesNotUpdateCache() {
	const name = "taco"
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	writer := &storage.ObjectWriter{
		Writer: &gostorage.Writer{ObjectAttrs: gostorage.ObjectAttrs{Name: name}},
	}
	minObj := &gcs.MinObject{Name: name, Size: 789}

	ExpectCall(t.wrapped, "FlushPendingWrites")(ctx, writer).WillOnce(Return(minObj, nil))
	ExpectCall(t.cache, "Erase")(name).WillOnce(Return())
	// Because context is cancelled, insertMinObject should not insert into cache.
	ExpectCall(t.cache, "Insert")(Any(), Any()).Times(0)

	m, err := t.bucket.FlushPendingWrites(ctx, writer)
	AssertEq(nil, err)
	ExpectEq(minObj, m)
}

func (t *FastStatBucketAdvTest) TestAdv_ListObjects_TypeCacheNotDeprecated() {
	// Recreate bucket with isTypeCacheDeprecated = false
	bucket := caching.NewFastStatBucket(
		time.Second,
		t.cache,
		&t.clock,
		t.wrapped,
		time.Second*5,
		false,
		true,
	)

	req := &gcs.ListObjectsRequest{Prefix: "dir/"}
	o0 := &gcs.MinObject{Name: "dir/file.txt", Size: 10}
	listing := &gcs.Listing{
		MinObjects:    []*gcs.MinObject{o0},
		CollapsedRuns: []string{"dir/sub/"},
	}

	ExpectCall(t.wrapped, "BucketType")().WillOnce(Return(gcs.BucketType{Hierarchical: false}))
	ExpectCall(t.wrapped, "ListObjects")(Any(), req).WillOnce(Return(listing, nil))

	// When isTypeCacheDeprecated is false, only MinObjects are cached via insertMultipleMinObjects.
	// Implicit dirs (dir/ and CollapsedRuns dir/sub/) should NOT be cached.
	ExpectCall(t.cache, "Insert")(o0, timeutil.TimeEq(t.clock.Now().Add(time.Second))).Times(1)
	ExpectCall(t.cache, "InsertImplicitDir")(Any(), Any()).Times(0)

	gotListing, err := bucket.ListObjects(context.Background(), req)
	AssertEq(nil, err)
	ExpectEq(listing, gotListing)
}

func (t *FastStatBucketAdvTest) TestAdv_GetFolder_NotFoundErrorAddsNegativeEntry() {
	const name = "missing/"
	req := &gcs.GetFolderRequest{Name: name}

	ExpectCall(t.cache, "LookUpFolder")(name, Any()).WillOnce(Return(false, nil))
	ExpectCall(t.wrapped, "GetFolder")(Any(), req).WillOnce(Return(nil, &gcs.NotFoundError{Err: errors.New("not found")}))
	ExpectCall(t.cache, "AddNegativeEntryForFolder")(name, timeutil.TimeEq(t.clock.Now().Add(time.Second*5))).Times(1)

	_, err := t.bucket.GetFolder(context.Background(), req)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *FastStatBucketAdvTest) TestAdv_ListObjects_Hierarchical_MalformedPrefixIgnored() {
	req := &gcs.ListObjectsRequest{}
	listing := &gcs.Listing{
		CollapsedRuns: []string{"malformed_prefix", "valid_folder/"},
	}

	ExpectCall(t.wrapped, "BucketType")().WillOnce(Return(gcs.BucketType{Hierarchical: true}))
	ExpectCall(t.wrapped, "ListObjects")(Any(), req).WillOnce(Return(listing, nil))

	// Only "valid_folder/" should be inserted as folder; "malformed_prefix" should be logged and skipped.
	validFolder := gcs.Folder{Name: "valid_folder/"}
	ExpectCall(t.cache, "InsertFolder")(Pointee(DeepEquals(validFolder)), timeutil.TimeEq(t.clock.Now().Add(time.Second))).Times(1)

	gotListing, err := t.bucket.ListObjects(context.Background(), req)
	AssertEq(nil, err)
	ExpectEq(listing, gotListing)
}

func (t *FastStatBucketAdvTest) TestAdv_RenameFolder_WrappedFailsDoesNotModifyCache() {
	const name = "old/"
	const newName = "new/"

	ExpectCall(t.wrapped, "RenameFolder")(Any(), name, newName).WillOnce(Return(nil, errors.New("rename failed")))
	ExpectCall(t.cache, "EraseEntriesWithGivenPrefix")(Any()).Times(0)
	ExpectCall(t.cache, "InsertFolder")(Any(), Any()).Times(0)

	_, err := t.bucket.RenameFolder(context.Background(), name, newName)
	AssertNe(nil, err)
}

func (t *FastStatBucketAdvTest) TestAdv_UpdateObject_ContextCancelledDoesNotUpdateCache() {
	const name = "taco"
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	req := &gcs.UpdateObjectRequest{Name: name}
	obj := &gcs.Object{Name: name, Size: 100}
	ExpectCall(t.wrapped, "UpdateObject")(ctx, req).WillOnce(Return(obj, nil))
	ExpectCall(t.cache, "Erase")(name).WillOnce(Return())
	// In an adversarial test for context cancellation consistency, if ctx is cancelled,
	// UpdateObject should NOT insert into cache.
	ExpectCall(t.cache, "Insert")(Any(), Any()).Times(0)

	o, err := t.bucket.UpdateObject(ctx, req)
	AssertEq(nil, err)
	ExpectEq(obj, o)
}

func (t *FastStatBucketAdvTest) TestAdv_RenameFolder_ContextCancelledDoesNotUpdateCache() {
	const name = "old/"
	const newName = "new/"
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	folder := &gcs.Folder{Name: newName}
	ExpectCall(t.wrapped, "RenameFolder")(ctx, name, newName).WillOnce(Return(folder, nil))
	ExpectCall(t.cache, "EraseEntriesWithGivenPrefix")(name).WillOnce(Return())
	// In an adversarial test for context cancellation consistency, if ctx is cancelled,
	// RenameFolder should NOT insert into cache.
	ExpectCall(t.cache, "InsertFolder")(Any(), Any()).Times(0)

	f, err := t.bucket.RenameFolder(ctx, name, newName)
	AssertEq(nil, err)
	ExpectEq(folder, f)
}
