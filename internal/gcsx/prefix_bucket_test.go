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

package gcsx_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vipnydav/gcsfuse/v3/internal/storage/fake"
	"github.com/vipnydav/gcsfuse/v3/internal/storage/gcs"
	"github.com/vipnydav/gcsfuse/v3/internal/storage/storageutil"
	"golang.org/x/net/context"

	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/internal/gcsx"

	"github.com/jacobsa/timeutil"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type PrefixBucketTest struct {
	suite.Suite
	ctx     context.Context
	prefix  string
	wrapped gcs.Bucket
	bucket  gcs.Bucket
}

func TestPrefixBucket(t *testing.T) {
	suite.Run(t, new(PrefixBucketTest))
}

func (t *PrefixBucketTest) SetupTest() {
	var err error

	t.ctx = context.Background()
	t.prefix = "foo_"
	t.wrapped = fake.NewFakeBucket(timeutil.RealClock(), "some_bucket", gcs.BucketType{})

	t.bucket, err = gcsx.NewPrefixBucket(t.prefix, t.wrapped)
	assert.NoError(t.T(), err)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *PrefixBucketTest) Test_Name() {
	assert.Equal(t.T(), t.wrapped.Name(), t.bucket.Name())
}

func (t *PrefixBucketTest) Test_NewReader() {
	var err error
	suffix := "taco"
	name := t.prefix + suffix
	contents := "foobar"

	// Create an object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte(contents))
	assert.Equal(t.T(), nil, err)

	// Read it through the prefix bucket.
	rc, err := t.bucket.NewReaderWithReadHandle(
		t.ctx,
		&gcs.ReadObjectRequest{
			Name: suffix,
		})

	assert.Equal(t.T(), nil, err)
	defer rc.Close()

	actual, err := io.ReadAll(rc)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), contents, string(actual))
}

func (t *PrefixBucketTest) Test_NewReaderWithReadHandle() {
	var err error
	suffix := "taco"
	name := t.prefix + suffix
	contents := "foobar"
	// Create an object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte(contents))
	assert.Equal(t.T(), nil, err)

	// Read it through the prefix bucket with read handle.
	rc, err := t.bucket.NewReaderWithReadHandle(
		t.ctx,
		&gcs.ReadObjectRequest{
			Name:       suffix,
			ReadHandle: []byte("new-handle"),
		})

	assert.Equal(t.T(), nil, err)
	defer rc.Close()
	actual, err := io.ReadAll(rc)
	assert.NoError(t.T(), nil, err)
	assert.Equal(t.T(), contents, string(actual))
	assert.Equal(t.T(), string(rc.ReadHandle()), "opaque-handle")
}

func (t *PrefixBucketTest) Test_NewReaderWithNilReadHandle() {
	var err error
	suffix := "taco"
	name := t.prefix + suffix
	contents := "foobar"
	// Create an object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte(contents))
	assert.Equal(t.T(), nil, err)

	// Read it through the prefix bucket with out read handle.
	rc, err := t.bucket.NewReaderWithReadHandle(
		t.ctx,
		&gcs.ReadObjectRequest{
			Name: suffix,
		})

	assert.NoError(t.T(), err)
	defer rc.Close()
	actual, err := io.ReadAll(rc)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), contents, string(actual))
	assert.Equal(t.T(), string(rc.ReadHandle()), "opaque-handle")
}

func (t *PrefixBucketTest) Test_NewMultiRangeReader_WithFullContentRead() {
	var err error
	suffix := "taco"
	name := t.prefix + suffix
	contents := "foobar"
	// Create an object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte(contents))
	assert.Equal(t.T(), nil, err)

	// Read it through the prefix bucket.
	mrd, err := t.bucket.NewMultiRangeDownloader(
		t.ctx,
		&gcs.MultiRangeDownloaderRequest{
			Name: suffix,
		})

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), mrd)
	defer func() {
		assert.NoError(t.T(), mrd.Close())
	}()

	size := int64(len(contents))
	var outputString string
	outputWriter := bytes.NewBufferString(outputString)
	mrd.Add(outputWriter, 0, size, func(int64, int64, error) {})
	mrd.Wait()

	assert.Equal(t.T(), contents, outputWriter.String())
}

