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

	"golang.org/x/net/context"
)

// A function used by read proxies to refresh their contents. See notes on
// NewReadProxy.
type RefreshContentsFunc func(context.Context) (io.ReadCloser, error)

// Create a read proxy.
//
// The supplied function will be used to obtain the proxy's contents whenever
// the supplied file leaser decides to expire the temporary copy thus obtained.
// It must return the same contents every time, and the contents must be of the
// given size.
//
// If rl is non-nil, it will be used as the first temporary copy of the
// contents, and must match what refresh returns.
func NewReadProxy(
	fl FileLeaser,
	size int64,
	refresh RefreshContentsFunc,
	rl ReadLease) (rp *ReadProxy) {
	rp = &ReadProxy{
		leaser:  fl,
		size:    size,
		refresh: refresh,
		lease:   rl,
	}

	return
}

// A wrapper around a read lease, exposing a similar interface with the
// following differences:
//
//  *  Contents are fetched and re-fetched automatically when needed. Therefore
//     the user need not worry about lease expiration.
//
//  *  Methods that may involve fetching the contents (reading, seeking) accept
//     context arguments, so as to be cancellable.
//
// External synchronization is required.
type ReadProxy struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	size int64

	/////////////////////////
	// Dependencies
	/////////////////////////

	leaser  FileLeaser
	refresh RefreshContentsFunc

	/////////////////////////
	// Mutable state
	/////////////////////////

	// The current wrapped lease, or nil if one has never been issued.
	lease ReadLease
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

func isRevokedErr(err error) bool {
	_, ok := err.(*RevokedError)
	return ok
}

// Set up a read/write lease and fill in our contents.
//
// REQUIRES: The caller has observed that rp.lease has expired.
func (rp *ReadProxy) getContents(
	ctx context.Context) (rwl ReadWriteLease, err error) {
	// Obtain some space to write the contents.
	rwl, err = rp.leaser.NewFile()
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
	rc, err := rp.refresh(ctx)
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
	if copied != rp.Size() {
		err = fmt.Errorf("Copied %v bytes; expected %v", copied, rp.Size())
		return
	}

	return
}

// Downgrade and save the supplied read/write lease obtained with getContents
// for later use.
func (rp *ReadProxy) saveContents(rwl ReadWriteLease) {
	downgraded, err := rwl.Downgrade()
	if err != nil {
		log.Printf("Failed to downgrade write lease (%q); abandoning.", err.Error())
		return
	}

	rp.lease = downgraded
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

// Semantics matching io.Reader, except with context support.
func (rp *ReadProxy) Read(
	ctx context.Context,
	p []byte) (n int, err error) {
	// Common case: is the existing lease still valid?
	if rp.lease != nil {
		n, err = rp.lease.Read(p)
		if !isRevokedErr(err) {
			return
		}

		// Clear the revoked error.
		err = nil
	}

	// Get hold of a read/write lease containing our contents.
	rwl, err := rp.getContents(ctx)
	if err != nil {
		err = fmt.Errorf("getContents: %v", err)
		return
	}

	defer rp.saveContents(rwl)

	// Serve from the read/write lease.
	n, err = rwl.Read(p)

	return
}

// Semantics matching io.Seeker, except with context support.
func (rp *ReadProxy) Seek(
	ctx context.Context,
	offset int64,
	whence int) (off int64, err error) {
	// Common case: is the existing lease still valid?
	if rp.lease != nil {
		off, err = rp.lease.Seek(offset, whence)
		if !isRevokedErr(err) {
			return
		}

		// Clear the revoked error.
		err = nil
	}

	// Get hold of a read/write lease containing our contents.
	rwl, err := rp.getContents(ctx)
	if err != nil {
		err = fmt.Errorf("getContents: %v", err)
		return
	}

	defer rp.saveContents(rwl)

	// Serve from the read/write lease.
	off, err = rwl.Seek(offset, whence)

	return
}

// Semantics matching io.ReaderAt, except with context support.
func (rp *ReadProxy) ReadAt(
	ctx context.Context,
	p []byte,
	off int64) (n int, err error) {
	// Common case: is the existing lease still valid?
	if rp.lease != nil {
		n, err = rp.lease.ReadAt(p, off)
		if !isRevokedErr(err) {
			return
		}

		// Clear the revoked error.
		err = nil
	}

	// Get hold of a read/write lease containing our contents.
	rwl, err := rp.getContents(ctx)
	if err != nil {
		err = fmt.Errorf("getContents: %v", err)
		return
	}

	defer rp.saveContents(rwl)

	// Serve from the read/write lease.
	n, err = rwl.ReadAt(p, off)

	return
}

// Return the size of the proxied content. Guarantees to not block.
func (rp *ReadProxy) Size() (size int64) {
	size = rp.size
	return
}

// Return a read/write lease for the proxied contents. The read proxy must not
// be used after calling this method.
func (rp *ReadProxy) Upgrade(
	ctx context.Context) (rwl ReadWriteLease, err error) {
	// If we succeed, we are now destroyed.
	defer func() {
		if err == nil {
			rp.Destroy()
		}
	}()

	// Common case: is the existing lease still valid?
	if rp.lease != nil {
		rwl, err = rp.lease.Upgrade()
		if !isRevokedErr(err) {
			return
		}

		// Clear the revoked error.
		err = nil
	}

	// Build the read/write lease anew.
	rwl, err = rp.getContents(ctx)
	if err != nil {
		err = fmt.Errorf("getContents: %v", err)
		return
	}

	return
}

// Destroy any resources in use by the read proxy. It must not be used further.
func (rp *ReadProxy) Destroy() {
	if rp.lease != nil {
		rp.lease.Revoke()
	}

	// Make use-after-destroy errors obvious.
	rp.size = 0
	rp.leaser = nil
	rp.refresh = nil
	rp.lease = nil
}
