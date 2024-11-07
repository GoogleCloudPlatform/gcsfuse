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

package bufferedwrites

import (
	"bytes"
	"io"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

type MockWriter struct {
	io.WriteCloser
	buf bytes.Buffer
	storage.ObjectAttrs
}

func (w *MockWriter) Write(p []byte) (n int, err error) {
	return w.buf.Write(p)
}

func (w *MockWriter) Close() error {
	return nil
}

func (w *MockWriter) ObjectName() string {
	return w.Name
}
func (w *MockWriter) Attrs() *storage.ObjectAttrs {
	return &w.ObjectAttrs
}

func NewMockWriter(objName string) gcs.Writer {
	wr := &MockWriter{
		buf: bytes.Buffer{},
		ObjectAttrs: storage.ObjectAttrs{
			Name: objName,
		},
	}

	return wr
}
