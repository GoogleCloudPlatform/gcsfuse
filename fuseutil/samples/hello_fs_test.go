// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package samples_test

import (
	"testing"

	. "github.com/jacobsa/ogletest"
)

func TestHelloFS(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type HelloFSTest struct {
}

var _ SetUpInterface = &HelloFSTest{}

func init() { RegisterTestSuite(&HelloFSTest{}) }

func (t *HelloFSTest) SetUp(ti *TestInfo)

////////////////////////////////////////////////////////////////////////
// Test functions
////////////////////////////////////////////////////////////////////////

func (t *HelloFSTest) DoesFoo() {
	AssertTrue(false, "TODO")
}
