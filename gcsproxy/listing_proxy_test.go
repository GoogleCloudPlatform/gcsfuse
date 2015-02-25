// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package gcsproxy_test

import (
	"testing"

	"github.com/jacobsa/gcloud/gcs/mock_gcs"
	"github.com/jacobsa/gcsfuse/gcsproxy"
	"github.com/jacobsa/gcsfuse/timeutil"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"
)

func TestListingProxy(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Invariant-checking listing proxy
////////////////////////////////////////////////////////////////////////

// A wrapper around ListingProxy that calls CheckInvariants whenever invariants
// should hold. For catching logic errors early in the test.
type checkingListingProxy struct {
	wrapped *gcsproxy.ListingProxy
}

func (lp *checkingListingProxy) Name() string {
	lp.wrapped.CheckInvariants()
	defer lp.wrapped.CheckInvariants()
	return lp.wrapped.Name()
}

func (lp *checkingListingProxy) List() ([]*storage.Object, []string, error) {
	lp.wrapped.CheckInvariants()
	defer lp.wrapped.CheckInvariants()
	return lp.wrapped.List(context.Background())
}

func (lp *checkingListingProxy) NoteNewObject(o *storage.Object) error {
	lp.wrapped.CheckInvariants()
	defer lp.wrapped.CheckInvariants()
	return lp.wrapped.NoteNewObject(o)
}

func (lp *checkingListingProxy) NoteNewSubdirectory(name string) error {
	lp.wrapped.CheckInvariants()
	defer lp.wrapped.CheckInvariants()
	return lp.wrapped.NoteNewSubdirectory(name)
}

func (lp *checkingListingProxy) NoteRemoval(name string) error {
	lp.wrapped.CheckInvariants()
	defer lp.wrapped.CheckInvariants()
	return lp.wrapped.NoteRemoval(name)
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ListingProxyTest struct {
	dirName string
	bucket  mock_gcs.MockBucket
	clock   timeutil.SimulatedClock
	lp      checkingListingProxy
}

var _ SetUpInterface = &ListingProxyTest{}

func init() { RegisterTestSuite(&ListingProxyTest{}) }

func (t *ListingProxyTest) SetUp(ti *TestInfo) {
	t.dirName = "some/dir/"
	t.bucket = mock_gcs.NewMockBucket(ti.MockController, "bucket")

	var err error
	t.lp.wrapped, err = gcsproxy.NewListingProxy(t.bucket, &t.clock, t.dirName)
	if err != nil {
		panic(err)
	}
}

////////////////////////////////////////////////////////////////////////
// Test functions
////////////////////////////////////////////////////////////////////////

func (t *ListingProxyTest) DoesFoo() {
	AssertTrue(false, "TODO")
}
