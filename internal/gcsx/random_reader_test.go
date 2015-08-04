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

package gcsx

import (
	"testing"

	"github.com/jacobsa/gcloud/gcs/mock_gcs"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
)

func TestRandomReader(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Invariant-checking random reader
////////////////////////////////////////////////////////////////////////

type checkingRandomReader struct {
	ctx     context.Context
	wrapped RandomReader
}

func (rr *checkingRandomReader) ReadAt(p []byte, offset int64) (int, error) {
	rr.wrapped.CheckInvariants()
	defer rr.wrapped.CheckInvariants()
	return rr.wrapped.ReadAt(rr.ctx, p, offset)
}

func (rr *checkingRandomReader) Destroy() {
	rr.wrapped.CheckInvariants()
	rr.wrapped.Destroy()
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type RandomReaderTest struct {
	bucket mock_gcs.MockBucket
	rr     checkingRandomReader
}

func init() { RegisterTestSuite(&RandomReaderTest{}) }

var _ SetUpInterface = &RandomReaderTest{}
var _ TearDownInterface = &RandomReaderTest{}

func (t *RandomReaderTest) SetUp(ti *TestInfo) {
	panic("TODO")
}

func (t *RandomReaderTest) TearDown() {
	t.rr.Destroy()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *RandomReaderTest) EmptyRead() {
	AssertTrue(false, "TODO")
}

func (t *RandomReaderTest) NoExistingReader() {
	AssertTrue(false, "TODO")
}

func (t *RandomReaderTest) ExistingReader_WrongOffset() {
	AssertTrue(false, "TODO")
}

func (t *RandomReaderTest) NewReaderReturnsError() {
	AssertTrue(false, "TODO")
}

func (t *RandomReaderTest) ReaderFails() {
	AssertTrue(false, "TODO")
}

func (t *RandomReaderTest) ReaderOvershootsRange() {
	AssertTrue(false, "TODO")
}

func (t *RandomReaderTest) ReaderExhausted_ReadFinished() {
	AssertTrue(false, "TODO")
}

func (t *RandomReaderTest) ReaderExhausted_ReadNotFinished() {
	AssertTrue(false, "TODO")
}

func (t *RandomReaderTest) PropagatesCancellation() {
	AssertTrue(false, "TODO")
}

func (t *RandomReaderTest) DoesntPropagateCancellationAfterReturning() {
	AssertTrue(false, "TODO")
}

func (t *RandomReaderTest) UpgradesReadsToMinimumSize() {
	AssertTrue(false, "TODO")
}

func (t *RandomReaderTest) UpgradesSequentialReads() {
	AssertTrue(false, "TODO")
}