func (t *PrefixBucketTest) Test_NewMultiRangeReader_WithoutWait() {
	var err error
	suffix := "taco"
	name := t.prefix + suffix
	contents := "foobar"
	// Create an object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte(contents))
	assert.Equal(t.T(), nil, err)

	// Read it through the prefix bucket with out read handle.
	mrd, err := t.bucket.NewMultiRangeDownloader(
		t.ctx,
		&gcs.MultiRangeDownloaderRequest{
			Name: suffix,
		})

	var outputString string
	outputWriter := bytes.NewBufferString(outputString)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), mrd)
	defer func() {
		assert.NoError(t.T(), mrd.Close())
		assert.Equal(t.T(), contents, outputWriter.String())
	}()

	size := int64(len(contents))
	mrd.Add(outputWriter, 0, size, func(offset, length int64, err error) {})
}

func (t *PrefixBucketTest) Test_NewMultiRangeReader_WithMultipleReads() {
	var err error
	suffix := "taco"
	name := t.prefix + suffix
	contents := "foobar"
	// Create an object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte(contents))
	assert.Equal(t.T(), nil, err)

	// Read it through the prefix bucket with out read handle.
	mrd, err := t.bucket.NewMultiRangeDownloader(
		t.ctx,
		&gcs.MultiRangeDownloaderRequest{
			Name: suffix,
		})

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), mrd)
	defer func() {
		assert.NoError(t.T(), mrd.Close())
	}()

	size := int64(len(contents))
	halfSize := size / 2
	var outputString1 string
	outputWriter1 := bytes.NewBufferString(outputString1)
	mrd.Add(outputWriter1, 0, halfSize, func(offset, length int64, err error) {})

	var outputString2 string
	outputWriter2 := bytes.NewBufferString(outputString2)
	mrd.Add(outputWriter2, halfSize, halfSize, func(offset, length int64, err error) {})

	mrd.Wait()

	assert.Equal(t.T(), "foo", outputWriter1.String())
	assert.Equal(t.T(), "bar", outputWriter2.String())
}

func (t *PrefixBucketTest) Test_NewMultiRangeReader_WithOutOfBoundsReadError() {
	var err error
	suffix := "taco"
	name := t.prefix + suffix
	contents := "foobar"
	// Create an object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte(contents))
	assert.Equal(t.T(), nil, err)

	// Read it through the prefix bucket with out read handle.
	mrd, err := t.bucket.NewMultiRangeDownloader(
		t.ctx,
		&gcs.MultiRangeDownloaderRequest{
			Name: suffix,
		})

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), mrd)
	defer func() {
		assert.Error(t.T(), mrd.Close())
	}()

	size := int64(len(contents))
	var outputString string
	outputWriter := bytes.NewBufferString(outputString)
	mrd.Add(outputWriter, size+1, 1, func(offset, length int64, err error) {})
}

func (t *PrefixBucketTest) Test_NewMultiRangeReader_WithNonexistentObjectError() {
	var err error

	// Read it through the prefix bucket with out read handle.
	mrd, err := t.bucket.NewMultiRangeDownloader(
		t.ctx,
		&gcs.MultiRangeDownloaderRequest{
			Name: "taco",
		})

	assert.Error(t.T(), err)
	assert.Nil(t.T(), mrd)
}

func (t *PrefixBucketTest) Test_CreateObject() {
	var err error
	suffix := "taco"
	contents := "foobar"

	// Create the object.
	o, err := t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:            suffix,
			ContentLanguage: "en-GB",
			Contents:        strings.NewReader(contents),
		})

	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), suffix, o.Name)
	assert.Equal(t.T(), "en-GB", o.ContentLanguage)

	// Read it through the back door.
	actual, err := storageutil.ReadObject(t.ctx, t.wrapped, t.prefix+suffix)
	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), contents, string(actual))
}

