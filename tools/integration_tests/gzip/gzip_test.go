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
	"compress/gzip"
	"crypto/md5"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const TempFileStrLine = "This is a test file"

// Creates a temporary file (name-collision-safe) in /tmp with given size in bytes.
// If gzipCompress is true, output file is a gzip. The fileSize is the size of the
// original uncompressed content, and the actual output file size will be
// that of the compressed file and thus different from fileSize.
// Caller is responsible for deleting when done.
// TODO: move this to integration_tests/util/something
func createLocalTempFile(fileSize int, gzipCompress bool) (path string, err error) {
	if fileSize <= 0 {
		return "", fmt.Errorf("unsupported fileSize: %d", fileSize)
	}

	var contentBuilder strings.Builder

	const tempStr = TempFileStrLine + "\n"

	for ; fileSize >= len(tempStr); fileSize -= len(tempStr) {
		contentBuilder.WriteString(tempStr)
	}

	if fileSize > 0 {
		fmt.Println("fileSize remaining = ", fileSize)
		contentBuilder.WriteString(tempStr[0:fileSize])
	}

	fileSize = contentBuilder.Len()
	content := contentBuilder.String()

	filenameTemplate := "testfile-*.txt"
	if gzipCompress {
		filenameTemplate += ".gz"
	}

	f, err := os.CreateTemp("/tmp", filenameTemplate)
	if err != nil {
		return "", err
	} else if f == nil {
		return "", fmt.Errorf("nil file handle returned from os.CreateTemp")
	}

	filepath := f.Name()

	defer f.Close()

	if gzipCompress {
		w := gzip.NewWriter(f)
		n, err := w.Write([]byte(content))
		if err != nil {
			return "", err
		} else if n != fileSize {
			return "", fmt.Errorf("failed to write to gzip file %s. Content-size: %d bytes, wrote = %d bytes", filepath, fileSize, n)
		}

		defer w.Close()
	} else {
		n, err := f.WriteString(content)
		if err != nil {
			return "", err
		} else if n != fileSize {
			return "", fmt.Errorf("failed to write to text file %s. Content-size: %d bytes, wrote = %d bytes", filepath, fileSize, n)
		}
	}

	return filepath, nil
}

// Finds if two files have identical content
// Needs read permission on files and files
// to exist.
// Returns 0 if no error and files match
// Returns 1 if files don't match
// Returns 2 if any error
// TODO: move this to integration_tests/util/something
func DiffFiles(file1, file2 string) (diffval int, err error) {
	bytes1, err := os.ReadFile(file1)
	if err != nil || bytes1 == nil {
		return 2, fmt.Errorf("Failed to read file %s", file1)
	}

	bytes2, err := os.ReadFile(file2)
	if err != nil || bytes2 == nil {
		return 2, fmt.Errorf("Failed to read file %s", file2)
	}

	if len(bytes1) != len(bytes2) {
		return 1, fmt.Errorf("Files don't match in size: %s (%d bytes), %s (%d bytes)", file1, len(bytes1), file2, len(bytes2))
	} else if md5.Sum(bytes1) != md5.Sum(bytes2) {
		return 1, fmt.Errorf("Files don't match in content: %s, %s", file1, file2)
	}

	return 0, nil
}

const (
	seqReadSizeMb   = 1
	textContentSize = 10 * 1e6 * seqReadSizeMb

	textContentWithContentEncodingWithNoTransformFilename    = "textContentWithContentEncodingWithNoTransform.txt"
	textContentWithContentEncodingWithoutNoTransformFilename = "textContentWithContentEncodingWithoutNoTransform.txt"

	gzipContentWithoutContentEncodingFilename = "gzipContentWithoutContentEncoding.txt.gz"

	gzipContentWithContentEncodingWithNoTransformFilename    = "gzipContentWithContentEncodingWithNoTransform.txt.gz"
	gzipContentWithContentEncodingWithoutNoTransformFilename = "gzipContentWithContentEncodingWithoutNoTransform.txt.gz"

	gzipTestResPath = "" // setting this non-empty is causing unexpected failures. needs to be debugged.
)

