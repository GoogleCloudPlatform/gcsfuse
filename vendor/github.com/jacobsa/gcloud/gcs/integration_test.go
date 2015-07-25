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

// An integration test that uses the real GCS. Run it with appropriate flags as
// follows:
//
//     go test -v -tags integration . -bucket <bucket name>
//
// The bucket's contents are not preserved.
//
// The first time you run the test, it may die with a URL to visit to obtain an
// authorization code after authorizing the test to access your bucket. Run it
// again with the "-oauthutil.auth_code" flag afterward.

// Restrict this (slow) test to builds that specify the tag 'integration'.
// +build integration

package gcs_test

import (
	"flag"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcstesting"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
)

var fBucket = flag.String(
	"bucket", "",
	"Bucket to use for testing.")

var fUseRetry = flag.Bool(
	"use_retry",
	false,
	"Whether to use retry with exponential backoff.")

var fDebugGCS = flag.Bool(
	"debug_gcs", false,
	"Enable debug info about GCS requests and timing.")

var fDebugHTTP = flag.Bool(
	"debug_http", false,
	"Enable HTTP request/response debugging.")

////////////////////////////////////////////////////////////////////////
// Registration
////////////////////////////////////////////////////////////////////////

func TestOgletest(t *testing.T) { RunTests(t) }

func init() {
	makeDeps := func(ctx context.Context) (deps gcstesting.BucketTestDeps) {
		var err error

		// Set up the token source.
		const scope = gcs.Scope_FullControl
		tokenSrc, err := google.DefaultTokenSource(context.Background(), scope)
		AssertEq(nil, err)

		// Use that to create a GCS connection, enabling retry if requested.
		cfg := &gcs.ConnConfig{
			TokenSource: tokenSrc,
		}

		if *fUseRetry {
			cfg.MaxBackoffSleep = 5 * time.Minute
			deps.BuffersEntireContentsForCreate = true
		}

		if *fDebugGCS {
			cfg.GCSDebugLogger = log.New(os.Stderr, "gcs: ", 0)
		}

		if *fDebugHTTP {
			cfg.HTTPDebugLogger = log.New(os.Stderr, "http: ", 0)
		}

		conn, err := gcs.NewConn(cfg)
		AssertEq(nil, err)

		// Open the bucket.
		deps.Bucket, err = conn.OpenBucket(ctx, *fBucket)
		AssertEq(nil, err)

		// Clear the bucket.
		err = gcsutil.DeleteAllObjects(ctx, deps.Bucket)
		if err != nil {
			panic("DeleteAllObjects: " + err.Error())
		}

		// Set up other information.
		deps.Clock = timeutil.RealClock()
		deps.SupportsCancellation = true

		return
	}

	gcstesting.RegisterBucketTests(makeDeps)
}
