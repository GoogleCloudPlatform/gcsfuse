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

// FakeObjectWriter is a mock implementation of storage.Writer
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
