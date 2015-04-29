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

// An integration test that uses real GCS.

// Restrict this (slow) test to builds that specify the tag 'integration'.
// +build integration

package fs_test

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/fs/fstesting"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/gcloud/gcs/gcstesting"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	"github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
)

func TestIntegrationTest(t *testing.T) { ogletest.RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Registration
////////////////////////////////////////////////////////////////////////

func init() {
	fstesting.RegisterFSTests(
		"RealGCS",
		func() (cfg fstesting.FSTestConfig) {
			cfg.ServerConfig.Bucket = gcstesting.IntegrationTestBucketOrDie()
			cfg.ServerConfig.Clock = timeutil.RealClock()

			err := gcsutil.DeleteAllObjects(
				context.Background(),
				cfg.ServerConfig.Bucket)

			if err != nil {
				panic("DeleteAllObjects: " + err.Error())
			}

			return
		})
}
