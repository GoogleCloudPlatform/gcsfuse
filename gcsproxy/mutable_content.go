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
	initialContents lease.ReadProxy,
	clock timeutil.Clock) (mc *MutableContent)

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

// Panic if any internal invariants are violated. Careful users can call this
// at appropriate times to help debug weirdness. Consider using
// syncutil.InvariantMutex to automate the process.
func (mc *MutableContent) CheckInvariants()

// Destroy any state used by the object, putting it into an indeterminate
// state. The object must not be used again.
func (mc *MutableContent) Destroy()

// Read part of the content, with semantics equivalent to io.ReaderAt aside
// from context support.
func (mc *MutableContent) ReadAt(
	ctx context.Context,
	buf []byte,
	offset int64) (n int, err error)

// Return information about the current state of the content.
func (mc *MutableContent) Stat(ctx context.Context) (sr StatResult, err error)

// Write into the content, with semantics equivalent to io.WriterAt aside from
// context support.
func (mc *MutableContent) WriteAt(
	ctx context.Context,
	buf []byte,
	offset int64) (n int, err error)

// Truncate our the content to the given number of bytes, extending if n is
// greater than the current size.
func (mc *MutableContent) Truncate(
	ctx context.Context,
	n int64) (err error)
