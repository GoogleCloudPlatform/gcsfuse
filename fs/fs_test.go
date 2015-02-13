// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)
//
// A large test that uses a fake GCS bucket.

package fs_test

import (
	"testing"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	"github.com/jacobsa/gcsfuse/fs/fstesting"
	"github.com/jacobsa/ogletest"
)

func TestOgletest(t *testing.T) { ogletest.RunTests(t) }

func init() {
	fstesting.RegisterFSTests(
		"FakeGCS",
		func() gcs.Bucket {
			return gcsfake.NewFakeBucket("some_bucket")
		})
}
