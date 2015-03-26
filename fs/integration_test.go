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
	"flag"
	"log"
	"net/http"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/fs/fstesting"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	"github.com/jacobsa/gcloud/oauthutil"
	"github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
	storagev1 "google.golang.org/api/storage/v1"
)

func TestIntegrationTest(t *testing.T) { ogletest.RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Wiring code
////////////////////////////////////////////////////////////////////////

var fKeyFile = flag.String("key_file", "", "Path to a JSON key for a service account created on the Google Developers Console.")
var fBucket = flag.String("bucket", "", "Empty bucket to use for storage.")

func getHttpClientOrDie() *http.Client {
	if *fKeyFile == "" {
		panic("You must set --key_file.")
	}

	const scope = storagev1.DevstorageRead_writeScope
	httpClient, err := oauthutil.NewJWTHttpClient(*fKeyFile, []string{scope})
	if err != nil {
		panic("oauthutil.NewJWTHttpClient: " + err.Error())
	}

	return httpClient
}

func getBucketNameOrDie() string {
	s := *fBucket
	if s == "" {
		log.Fatalln("You must set --bucket.")
	}

	return s
}

// Return a bucket based on the contents of command-line flags, exiting the
// process if misconfigured.
func getBucketOrDie() gcs.Bucket {
	// A project ID is apparently only needed for creating and listing buckets,
	// presumably since a bucket ID already maps to a unique project ID (cf.
	// http://goo.gl/Plh3rb). This doesn't currently matter to us.
	const projectId = "some_project_id"

	// Set up a GCS connection.
	conn, err := gcs.NewConn(projectId, getHttpClientOrDie())
	if err != nil {
		log.Fatalf("gcs.NewConn: %v", err)
	}

	// Open the bucket.
	return conn.GetBucket(getBucketNameOrDie())
}

////////////////////////////////////////////////////////////////////////
// Registration
////////////////////////////////////////////////////////////////////////

func init() {
	fstesting.RegisterFSTests(
		"RealGCS",
		func() (cfg fstesting.FSTestConfig) {
			cfg.ServerConfig.Bucket = getBucketOrDie()
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
