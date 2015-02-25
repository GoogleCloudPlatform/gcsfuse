// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package gcsproxy_test

import (
	"errors"
	"sort"
	"testing"

	"github.com/jacobsa/gcloud/gcs/mock_gcs"
	"github.com/jacobsa/gcsfuse/gcsproxy"
	"github.com/jacobsa/gcsfuse/timeutil"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"
)

func TestListingProxy(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type ObjectSlice []*storage.Object

func (s ObjectSlice) Len() int           { return len(s) }
func (s ObjectSlice) Less(i, j int) bool { return s[i].Name < s[j].Name }
func (s ObjectSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func sortObjectsByName(s []*storage.Object) []*storage.Object {
	sortable := ObjectSlice(s)
	sort.Sort(sortable)
	return sortable
}

func sortStrings(s []string) []string {
	sortable := sort.StringSlice(s)
	sort.Sort(sortable)
	return sortable
}

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

func (t *ListingProxyTest) CreateForRootDirectory() {
	_, err := gcsproxy.NewListingProxy(t.bucket, &t.clock, "")
	AssertEq(nil, err)
}

func (t *ListingProxyTest) CreateForIllegalDirectoryName() {
	_, err := gcsproxy.NewListingProxy(t.bucket, &t.clock, "foo/bar")

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("foo/bar")))
	ExpectThat(err, Error(HasSubstr("directory name")))
}

func (t *ListingProxyTest) Name() {
	ExpectEq(t.dirName, t.lp.Name())
}

func (t *ListingProxyTest) List_CallsBucket() {
	// Bucket.ListObjects
	var query *storage.Query
	saveQuery := func(
		ctx context.Context,
		q *storage.Query) (*storage.Objects, error) {
		query = q
		return nil, errors.New("")
	}

	ExpectCall(t.bucket, "ListObjects")(Any(), Any()).
		WillOnce(oglemock.Invoke(saveQuery))

	// List
	t.lp.List()

	AssertNe(nil, query)
	ExpectEq("/", query.Delimiter)
	ExpectEq(t.dirName, query.Prefix)
	ExpectFalse(query.Versions)
	ExpectEq("", query.Cursor)
	ExpectEq(0, query.MaxResults)
}

func (t *ListingProxyTest) List_BucketFails() {
	// Bucket.ListObjects
	ExpectCall(t.bucket, "ListObjects")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// List
	_, _, err := t.lp.List()

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("List")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ListingProxyTest) List_BucketReturnsIllegalObjectName() {
	badObj := &storage.Object{
		Name: t.dirName + "foo/",
	}

	badListing := &storage.Objects{
		Results: []*storage.Object{badObj},
	}

	// Bucket.ListObjects
	ExpectCall(t.bucket, "ListObjects")(Any(), Any()).
		WillOnce(oglemock.Return(badListing, nil))

	// List
	_, _, err := t.lp.List()

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("object name")))
	ExpectThat(err, Error(HasSubstr(badObj.Name)))
}

func (t *ListingProxyTest) List_BucketReturnsIllegalDirectoryName() {
	badListing := &storage.Objects{
		Prefixes: []string{
			t.dirName + "foo/",
			t.dirName + "bar",
			t.dirName + "baz/",
		},
	}

	// Bucket.ListObjects
	ExpectCall(t.bucket, "ListObjects")(Any(), Any()).
		WillOnce(oglemock.Return(badListing, nil))

	// List
	_, _, err := t.lp.List()

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("directory name")))
	ExpectThat(err, Error(HasSubstr(badListing.Prefixes[1])))
}

func (t *ListingProxyTest) List_BucketReturnsNonDescendantObject() {
	badObj := &storage.Object{
		Name: "some/other/dir/obj",
	}

	badListing := &storage.Objects{
		Results: []*storage.Object{badObj},
	}

	// Bucket.ListObjects
	ExpectCall(t.bucket, "ListObjects")(Any(), Any()).
		WillOnce(oglemock.Return(badListing, nil))

	// List
	_, _, err := t.lp.List()

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("object")))
	ExpectThat(err, Error(HasSubstr(badObj.Name)))
	ExpectThat(err, Error(HasSubstr("descendant")))
	ExpectThat(err, Error(HasSubstr(t.dirName)))
}

func (t *ListingProxyTest) List_BucketReturnsNonDescendantPrefix() {
	AssertTrue(false, "TODO")
}

func (t *ListingProxyTest) List_EmptyResult() {
	// Bucket.ListObjects
	listing := &storage.Objects{}
	ExpectCall(t.bucket, "ListObjects")(Any(), Any()).
		WillOnce(oglemock.Return(listing, nil))

	// List
	objects, subdirs, err := t.lp.List()

	AssertEq(nil, err)
	ExpectThat(objects, ElementsAre())
	ExpectThat(subdirs, ElementsAre())
}

