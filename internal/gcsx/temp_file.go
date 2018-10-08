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
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jacobsa/timeutil"
	"sync"
	"log"
	"path"
	"io/ioutil"
)

// A temporary file that keeps track of the lowest offset at which it has been
// modified.
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

	// Return information about the current state of the content. May invalidate
	// the seek position.
	Stat() (sr StatResult, err error)

	// Explicitly set the mtime that will return in stat results. This will stick
	// until another method that modifies the file is called.
	SetMtime(mtime time.Time)

	// Throw away the resources used by the temporary file. The object must not
	// be used again.
	Destroy()

	SetDirtyThreshold(t int64)

	SyncLocal() error

	GetFileRO() *os.File
}

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

type progressWriter struct{
	written int64
}

func (pw *progressWriter) Write(data []byte) (int, error) {
	pw.written += int64(len(data))
	return len(data), nil
}

// Create a temp file whose initial contents are given by the supplied reader.
// dir is a directory on whose file system the inode will live, or the system
// default temporary location if empty.
func NewTempFile(
	content io.Reader,
	dir string,
	clock timeutil.Clock, readMode bool, close func() error) (tf TempFile, err error) {
	// Create an anonymous file to wrap. When we close it, its resources will be
	// magically cleaned up.
	f, fro, err := AnonymousFile(dir)
	if err != nil {
		err = fmt.Errorf("AnonymousFile: %v", err)
		return
	}
	pw := &progressWriter{}
	tempFile := &tempFile{
		clock:          clock,
		f:              f,
		fro: fro,
		pw: pw,
	}
	tempFile.mu.Lock()
	tf = tempFile

	fc := func() {
		defer func(){
			if close != nil {
				close()
			}
			tempFile.dpmu.Lock()
			tempFile.downloadInProgress = false
			tempFile.dpmu.Unlock()
			tempFile.mu.Unlock()
		}()
		// Copy into the file.
		size, err := io.Copy(f, io.TeeReader(content, pw))
		if err != nil {
			tempFile.err = fmt.Errorf("copy: %v", err)
			return
		}
		tempFile.dirtyThreshold = size
	}
	if readMode {
		//go fc()
	} else {
		tempFile.downloadInProgress = true
		fc()
		err = tempFile.err
	}
	return
}


func AnonymousFile(dir string) (frw *os.File, fro *os.File, err error){
	// Choose a prefix based on the binary name.
	prefix := path.Base(os.Args[0])

	// Create the file.
	frw, err = ioutil.TempFile(dir, prefix)
	if err != nil {
		err = fmt.Errorf("TempFile: %v", err)
		return
	}

	fro, err = os.Open(frw.Name())
	if err != nil {
		err = fmt.Errorf("TempFile: %v", err)
		return
	}

	//// Unlink it.
	//err = os.Remove(frw.Name())
	//if err != nil {
	//	err = fmt.Errorf("Remove: %v", err)
	//	return
	//}
	return
}

type tempFile struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	clock timeutil.Clock

	/////////////////////////
	// Mutable state
	/////////////////////////

	destroyed bool

	// A file containing our current contents.
	f *os.File

	// A file descriptior for sync process
	fro *os.File

	// The lowest byte index that has been modified from the initial contents.
	//
	// INVARIANT: Stat().DirtyThreshold <= Stat().Size
	dirtyThreshold int64

	// The time at which a method that modifies our contents was last called, or
	// nil if never.
	//
	// INVARIANT: mtime == nil => Stat().DirtyThreshold == Stat().Size
	mtime *time.Time

	mu sync.Mutex

	dpmu sync.Mutex
	downloadInProgress bool

	pw *progressWriter
	err error
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

func (tf *tempFile) CheckInvariants() {
	if tf.destroyed {
		panic("Use of destroyed tempFile object.")
	}

	// Restore the seek position after using Stat below.
	pos, err := tf.Seek(0, 1)
	if err != nil {
		panic(fmt.Sprintf("Seek: %v", err))
	}

	defer func() {
		_, err := tf.Seek(pos, 0)
		if err != nil {
			panic(fmt.Sprintf("Seek: %v", err))
		}
	}()

	// INVARIANT: Stat().DirtyThreshold <= Stat().Size
	sr, err := tf.Stat()
	if err != nil {
		panic(fmt.Sprintf("Stat: %v", err))
	}

	if !(sr.DirtyThreshold <= sr.Size) {
		panic(fmt.Sprintf("Mismatch: %d vs. %d", sr.DirtyThreshold, sr.Size))
	}

	// INVARIANT: mtime == nil => Stat().DirtyThreshold == Stat().Size
	if tf.mtime == nil && sr.DirtyThreshold != sr.Size {
		panic(fmt.Sprintf("Mismatch: %d vs. %d", sr.DirtyThreshold, sr.Size))
	}
}

func (tf *tempFile) Destroy() {
	tf.destroyed = true

	tf.f.Close()
	tf.fro.Close()
	// Throw away the file.
	if err := os.Remove(tf.f.Name()); err != nil {
		log.Println("failed to remove fuse local temp file.", tf.f.Name(), err)
	}
	tf.f = nil
	tf.fro = nil
}

func (tf *tempFile) Read(p []byte) (int, error) {
	return tf.fro.Read(p)
}

func (tf *tempFile) Seek(offset int64, whence int) (int64, error) {
	return tf.f.Seek(offset, whence)
}

func (tf *tempFile) ReadAt(p []byte, offset int64) (int, error) {
	for {
		tf.dpmu.Lock()
		if !tf.downloadInProgress {
			tf.dpmu.Unlock()
			if tf.err != nil && tf.err != io.EOF {
				return 0, tf.err
			}
			break
		}
		tf.dpmu.Unlock()
		bl := offset+int64(len(p))
		wr := tf.pw.written
		if bl <= wr {
			break
		}
		log.Println("fuse: waiting download", bl, wr)
		time.Sleep(time.Second)
	}
	return tf.fro.ReadAt(p, offset)
}

func (tf *tempFile) Stat() (sr StatResult, err error) {
	sr.DirtyThreshold = tf.dirtyThreshold
	sr.Mtime = tf.mtime

	// Get the size from the file.
	sr.Size, err = tf.f.Seek(0, 2)
	if err != nil {
		err = fmt.Errorf("Seek: %v", err)
		return
	}

	return
}

func (tf *tempFile) WriteAt(p []byte, offset int64) (int, error) {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	// Update our state regarding being dirty.
	tf.dirtyThreshold = minInt64(tf.dirtyThreshold, offset)

	newMtime := tf.clock.Now()
	tf.mtime = &newMtime

	// Call through.
	return tf.f.WriteAt(p, offset)
}

func (tf *tempFile) Truncate(n int64) error {
	// Update our state regarding being dirty.
	tf.dirtyThreshold = minInt64(tf.dirtyThreshold, n)

	newMtime := tf.clock.Now()
	tf.mtime = &newMtime

	// Call through.
	return tf.f.Truncate(n)
}

func (tf *tempFile) SetMtime(mtime time.Time) {
	tf.mtime = &mtime
}

func (tf *tempFile) SetDirtyThreshold(t int64) {
	tf.dirtyThreshold = t
}

func (tf *tempFile) SyncLocal() error {
	return tf.f.Sync()
}

func (tf *tempFile) GetFileRO() *os.File {
	return tf.fro
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