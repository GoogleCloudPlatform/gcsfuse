// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package gcsproxy_test

import (
	"testing"

	"github.com/jacobsa/gcsfuse/gcsproxy"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"
)

func TestOgletest(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Invariant-checking object proxy
////////////////////////////////////////////////////////////////////////

// A wrapper around ObjectProxy that calls CheckInvariants whenever invariants
// should hold. For catching logic errors early in the test.
type checkingObjectProxy struct {
	wrapped *gcsproxy.ObjectProxy
}

func (op *checkingObjectProxy) NoteLatest(o *storage.Object) (err error)
func (op *checkingObjectProxy) Size() (n uint64, err error)
func (op *checkingObjectProxy) ReadAt(b []byte, o int64) (int, error)
func (op *checkingObjectProxy) WriteAt(b []byte, o int64) (int, error)
func (op *checkingObjectProxy) Truncate(n uint64) error
func (op *checkingObjectProxy) Sync(ctx context.Context) (*storage.Object, error)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ObjectProxyTest struct {
	op *checkingObjectProxy
}

var _ SetUpInterface = &ObjectProxyTest{}

func init() { RegisterTestSuite(&ObjectProxyTest{}) }

func (t *ObjectProxyTest) SetUp(ti *TestInfo)

////////////////////////////////////////////////////////////////////////
// Test functions
////////////////////////////////////////////////////////////////////////

func (t *ObjectProxyTest) DoesFoo() {
	AssertTrue(false, "TODO")
}
