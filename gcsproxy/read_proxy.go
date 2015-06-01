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
	"fmt"
	"io"

	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

func makeRefreshers(
	chunkSize uint64,
	o *gcs.Object,
	bucket gcs.Bucket) (refreshers []lease.Refresher) {
	// Iterate over each chunk of the object.
	for startOff := uint64(0); startOff < o.Size; startOff += chunkSize {
		r := gcs.ByteRange{startOff, startOff + chunkSize}

		// Clip the range so that objectRefresher can report the correct size.
		if r.Limit > o.Size {
			r.Limit = o.Size
		}

		refresher := &objectRefresher{
			O:      o,
			Bucket: bucket,
			Range:  &r,
		}

		refreshers = append(refreshers, refresher)
	}

	return
}

// Create a view on the given GCS object generation. If rl is non-nil, it must
// contain a lease for the contents of the object and will be used when
// possible instead of re-reading the object.
//
// If the object is larger than the given chunk size, we will only read
// and cache portions of it at a time.
func NewReadProxy(
	o *gcs.Object,
	rl lease.ReadLease,
	chunkSize uint64,
	leaser lease.FileLeaser,
	bucket gcs.Bucket) (rp lease.ReadProxy) {
	// Sanity check: the read lease's size should match the object's size if it
	// is present.
	if rl != nil && uint64(rl.Size()) != o.Size {
		panic(fmt.Sprintf(
			"Read lease size %d doesn't match object size %d",
			rl.Size(),
			o.Size))
	}

	// Special case: don't bring in the complication of a multi-read proxy if we
	// have only one refresher.
	refreshers := makeRefreshers(chunkSize, o, bucket)
	if len(refreshers) == 1 {
		rp = lease.NewReadProxy(leaser, refreshers[0], rl)
	} else {
		rp = lease.NewMultiReadProxy(leaser, refreshers, rl)
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// A refresher that returns the contents of a particular generation of a GCS
// object. Optionally, only a particular range is returned.
type objectRefresher struct {
	Bucket gcs.Bucket
	O      *gcs.Object
	Range  *gcs.ByteRange
}

func (r *objectRefresher) Size() (size int64) {
	if r.Range != nil {
		size = int64(r.Range.Limit - r.Range.Start)
		return
	}

	size = int64(r.O.Size)
	return
}

func (r *objectRefresher) Refresh(
	ctx context.Context) (rc io.ReadCloser, err error) {
	req := &gcs.ReadObjectRequest{
		Name:       r.O.Name,
		Generation: r.O.Generation,
		Range:      r.Range,
	}

	rc, err = r.Bucket.NewReader(ctx, req)
	if err != nil {
		err = fmt.Errorf("NewReader: %v", err)
		return
	}

	return
}
