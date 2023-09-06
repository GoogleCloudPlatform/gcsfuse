// Copyright 2023 Google Inc. All Rights Reserved.
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

package multimounts_test

import (
	"os"
	"path"
	"strings"
	"sync"
	"syscall"
	"testing"

	operations "github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	. "github.com/jacobsa/ogletest"
)

const TwoHundredMiB = 1024 * 1024 * 200
const InputOutputErrorMsg = "input/output error"

func getMountPath1() string {
	return path.Join(setup.MntDir(), MountName1)
}

func getMountPath2() string {
	return path.Join(setup.MntDir(), MountName2)
}

func getFilePathInMount1() string {
	return path.Join(getMountPath1(), CommonFileName)
}

func getFilePathInMount2() string {
	return path.Join(getMountPath2(), CommonFileName)
}

func getGCSObjPath() string {
	return path.Join(setup.TestBucket(), CommonFileName)
}

func openFile(t *testing.T, filePath string) (file *os.File) {
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, operations.FilePermission_0600)
	if err != nil {
		t.Errorf("error in opening file %s: %v", file.Name(), err)
	}
	return
}

func writeToFile(t *testing.T, file *os.File, fileContent string) {
	_, err := file.WriteString(fileContent)
	if err != nil {
		t.Errorf("error in writing to the file %s: %v", file.Name(), err)
	}
}

func syncFile(t *testing.T, file *os.File) {
	err := file.Sync()
	if err != nil {
		t.Errorf("error in syncing file %s: %v", file.Name(), err)
	}
}

func cleanAndCreateCommonFileInTestBucket(t *testing.T) {
	// Cleaning one mount is sufficient as the test bucket is common.
	err := operations.EmptyDir(getMountPath1())
	if err != nil {
		t.Errorf("Error while cleaning mount %v", err)
	}

	// Create file in mount 1.
	commonFile, err := os.Create(getFilePathInMount1())
	if err != nil {
		t.Errorf("Error creating common file %v", err)
	}
	operations.CloseFile(commonFile)
}

func compareGCSObjContent(t *testing.T, expectedContent string) {
	actualContent, err := operations.GetGCSObject(getGCSObjPath())
	if err != nil {
		t.Errorf("Error while getting object from GCS %v", err)
	}

	if actualContent != expectedContent {
		t.Errorf("The content in GCS object: %s is different from expected content %s", actualContent, expectedContent)
	}
}

func TestW1OpensW2OpensW1WritesFlushesW2WritesFlushes(t *testing.T) {
	cleanAndCreateCommonFileInTestBucket(t)
	//w1 opens
	w1File := openFile(t, getFilePathInMount1())
	defer operations.CloseFile(w1File)
	//w2 opens
	w2File := openFile(t, getFilePathInMount2())
	defer operations.CloseFile(w2File)
	dataToWrite1 := "sample text to write in file by W1\n"
	// w1 writes
	writeToFile(t, w1File, dataToWrite1)
	// w1 flushes
	syncFile(t, w1File)
	// Check if content is flushed to GCS
	compareGCSObjContent(t, dataToWrite1)

	// w2 tries writing when w1 has already flushed. w2 should fail in the read
	// that happens in the write flow to load file content from GCS into disk.
	dataToWrite2 := "sample text to write in file by W2\n"
	_, err := w2File.WriteString(dataToWrite2)

	// Currently, the read flow returns ENOENT error when the object to be read is
	// clobbered.
	ExpectEq(true, strings.Contains(err.Error(), "no such file or directory"))
	// Ensure the object in GCS is not modified.
	compareGCSObjContent(t, dataToWrite1)
}

// This tests the case of clobbered error when two writers try to flush to same
// object concurrently.
func TestW1OpensWritesW2OpensWritesW1FlushesW2Flushes(t *testing.T) {
	cleanAndCreateCommonFileInTestBucket(t)
	//w1 opens
	w1File := openFile(t, getFilePathInMount1())
	defer operations.CloseFile(w1File)
	// w1 writes
	dataToWrite1 := "sample text to write in file by W1\n"
	writeToFile(t, w1File, dataToWrite1)
	//w2 opens
	w2File := openFile(t, getFilePathInMount2())
	defer operations.CloseFile(w2File)
	// w2 writes
	dataToWrite2 := "sample text to write in file by W2\n"
	writeToFile(t, w2File, dataToWrite2)
	// w1 flushes
	syncFile(t, w1File)

	// w2 tries to flush the file but gets I/O error because the file in the GCS
	// has been clobbered by w1.
	err := w2File.Sync()

	ExpectEq(true, strings.Contains(err.Error(), InputOutputErrorMsg))
	// check the GCS object still has the contents written by w1.
	compareGCSObjContent(t, dataToWrite1)
}

// This tests the case of precondition error when two writers concurrently flush
// to same object.
func TestW1OpensWritesW2OpensWritesW1W2Flushes(t *testing.T) {
	cleanAndCreateCommonFileInTestBucket(t)
	//w1 opens
	w1File := openFile(t, getFilePathInMount1())
	defer operations.CloseFile(w1File)
	// w1 writes (writing large file so that it takes some time to flush to GCS)
	dataToWrite1 := strings.Repeat("1", TwoHundredMiB)
	writeToFile(t, w1File, dataToWrite1)
	//w2 opens
	w2File := openFile(t, getFilePathInMount2())
	defer operations.CloseFile(w2File)
	// w2 writes
	dataToWrite2 := strings.Repeat("2", TwoHundredMiB)
	writeToFile(t, w2File, dataToWrite2)
	// Go routines for W1 and W2 for concurrent flushing.
	var err1, err2 error
	var wg sync.WaitGroup
	wg.Add(2)
	syncW1 := func() {
		err1 = w1File.Sync()
		wg.Done()
	}
	syncW2 := func() {
		err2 = w2File.Sync()
		wg.Done()
	}

	// Both W1 and W2 tries to flush concurrently but one of them should fail with
	// I/O error. Because the file size they are flushing is large, it is expected
	// that both will pass clobbered check and the I/O will come because of
	// precondition error
	go syncW1()
	go syncW2()
	wg.Wait()

	// if W1 syncs successfully, the other one should fail and vice-versa
	if err1 == nil {
		ExpectEq(true, strings.Contains(err2.Error(), InputOutputErrorMsg))
		compareGCSObjContent(t, dataToWrite1)
	} else {
		ExpectEq(true, strings.Contains(err1.Error(), InputOutputErrorMsg))
		compareGCSObjContent(t, dataToWrite2)
	}
}

func TestW1OpensWritesFlushesW2OpensWritesFlushes(t *testing.T) {
	cleanAndCreateCommonFileInTestBucket(t)
	//w1 opens
	w1File := openFile(t, getFilePathInMount1())
	defer operations.CloseFile(w1File)
	// w1 writes
	dataToWrite1 := "sample text to write in file by W1\n"
	writeToFile(t, w1File, dataToWrite1)
	// w1 flushes
	syncFile(t, w1File)
	compareGCSObjContent(t, dataToWrite1)
	// w2 opens
	w2File := openFile(t, getFilePathInMount2())
	defer operations.CloseFile(w2File)
	// w2 writes
	dataToWrite2 := "sample text to write in file by W2\n"
	writeToFile(t, w2File, dataToWrite2)

	// w2 flushes
	err := w2File.Sync()

	// w2 flush should pass as w2 opened the file after w1 wrote and flushed it.
	ExpectEq(nil, err)
	// GCS object should have content written by w2
	compareGCSObjContent(t, dataToWrite2)
}
