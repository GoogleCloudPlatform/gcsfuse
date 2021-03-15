// Copyright 2021 Google Inc. All Rights Reserved.
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

package integration_test

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"testing"

	. "github.com/jacobsa/ogletest"
)

var testBucket = flag.String("test-bucket", "", "Name of the bucket used for the e2e test.")

func TestGcsfuseE2E(t *testing.T) {RunTests(t)}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

// Test the mounted Gcsfuse from end to end, involving the FUSE APIs and GCS
// APIs. Unlike GcsfuseTest where the GCS API is mocked, this test calls GCS 
// APIs directly, which will incur charges in the GCP project.
type GcsfuseE2ETest struct {
	// A real GCS bucket that the test uses.
	bucketName string
	
	// Logs generated from the mounted Gcsfuse during the test.
	logFile string

	// A temporary directory into which a file system may be mounted. Removed in
	// TearDown.
	dir string
}

var _ SetUpInterface = &GcsfuseE2ETest{}
var _ TearDownInterface = &GcsfuseE2ETest{}

func init() { 
	if *testBucket != "" {
		RegisterTestSuite(&GcsfuseE2ETest{}) 
	}
}

func (t *GcsfuseE2ETest) SetUp(_ *TestInfo) {
	t.bucketName = *testBucket

	var err error
	t.dir, err = ioutil.TempDir("", "gcsfuse_e2e_test_")
	AssertEq(nil, err)

	logFile, err := ioutil.TempFile("", "gcsfuse_log_")
	AssertEq(nil, err)
	t.logFile = logFile.Name()
	logFile.Close()
	log.Printf("gcsfuse logs: %s\n", t.logFile)

	cmd := exec.Command(
		path.Join(gBuildDir, "bin/gcsfuse"),  // gcsfuse binary
		"--implicit-dirs",
		"--debug_gcs",
		"--debug_fuse",
		"--log-file", 
		t.logFile,
		t.bucketName,
		t.dir,
	)
	_, err = cmd.CombinedOutput()
	AssertEq(nil, err)
	log.Printf("%s is mounted!\n", t.bucketName)
}

func (t *GcsfuseE2ETest) TearDown() {
	err := unmount(t.dir)
	AssertEq(nil, err)

	err = os.Remove(t.dir)
	AssertEq(nil, err)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *GcsfuseE2ETest) ReadAfterWrite() {
	testDir, err := ioutil.TempDir(t.dir, "testDir")
	AssertEq(nil, err)

	// Create file and write first line
	file, err := ioutil.TempFile(testDir, "testFile")
	AssertEq(nil, err)
	fileName := file.Name()
	file.WriteString("line 1\n")
	file.Close()

	file,err  = os.Open(fileName)
	AssertEq(nil, err)
	content, err := ioutil.ReadAll(file)
	AssertEq(nil, err)
	AssertEq("line 1\n", string(content))
}