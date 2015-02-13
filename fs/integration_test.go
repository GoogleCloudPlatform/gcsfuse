// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)
//
// An integration test that uses real GCS.

// Restrict this (slow) test to builds that specify the tag 'integration'.
// +build integration

package fs_test

import (
	"flag"
	"log"
	"net/http"
	"testing"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	"github.com/jacobsa/gcloud/oauthutil"
	"github.com/jacobsa/gcsfuse/fs/fstesting"
	"github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	storagev1 "google.golang.org/api/storage/v1"
)

func TestIntegrationTest(t *testing.T) { ogletest.RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Wiring code
////////////////////////////////////////////////////////////////////////

var fBucket = flag.String("bucket", "", "Empty bucket to use for storage.")

func getHttpClientOrDie() *http.Client {
	// Set up a token source.
	config := &oauth2.Config{
		ClientID:     "501259388845-j47fftkfn6lhp4o80ajg38cs8jed2dmj.apps.googleusercontent.com",
		ClientSecret: "-z3_0mx4feP2mqOGhRIEk_DN",
		RedirectURL:  "urn:ietf:wg:oauth:2.0:oob",
		Scopes:       []string{storagev1.DevstorageRead_writeScope},
		Endpoint:     google.Endpoint,
	}

	const cacheFileName = ".gcsfuse_integration_test.token_cache.json"
	httpClient, err := oauthutil.NewTerribleHttpClient(config, cacheFileName)
	if err != nil {
		panic("NewTerribleHttpClient: " + err.Error())
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
		func() gcs.Bucket {
			bucket := getBucketOrDie()

			if err := gcsutil.DeleteAllObjects(context.Background(), bucket); err != nil {
				panic("DeleteAllObjects: " + err.Error())
			}

			return bucket
		})
}
