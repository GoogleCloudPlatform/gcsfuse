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

package storage

import (
	"cloud.google.com/go/storage"
)

// ObjectWriter is a wrapper over storage.Writer and implements
// gcs.Writer interface (used by bucket.go).
// It is used to write content to GCS object via resumable upload API.
type ObjectWriter struct {
	*storage.Writer
}

func (e *ObjectWriter) ObjectName() string {
	return e.Writer.Name
}

func (e *ObjectWriter) Attrs() *storage.ObjectAttrs {
	return e.Writer.Attrs()
}
