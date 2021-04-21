// Copyright 2020 Google Inc. All Rights Reserved.
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
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/jacobsa/fuse/fusetesting"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type AllBucketsTest struct {
	fsTest
}

func init() { RegisterTestSuite(&AllBucketsTest{}) }

func (t *AllBucketsTest) SetUp(ti *TestInfo) {
	t.mtimeClock = timeutil.RealClock()
	t.buckets = map[string]gcs.Bucket{
		"bucket-0": gcsfake.NewFakeBucket(t.mtimeClock, "bucket-0"),
		"bucket-1": gcsfake.NewFakeBucket(t.mtimeClock, "bucket-1"),
		"bucket-2": gcsfake.NewFakeBucket(t.mtimeClock, "bucket-2"),
	}
	// buckets: {"some_bucket", "bucket-1", "bucket-2"}
	t.fsTest.SetUp(ti)
	AssertEq("", t.serverCfg.BucketName)
	AssertEq(t.Dir, t.mfs.Dir())
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *AllBucketsTest) BaseDir_Ls() {
	entries, err := ioutil.ReadDir(t.Dir)
	AssertEq(nil, err)
	AssertEq(3, len(entries))
	ExpectEq("bucket-0", entries[0].Name())
	ExpectEq("bucket-1", entries[1].Name())
	ExpectEq("bucket-2", entries[2].Name())

	entries, err = fusetesting.ReadDirPicky(t.Dir)
	AssertEq(nil, err)
	AssertEq(3, len(entries))
	ExpectEq("bucket-0", entries[0].Name())
	ExpectEq("bucket-1", entries[1].Name())
	ExpectEq("bucket-2", entries[2].Name())
}

func (t *AllBucketsTest) BaseDir_Write() {
	filename := path.Join(t.mfs.Dir(), "foo")
	err := ioutil.WriteFile(
		filename, []byte("content"), os.FileMode(0644))
	ExpectThat(err, Error(HasSubstr("input/output error")))
}

func (t *AllBucketsTest) BaseDir_Rename() {
	err := os.Rename(t.Dir+"/bucket-0", t.Dir+"/bucket-1/foo")
	ExpectThat(err, Error(HasSubstr("operation not supported")))

	filename := path.Join(t.Dir + "/bucket-0/foo")
	err = ioutil.WriteFile(
		filename, []byte("content"), os.FileMode(0644))
	AssertEq(nil, err)

	err = os.Rename(t.Dir+"/bucket-0/foo", t.Dir+"/bucket-1/foo")
	ExpectThat(err, Error(HasSubstr("operation not supported")))

	err = os.Rename(t.Dir+"/bucket-0/foo", t.Dir+"/bucket-99")
	ExpectThat(err, Error(HasSubstr("input/output error")))

}

func (t *AllBucketsTest) SingleBucket_ReadAfterWrite() {
	var err error
	filename := path.Join(t.mfs.Dir(), "bucket-1/foo")

	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			filename,
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	t.f1, err = os.OpenFile(filename, os.O_RDWR, 0)
	AssertEq(nil, err)

	// Write to the start of the file using File.Write.
	_, err = t.f1.Write([]byte("000"))
	AssertEq(nil, err)

	// Write to the middle of the file using File.WriteAt.
	_, err = t.f1.WriteAt([]byte("111"), 4)
	AssertEq(nil, err)

	// Seek and write past the end of the file.
	_, err = t.f1.Seek(int64(len(contents)), 0)
	AssertEq(nil, err)

	_, err = t.f1.Write([]byte("222"))
	AssertEq(nil, err)

	// Check the size now.
	fi, err := t.f1.Stat()
	AssertEq(nil, err)
	ExpectEq(len(contents)+len("222"), fi.Size())

	// Read some contents with Seek and Read.
	_, err = t.f1.Seek(4, 0)
	AssertEq(nil, err)

	buf := make([]byte, 4)
	_, err = io.ReadFull(t.f1, buf)

	AssertEq(nil, err)
	ExpectEq("111r", string(buf))

	// Read the full contents with ReadAt.
	buf = make([]byte, len(contents)+len("222"))
	_, err = t.f1.ReadAt(buf, 0)

	AssertEq(nil, err)
	ExpectEq("000o111ritoenchilada222", string(buf))

	// Close the file.
	AssertEq(nil, t.f1.Close())
	t.f1 = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(filename)

	AssertEq(nil, err)
	ExpectEq("000o111ritoenchilada222", string(fileContents))
}