func (t *ListingProxyTest) List_OnlyPlaceholderForProxiedDir() {
	// Bucket.ListObjects
	listing := &storage.Objects{
		Results: []*storage.Object{
			&storage.Object{Name: t.dirName},
		},
	}

	ExpectCall(t.bucket, "ListObjects")(Any(), Any()).
		WillOnce(oglemock.Return(listing, nil))

	// List
	objects, subdirs, err := t.lp.List()

	AssertEq(nil, err)
	ExpectThat(objects, ElementsAre())
	ExpectThat(subdirs, ElementsAre())
}

func (t *ListingProxyTest) List_NonEmptyResult_PlaceholderForProxiedDirPresent() {
	// Bucket.ListObjects
	listing := &storage.Objects{
		Results: []*storage.Object{
			&storage.Object{Name: t.dirName},
			&storage.Object{Name: t.dirName + "bar"},
			&storage.Object{Name: t.dirName + "foo"},
		},
		Prefixes: []string{
			t.dirName + "baz/",
			t.dirName + "qux/",
		},
	}

	ExpectCall(t.bucket, "ListObjects")(Any(), Any()).
		WillOnce(oglemock.Return(listing, nil))

	// List
	objects, subdirs, err := t.lp.List()

	objects = sortObjectsByName(objects)
	subdirs = sortStrings(subdirs)

	AssertEq(nil, err)
	ExpectThat(objects, ElementsAre(listing.Results[1], listing.Results[2]))
	ExpectThat(subdirs, ElementsAre(listing.Prefixes[0], listing.Prefixes[1]))
}

func (t *ListingProxyTest) List_NonEmptyResult_PlaceholderForProxiedDirNotPresent() {
	// Bucket.ListObjects
	listing := &storage.Objects{
		Results: []*storage.Object{
			&storage.Object{Name: t.dirName + "bar"},
			&storage.Object{Name: t.dirName + "foo"},
		},
		Prefixes: []string{
			t.dirName + "baz/",
			t.dirName + "qux/",
		},
	}

	ExpectCall(t.bucket, "ListObjects")(Any(), Any()).
		WillOnce(oglemock.Return(listing, nil))

	// List
	objects, subdirs, err := t.lp.List()

	objects = sortObjectsByName(objects)
	subdirs = sortStrings(subdirs)

	AssertEq(nil, err)
	ExpectThat(objects, ElementsAre(listing.Results[0], listing.Results[1]))
	ExpectThat(subdirs, ElementsAre(listing.Prefixes[0], listing.Prefixes[1]))
}

func (t *ListingProxyTest) List_CacheIsValid() {
	AssertTrue(false, "TODO")
}

func (t *ListingProxyTest) List_CacheHasExpired() {
	AssertTrue(false, "TODO")
}

func (t *ListingProxyTest) NoteNewObject_IllegalNames() {
	AssertTrue(false, "TODO")
}

func (t *ListingProxyTest) NoteNewObject_NoPreviousListing() {
	AssertTrue(false, "TODO")
}

func (t *ListingProxyTest) NoteNewObject_PrevListingConflicts() {
	AssertTrue(false, "TODO")
}

func (t *ListingProxyTest) NoteNewObject_PrevListingDoesntConflict() {
	AssertTrue(false, "TODO")
}

func (t *ListingProxyTest) NoteNewObject_PreviousAddition() {
	AssertTrue(false, "TODO")
}

func (t *ListingProxyTest) NoteNewObject_PreviousRemoval() {
	AssertTrue(false, "TODO")
}

func (t *ListingProxyTest) NoteNewSubdirectory_IllegalNames() {
	AssertTrue(false, "TODO")
}

func (t *ListingProxyTest) NoteNewSubdirectory_NoPreviousListing() {
	AssertTrue(false, "TODO")
}

func (t *ListingProxyTest) NoteNewSubdirectory_PrevListingConflicts() {
	AssertTrue(false, "TODO")
}

func (t *ListingProxyTest) NoteNewSubdirectory_PrevListingDoesntConflict() {
	AssertTrue(false, "TODO")
}

func (t *ListingProxyTest) NoteNewSubdirectory_PreviousAddition() {
	AssertTrue(false, "TODO")
}

func (t *ListingProxyTest) NoteNewSubdirectory_PreviousRemoval() {
	AssertTrue(false, "TODO")
}

func (t *ListingProxyTest) NoteRemoval_NoPreviousListing() {
	AssertTrue(false, "TODO")
}

func (t *ListingProxyTest) NoteRemoval_PrevListingConflicts() {
	AssertTrue(false, "TODO")
}

func (t *ListingProxyTest) NoteRemoval_PrevListingDoesntConflict() {
	AssertTrue(false, "TODO")
}

func (t *ListingProxyTest) NoteRemoval_PreviousAddition() {
	AssertTrue(false, "TODO")
}

func (t *ListingProxyTest) NoteRemoval_PreviousRemoval() {
	AssertTrue(false, "TODO")
}
