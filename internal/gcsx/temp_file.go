// Copyright 2015 Google LLC
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
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"github.com/jacobsa/fuse/fsutil"
	"github.com/jacobsa/timeutil"
)

// TempFile is a temporary file that keeps track of the lowest offset at which
// it has been modified.
//
// Not safe for concurrent access.
type TempFile interface {
	// Panic if any internal invariants are violated.
	CheckInvariants()

	// Semantics matching os.File.
	io.ReadSeeker
	io.ReaderAt
	io.WriterAt
	Truncate(n int64) (err error)

	// Retrieve the file name
	Name() string

	// Return information about the current state of the content. May invalidate
	// the seek position.
	Stat() (sr StatResult, err error)

	// Explicitly set the mtime that will return in stat results. This will stick
	// until another method that modifies the file is called.
	SetMtime(mtime time.Time)

	// Throw away the resources used by the temporary file. The object must not
	// be used again.
	Destroy()
}

// StatResult stores the result of a stat operation.
type StatResult struct {
	// The current size in bytes of the content.
	Size int64

	// The largest value T such that we are sure that the range of bytes [0, T)
	// is unmodified from the original content with which the temp file was
	// created.
	DirtyThreshold int64

	// The mtime of the temp file is updated according to the temp file's clock
	// with each call to a method that modified its content, and is also updated
	// when the user explicitly calls SetMtime.
	//
	// If neither of those things has ever happened, it is nil. This implies that
	// DirtyThreshold == Size.
	Mtime *time.Time
}

// NewTempFile creates a temp file whose initial contents are given by the
// supplied reader. dir is a directory on whose file system the inode will live,
// or the system default temporary location if empty.
func NewTempFile(
	source io.ReadCloser,
	dir string,
	clock timeutil.Clock) (tf TempFile, err error) {
	// Create an anonymous file to wrap. When we close it, its resources will be
	// magically cleaned up.
	f, err := fsutil.AnonymousFile(dir)
	if err != nil {
		err = fmt.Errorf("AnonymousFile: %w", err)
		return
	}

	tf = &tempFile{
		source:         source,
		state:          fileIncomplete,
		clock:          clock,
		f:              f,
		dirtyThreshold: 0,
	}

	return
}

// NewCacheFile creates a wrapper temp file whose initial contents are given by the
// supplied source. dir is a directory on whose file system the file will live,
// or the system default temporary location if empty.
func NewCacheFile(
	source io.ReadCloser,
	f *os.File,
	dir string,
	clock timeutil.Clock) (tf TempFile) {

	tf = &tempFile{
		source:         source,
		state:          fileIncomplete,
		clock:          clock,
		f:              f,
		dirtyThreshold: 0,
	}

	return
}

func RecoverCacheFile(
	source *os.File,
	dir string,
	clock timeutil.Clock) (tf TempFile, err error) {

	stat, err := source.Stat()

	if err != nil {
		return nil, fmt.Errorf("could not retrieve file stat: %w", err)
	}

	tf = &tempFile{
		source:         source,
		state:          fileComplete,
		clock:          clock,
		f:              source,
		dirtyThreshold: stat.Size(),
	}

	return
}

type fileState string

const (
	fileIncomplete fileState = "fileIncomplete"
	fileComplete             = "fileComplete"
	fileDirty                = "fileDirty"
	fileDestroyed            = "fileDestroyed"
)

type tempFile struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	clock timeutil.Clock

	source io.ReadCloser

	/////////////////////////
	// Mutable state
	/////////////////////////
	state fileState

	// A file containing our current contents.
	f *os.File

	// The lowest byte index that has been modified from the initial contents.
	//
	// INVARIANT: Stat().DirtyThreshold <= Stat().Size
	dirtyThreshold int64

	// The time at which a method that modifies our contents was last called, or
	// nil if never.
	//
	// INVARIANT: mtime == nil => Stat().DirtyThreshold == Stat().Size
	mtime *time.Time
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

