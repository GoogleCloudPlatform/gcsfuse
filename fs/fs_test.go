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

// A large test that uses a fake GCS bucket.

package fs_test

import (
	"testing"
	"time"

	"github.com/jacobsa/gcloud/gcs/gcsfake"
	"github.com/googlecloudplatform/gcsfuse/fs/fstesting"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/ogletest"
)

func TestOgletest(t *testing.T) { ogletest.RunTests(t) }

func init() {
	fstesting.RegisterFSTests(
		"FakeGCS",
		func() (deps fstesting.FSTestDeps) {
			// Set up a fixed, non-zero time.
			clock := &timeutil.SimulatedClock{}
			clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))
			deps.Clock = clock

			// Set up the bucket.
			deps.Bucket = gcsfake.NewFakeBucket(clock, "some_bucket")

			return
		})
}
