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
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/lease"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestFileLeaser(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func panicIf(err *error) {
	if *err != nil {
		panic(*err)
	}
}

// Create a read/write lease and fill it in with data of the specified length.
// Panic on failure.
func newFileOfLength(
	fl *lease.FileLeaser,
	length int) (rwl lease.ReadWriteLease) {
	var err error
	defer panicIf(&err)

	// Create the lease.
	rwl, err = fl.NewFile()
	if err != nil {
		err = fmt.Errorf("NewFile: %v", err)
		return
	}

	// Write the contents.
	_, err = rwl.Write(bytes.Repeat([]byte("a"), length))
	if err != nil {
		err = fmt.Errorf("Write: %v", err)
		return
	}

	return
}

// Downgrade the supplied lease or panic.
func downgrade(rwl lease.ReadWriteLease) (rl lease.ReadLease) {
	var err error
	defer panicIf(&err)

	// Attempt to downgrade.
	rl, err = rwl.Downgrade()

	return
}

// Check whether the lease has been revoked. Note the inherent race here.
func isRevoked(rl lease.ReadLease) (revoked bool) {
	var err error
	defer panicIf(&err)

	_, err = rl.ReadAt([]byte{}, 0)
	if _, ok := err.(*lease.RevokedError); ok {
		err = nil
		revoked = true
	}

	return
}

func touchWithRead(r io.Reader) {
	panic("TODO")
}

func touchWithReadAt(r io.ReaderAt) {
	panic("TODO")
}

