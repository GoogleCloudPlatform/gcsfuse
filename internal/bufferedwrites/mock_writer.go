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

package bufferedwrites

import (
	"bytes"
	"io"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

// MockWriter implements io.WriteCloser and is used in unit tests to mock
// the behavior of a GCS object writer. This is particular used with
// storage.TestifyMockBucket implementation and allows for controlled testing of
// interactions with the writer without relying on actual GCS operations.
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
