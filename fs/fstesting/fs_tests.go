// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)
//
// Tests registered by RegisterFSTests.

package fstesting

import (
	"github.com/jacobsa/gcloud/gcs"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
)

////////////////////////////////////////////////////////////////////////
// Common
////////////////////////////////////////////////////////////////////////

type fsTest struct {
	ctx    context.Context
	bucket gcs.Bucket
}

var _ fsTestInterface = &fsTest{}

func (t *fsTest) setUpFsTest(b gcs.Bucket) {
	t.bucket = b
	t.ctx = context.Background()
}

func (t *fsTest) tearDownFsTest() {
}

////////////////////////////////////////////////////////////////////////
// Read-only interaction
////////////////////////////////////////////////////////////////////////

type readOnlyTest struct {
	fsTest
}

func (t *readOnlyTest) DoesFoo() {
	AssertTrue(false, "TODO")
}
