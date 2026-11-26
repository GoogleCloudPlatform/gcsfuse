// Copyright 2026 Google LLC
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

package fake

import (
	"context"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v4/internal/storage/gcs"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeObjectWriter_OperationsAfterAbortFail(t *testing.T) {
	testCases := []struct {
		name string
		op   func(bucket gcs.Bucket, w gcs.Writer) error
	}{
		{
			name: "Write",
			op: func(_ gcs.Bucket, w gcs.Writer) error {
				_, err := w.Write([]byte("baz"))
				return err
			},
		},
		{
			name: "FinalizeUpload",
			op: func(bucket gcs.Bucket, w gcs.Writer) error {
				_, err := bucket.FinalizeUpload(context.Background(), w)
				return err
			},
		},
		{
			name: "FlushPendingWrites",
			op: func(bucket gcs.Bucket, w gcs.Writer) error {
				_, err := bucket.FlushPendingWrites(context.Background(), w)
				return err
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bucket := NewFakeBucket(timeutil.RealClock(), "some_bucket", gcs.BucketType{})
			w, err := bucket.CreateObjectChunkWriter(context.Background(), &gcs.CreateObjectRequest{Name: "foo"}, nil)
			require.NoError(t, err)
			_, err = w.Write([]byte("bar"))
			require.NoError(t, err)
			require.NoError(t, bucket.AbortUpload(context.Background(), w))

			err = tc.op(bucket, w)

			assert.ErrorIs(t, err, errWriterAborted)
			_, _, err = bucket.StatObject(context.Background(), &gcs.StatObjectRequest{Name: "foo"})
			var notFoundErr *gcs.NotFoundError
			assert.ErrorAs(t, err, &notFoundErr)
		})
	}
}
