// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package gcsproxy

import (
	"github.com/jacobsa/gcloud/gcs"
	"google.golang.org/cloud/storage"
)

// XXX: Comments
// XXX: Thread safety
type ProxyObject struct {
}

func NewProxyObject(
	bucket gcs.Bucket,
	name string) (*ProxyObject, error)

func (po *ProxyObject) Size() uint64

func (po *ProxyObject) ReadAt(buf []byte, offset int64) (int, error)

func (po *ProxyObject) WriterAt(buf []byte, offset int64) (int, error)

func (po *ProxyObject) Truncate(n uint64) error

func (po *ProxyObject) Sync() (storage.Object, error)
