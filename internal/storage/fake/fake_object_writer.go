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

// For now, we are not writing the unit test, which requires multiple
// version of same object. As this is not supported by fake-storage-server.
// Although API is exposed to enable the object versioning for a bucket,
// but it returns "method not allowed" when we call it.

package fake

import (
	"bytes"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
)

// FakeObjectWriter is a mock implementation of storage.Writer used by FakeBucket.
type FakeObjectWriter struct {
	io.WriteCloser
	buf bytes.Buffer
	storage.ObjectAttrs
	bkt    *bucket
	req    *gcs.CreateObjectRequest
	Object *gcs.MinObject // Object created by writer
}

func (w *FakeObjectWriter) Write(p []byte) (n int, err error) {
	contents := w.buf.Bytes()
	// Validate for preconditions.
	if err := preconditionChecks(w.bkt, w.req, contents); err != nil {
		return 0, err
	}
	return w.buf.Write(p)
}

func (w *FakeObjectWriter) Close() error {
	contents := w.buf.Bytes()

	// Validate for preconditions.
	if err := preconditionChecks(w.bkt, w.req, contents); err != nil {
		return err
	}

	o, err := createOrUpdateFakeObject(true, w.bkt, w.req, contents)
	if err == nil {
		w.Object = storageutil.ConvertObjToMinObject(o)
	}

	return err
}

func (w *FakeObjectWriter) Flush() (int64, error) {
	if !w.bkt.BucketType().Zonal {
		return 0, fmt.Errorf("flush is not supported on non zonal buckets")
	}
	contents := w.buf.Bytes()
	// Validate for preconditions.
	if err := preconditionChecks(w.bkt, w.req, contents); err != nil {
		return 0, err
	}
	o, err := createOrUpdateFakeObject(false, w.bkt, w.req, contents)
	if err != nil {
		return 0, err
	}
	w.Object = storageutil.ConvertObjToMinObject(o)
	return int64(w.buf.Len()), nil
}

func (w *FakeObjectWriter) ObjectName() string {
	return w.Name
}
func (w *FakeObjectWriter) Attrs() *storage.ObjectAttrs {
	return &w.ObjectAttrs
}

func NewFakeObjectWriter(b *bucket, req *gcs.CreateObjectRequest) (w gcs.Writer, err error) {
	// Check that the name is legal.
	err = checkName(req.Name)
	if err != nil {
		return
	}

	wr := &FakeObjectWriter{
		buf: bytes.Buffer{},
		bkt: b,
		req: req,
		ObjectAttrs: storage.ObjectAttrs{
			Name: req.Name,
		},
	}
	wr.ContentType = req.ContentType

	return wr, nil
}
