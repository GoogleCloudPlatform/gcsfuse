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
	"crypto/rand"
	"errors"
	"fmt"
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

func (oc *appendObjectCreator) chooseName() (name string, err error) {
	// Generate a good 64-bit random number.
	var buf [8]byte
	_, err = io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		err = fmt.Errorf("ReadFull: %v", err)
		return
	}

	x := uint64(buf[0])<<0 |
		uint64(buf[1])<<8 |
		uint64(buf[2])<<16 |
		uint64(buf[3])<<24 |
		uint64(buf[4])<<32 |
		uint64(buf[5])<<40 |
		uint64(buf[6])<<48 |
		uint64(buf[7])<<56

	// Turn it into a name.
	name = fmt.Sprintf("%s%016x", oc.prefix, x)

	return
}

func (oc *appendObjectCreator) Create(
	ctx context.Context,
	srcObject *gcs.Object,
	r io.Reader) (o *gcs.Object, err error) {
	// Choose a name for a temporary object.
	tmpName, err := oc.chooseName()
	if err != nil {
		err = fmt.Errorf("chooseName: %v", err)
		return
	}

	// Create a temporary object containing the additional contents.
	_, err = oc.bucket.CreateObject(
		ctx,
		&gcs.CreateObjectRequest{
			Name:     tmpName,
			Contents: r,
		})

	if err != nil {
		err = fmt.Errorf("CreateObject: %v", err)
		return
	}

	err = errors.New("TODO")
	return
}