func (t *PrefixBucketTest) TestCreateObjectChunkWriterAndFinalizeUpload() {
	var err error
	suffix := "taco"
	content := []byte("foobar")

	// Create the object.
	w, err := t.bucket.CreateObjectChunkWriter(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:            suffix,
			ContentEncoding: "gzip",
			Contents:        nil,
		},
		1024, nil)
	assert.NoError(t.T(), err)
	_, err = w.Write(content)
	assert.NoError(t.T(), err)
	o, err := t.bucket.FinalizeUpload(t.ctx, w)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), suffix, o.Name)
	assert.Equal(t.T(), "gzip", o.ContentEncoding)
	// Read it through the back door.
	actual, err := storageutil.ReadObject(t.ctx, t.wrapped, t.prefix+suffix)
	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), string(content), string(actual))
}

func (t *PrefixBucketTest) TestCreateObjectChunkWriterAndFlushPendingWrites() {
	var err error
	suffix := "taco"
	content := []byte("foobar")

	// Create the object.
	w, err := t.bucket.CreateObjectChunkWriter(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:            suffix,
			ContentEncoding: "gzip",
			Contents:        nil,
		},
		1024, nil)
	assert.NoError(t.T(), err)
	_, err = w.Write(content)
	assert.NoError(t.T(), err)
	o, err := t.bucket.FlushPendingWrites(t.ctx, w)

	assert.NoError(t.T(), err)
	assert.EqualValues(t.T(), int64(len(content)), o.Size)
	assert.Equal(t.T(), suffix, o.Name)
	// Read it through the back door.
	actual, err := storageutil.ReadObject(t.ctx, t.wrapped, t.prefix+suffix)
	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), string(content), string(actual))
}

func (t *PrefixBucketTest) TestCreateAppendableObjectWriterAndFlush() {
	var err error
	suffix := "taco"
	content := []byte("foobar")

	// Create the object writer.
	w, err := t.bucket.CreateAppendableObjectWriter(
		t.ctx,
		&gcs.CreateObjectChunkWriterRequest{
			CreateObjectRequest: gcs.CreateObjectRequest{
				Name: suffix,
			},
			ChunkSize: 1024,
			Offset:    10,
		})
	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), w)
	_, err = w.Write(content)
	assert.NoError(t.T(), err)
	o, err := t.bucket.FlushPendingWrites(t.ctx, w)

	assert.NoError(t.T(), err)
	assert.EqualValues(t.T(), int64(len(content)), o.Size)
	// Read it through the back door.
	actual, err := storageutil.ReadObject(t.ctx, t.wrapped, t.prefix+suffix)
	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), string(content), string(actual))
}

func (t *PrefixBucketTest) TestCreateAppendableObjectWriterAndClose() {
	var err error
	suffix := "taco"
	content := []byte("foobar")

	// Create the object writer.
	w, err := t.bucket.CreateAppendableObjectWriter(
		t.ctx,
		&gcs.CreateObjectChunkWriterRequest{
			CreateObjectRequest: gcs.CreateObjectRequest{
				Name: suffix,
			},
			ChunkSize: 1024,
			Offset:    10,
		})
	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), w)
	_, err = w.Write(content)
	assert.NoError(t.T(), err)
	o, err := t.bucket.FinalizeUpload(t.ctx, w)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), suffix, o.Name)

	// Read it through the back door.
	actual, err := storageutil.ReadObject(t.ctx, t.wrapped, t.prefix+suffix)
	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), string(content), string(actual))
}

func (t *PrefixBucketTest) Test_CopyObject() {
	var err error
	suffix := "taco"
	name := t.prefix + suffix
	contents := "foobar"

	// Create an object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte(contents))
	assert.Equal(t.T(), nil, err)

	// Copy it to a new name.
	newSuffix := "burrito"
	o, err := t.bucket.CopyObject(
		t.ctx,
		&gcs.CopyObjectRequest{
			SrcName: suffix,
			DstName: newSuffix,
		})

	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), newSuffix, o.Name)

	// Read it through the back door.
	actual, err := storageutil.ReadObject(t.ctx, t.wrapped, t.prefix+newSuffix)
	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), contents, string(actual))
}

