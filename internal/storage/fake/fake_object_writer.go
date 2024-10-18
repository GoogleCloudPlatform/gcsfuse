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

package fake

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

// FakeObjectWriter is a mock implementation of storage.Writer used by FakeBucket.
type FakeObjectWriter struct {
	io.WriteCloser
	buf bytes.Buffer
	storage.ObjectAttrs
	ChunkSize    int
	ProgressFunc func(_ int64)
	bkt          *bucket
	req          *gcs.CreateObjectRequest
	Object       *gcs.Object // Object created by writer
}

func (w *FakeObjectWriter) Write(p []byte) (n int, err error) {
	return w.buf.Write(p)
}

func (w *FakeObjectWriter) Close() error {
	// Validate for precondition: DoesNotExist.
	// Find any existing record for this name.
	existingIndex := w.bkt.objects.find(w.req.Name)
	if existingIndex < len(w.bkt.objects) {
		err := &gcs.PreconditionError{
			Err: errors.New("precondition failed: object exists"),
		}
		return err
	}

	// Create an object record from the given attributes.
	var fo fakeObject = w.bkt.mintObject(w.req, w.buf.Bytes())
	fo.data = w.buf.Bytes()
	w.Object = copyObject(&fo.metadata)

	// Add an entry to our list of objects.
	w.bkt.objects = append(w.bkt.objects, fo)
	sort.Sort(w.bkt.objects)

	if w.bkt.BucketType() == gcs.Hierarchical {
		w.bkt.addFolderEntry(w.req.Name)
	}
	return nil
}

func (w *FakeObjectWriter) ObjectName() string {
	return w.Name
}
func (w *FakeObjectWriter) Attrs() *storage.ObjectAttrs {
	return &w.ObjectAttrs
}

func NewFakeObjectWriter(b *bucket, req *gcs.CreateObjectRequest, chunkSize int, callback func(int64)) (w gcs.Writer, err error) {
	// Check that the name is legal.
	err = checkName(req.Name)
	if err != nil {
		return
	}

	// Check preconditions.
	if req.GenerationPrecondition != nil && *req.GenerationPrecondition != 0 {
		return nil, fmt.Errorf("storage.Writer can only be created for new objects")
	}

	wr := &FakeObjectWriter{
		buf:          bytes.Buffer{},
		bkt:          b,
		req:          req,
		ChunkSize:    chunkSize,
		ProgressFunc: callback,
		ObjectAttrs: storage.ObjectAttrs{
			Name: req.Name,
		},
	}
	wr.ContentType = req.ContentType

	return wr, nil
}
