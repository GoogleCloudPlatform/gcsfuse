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

// Provides integration tests for gzip objects in gcsfuse mounts.
package gzip_test

import (
	"os"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

// Returns size of a give GCS object with path (without 'gs://').
// Uses 'gsutil du -s'.
// Fails if the object doesn't exist, the passed path is prefixied with
// TODO: move this to integration_tests/util/something
func GetGcsObjectSize(gcsObjPath string) (size int, err error) {
	gcsObjPath = strings.TrimSpace(gcsObjPath)
	stdout, err := setup.RunScriptForOutput("testdata/get_gcs_object_size.sh", gcsObjPath)
	if err != nil {
		return 0, err
	}

	gcsObjectSize, err := strconv.Atoi(strings.TrimSpace(string(stdout)))
	if err != nil {
		return gcsObjectSize, err
	}

	return gcsObjectSize, nil
}

func genericFileSizeStatTest(t *testing.T, filename string) {
	mountedFilePath := path.Join(setup.MntDir(), gzipTestResPath, filename)
	gcsObjectPath := path.Join(setup.TestBucket(), gzipTestResPath, filename)
	gcsObjectSize, err := GetGcsObjectSize(gcsObjectPath)
	if err != nil {
		t.Fatalf("Failed to get size of gcs object %s: %v\n", gcsObjectPath, err)
	}

	fi, err := os.Stat(mountedFilePath)
	if err != nil {
		t.Fatalf("Failed to get stat info of mounted file %s: %v\n", mountedFilePath, err)
	}

	if fi.Size() != int64(gcsObjectSize) {
		t.Fatalf("Size of file mounted through gcsfuse (%s, %d) doesn't match the size of the file on GCS (%s, %d)",
			mountedFilePath, fi.Size(), gcsObjectPath, gcsObjectSize)
	}
}

func genericFullFileReadTest(t *testing.T, filename string) {
	mountedFilePath := path.Join(setup.MntDir(), gzipTestResPath, filename)
	data, err := os.ReadFile(mountedFilePath)
	if err != nil {
		t.Errorf("Failed to read mounted file %s: %v", mountedFilePath, err)
	}

	gcsObjectPath := path.Join(setup.TestBucket(), gzipTestResPath, filename)
	gcsObjectSize, err := GetGcsObjectSize(gcsObjectPath)
	if err != nil {
		t.Fatalf("Failed to get size of gcs object %s: %v\n", gcsObjectPath, err)
	}
	if len(data) != gcsObjectSize {
		t.Fatalf("Read size for %s (%d bytes) didn't match the size of the GCS object gs://%s (%d bytes)", mountedFilePath, len(data), gcsObjectPath, gcsObjectSize)
	}
	if strings.Contains(string(data), TempFileStrLine) {
		t.Fatalf("Read data contains '%s' but shouldn't", TempFileStrLine)
	}
}

func genericRangedReadTest(t *testing.T, filename string) {
	mountedFilePath := path.Join(setup.MntDir(), gzipTestResPath, filename)

	gcsObjectPath := path.Join(setup.TestBucket(), gzipTestResPath, filename)
	gcsObjectSize, err := GetGcsObjectSize(gcsObjectPath)
	if err != nil {
		t.Fatalf("Failed to get size of gcs object %s: %v\n", gcsObjectPath, err)
	}

	readSize := gcsObjectSize / 10
	readOffset := readSize / 10
	f, err := os.Open(mountedFilePath)
	if err != nil || f == nil {
		t.Fatalf("Failed to open local mounted file %s: %v", mountedFilePath, err)
	}

	buf := make([]byte, readSize)
	for _, offsetMultiplier := range []int64{1, 3, 5, 7, 9} {
		n, err := f.ReadAt(buf, offsetMultiplier*int64(readOffset))
		if err != nil {
			t.Fatalf("Failed to read mounted file %s: %v", mountedFilePath, err)
		}

		if readSize != n {
			t.Fatalf("Read size for %s (%d bytes) didn't match the size of the requested read-size (%d bytes)", mountedFilePath, n, readSize)
		}

		if strings.Contains(string(buf), TempFileStrLine) {
			t.Fatalf("Read buffer contains '%s' but shouldn't", TempFileStrLine)
		}
	}
}

func TestGzipEncodedTextFileWithNoTransformSize(t *testing.T) {
	genericFileSizeStatTest(t, textContentWithContentEncodingWithNoTransformFilename)
}

func TestGzipEncodedTextFileWithNoTransformFullRead(t *testing.T) {
	genericFullFileReadTest(t, textContentWithContentEncodingWithNoTransformFilename)
}

func TestGzipEncodedTextFileWithNoTransformRangedRead(t *testing.T) {
	genericRangedReadTest(t, textContentWithContentEncodingWithNoTransformFilename)
}

func TestGzipEncodedTextFileWithoutNoTransformSize(t *testing.T) {
	genericFileSizeStatTest(t, textContentWithContentEncodingWithoutNoTransformFilename)
}

func TestGzipEncodedTextFileWithoutNoTransformFullRead(t *testing.T) {
	genericFullFileReadTest(t, textContentWithContentEncodingWithoutNoTransformFilename)
}

func TestGzipEncodedTextFileWithoutNoTransformRangedRead(t *testing.T) {
	genericRangedReadTest(t, textContentWithContentEncodingWithoutNoTransformFilename)
}

func TestGzipUnencodedGzipFileSize(t *testing.T) {
	genericFileSizeStatTest(t, gzipContentWithoutContentEncodingFilename)
}

func TestGzipUnencodedGzipFileFullRead(t *testing.T) {
	genericFullFileReadTest(t, gzipContentWithoutContentEncodingFilename)
}

func TestGzipUnencodedGzipFileRangedRead(t *testing.T) {
	genericRangedReadTest(t, gzipContentWithoutContentEncodingFilename)
}

func TestGzipEncodedGzipFileWithNoTransformSize(t *testing.T) {
	genericFileSizeStatTest(t, gzipContentWithContentEncodingWithNoTransformFilename)
}

func TestGzipEncodedGzipFileWithNoTransformFullRead(t *testing.T) {
	genericFullFileReadTest(t, gzipContentWithContentEncodingWithNoTransformFilename)
}

func TestGzipEncodedGzipFileWithNoTransformRangedRead(t *testing.T) {
	genericRangedReadTest(t, gzipContentWithContentEncodingWithNoTransformFilename)
}

func TestGzipEncodedGzipFileWithoutNoTransformSize(t *testing.T) {
	genericFileSizeStatTest(t, gzipContentWithContentEncodingWithoutNoTransformFilename)
}

func TestGzipEncodedGzipFileWithoutNoTransformFullRead(t *testing.T) {
	genericFullFileReadTest(t, gzipContentWithContentEncodingWithoutNoTransformFilename)
}

func TestGzipEncodedGzipFileWithoutNoTransformRangedRead(t *testing.T) {
	genericRangedReadTest(t, gzipContentWithContentEncodingWithoutNoTransformFilename)
}
