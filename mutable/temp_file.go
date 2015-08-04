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
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
)

// A temporary file that keeps track of the lowest offset at which it has been
// modified.
type TempFile interface {
	// Panic if any internal invariants are violated.
	CheckInvariants()

	// Semantics matching os.File.
	io.ReadSeeker
	io.ReaderAt
	io.WriterAt
	Truncate(n int64) (err error)

	// Return information about the current state of the content.
	Stat() (sr StatResult, err error)

	// Throw away the resources used by the temporary file. The object must not
	// be used again.
	Destroy()
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

// Create a temp file whose initial contents are given by the supplied reader.
func NewTempFile(
	content io.Reader,
	clock timeutil.Clock) (tf TempFile, err error) {
	err = errors.New("TODO")
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

func (mc *mutableContent) ReadAt(
	ctx context.Context,
	buf []byte,
	offset int64) (n int, err error) {
	n, err = mc.contents.ReadAt(buf, offset)
	return
}

func (mc *mutableContent) Stat(
	ctx context.Context) (sr StatResult, err error) {
	sr.DirtyThreshold = mc.dirtyThreshold
	sr.Mtime = mc.mtime

	// Get the size from the file.
	sr.Size, err = mc.contents.Seek(0, 2)
	if err != nil {
		err = fmt.Errorf("Seek: %v", err)
		return
	}

	return
}

func (mc *mutableContent) WriteAt(
	ctx context.Context,
	buf []byte,
	offset int64) (n int, err error) {
	// Update our state regarding being dirty.
	mc.dirtyThreshold = minInt64(mc.dirtyThreshold, offset)

	newMtime := mc.clock.Now()
	mc.mtime = &newMtime

	// Call through.
	n, err = mc.contents.WriteAt(buf, offset)

	return
}

func (mc *mutableContent) Truncate(
	ctx context.Context,
	n int64) (err error) {
	// Update our state regarding being dirty.
	mc.dirtyThreshold = minInt64(mc.dirtyThreshold, n)

	newMtime := mc.clock.Now()
	mc.mtime = &newMtime

	// Call through.
	err = mc.contents.Truncate(int64(n))

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
