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

package mutable

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
)

// A mutable view on some content. Created with an initial read-only view,
// which then can be modified by the user and read back. Keeps track of which
// portion of the content has been dirtied.
//
// External synchronization is required.
type Content interface {
	// Panic if any internal invariants are violated.
	CheckInvariants()

	// Destroy any state used by the object, putting it into an indeterminate
	// state. The object must not be used again.
	Destroy()

	// Read part of the content, with semantics equivalent to io.ReaderAt aside
	// from context support.
	ReadAt(ctx context.Context, buf []byte, offset int64) (n int, err error)

	// Return information about the current state of the content.
	Stat(ctx context.Context) (sr StatResult, err error)

	// Write into the content, with semantics equivalent to io.WriterAt aside from
	// context support.
	WriteAt(ctx context.Context, buf []byte, offset int64) (n int, err error)

	// Truncate our the content to the given number of bytes, extending if n is
	// greater than the current size.
	Truncate(ctx context.Context, n int64) (err error)
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
// supplied file. The caller must ensure that the file is not used further by
// anybody else.
func NewContent(
	initialContent *os.File,
	clock timeutil.Clock) (mc Content, err error) {
	// Find the file's size.
	size, err := initialContent.Seek(0, 2)
	if err != nil {
		err = fmt.Errorf("Seek: %v", err)
		return
	}

	// Create the Content.
	mc = &mutableContent{
		clock:          clock,
		contents:       initialContent,
		dirtyThreshold: size,
	}

	return
}

type mutableContent struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	clock timeutil.Clock

	/////////////////////////
	// Mutable state
	/////////////////////////

	destroyed bool

	// Have we been dirtied since we were created?
	dirty bool

	// A file containing our current contents.
	contents *os.File

	// The lowest byte index that has been modified from the initial contents.
	//
	// INVARIANT: !dirty => Stat().DirtyThreshold == Stat().Size
	dirtyThreshold int64

	// The time at which a method that modifies our contents was last called, or
	// nil if never.
	//
	// INVARIANT: dirty => mtime != nil
	mtime *time.Time
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

func (mc *mutableContent) CheckInvariants() {
	if mc.destroyed {
		panic("Use of destroyed mutableContent object.")
	}

	// INVARIANT: !dirty => Stat().DirtyThreshold == Stat().Size
	if !mc.dirty {
		sr, err := mc.Stat(context.Background())
		if err != nil {
			panic(fmt.Sprintf("Stat: %v", err))
		}

		if sr.DirtyThreshold != sr.Size {
			panic(fmt.Sprintf("Mismatch: %d vs. %d", sr.DirtyThreshold, sr.Size))
		}
	}

	// INVARIANT: dirty => mtime != nil
	if mc.dirty && mc.mtime == nil {
		panic("Expected a non-nil mtime")
	}
}

func (mc *mutableContent) Destroy() {
	mc.destroyed = true

	// Throw away the file.
	mc.contents.Close()
	mc.contents = nil
}

func (mc *mutableContent) Release() (rwl lease.ReadWriteLease) {
	if !mc.dirty() {
		return
	}

	rwl = mc.readWriteLease
	mc.readWriteLease = nil
	mc.Destroy()

	return
}

func (mc *mutableContent) ReadAt(
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

func (mc *mutableContent) Stat(
	ctx context.Context) (sr StatResult, err error) {
	sr.DirtyThreshold = mc.dirtyThreshold
	sr.Mtime = mc.mtime

	// Get the size from the appropriate place.
	if mc.dirty() {
		sr.Size, err = mc.readWriteLease.Size()
		if err != nil {
			return
		}
	} else {
		sr.Size = mc.initialContent.Size()
	}

	return
}

func (mc *mutableContent) WriteAt(
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

func (mc *mutableContent) Truncate(
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

// Ensure that mc.readWriteLease is non-nil with an authoritative view of mc's
// contents.
func (mc *mutableContent) ensureReadWriteLease(
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