func growBy(w io.WriteSeeker, n int) {
	panic("TODO")
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const limitBytes = 17

type FileLeaserTest struct {
	fl *lease.FileLeaser
}

var _ SetUpInterface = &FileLeaserTest{}

func init() { RegisterTestSuite(&FileLeaserTest{}) }

func (t *FileLeaserTest) SetUp(ti *TestInfo) {
	t.fl = lease.NewFileLeaser("", limitBytes)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *FileLeaserTest) ReadWriteLeaseInitialState() {
	var n int
	var off int64
	var err error
	buf := make([]byte, 1024)

	// Create
	rwl, err := t.fl.NewFile()
	AssertEq(nil, err)

	// Size
	size, err := rwl.Size()
	AssertEq(nil, err)
	ExpectEq(0, size)

	// Seek
	off, err = rwl.Seek(0, 2)
	AssertEq(nil, err)
	ExpectEq(0, off)

	// Read
	n, err = rwl.Read(buf)
	ExpectEq(io.EOF, err)
	ExpectEq(0, n)

	// ReadAt
	n, err = rwl.ReadAt(buf, 0)
	ExpectEq(io.EOF, err)
	ExpectEq(0, n)
}

func (t *FileLeaserTest) ModifyThenObserveReadWriteLease() {
	var n int
	var off int64
	var size int64
	var err error
	buf := make([]byte, 1024)

	// Create
	rwl, err := t.fl.NewFile()
	AssertEq(nil, err)

	// Write, then check size and offset.
	n, err = rwl.Write([]byte("tacoburrito"))
	AssertEq(nil, err)
	ExpectEq(len("tacoburrito"), n)

	size, err = rwl.Size()
	AssertEq(nil, err)
	ExpectEq(len("tacoburrito"), size)

	off, err = rwl.Seek(0, 1)
	AssertEq(nil, err)
	ExpectEq(len("tacoburrito"), off)

	// Pwrite, then check size.
	n, err = rwl.WriteAt([]byte("enchilada"), 4)
	AssertEq(nil, err)
	ExpectEq(len("enchilada"), n)

	size, err = rwl.Size()
	AssertEq(nil, err)
	ExpectEq(len("tacoenchilada"), size)

	// Truncate downward, then check size.
	err = rwl.Truncate(4)
	AssertEq(nil, err)

	size, err = rwl.Size()
	AssertEq(nil, err)
	ExpectEq(len("taco"), size)

	// Seek, then read everything.
	off, err = rwl.Seek(0, 0)
	AssertEq(nil, err)
	ExpectEq(0, off)

	n, err = rwl.Read(buf)
	ExpectThat(err, AnyOf(nil, io.EOF))
	ExpectEq("taco", string(buf[0:n]))
}

func (t *FileLeaserTest) DowngradeThenObserve() {
	var n int
	var off int64
	var size int64
	var err error
	buf := make([]byte, 1024)

	// Create and write some data.
	rwl, err := t.fl.NewFile()
	AssertEq(nil, err)

	n, err = rwl.Write([]byte("taco"))
	AssertEq(nil, err)

	// Downgrade.
	rl, err := rwl.Downgrade()
	AssertEq(nil, err)

	// Interacting with the read/write lease should no longer work.
	_, err = rwl.Read(buf)
	ExpectThat(err, HasSameTypeAs(&lease.RevokedError{}))

	_, err = rwl.Write(buf)
	ExpectThat(err, HasSameTypeAs(&lease.RevokedError{}))

	_, err = rwl.Seek(0, 0)
	ExpectThat(err, HasSameTypeAs(&lease.RevokedError{}))

	_, err = rwl.ReadAt(buf, 0)
	ExpectThat(err, HasSameTypeAs(&lease.RevokedError{}))

	_, err = rwl.WriteAt(buf, 0)
	ExpectThat(err, HasSameTypeAs(&lease.RevokedError{}))

	err = rwl.Truncate(0)
	ExpectThat(err, HasSameTypeAs(&lease.RevokedError{}))

	_, err = rwl.Size()
	ExpectThat(err, HasSameTypeAs(&lease.RevokedError{}))

	_, err = rwl.Downgrade()
	ExpectThat(err, HasSameTypeAs(&lease.RevokedError{}))

	// Observing via the read lease should work fine.
	size = rl.Size()
	ExpectEq(len("taco"), size)

	off, err = rl.Seek(-4, 2)
	AssertEq(nil, err)
	ExpectEq(0, off)

	n, err = rl.Read(buf)
	ExpectThat(err, AnyOf(nil, io.EOF))
	ExpectEq("taco", string(buf[0:n]))

	n, err = rl.ReadAt(buf[0:2], 1)
	AssertEq(nil, err)
	ExpectEq("ac", string(buf[0:2]))
}

func (t *FileLeaserTest) DowngradeThenUpgradeThenObserve() {
	var n int
	var off int64
	var size int64
	var err error
	buf := make([]byte, 1024)

	// Create and write some data.
	rwl, err := t.fl.NewFile()
	AssertEq(nil, err)

	n, err = rwl.Write([]byte("taco"))
	AssertEq(nil, err)

	// Downgrade.
	rl, err := rwl.Downgrade()
	AssertEq(nil, err)

	// Upgrade again.
	rwl = rl.Upgrade()
	AssertNe(nil, rwl)

	// Interacting with the read lease should no longer work.
	_, err = rl.Read(buf)
	ExpectThat(err, HasSameTypeAs(&lease.RevokedError{}))

	_, err = rl.Seek(0, 0)
	ExpectThat(err, HasSameTypeAs(&lease.RevokedError{}))

	_, err = rl.ReadAt(buf, 0)
	ExpectThat(err, HasSameTypeAs(&lease.RevokedError{}))

	tmp := rl.Upgrade()
	ExpectEq(nil, tmp)

	// Calling Revoke should cause nothing nasty to happen.
	rl.Revoke()

	// Observing via the new read/write lease should work fine.
	size, err = rwl.Size()
	AssertEq(nil, err)
	ExpectEq(len("taco"), size)

	off, err = rwl.Seek(-4, 2)
	AssertEq(nil, err)
	ExpectEq(0, off)

	n, err = rwl.Read(buf)
	ExpectThat(err, AnyOf(nil, io.EOF))
	ExpectEq("taco", string(buf[0:n]))

	n, err = rwl.ReadAt(buf[0:2], 1)
	AssertEq(nil, err)
	ExpectEq("ac", string(buf[0:2]))
}

func (t *FileLeaserTest) DowngradeFileWhoseSizeIsAboveLimit() {
	var err error
	buf := make([]byte, 1024)

	// Create and write data larger than the capacity.
	rwl, err := t.fl.NewFile()
	AssertEq(nil, err)

	_, err = rwl.Write(bytes.Repeat([]byte("a"), limitBytes+1))
	AssertEq(nil, err)

	// Downgrade.
	rl, err := rwl.Downgrade()
	AssertEq(nil, err)

	// The read lease should be revoked on arrival.
	_, err = rl.Read(buf)
	ExpectThat(err, HasSameTypeAs(&lease.RevokedError{}))

	_, err = rl.Seek(0, 0)
	ExpectThat(err, HasSameTypeAs(&lease.RevokedError{}))

	_, err = rl.ReadAt(buf, 0)
	ExpectThat(err, HasSameTypeAs(&lease.RevokedError{}))

	tmp := rl.Upgrade()
	ExpectEq(nil, tmp)
}

func (t *FileLeaserTest) WriteCausesEviction() {
	var err error

	// Set up a read lease whose size is right at the limit.
	rl := downgrade(newFileOfLength(t.fl, limitBytes))
	AssertFalse(isRevoked(rl))

	// Set up a new read/write lease. The read lease should still be unrevoked.
	rwl, err := t.fl.NewFile()
	AssertEq(nil, err)

	AssertFalse(isRevoked(rl))

	// Writing zero bytes shouldn't cause trouble.
	_, err = rwl.Write([]byte(""))
	AssertEq(nil, err)

	AssertFalse(isRevoked(rl))

	// But the next byte should.
	_, err = rwl.Write([]byte("a"))
	AssertEq(nil, err)

	ExpectTrue(isRevoked(rl))
}

func (t *FileLeaserTest) WriteAtCausesEviction() {
	var err error
	AssertLt(3, limitBytes)

	// Set up a read lease whose size is three bytes below the limit.
	rl := downgrade(newFileOfLength(t.fl, limitBytes-3))
	AssertFalse(isRevoked(rl))

	// Set up a new read/write lease. The read lease should still be unrevoked.
	rwl, err := t.fl.NewFile()
	AssertEq(nil, err)

	AssertFalse(isRevoked(rl))

	// Write in three bytes. Everything should be fine.
	_, err = rwl.Write([]byte("foo"))
	AssertEq(nil, err)

	// Overwriting a byte shouldn't cause trouble.
	_, err = rwl.WriteAt([]byte("p"), 0)
	AssertEq(nil, err)

	AssertFalse(isRevoked(rl))

	// But extending the file by one byte should.
	_, err = rwl.WriteAt([]byte("taco"), 0)
	AssertEq(nil, err)

	ExpectTrue(isRevoked(rl))
}

func (t *FileLeaserTest) TruncateCausesEviction() {
	var err error
	AssertLt(3, limitBytes)

	// Set up a read lease whose size is three bytes below the limit.
	rl := downgrade(newFileOfLength(t.fl, limitBytes-3))
	AssertFalse(isRevoked(rl))

	// Set up a new read/write lease. The read lease should still be unrevoked.
	rwl, err := t.fl.NewFile()
	AssertEq(nil, err)

	AssertFalse(isRevoked(rl))

	// Truncate up to the limit. Nothing should happen.
	err = rwl.Truncate(3)
	AssertEq(nil, err)

	AssertFalse(isRevoked(rl))

	// Truncate downward. Again, nothing should happen.
	err = rwl.Truncate(2)
	AssertEq(nil, err)

	AssertFalse(isRevoked(rl))

	// But extending to four bytes should cause revocation.
	err = rwl.Truncate(4)
	AssertEq(nil, err)

	ExpectTrue(isRevoked(rl))
}

func (t *FileLeaserTest) EvictionIsLRU() {
	AssertLt(4, limitBytes)

	// Arrange for four read leases, with a known order of recency of usage.
	rl0 := downgrade(newFileOfLength(t.fl, 1))
	rl2 := downgrade(newFileOfLength(t.fl, 1))
	rl3 := downgrade(newFileOfLength(t.fl, 1))

	touchWithRead(rl0)                         // Least recent
	rl1 := downgrade(newFileOfLength(t.fl, 1)) // Second least recent
	touchWithRead(rl2)                         // Third least recent
	touchWithReadAt(rl3)                       // Fourth least recent

	// Fill up the remaining space. All read leases should still be valid.
	rwl := newFileOfLength(t.fl, limitBytes-4)

	panic("TODO: isRevoked should not change recency of use")

	AssertFalse(isRevoked(rl0))
	AssertFalse(isRevoked(rl1))
	AssertFalse(isRevoked(rl2))
	AssertFalse(isRevoked(rl3))

	// Use up one more byte. The least recently used lease should be revoked.
	growBy(rwl, 1)

	AssertTrue(isRevoked(rl0))
	AssertFalse(isRevoked(rl1))
	AssertFalse(isRevoked(rl2))
	AssertFalse(isRevoked(rl3))

	// Two more bytes. Now the next two should go.
	growBy(rwl, 2)

	AssertTrue(isRevoked(rl0))
	AssertTrue(isRevoked(rl1))
	AssertTrue(isRevoked(rl2))
	AssertFalse(isRevoked(rl3))

	// Downgrading and upgrading the read/write lease should change nothing.
	rwl = downgrade(rwl).Upgrade()
	AssertNe(nil, rwl)

	AssertTrue(isRevoked(rl0))
	AssertTrue(isRevoked(rl1))
	AssertTrue(isRevoked(rl2))
	AssertFalse(isRevoked(rl3))

	// But writing one more byte should boot the last one.
	growBy(rwl, 1)

	AssertTrue(isRevoked(rl0))
	AssertTrue(isRevoked(rl1))
	AssertTrue(isRevoked(rl2))
	AssertTrue(isRevoked(rl3))
}

func (t *FileLeaserTest) NothingAvailableToEvict() {
	AssertFalse(true, "TODO")
}

func (t *FileLeaserTest) RevokeVoluntarily() {
	// TODO(jacobsa): Test that methods return RevokedError and that capacity in
	// the leaser is freed up.
	AssertFalse(true, "TODO")
}
