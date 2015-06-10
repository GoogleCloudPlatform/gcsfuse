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

package fs_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/fs"
	"github.com/googlecloudplatform/gcsfuse/perms"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	"github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

const (
	filePerms os.FileMode = 0740
	dirPerms              = 0754
)

func TestFS(t *testing.T) { ogletest.RunTests(t) }

// Install a SIGINT handler that exits gracefully once the current test is
// finished. It's not safe to exit in the middle of a test because closing any
// open files may require the fuse daemon to still be responsive.
func init() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		log.Println("Received SIGINT; exiting after running tests complete.")
		ogletest.StopRunningTests()
	}()
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

// A struct that can be embedded to inherit common file system test behaviors.
type fsTest struct {
	// Configuration
	serverCfg fs.ServerConfig
	mountCfg  fuse.MountConfig

	// Dependencies. If bucket is set before SetUp is called, it will be used
	// rather than creating a default one.
	clock  timeutil.SimulatedClock
	bucket gcs.Bucket

	// Mount information
	mfs *fuse.MountedFileSystem
	Dir string

	// Files to close when tearing down. Nil entries are skipped.
	f1 *os.File
	f2 *os.File
}

var _ ogletest.SetUpInterface = &fsTest{}
var _ ogletest.TearDownInterface = &fsTest{}

func (s *fsTest) SetUp(t *ogletest.T) {
	var err error

	// Set up the clock.
	s.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))
	s.serverCfg.Clock = &s.clock

	// And the bucket.
	if s.bucket == nil {
		s.bucket = gcsfake.NewFakeBucket(&s.clock, "some_bucket")
	}

	s.serverCfg.Bucket = s.bucket

	// Set up ownership.
	s.serverCfg.Uid, s.serverCfg.Gid, err = perms.MyUserAndGroup()
	t.AssertEq(nil, err)

	// Set up permissions.
	s.serverCfg.FilePerms = filePerms
	s.serverCfg.DirPerms = dirPerms

	// Use some temporary space to speed tests.
	s.serverCfg.TempDirLimitNumFiles = 16
	s.serverCfg.TempDirLimitBytes = 1 << 22 // 4 MiB

	// Set up the append optimization.
	s.serverCfg.AppendThreshold = 0
	s.serverCfg.TmpObjectPrefix = ".gcsfuse_tmp/"

	// Set up a temporary directory for mounting.
	s.Dir, err = ioutil.TempDir("", "fs_test")
	t.AssertEq(nil, err)

	// Create a file system server.
	server, err := fs.NewServer(&s.serverCfg)
	t.AssertEq(nil, err)

	// Mount the file system.
	mountCfg := s.mountCfg
	mountCfg.OpContext = t.Ctx

	s.mfs, err = fuse.Mount(s.Dir, server, &mountCfg)
	t.AssertEq(nil, err)
}

func (s *fsTest) TearDown(t *ogletest.T) {
	var err error

	// Close any files we opened.
	if s.f1 != nil {
		t.ExpectEq(nil, s.f1.Close())
	}

	if s.f2 != nil {
		t.ExpectEq(nil, s.f2.Close())
	}

	// Unmount the file system. Try again on "resource busy" errors.
	delay := 10 * time.Millisecond
	for {
		err := fuse.Unmount(s.mfs.Dir())
		if err == nil {
			break
		}

		if strings.Contains(err.Error(), "resource busy") {
			log.Println("Resource busy error while unmounting; trying again")
			time.Sleep(delay)
			delay = time.Duration(1.3 * float64(delay))
			continue
		}

		t.AddFailure("MountedFileSystem.Unmount: %v", err)
		t.AbortTest()
	}

	if err := s.mfs.Join(t.Ctx); err != nil {
		t.AssertEq(nil, err)
	}

	// Unlink the mount point.
	if err = os.Remove(s.Dir); err != nil {
		err = fmt.Errorf("Unlinking mount point: %v", err)
		return
	}
}

func (s *fsTest) createWithContents(
	t *ogletest.T,
	name string,
	contents string) error {
	return s.createObjects(t, map[string]string{name: contents})
}

func (s *fsTest) createObjects(t *ogletest.T, in map[string]string) error {
	err := gcsutil.CreateObjects(t.Ctx, s.bucket, in)
	return err
}

func (s *fsTest) createEmptyObjects(t *ogletest.T, names []string) error {
	err := gcsutil.CreateEmptyObjects(t.Ctx, s.bucket, names)
	return err
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

func currentUid(t *ogletest.T) uint32 {
	user, err := user.Current()
	t.AssertEq(nil, err)

	uid, err := strconv.ParseUint(user.Uid, 10, 32)
	t.AssertEq(nil, err)

	return uint32(uid)
}

func currentGid(t *ogletest.T) uint32 {
	user, err := user.Current()
	t.AssertEq(nil, err)

	gid, err := strconv.ParseUint(user.Gid, 10, 32)
	t.AssertEq(nil, err)

	return uint32(gid)
}
