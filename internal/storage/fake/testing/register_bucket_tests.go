// Copyright 2023 Google LLC
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

package testing

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"golang.org/x/net/context"

	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/suite"
)

// Dependencies needed for tests registered by RunBucketTests.
type BucketTestDeps struct {
	// A context that should be used for all blocking operations.
	ctx context.Context

	// An initialized, empty bucket.
	Bucket gcs.Bucket

	// A clock matching the bucket's notion of time.
	Clock timeutil.Clock

	// Does the bucket support cancellation?
	SupportsCancellation bool

	// Does the bucket buffer all contents before creating in GCS?
	BuffersEntireContentsForCreate bool
}

func RunBucketTests(t *testing.T, makeDeps func(context.Context) BucketTestDeps) {
	suite.Run(t, &createTest{bucketTest: bucketTest{MakeDeps: makeDeps}})
	suite.Run(t, &copyTest{bucketTest: bucketTest{MakeDeps: makeDeps}})
	suite.Run(t, &composeTest{bucketTest: bucketTest{MakeDeps: makeDeps}})
	suite.Run(t, &readTest{bucketTest: bucketTest{MakeDeps: makeDeps}})
	suite.Run(t, &readMultiRangeTest{bucketTest: bucketTest{MakeDeps: makeDeps}})
	suite.Run(t, &statTest{bucketTest: bucketTest{MakeDeps: makeDeps}})
	suite.Run(t, &updateTest{bucketTest: bucketTest{MakeDeps: makeDeps}})
	suite.Run(t, &deleteTest{bucketTest: bucketTest{MakeDeps: makeDeps}})
	suite.Run(t, &listTest{bucketTest: bucketTest{MakeDeps: makeDeps}})
	suite.Run(t, &cancellationTest{bucketTest: bucketTest{MakeDeps: makeDeps}})
}