func (tf *tempFile) CheckInvariants() {
	if tf.state == fileDestroyed {
		panic("Use of destroyed tempFile object.")
	}

	// Restore the seek position after using Stat below.
	pos, err := tf.Seek(0, 1)
	if err != nil {
		panic(fmt.Errorf("seek: %w", err))
	}

	defer func() {
		_, err := tf.Seek(pos, 0)
		if err != nil {
			panic(fmt.Errorf("seek: %w", err))
		}
	}()

	// INVARIANT: Stat().DirtyThreshold <= Stat().Size
	sr, err := tf.Stat()
	if err != nil {
		panic(fmt.Errorf("stat: %w", err))
	}

	if !(sr.DirtyThreshold <= sr.Size) {
		panic(fmt.Errorf("mismatch: %d vs. %d", sr.DirtyThreshold, sr.Size))
	}

	// INVARIANT: mtime == nil => Stat().DirtyThreshold == Stat().Size
	if tf.mtime == nil && sr.DirtyThreshold != sr.Size {
		panic(fmt.Errorf("mismatch: %d vs. %d", sr.DirtyThreshold, sr.Size))
	}
}

func (tf *tempFile) Destroy() {
	tf.state = fileDestroyed
	// Throw away the file (for anonymous files).
	tf.f.Close()

	tf.f = nil
}

func (tf *tempFile) Read(p []byte) (int, error) {
	err := tf.ensureComplete()
	if err != nil {
		return 0, fmt.Errorf("cannot Read incomplete file: %w", err)
	}
	return tf.f.Read(p)
}

func (tf *tempFile) Seek(offset int64, whence int) (int64, error) {
	err := tf.ensureComplete()
	if err != nil {
		return 0, fmt.Errorf("cannot Seek incomplete file: %w", err)
	}
	return tf.f.Seek(offset, whence)
}

func (tf *tempFile) ReadAt(p []byte, offset int64) (int, error) {
	err := tf.ensureComplete()
	if err != nil {
		return 0, fmt.Errorf("cannot ReadAt incomplete file: %w", err)
	}
	return tf.f.ReadAt(p, offset)
}

func (tf *tempFile) Stat() (sr StatResult, err error) {
	err = tf.ensureComplete()
	if err != nil {
		err = fmt.Errorf("cannot Stat incomplete file: %w", err)
		return
	}
	sr.DirtyThreshold = tf.dirtyThreshold
	sr.Mtime = tf.mtime

	// Get the size from the file.
	sr.Size, err = tf.f.Seek(0, 2)
	if err != nil {
		err = fmt.Errorf("seek: %w", err)
		return
	}

	return
}

func (tf *tempFile) WriteAt(p []byte, offset int64) (int, error) {
	err := tf.ensureComplete()
	if err != nil {
		return 0, fmt.Errorf("cannot WriteAt incomplete file: %w", err)
	}

	// Update our state regarding being dirty.
	tf.dirtyThreshold = minInt64(tf.dirtyThreshold, offset)

	tf.state = fileDirty

	newMtime := tf.clock.Now()
	tf.mtime = &newMtime

	// Call through.
	return tf.f.WriteAt(p, offset)
}

func (tf *tempFile) Truncate(n int64) error {
	err := tf.ensureComplete()
	if err != nil {
		return fmt.Errorf("cannot Truncate incomplete file: %w", err)
	}

	// Update our state regarding being dirty.
	tf.dirtyThreshold = minInt64(tf.dirtyThreshold, n)

	tf.state = fileDirty

	newMtime := tf.clock.Now()
	tf.mtime = &newMtime

	// Call through.
	return tf.f.Truncate(n)
}

func (tf *tempFile) SetMtime(mtime time.Time) {
	tf.mtime = &mtime
}

func (tf *tempFile) Name() string {
	return tf.f.Name()
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

const (
	minCopyLength = 64 * 1024 * 1024 // 64 MB
)

func (tf *tempFile) ensure(limit int64) error {
	switch tf.state {
	case fileIncomplete:
		size, err := tf.f.Seek(0, 2)
		if size >= limit {
			return nil
		}
		n := limit - size
		if n < minCopyLength {
			n = minCopyLength
		}
		n, err = io.CopyN(tf.f, tf.source, n)
		if err == io.EOF {
			tf.source.Close()
			tf.dirtyThreshold = size + n
			tf.state = fileComplete
			err = nil
		}
		return err
	case fileComplete, fileDirty:
		// already completed
		return nil
	case fileDestroyed:
		return fmt.Errorf("file destroyed")
	}
	return nil
}

func (tf *tempFile) ensureComplete() error {
	err := tf.ensure(math.MaxInt64)
	if err != nil {
		err = fmt.Errorf("load temp file: %w", err)
	}
	return err
}
