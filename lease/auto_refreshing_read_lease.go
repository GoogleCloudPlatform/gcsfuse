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
	"fmt"
	"io"
	"log"
	"sync"
)

// Create a ReadLease that never expires, unless voluntarily revoked or
// upgraded.
//
// The supplied function will be used to obtain the read lease contents, the
// first time and whenever the supplied file leaser decides to expire the
// temporary copy thus obtained. It must return the same contents every time,
// and the contents must be of the given size.
//
// This magic is not preserved after the lease is upgraded.
func NewAutoRefreshingReadLease(
	fl FileLeaser,
	size int64,
	f func() (io.ReadCloser, error)) (rl ReadLease) {
	rl = &autoRefreshingReadLease{
		leaser: fl,
		size:   size,
		f:      f,
	}

	return
}

type autoRefreshingReadLease struct {
	mu sync.Mutex

	/////////////////////////
	// Constant data
	/////////////////////////

	size int64

	/////////////////////////
	// Dependencies
	/////////////////////////

	leaser FileLeaser
	f      func() (io.ReadCloser, error)

	/////////////////////////
	// Mutable state
	/////////////////////////

	// The current wrapped lease, or nil if one has never been issued.
	//
	// GUARDED_BY(mu)
	wrapped ReadLease
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// Attempt to clean up after the supplied read/write lease.
func destroyReadWriteLease(rwl ReadWriteLease) {
	var err error
	defer func() {
		if err != nil {
			log.Printf("Error destroying read/write lease: %v", err)
		}
	}()

	// Downgrade to a read lease.
	rl, err := rwl.Downgrade()
	if err != nil {
		err = fmt.Errorf("Downgrade: %v", err)
		return
	}

	// Revoke the read lease.
	rl.Revoke()
}

// Set up a read/write lease and fill in our contents.
//
// REQUIRES: The caller has observed that rl.lease has expired.
//
// LOCKS_REQUIRED(rl.mu)
func (rl *autoRefreshingReadLease) getContents() (
	rwl ReadWriteLease, err error) {
	// Obtain some space to write the contents.
	rwl, err = rl.leaser.NewFile()
	if err != nil {
		err = fmt.Errorf("NewFile: %v", err)
		return
	}

	// Attempt to clean up if we exit early.
	defer func() {
		if err != nil {
			destroyReadWriteLease(rwl)
		}
	}()

	// Obtain the reader for our contents.
	rc, err := rl.f()
	if err != nil {
		err = fmt.Errorf("User function: %v", err)
		return
	}

	defer func() {
		closeErr := rc.Close()
		if closeErr != nil && err == nil {
			err = fmt.Errorf("Close: %v", closeErr)
		}
	}()

	// Copy into the read/write lease.
	copied, err := io.Copy(rwl, rc)
	if err != nil {
		err = fmt.Errorf("Copy: %v", err)
		return
	}

	// Did the user lie about the size?
	if copied != rl.Size() {
		err = fmt.Errorf("Copied %v bytes; expected %v", copied, rl.Size())
		return
	}

	return
}

// Downgrade and save the supplied read/write lease obtained with getContents
// for later use.
//
// LOCKS_REQUIRED(rl.mu)
func (rl *autoRefreshingReadLease) saveContents(rwl ReadWriteLease) {
	downgraded, err := rwl.Downgrade()
	if err != nil {
		log.Printf("Failed to downgrade write lease (%q); abandoning.", err.Error())
		return
	}

	rl.wrapped = downgraded
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

func (rl *autoRefreshingReadLease) Read(p []byte) (n int, err error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Common case: is the existing lease still valid?
	if rl.wrapped != nil {
		panic("TODO")
	}

	// Get hold of a read/write lease containing our contents.
	rwl, err := rl.getContents()
	if err != nil {
		err = fmt.Errorf("getContents: %v", err)
		return
	}

	defer rl.saveContents(rwl)

	// Serve from the read/write lease.
	n, err = rwl.Read(p)

	return
}

func (rl *autoRefreshingReadLease) Seek(
	offset int64,
	whence int) (off int64, err error) {
	panic("TODO")
}

func (rl *autoRefreshingReadLease) ReadAt(
	p []byte,
	off int64) (n int, err error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Common case: is the existing lease still valid?
	if rl.wrapped != nil {
		panic("TODO")
	}

	// Get hold of a read/write lease containing our contents.
	rwl, err := rl.getContents()
	if err != nil {
		err = fmt.Errorf("getContents: %v", err)
		return
	}

	defer rl.saveContents(rwl)

	// Serve from the read/write lease.
	n, err = rwl.ReadAt(p, off)

	return
}

func (rl *autoRefreshingReadLease) Size() (size int64) {
	size = rl.size
	return
}

func (rl *autoRefreshingReadLease) Revoked() (revoked bool) {
	panic("TODO")
}

func (rl *autoRefreshingReadLease) Upgrade() (rwl ReadWriteLease, err error) {
	panic("TODO")
}

func (rl *autoRefreshingReadLease) Revoke() {
	panic("TODO")
}
