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

// Provides integration tests for reads and writes through FUSE and GCS APIs.
package readwrite_test

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/util"
)

var testBucket = flag.String("testbucket", "", "The GCS bucket used for the test.")

var (
	binFile string
	logFile string
	mntDir string
)

func TestMain(m *testing.M) {
	flag.Parse()

	if *testBucket == "" {
		log.Fatalf("--testbucket must be specified")
		return
	}

	testDir, err := ioutil.TempDir("", "gcsfuse_readwrite_test_")
	if err != nil {
		log.Fatalf("TempDir: %v\n", err)
	}
	defer os.Remove(testDir)

	err = util.BuildGcsfuse(testDir)
	if err != nil {
		log.Printf("BuildGcsfuse(%q): %v\n", testDir, err)
		return
	}

	binFile = path.Join(testDir, "bin/gcsfuse")
	logFile = path.Join(testDir, "gcsfuse.log")
	mntDir = path.Join(testDir, "mnt")

	err = os.Mkdir(mntDir, 0755)
	if err != nil {
		log.Printf("Mkdir(%q): %v\n", mntDir, err)
		return
	}

	mountCmd := exec.Command(
		binFile,
		"--implicit-dirs",
		"--debug_gcs",
		"--debug_fuse",
		"--log-file=" + logFile,
		"--log-format=text",
		*testBucket,
		mntDir,
	)
	output, err := mountCmd.CombinedOutput()
	if err != nil {
		log.Printf("Cannot mount gcsfuse: %v\n", err)
		return
	}
	if lines := bytes.Count(output, []byte{'\n'}); lines > 1 {
		log.Printf("Mount failure: %q\n", output)
		return
	}
	defer util.Unmount(mntDir)

	ret := m.Run()
	os.Exit(ret)
}

func TestReadAfterWrite(t *testing.T) {
	tmpDir, err := ioutil.TempDir(mntDir, "tmpDir")
	if err != nil {
		t.Errorf("Mkdir at %q: %v", mntDir, err)
		return
	}

	tmpFile, err := ioutil.TempFile(tmpDir, "tmpFile")
	if err != nil {
		t.Errorf("Create file at %q: %v", tmpDir, err)
		return
	}

	fileName := tmpFile.Name()
	if _, err := tmpFile.WriteString("line 1\n"); err != nil {
		t.Errorf("WriteString: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}

	tmpFile, err = os.Open(fileName)
	if err != nil {
		t.Errorf("Open %q: %v", fileName, err)
		return
	}

	content, err := ioutil.ReadAll(tmpFile)
	if err != nil {
		t.Errorf("ReadAll: %v",  err)
	}
	if got, want := string(content), "line 1\n"; got != want {
		t.Errorf("File content %q not match %q", got, want)
	}
}