func (t *PrefixBucketTest) Test_ComposeObjects() {
	var err error

	suffix0 := "taco"
	contents0 := "foo"

	suffix1 := "burrito"
	contents1 := "bar"

	// Create two objects through the back door.
	err = storageutil.CreateObjects(
		t.ctx,
		t.wrapped,
		map[string][]byte{
			t.prefix + suffix0: []byte(contents0),
			t.prefix + suffix1: []byte(contents1),
		})

	assert.Equal(t.T(), nil, err)

	// Compose them.
	newSuffix := "enchilada"
	o, err := t.bucket.ComposeObjects(
		t.ctx,
		&gcs.ComposeObjectsRequest{
			DstName: newSuffix,
			Sources: []gcs.ComposeSource{
				{Name: suffix0},
				{Name: suffix1},
			},
		})

	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), newSuffix, o.Name)

	// Read it through the back door.
	actual, err := storageutil.ReadObject(t.ctx, t.wrapped, t.prefix+newSuffix)
	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), contents0+contents1, string(actual))
}

func (t *PrefixBucketTest) Test_StatObject() {
	var err error
	suffix := "taco"
	name := t.prefix + suffix
	contents := "foobar"

	// Create an object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte(contents))
	assert.Equal(t.T(), nil, err)

	// Stat it.
	m, _, err := t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{
			Name: suffix,
		})

	assert.Equal(t.T(), nil, err)
	assert.NotEqual(t.T(), nil, m)
	assert.Equal(t.T(), suffix, m.Name)
	assert.Equal(t.T(), uint64(len(contents)), m.Size)
}

func (t *PrefixBucketTest) Test_ListObjects_NoOptions() {
	var err error

	// Create a few objects.
	err = storageutil.CreateObjects(
		t.ctx,
		t.wrapped,
		map[string][]byte{
			t.prefix + "burrito":   []byte(""),
			t.prefix + "enchilada": []byte(""),
			t.prefix + "taco":      []byte(""),
			"some_other":           []byte(""),
		})

	assert.Equal(t.T(), nil, err)

	// List.
	l, err := t.bucket.ListObjects(
		t.ctx,
		&gcs.ListObjectsRequest{})

	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), "", l.ContinuationToken)
	assert.Empty(t.T(), l.CollapsedRuns)

	assert.Equal(t.T(), 3, len(l.MinObjects))
	assert.Equal(t.T(), "burrito", l.MinObjects[0].Name)
	assert.Equal(t.T(), "enchilada", l.MinObjects[1].Name)
	assert.Equal(t.T(), "taco", l.MinObjects[2].Name)
}

func (t *PrefixBucketTest) Test_ListObjects_Prefix() {
	var err error

	// Create a few objects.
	err = storageutil.CreateObjects(
		t.ctx,
		t.wrapped,
		map[string][]byte{
			t.prefix + "burritn":  []byte(""),
			t.prefix + "burrito0": []byte(""),
			t.prefix + "burrito1": []byte(""),
			t.prefix + "burritp":  []byte(""),
			"some_other":          []byte(""),
		})

	assert.Equal(t.T(), nil, err)

	// List, with a prefix.
	l, err := t.bucket.ListObjects(
		t.ctx,
		&gcs.ListObjectsRequest{
			Prefix: "burrito",
		})

	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), "", l.ContinuationToken)
	assert.Empty(t.T(), l.CollapsedRuns)

	assert.Equal(t.T(), 2, len(l.MinObjects))
	assert.Equal(t.T(), "burrito0", l.MinObjects[0].Name)
	assert.Equal(t.T(), "burrito1", l.MinObjects[1].Name)
}

