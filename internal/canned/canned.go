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

// Helper code for creating buckets with canned contents, for testing.
//
// The details of this package are subject to change.
package canned

import (
	"log"
	"strings"

	"golang.org/x/net/context"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	"github.com/jacobsa/timeutil"
)

// The name of a fake bucket supported by gcsfuse. This is intentionally an
// illegal name (cf. https://cloud.google.com/storage/docs/bucket-naming).
//
// The initial contents of the bucket are objects with names given by the
// following constants:
//
//     TopLevelFile
//     TopLevelDir
//     ImplicitDirFile
//
const FakeBucketName = "fake@bucket"

// See notes on FakeBucketName.
const (
	TopLevelFile          = "foo"
	TopLevelFile_Contents = "taco"

	TopLevelDir          = "bar/"
	TopLevelDir_Contents = ""

	ExplicitDirFile          = "bar/f"
	ExplicitDirFile_Contents = "burrito"

	ImplicitDirFile          = "baz/qux"
	ImplicitDirFile_Contents = "enchilada"
)

// Create a fake bucket with canned contents as described in the comments for
// FakeBucketName.
func MakeFakeBucket(ctx context.Context) (b gcs.Bucket) {
	b = gcsfake.NewFakeBucket(timeutil.RealClock(), FakeBucketName)

	// Set up contents.
	contents := map[string]string{
		TopLevelFile:    TopLevelFile_Contents,
		TopLevelDir:     TopLevelDir_Contents,
		ExplicitDirFile: ExplicitDirFile_Contents,
		ImplicitDirFile: ImplicitDirFile_Contents,
	}

	for k, v := range contents {
		_, err := b.CreateObject(
			ctx,
			&gcs.CreateObjectRequest{
				Name:     k,
				Contents: strings.NewReader(v),
			})

		if err != nil {
			log.Panicf("CreateObject: %v", err)
		}
	}

	return
}
