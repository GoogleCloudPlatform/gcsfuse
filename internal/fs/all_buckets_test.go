// Copyright 2020 Google LLC
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
	"os"
	"path"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
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

func init() {
	RegisterTestSuite(&AllBucketsTest{})
}

func (t *AllBucketsTest) SetUpTestSuite() {
	mtimeClock = timeutil.RealClock()
	buckets = map[string]gcs.Bucket{
		"bucket-0": fake.NewFakeBucket(mtimeClock, "bucket-0", gcs.BucketType{}),
		"bucket-1": fake.NewFakeBucket(mtimeClock, "bucket-1", gcs.BucketType{}),
		"bucket-2": fake.NewFakeBucket(mtimeClock, "bucket-2", gcs.BucketType{}),
	}
	// buckets: {"some_bucket", "bucket-1", "bucket-2"}
	t.fsTest.SetUpTestSuite()
	AssertEq("", t.serverCfg.BucketName)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *AllBucketsTest) BaseDir_Ls() {
	_, err := os.ReadDir(mntDir)
	ExpectThat(err, Error(HasSubstr("operation not supported")))
}

func (t *AllBucketsTest) BaseDir_Write() {
	filename := path.Join(mntDir, "foo")
	err := os.WriteFile(
		filename, []byte("content"), os.FileMode(0644))
	ExpectThat(err, Error(HasSubstr("input/output error")))
}

func (t *AllBucketsTest) BaseDir_Rename() {
	err := os.Rename(mntDir+"/bucket-0", mntDir+"/bucket-1/foo")
	ExpectThat(err, Error(HasSubstr("operation not supported")))

	filename := path.Join(mntDir + "/bucket-0/foo")
	err = os.WriteFile(
		filename, []byte("content"), os.FileMode(0644))
	AssertEq(nil, err)

	err = os.Rename(mntDir+"/bucket-0/foo", mntDir+"/bucket-1/foo")
	ExpectThat(err, Error(HasSubstr("operation not supported")))

	err = os.Rename(mntDir+"/bucket-0/foo", mntDir+"/bucket-99")
	ExpectThat(err, Error(HasSubstr("input/output error")))

}

func (t *AllBucketsTest) SingleBucket_ReadAfterWrite() {
	var err error
	filename := path.Join(mntDir, "bucket-1/foo")

	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		os.WriteFile(
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
	fileContents, err := os.ReadFile(filename)

	AssertEq(nil, err)
	ExpectEq("000o111ritoenchilada222", string(fileContents))
}