func (t *PrefixBucketTest) Test_ListObjects_Delimeter() {
	var err error

	// Create a few objects.
	err = storageutil.CreateObjects(
		t.ctx,
		t.wrapped,
		map[string][]byte{
			t.prefix + "burrito":     []byte(""),
			t.prefix + "burrito_0":   []byte(""),
			t.prefix + "burrito_1":   []byte(""),
			t.prefix + "enchilada_0": []byte(""),
			"some_other":             []byte(""),
		})

	assert.Equal(t.T(), nil, err)

	// List, with a delimiter. Make things extra interesting by using a delimiter
	// that is contained within the bucket prefix.
	assert.NotEqual(t.T(), -1, strings.IndexByte(t.prefix, '_'))
	l, err := t.bucket.ListObjects(
		t.ctx,
		&gcs.ListObjectsRequest{
			Delimiter: "_",
		})

	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), "", l.ContinuationToken)

	assert.ElementsMatch(t.T(), l.CollapsedRuns, []string{"burrito_", "enchilada_"})

	assert.Equal(t.T(), 1, len(l.MinObjects))
	assert.Equal(t.T(), "burrito", l.MinObjects[0].Name)
}

func (t *PrefixBucketTest) Test_ListObjects_PrefixAndDelimeter() {
	var err error

	// Create a few objects.
	err = storageutil.CreateObjects(
		t.ctx,
		t.wrapped,
		map[string][]byte{
			t.prefix + "burrito":     []byte(""),
			t.prefix + "burrito_0":   []byte(""),
			t.prefix + "burrito_1":   []byte(""),
			t.prefix + "enchilada_0": []byte(""),
			"some_other":             []byte(""),
		})

	assert.Equal(t.T(), nil, err)

	// List, with a delimiter and a prefix. Make things extra interesting by
	// using a delimiter that is contained within the bucket prefix.
	assert.NotEqual(t.T(), -1, strings.IndexByte(t.prefix, '_'))
	l, err := t.bucket.ListObjects(
		t.ctx,
		&gcs.ListObjectsRequest{
			Delimiter: "_",
			Prefix:    "burrito",
		})

	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), "", l.ContinuationToken)

	assert.ElementsMatch(t.T(), l.CollapsedRuns, []string{"burrito_"})

	assert.Equal(t.T(), 1, len(l.MinObjects))
	assert.Equal(t.T(), "burrito", l.MinObjects[0].Name)
}

func (t *PrefixBucketTest) Test_UpdateObject() {
	var err error
	suffix := "taco"
	name := t.prefix + suffix
	contents := "foobar"

	// Create an object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte(contents))
	assert.Equal(t.T(), nil, err)

	// Update it.
	newContentLanguage := "en-GB"
	o, err := t.bucket.UpdateObject(
		t.ctx,
		&gcs.UpdateObjectRequest{
			Name:            suffix,
			ContentLanguage: &newContentLanguage,
		})

	assert.Equal(t.T(), nil, err)
	assert.Equal(t.T(), suffix, o.Name)
	assert.Equal(t.T(), newContentLanguage, o.ContentLanguage)
}

func (t *PrefixBucketTest) Test_DeleteObject() {
	var err error
	suffix := "taco"
	name := t.prefix + suffix
	contents := "foobar"

	// Create an object through the back door.
	_, err = storageutil.CreateObject(t.ctx, t.wrapped, name, []byte(contents))
	assert.Equal(t.T(), nil, err)

	// Delete it.
	err = t.bucket.DeleteObject(
		t.ctx,
		&gcs.DeleteObjectRequest{
			Name: suffix,
		})

	assert.Equal(t.T(), nil, err)

	// It should be gone.
	_, _, err = t.wrapped.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{
			Name: name,
		})

	var notFoundErr *gcs.NotFoundError
	assert.True(t.T(), errors.As(err, &notFoundErr))
}

func TestGetFolder_Prefix(t *testing.T) {
	prefix := "foo_"
	wrapped := fake.NewFakeBucket(timeutil.RealClock(), "some_bucket", gcs.BucketType{})
	bucket, err := gcsx.NewPrefixBucket(prefix, wrapped)
	require.Nil(t, err)
	folderName := "taco"
	name := "foo_" + folderName
	ctx := context.Background()
	_, err = wrapped.CreateFolder(ctx, name)
	require.Nil(t, err)

	result, err := bucket.GetFolder(
		ctx,
		folderName)

	assert.Nil(nil, err)
	assert.Equal(t, folderName, result.Name)
}

