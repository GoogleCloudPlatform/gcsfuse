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

package lease_test

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"

	. "github.com/jacobsa/ogletest"
)

func TestAutoRefreshingReadLease(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

const contents = "taco"

// A function that always successfully returns our contents constant.
func returnContents() (rc io.ReadCloser, err error) {
	rc = ioutil.NopCloser(strings.NewReader(contents))
	return
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type AutoRefreshingReadLeaseTest struct {
}

var _ SetUpInterface = &AutoRefreshingReadLeaseTest{}

func init() { RegisterTestSuite(&AutoRefreshingReadLeaseTest{}) }

func (t *AutoRefreshingReadLeaseTest) SetUp(ti *TestInfo) {
	// TODO
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *AutoRefreshingReadLeaseTest) Size() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) LeaserReturnsError() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) CallsFunc() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) FuncReturnsError() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) ContentsReturnReadError() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) ContentsReturnCloseError() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) ContentsAreWrongLength() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) DowngradesAfterRead() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) DowngradesAfterReadAt() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) DowngradesAfterSeek() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) Upgrade_Error() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) Upgrade_Success() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) Upgrade_Failure() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) SecondRead_StillValid() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) SecondRead_Revoked_ErrorReading() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) SecondRead_Revoked_Successful() {
	AssertTrue(false, "TODO")
}

func (t *AutoRefreshingReadLeaseTest) Revoke() {
	AssertTrue(false, "TODO")
}
