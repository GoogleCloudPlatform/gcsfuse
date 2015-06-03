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

package gcsproxy

import (
	"errors"
	"io"

	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

// Create an objectCreator that accepts a source object and the contents that
// should be "appended" to it, storing temporary objects using the supplied
// prefix.
//
// Note that the Create method will attempt to remove any temporary junk left
// behind, but it may fail to do so. Users should arrange for garbage collection.
func newAppendObjectCreator(
	prefix string,
	bucket gcs.Bucket) (oc objectCreator) {
	oc = &appendObjectCreator{
		prefix: prefix,
		bucket: bucket,
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type appendObjectCreator struct {
	prefix string
	bucket gcs.Bucket
}

func (oc *appendObjectCreator) Create(
	ctx context.Context,
	srcObject *gcs.Object,
	r io.Reader) (o *gcs.Object, err error) {
	err = errors.New("TODO")
	return
}