func TestDeleteFolder(t *testing.T) {
	prefix := "foo_"
	wrapped := fake.NewFakeBucket(timeutil.RealClock(), "some_bucket", gcs.BucketType{})
	bucket, err := gcsx.NewPrefixBucket(prefix, wrapped)
	require.Nil(t, err)
	folderName := "taco"
	name := "foo_" + folderName

	ctx := context.Background()
	_, err = wrapped.CreateFolder(ctx, name)
	require.Nil(t, err)

	err = bucket.DeleteFolder(
		ctx,
		folderName)

	if assert.Nil(t, err) {
		_, err = wrapped.GetFolder(
			ctx,
			folderName)
		var notFoundErr *gcs.NotFoundError
		assert.ErrorAs(t, err, &notFoundErr)
	}
}

func TestRenameFolder(t *testing.T) {
	prefix := "foo_"
	var err error
	old_suffix := "test"
	name := prefix + old_suffix
	new_suffix := "new_test"
	wrapped := fake.NewFakeBucket(timeutil.RealClock(), "some_bucket", gcs.BucketType{})
	bucket, err := gcsx.NewPrefixBucket(prefix, wrapped)
	require.Nil(t, err)
	ctx := context.Background()
	_, err = wrapped.CreateFolder(ctx, name)
	assert.Nil(t, err)

	f, err := bucket.RenameFolder(ctx, old_suffix, new_suffix)
	assert.Nil(t, err)
	assert.Equal(t, new_suffix, f.Name)

	// New folder should get created
	_, err = bucket.GetFolder(ctx, new_suffix)
	assert.Nil(t, err)
	// Old folder should be gone.
	_, err = bucket.GetFolder(ctx, old_suffix)
	var notFoundErr *gcs.NotFoundError
	assert.True(t, errors.As(err, &notFoundErr))
}

func TestCreateFolder(t *testing.T) {
	prefix := "foo_"
	var err error
	suffix := "test"
	wrapped := fake.NewFakeBucket(timeutil.RealClock(), "some_bucket", gcs.BucketType{})
	bucket, err := gcsx.NewPrefixBucket(prefix, wrapped)
	require.NoError(t, err)
	ctx := context.Background()

	f, err := bucket.CreateFolder(ctx, suffix)

	assert.Equal(t, f.Name, suffix)
	assert.NoError(t, err)
	// Folder should get created
	_, err = bucket.GetFolder(ctx, suffix)
	assert.NoError(t, err)
}

func TestMoveObject(t *testing.T) {
	var notFoundErr *gcs.NotFoundError
	var err error
	prefix := "foo_"
	suffix := "test"
	wrapped := fake.NewFakeBucket(timeutil.RealClock(), "some_bucket", gcs.BucketType{Hierarchical: true})
	bucket, err := gcsx.NewPrefixBucket(prefix, wrapped)
	require.NoError(t, err)
	ctx := context.Background()
	contents := "foobar"
	name := prefix + suffix
	// Create an object through the back door.
	_, err = storageutil.CreateObject(ctx, wrapped, name, []byte(contents))
	assert.NoError(t, err)

	// Move it to a new name.
	newSuffix := "burrito"
	o, err := bucket.MoveObject(
		ctx,
		&gcs.MoveObjectRequest{
			SrcName: suffix,
			DstName: newSuffix,
		})

	assert.NoError(t, err)
	assert.Equal(t, newSuffix, o.Name)

	newName := prefix + newSuffix
	// Read it through the back door.
	actual, err := storageutil.ReadObject(ctx, wrapped, newName)
	assert.NoError(t, err)
	assert.Equal(t, contents, string(actual))

	// Stat old object.
	m, _, err := bucket.StatObject(
		ctx,
		&gcs.StatObjectRequest{
			Name: suffix,
		})

	assert.True(t, errors.As(err, &notFoundErr))
	assert.Nil(t, m)
}
