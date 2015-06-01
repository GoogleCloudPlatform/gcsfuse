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

package gcsproxy

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"golang.org/x/net/context"
)

// A mutable view on some content. Created with an initial read-only view,
// which then can be modified by the user and read back. Keeps track of which
// portion of the content has been dirtied.
//
// External synchronization is required.
type MutableContent struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	clock timeutil.Clock

	/////////////////////////
	// Mutable state
	/////////////////////////

	destroyed bool

	// The initial contents with which this object was created, or nil if it has
	// been dirtied.
	//
	// INVARIANT: When non-nil, initialContent.CheckInvariants() does not panic.
	initialContent lease.ReadProxy

	// When dirty, a read/write lease containing our current contents. When
	// clean, nil.
	//
	// INVARIANT: (initialContent == nil) != (readWriteLease == nil)
	readWriteLease lease.ReadWriteLease

	// The lowest byte index that has been modified from the initial contents.
	//
	// INVARIANT: initialContent != nil => dirtyThreshold == initialContent.Size()
	dirtyThreshold int64

	// The time at which a method that modifies our contents was last called, or
	// nil if never.
	//
	// INVARIANT: If dirty(), then mtime != nil
	mtime *time.Time
}

type StatResult struct {
	// The current size in bytes of the content.
	Size int64

	// It is guaranteed that all bytes in the range [0, DirtyThreshold) are
	// unmodified from the original content with which the mutable content object
	// was created.
	DirtyThreshold int64

	// The time at which the content was last updated, or nil if we've never
	// changed it.
	Mtime *time.Time
}

// Create a mutable content object whose initial contents are given by the
// supplied read proxy.
func NewMutableContent(
	initialContent lease.ReadProxy,
	clock timeutil.Clock) (mc *MutableContent) {
	mc = &MutableContent{
		clock:          clock,
		initialContent: initialContent,
		dirtyThreshold: initialContent.Size(),
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

// Panic if any internal invariants are violated.
func (mc *MutableContent) CheckInvariants() {
	if mc.destroyed {
		panic("Use of destroyed MutableContent object.")
	}

	// INVARIANT: When non-nil, initialContent.CheckInvariants() does not panic.
	if mc.initialContent != nil {
		mc.initialContent.CheckInvariants()
	}

	// INVARIANT: (initialContent == nil) != (readWriteLease == nil)
	if mc.initialContent == nil && mc.readWriteLease == nil {
		panic("Both initialContent and readWriteLease are nil")
	}

	if mc.initialContent != nil && mc.readWriteLease != nil {
		panic("Both initialContent and readWriteLease are non-nil")
	}

	// INVARIANT: If dirty(), then mtime != nil
	if mc.dirty() && mc.mtime == nil {
		panic("Expected non-nil mtime.")
	}

	// INVARIANT: initialContent != nil => dirtyThreshold == initialContent.Size()
	if mc.initialContent != nil {
		if mc.dirtyThreshold != mc.initialContent.Size() {
			panic(fmt.Sprintf(
				"Dirty threshold mismatch: %d vs. %d",
				mc.dirtyThreshold,
				mc.initialContent.Size()))
		}
	}
}

// Destroy any state used by the object, putting it into an indeterminate
// state. The object must not be used again.
func (mc *MutableContent) Destroy() {
	mc.destroyed = true

	if mc.initialContent != nil {
		mc.initialContent.Destroy()
		mc.initialContent = nil
	}

	if mc.readWriteLease != nil {
		mc.readWriteLease.Downgrade().Revoke()
		mc.readWriteLease = nil
	}
}

// Read part of the content, with semantics equivalent to io.ReaderAt aside
// from context support.
func (mc *MutableContent) ReadAt(
	ctx context.Context,
	buf []byte,
	offset int64) (n int, err error) {
	// Serve from the appropriate place.
	if mc.dirty() {
		n, err = mc.readWriteLease.ReadAt(buf, offset)
	} else {
		n, err = mc.initialContent.ReadAt(ctx, buf, offset)
	}

	return
}

// Return information about the current state of the content.
func (mc *MutableContent) Stat(
	ctx context.Context) (sr StatResult, err error) {
	err = errors.New(
		"TODO: Make sure Stat tests are up to date with new interface.")
	return
}

// Write into the content, with semantics equivalent to io.WriterAt aside from
// context support.
func (mc *MutableContent) WriteAt(
	ctx context.Context,
	buf []byte,
	offset int64) (n int, err error) {
	// Make sure we have a read/write lease.
	if err = mc.ensureReadWriteLease(ctx); err != nil {
		err = fmt.Errorf("ensureReadWriteLease: %v", err)
		return
	}

	// Update our state regarding being dirty.
	mc.dirtyThreshold = minInt64(mc.dirtyThreshold, offset)

	newMtime := mc.clock.Now()
	mc.mtime = &newMtime

	// Call through.
	n, err = mc.readWriteLease.WriteAt(buf, offset)

	return
}

// Truncate our the content to the given number of bytes, extending if n is
// greater than the current size.
func (mc *MutableContent) Truncate(
	ctx context.Context,
	n int64) (err error) {
	// Make sure we have a read/write lease.
	if err = mc.ensureReadWriteLease(ctx); err != nil {
		err = fmt.Errorf("ensureReadWriteLease: %v", err)
		return
	}

	// Convert to signed, which is what lease.ReadWriteLease wants.
	if n > math.MaxInt64 {
		err = fmt.Errorf("Illegal offset: %v", n)
		return
	}

	// Update our state regarding being dirty.
	mc.dirtyThreshold = minInt64(mc.dirtyThreshold, n)

	newMtime := mc.clock.Now()
	mc.mtime = &newMtime

	// Call through.
	err = mc.readWriteLease.Truncate(int64(n))

	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func minInt64(a int64, b int64) int64 {
	if a < b {
		return a
	}

	return b
}

func (mc *MutableContent) dirty() bool {
	return mc.readWriteLease != nil
}

// Ensure that mc.readWriteLease is non-nil with an authoritative view of mc's
// contents.
func (mc *MutableContent) ensureReadWriteLease(
	ctx context.Context) (err error) {
	// Is there anything to do?
	if mc.readWriteLease != nil {
		return
	}

	// Set up the read/write lease.
	rwl, err := mc.initialContent.Upgrade(ctx)
	if err != nil {
		err = fmt.Errorf("initialContent.Upgrade: %v", err)
		return
	}

	mc.readWriteLease = rwl
	mc.initialContent = nil

	return
}
