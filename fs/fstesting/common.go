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

package fstesting

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/googlecloudplatform/gcsfuse/fs"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	"github.com/jacobsa/oglematchers"
	"github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
)

// A struct that can be embedded to inherit common file system test behaviors.
type fsTest struct {
	ctx    context.Context
	clock  timeutil.Clock
	bucket gcs.Bucket
	mfs    *fuse.MountedFileSystem

	// Files to close when tearing down. Nil entries are skipped.
	f1 *os.File
	f2 *os.File
}

var _ fsTestInterface = &fsTest{}

func (t *fsTest) setUpFSTest(cfg FSTestConfig) {
	t.ctx = context.Background()
	t.clock = cfg.ServerConfig.Clock
	t.bucket = cfg.ServerConfig.Bucket

	// Set up a temporary directory for mounting.
	mountPoint, err := ioutil.TempDir("", "fs_test")
	if err != nil {
		panic("ioutil.TempDir: " + err.Error())
	}

	// Create a file system server.
	server, err := fs.NewServer(&cfg.ServerConfig)
	if err != nil {
		panic("NewServer: " + err.Error())
	}

	// Mount the file system.
	t.mfs, err = fuse.Mount(mountPoint, server, &fuse.MountConfig{})
	if err != nil {
		panic("Mount: " + err.Error())
	}
}

func (t *fsTest) tearDownFsTest() {
	// Close any files we opened.
	if t.f1 != nil {
		ogletest.ExpectEq(nil, t.f1.Close())
	}

	if t.f2 != nil {
		ogletest.ExpectEq(nil, t.f2.Close())
	}

	// Unmount the file system. Try again on "resource busy" errors.
	delay := 10 * time.Millisecond
	for {
		err := fuse.Unmount(t.mfs.Dir())
		if err == nil {
			break
		}

		if strings.Contains(err.Error(), "resource busy") {
			log.Println("Resource busy error while unmounting; trying again")
			time.Sleep(delay)
			delay = time.Duration(1.3 * float64(delay))
			continue
		}

		panic("MountedFileSystem.Unmount: " + err.Error())
	}

	if err := t.mfs.Join(t.ctx); err != nil {
		panic("MountedFileSystem.Join: " + err.Error())
	}
}

func (t *fsTest) createWithContents(name string, contents string) error {
	return t.createObjects(map[string]string{name: contents})
}

func (t *fsTest) createObjects(in map[string]string) error {
	err := gcsutil.CreateObjects(t.ctx, t.bucket, in)
	return err
}

func (t *fsTest) createEmptyObjects(names []string) error {
	err := gcsutil.CreateEmptyObjects(t.ctx, t.bucket, names)
	return err
}

// Ensure that the clock will report a different time after returning.
func (t *fsTest) advanceTime() {
	// For simulated clocks, we can just advance the time.
	if c, ok := t.clock.(*timeutil.SimulatedClock); ok {
		c.AdvanceTime(time.Second)
		return
	}

	// Otherwise, sleep a moment.
	time.Sleep(time.Millisecond)
}

// Return a matcher that matches event times as reported by the bucket
// corresponding to the supplied start time as measured by the test.
func (t *fsTest) matchesStartTime(start time.Time) oglematchers.Matcher {
	// For simulated clocks we can use exact equality.
	if _, ok := t.clock.(*timeutil.SimulatedClock); ok {
		return timeutil.TimeEq(start)
	}

	// Otherwise, we need to take into account latency between the start of our
	// call and the time the server actually executed the operation.
	const slop = 60 * time.Second
	return timeutil.TimeNear(start, slop)
}

// Repeatedly call ioutil.ReadDir until an error is encountered or until the
// result has the given length. After each successful call with the wrong
// length, advance the clock by more than the directory listing cache TTL in
// order to flush the cache before the next call.
//
// This is a hacky workaround for the lack of list-after-write consistency in
// GCS that must be used when interacting with GCS through a side channel
// rather than through the file system. We set up some objects through a back
// door, then list repeatedly until we see the state we hope to see.
func (t *fsTest) readDirUntil(
	desiredLen int,
	dir string) (entries []os.FileInfo, err error) {
	startTime := time.Now()
	endTime := startTime.Add(5 * time.Second)

	for i := 0; ; i++ {
		entries, err = ioutil.ReadDir(dir)
		if err != nil || len(entries) == desiredLen {
			return
		}

		// Should we stop?
		if time.Now().After(endTime) {
			err = errors.New("Timeout waiting for the given length.")
			break
		}

		// Sleep for awhile.
		const baseDelay = 10 * time.Millisecond
		time.Sleep(time.Duration(math.Pow(1.3, float64(i)) * float64(baseDelay)))

		// If this is taking awhile, log that fact so that the user can tell why
		// the test is hanging.
		if time.Since(startTime) > time.Second {
			var names []string
			for _, fi := range entries {
				names = append(names, fi.Name())
			}

			log.Printf(
				"readDirUntil waiting for length %v. Current: %v, names: %v",
				desiredLen,
				len(entries),
				names)
		}
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Common helpers
////////////////////////////////////////////////////////////////////////

func getFileNames(entries []os.FileInfo) (names []string) {
	for _, e := range entries {
		names = append(names, e.Name())
	}

	return
}

// REQUIRES: n % 4 == 0
func randString(n int) string {
	bytes := make([]byte, n)
	for i := 0; i < n; i += 4 {
		u32 := rand.Uint32()
		bytes[i] = byte(u32 >> 0)
		bytes[i+1] = byte(u32 >> 8)
		bytes[i+2] = byte(u32 >> 16)
		bytes[i+3] = byte(u32 >> 24)
	}

	return string(bytes)
}

func readRange(r io.ReadSeeker, offset int64, n int) (s string, err error) {
	if _, err = r.Seek(offset, 0); err != nil {
		return
	}

	bytes := make([]byte, n)
	if _, err = io.ReadFull(r, bytes); err != nil {
		return
	}

	s = string(bytes)
	return
}

func currentUid() uint32 {
	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	uid, err := strconv.ParseUint(user.Uid, 10, 32)
	if err != nil {
		panic(err)
	}

	return uint32(uid)
}

func currentGid() uint32 {
	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	gid, err := strconv.ParseUint(user.Gid, 10, 32)
	if err != nil {
		panic(err)
	}

	return uint32(gid)
}