func setup_testdata(m *testing.M) error {
	fmds := []struct {
		filename                    string
		filesize                    int
		keepCacheControlNoTransform bool // if true, no-transform is reset as ''
		enableGzipEncodedContent    bool // if true, original file content is gzip-encoded
		enableGzipContentEncoding   bool // if true, the content is uploaded as gsutil cp -Z i.e. with content-encoding: gzip header in GCS
	}{
		{
			filename:                    textContentWithContentEncodingWithNoTransformFilename,
			filesize:                    textContentSize,
			keepCacheControlNoTransform: true,
			enableGzipEncodedContent:    false,
			enableGzipContentEncoding:   true,
		},
		{
			filename:                    textContentWithContentEncodingWithoutNoTransformFilename,
			filesize:                    textContentSize,
			keepCacheControlNoTransform: false,
			enableGzipEncodedContent:    false,
			enableGzipContentEncoding:   true,
		},
		{
			filename:                    gzipContentWithoutContentEncodingFilename,
			filesize:                    textContentSize,
			keepCacheControlNoTransform: true, // it's a don't care in this case
			enableGzipEncodedContent:    true,
			enableGzipContentEncoding:   false,
		}, {
			filename:                    gzipContentWithContentEncodingWithNoTransformFilename,
			filesize:                    textContentSize,
			keepCacheControlNoTransform: true,
			enableGzipEncodedContent:    true,
			enableGzipContentEncoding:   true,
		}, {
			filename:                    gzipContentWithContentEncodingWithoutNoTransformFilename,
			filesize:                    textContentSize,
			keepCacheControlNoTransform: false,
			enableGzipEncodedContent:    true,
			enableGzipContentEncoding:   true,
		},
	}

	for _, fmd := range fmds {
		var localFilePath string
		localFilePath, err := createLocalTempFile(fmd.filesize, fmd.enableGzipEncodedContent)
		if err != nil {
			return err
		}

		defer os.Remove(localFilePath)

		// upload to the test-bucket for testing
		gcsObjectPath := path.Join(setup.TestBucket(), gzipTestResPath, fmd.filename)

		if fmd.enableGzipContentEncoding {
			setup.RunScriptForTestData("testdata/upload_to_gcs.sh", localFilePath, gcsObjectPath, "1")
		} else {
			setup.RunScriptForTestData("testdata/upload_to_gcs.sh", localFilePath, gcsObjectPath)
		}

		if !fmd.keepCacheControlNoTransform {
			setup.RunScriptForTestData("testdata/remove_notransform_from_gcs_object.sh", gcsObjectPath)
		}
	}

	return nil
}

func destroy_testdata(m *testing.M) error {
	// TODO: Delete all objects starting with path.Join(setup.TestBucket(), gzipTestResPath)
	return nil
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// currently commented out to reduce runtime. Uncomment before merging.
	//flags := [][]string{{"--sequential-read-size-mb=1"}, {"--sequential-read-size-mb=1 --enable-storage-client-library=false"}}

	flags := [][]string{{"--sequential-read-size-mb=" + fmt.Sprint(seqReadSizeMb)}}

	// setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	if setup.MountedDirectory() != "" {
		log.Print("--mountedDirectory is ignored for gzip tests")
	}

	if setup.TestBucket() == "" {
		log.Print("Please pass the name of bucket mounted at mountedDirectory to --testBucket flag.")
		os.Exit(1)
	}

	// Run tests for testBucket
	setup.SetUpTestDirForTestBucketFlag()

	err := setup_testdata(m)
	if err != nil {
		fmt.Printf("Failed to setup test: %v", err)
		os.Exit(1)
	}

	successCode := static_mounting.RunTests(flags, m)

	setup.RemoveBinFileCopiedForTesting()

	err = destroy_testdata(m)
	if err != nil {
		fmt.Printf("Failed to setup test: %v", err)
		os.Exit(1)
	}

	os.Exit(successCode)
}
