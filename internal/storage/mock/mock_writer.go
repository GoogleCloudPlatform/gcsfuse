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

package mock

import (
	"io"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/mock"
)

// Writer implements io.WriteCloser and is used in unit tests to mock
// the behavior of a GCS object writer. This is particular used with
// storage.TestifyMockBucket implementation and allows for controlled testing of
// interactions with the writer without relying on actual GCS operations.
type Writer struct {
	io.WriteCloser
	storage.ObjectAttrs
	mock.Mock
}

func (mw *Writer) Write(p []byte) (n int, err error) {
	args := mw.Called(p)
	return args.Int(0), args.Error(1)
}
func (mw *Writer) Attrs() *storage.ObjectAttrs {
	args := mw.Called()
	return args.Get(0).(*storage.ObjectAttrs)
}

func (mw *Writer) Close() error {
	args := mw.Called()
	return args.Error(0)
}

func (mw *Writer) ObjectName() string {
	args := mw.Called()
	return args.String(0)
}
