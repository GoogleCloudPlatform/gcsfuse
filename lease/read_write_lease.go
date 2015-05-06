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

package lease

import (
	"errors"
	"io"
)

// A read-write wrapper around a file. Unlike a read lease, this cannot be
// revoked.
//
// All methods are safe for concurrent access.
type ReadWriteLease interface {
	// Methods with semantics matching *os.File.
	io.ReadWriteSeeker
	io.ReaderAt
	io.WriterAt
	Truncate(size int64) (err error)

	// Return the current size of the underlying file.
	Size() (size int64, err error)

	// Downgrade to a read lease, releasing any resources pinned by this lease to
	// the pool that may be revoked, as with any read lease. After successfully
	// downgrading, this lease must not be used again.
	Downgrade() (rl ReadLease, err error)
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type readWriteLease struct {
}

var _ ReadWriteLease = &readWriteLease{}

func newReadWriteLease() (rwl *readWriteLease) {
	// TODO
	rwl = &readWriteLease{}
	return
}

func (rwl *readWriteLease) Read(p []byte) (n int, err error) {
	err = errors.New("TODO")
	return
}

func (rwl *readWriteLease) Write(p []byte) (n int, err error) {
	err = errors.New("TODO")
	return
}

func (rwl *readWriteLease) Seek(
	offset int64,
	whence int) (off int64, err error) {
	err = errors.New("TODO")
	return
}

func (rwl *readWriteLease) ReadAt(p []byte, off int64) (n int, err error) {
	err = errors.New("TODO")
	return
}

func (rwl *readWriteLease) WriteAt(p []byte, off int64) (n int, err error) {
	err = errors.New("TODO")
	return
}

func (rwl *readWriteLease) Truncate(size int64) (err error) {
	err = errors.New("TODO")
	return
}

func (rwl *readWriteLease) Size() (size int64, err error) {
	err = errors.New("TODO")
	return
}

func (rwl *readWriteLease) Downgrade() (rl ReadLease, err error) {
	err = errors.New("TODO")
	return
}
