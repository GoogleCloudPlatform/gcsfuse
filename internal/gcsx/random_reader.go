// Copyright 2015 Google Inc. All Rights Reserved.
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

package gcsx

import (
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

// An object that knows how to read ranges within a particular generation of a
// particular GCS object. May make optimizations when it e.g. detects large
// sequential reads.
//
// Not safe for concurrent access.
type RandomReader interface {
	// Matches the semantics of io.ReaderAt, with the addition of context
	// support.
	ReadAt(ctx context.Context, p []byte, offset int64) (n int, err error)

	// Return the record for the object to which the reader is bound.
	Object() (o *gcs.Object)

	// Clean up any resources associated with the reader, which must not be used
	// again.
	Destroy()
}
