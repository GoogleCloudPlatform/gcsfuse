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

package gcsx_test

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

func TestPrefixBucket(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type PrefixBucketTest struct {
	ctx     context.Context
	prefix  string
	wrapped gcs.Bucket
	bucket  gcs.Bucket
}

var _ SetUpInterface = &PrefixBucketTest{}

func init() { RegisterTestSuite(&PrefixBucketTest{}) }

func (t *PrefixBucketTest) SetUp(ti *TestInfo) {
	var err error

	t.ctx = ti.Ctx
	t.prefix = "foo_"
	t.wrapped = gcsfake.NewFakeBucket(timeutil.RealClock(), "some_bucket")

	t.bucket, err = gcsx.NewPrefixBucket(t.prefix, t.wrapped)
	AssertEq(nil, err)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *PrefixBucketTest) DoesFoo() {
	AddFailure("TODO")
}
