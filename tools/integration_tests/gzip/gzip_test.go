// Copyright 2023 Google LLC
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
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const (
	SeqReadSizeMb   = 1
	TextContentSize = 10 * 1e6 * SeqReadSizeMb

	TextContentWithContentEncodingWithNoTransformFilename    = "textContentWithContentEncodingWithNoTransform.txt"
	TextContentWithContentEncodingWithoutNoTransformFilename = "textContentWithContentEncodingWithoutNoTransform.txt"

	GzipContentWithoutContentEncodingFilename = "gzipContentWithoutContentEncoding.txt.gz"

	GzipContentWithContentEncodingWithNoTransformFilename    = "gzipContentWithContentEncodingWithNoTransform.txt.gz"
	GzipContentWithContentEncodingWithoutNoTransformFilename = "gzipContentWithContentEncodingWithoutNoTransform.txt.gz"

	TextContentWithContentEncodingWithNoTransformToOverwrite    = "TextContentWithContentEncodingWithNoTransformToOverwrite.txt"
	TextContentWithContentEncodingWithoutNoTransformToOverwrite = "TextContentWithContentEncodingWithoutNoTransformToOverwrite.txt"

	GzipContentWithoutContentEncodingToOverwrite = "GzipContentWithoutContentEncodingToOverwrite.txt.gz"

	GzipContentWithContentEncodingWithNoTransformToOverwrite    = "GzipContentWithContentEncodingWithNoTransformToOverwrite.txt.gz"
	GzipContentWithContentEncodingWithoutNoTransformToOverwrite = "GzipContentWithContentEncodingWithoutNoTransformToOverwrite.txt.gz"

	TestBucketPrefixPath = "gzip"
	TempFileStrLine      = "This is a test file"
)

var (
	gcsObjectsToBeDeletedEventually []string
	storageClient                   *storage.Client
	ctx                             context.Context
)

func setup_testdata(m *testing.M) error {
	fmds := []struct {
		filename                    string
		filesize                    int
		keepCacheControlNoTransform bool // if true, no-transform is reset as ''
		enableGzipEncodedContent    bool // if true, original file content is gzip-encoded
		enableGzipContentEncoding   bool // if true, the content is uploaded as gcloud storage cp -Z i.e. with content-encoding: gzip header in GCS
	}{
		{
			filename:                    TextContentWithContentEncodingWithNoTransformFilename,
			filesize:                    TextContentSize,
			keepCacheControlNoTransform: true,
			enableGzipEncodedContent:    false,
			enableGzipContentEncoding:   true,
		},
		{
			filename:                    TextContentWithContentEncodingWithoutNoTransformFilename,
			filesize:                    TextContentSize,
			keepCacheControlNoTransform: false,
			enableGzipEncodedContent:    false,
			enableGzipContentEncoding:   true,
		},
		{
			filename:                    GzipContentWithoutContentEncodingFilename,
			filesize:                    TextContentSize,
			keepCacheControlNoTransform: true, // it's a don't care in this case
			enableGzipEncodedContent:    true,
			enableGzipContentEncoding:   false,
		}, {
			filename:                    GzipContentWithContentEncodingWithNoTransformFilename,
			filesize:                    TextContentSize,
			keepCacheControlNoTransform: true,
			enableGzipEncodedContent:    true,
			enableGzipContentEncoding:   true,
		}, {
			filename:                    GzipContentWithContentEncodingWithoutNoTransformFilename,
			filesize:                    TextContentSize,
			keepCacheControlNoTransform: false,
			enableGzipEncodedContent:    true,
			enableGzipContentEncoding:   true,
		},
		{
			filename:                    TextContentWithContentEncodingWithNoTransformToOverwrite,
			filesize:                    TextContentSize,
			keepCacheControlNoTransform: true,
			enableGzipEncodedContent:    false,
			enableGzipContentEncoding:   true,
		},
		{
			filename:                    TextContentWithContentEncodingWithoutNoTransformToOverwrite,
			filesize:                    TextContentSize,
			keepCacheControlNoTransform: false,
			enableGzipEncodedContent:    false,
			enableGzipContentEncoding:   true,
		},
		{
			filename:                    GzipContentWithoutContentEncodingToOverwrite,
			filesize:                    TextContentSize,
			keepCacheControlNoTransform: true, // it's a don't care in this case
			enableGzipEncodedContent:    true,
			enableGzipContentEncoding:   false,
		}, {
			filename:                    GzipContentWithContentEncodingWithNoTransformToOverwrite,
			filesize:                    TextContentSize,
			keepCacheControlNoTransform: true,
			enableGzipEncodedContent:    true,
			enableGzipContentEncoding:   true,
		}, {
			filename:                    GzipContentWithContentEncodingWithoutNoTransformToOverwrite,
			filesize:                    TextContentSize,
			keepCacheControlNoTransform: false,
			enableGzipEncodedContent:    true,
			enableGzipContentEncoding:   true,
		},
	}

	for _, fmd := range fmds {
		content, err := createContentOfSize(fmd.filesize)
		if err != nil {
			return fmt.Errorf("failed to create content for testing: %w", err)
		}
		localFilePath, err := operations.CreateLocalTempFile(content, fmd.enableGzipEncodedContent)
		if err != nil {
			return fmt.Errorf("failed to create local file: %w", err)
		}

		defer os.Remove(localFilePath)

		// upload to the test-bucket for testing
		objectPrefixPath := path.Join(TestBucketPrefixPath, fmd.filename)
		ctx := context.Background()

		err = client.UploadGcsObject(ctx, storageClient, localFilePath, setup.TestBucket(), objectPrefixPath, fmd.enableGzipContentEncoding)
		if err != nil {
			return err
		}

		gcsObjectPath := path.Join(setup.TestBucket(), objectPrefixPath)
		gcsObjectsToBeDeletedEventually = append(gcsObjectsToBeDeletedEventually, gcsObjectPath)

		if !fmd.keepCacheControlNoTransform {
			err = operations.ClearCacheControlOnGcsObject(gcsObjectPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func destroy_testdata(m *testing.M) error {
	for _, gcsObjectPath := range gcsObjectsToBeDeletedEventually {
		err := operations.DeleteGcsObject(gcsObjectPath)
		if err != nil {
			return fmt.Errorf("Failed to delete gcs object gs://%s", gcsObjectPath)
		}
	}

	return nil
}

// createContentOfSize generates a string of the specified content size in bytes.
// Failure cases:
// 1. contentSize <= 0
func createContentOfSize(contentSize int) (string, error) {
	// fail if contentSize <= 0
	if contentSize <= 0 {
		return "", fmt.Errorf("unsupported fileSize: %d", contentSize)
	}

	// Create text-content of given size.
	// strings.builder is used as opposed to string appends
	// as this is much more efficient when multiple concatenations
	// are required.
	var contentBuilder strings.Builder
	const tempStr = TempFileStrLine + "\n"

	for ; contentSize >= len(tempStr); contentSize -= len(tempStr) {
		n, err := contentBuilder.WriteString(tempStr)
		if err != nil {
			return "", fmt.Errorf("failed to write to string builder: %w", err)
		}
		if n != len(tempStr) {
			return "", fmt.Errorf("unexpected number of bytes written: expected %d, got %d", len(tempStr), n)
		}
	}

	if contentSize > 0 {
		n, err := contentBuilder.WriteString(tempStr[0:contentSize])
		if err != nil {
			return "", fmt.Errorf("failed to write to string builder: %w", err)
		}
		if n != contentSize {
			return "", fmt.Errorf("unexpected number of bytes written: expected %d, got %d", contentSize, n)
		}
	}

	content := contentBuilder.String()
	return content, nil
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	var err error
	ctx = context.Background()
	storageClient, err = client.CreateStorageClient(ctx)
	if err != nil {
		log.Printf("Error creating storage client: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := storageClient.Close(); err != nil {
			log.Printf("failed to close storage client: %v", err)
		}
	}()

	commonFlags := []string{"--sequential-read-size-mb=" + fmt.Sprint(SeqReadSizeMb), "--implicit-dirs"}
	flags := [][]string{commonFlags}

	if !testing.Short() {
		gRPCFlags := append(commonFlags, "--client-protocol=grpc")
		flags = append(flags, gRPCFlags)
	}

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	if setup.TestBucket() == "" && setup.MountedDirectory() != "" {
		log.Print("Please pass the name of bucket mounted at mountedDirectory to --testBucket flag.")
		os.Exit(1)
	}

	err = setup_testdata(m)
	if err != nil {
		log.Printf("Failed to setup test data: %v", err)
		os.Exit(1)
	}

	defer func() {
		err := destroy_testdata(m)
		if err != nil {
			log.Printf("Failed to destoy gzip test data: %v", err)
		}
	}()

	// Run tests for mountedDirectory only if --mountedDirectory flag is set.
	setup.RunTestsForMountedDirectoryFlag(m)

	// Run tests for testBucket
	setup.SetUpTestDirForTestBucketFlag()

	successCode := static_mounting.RunTests(flags, m)

	os.Exit(successCode)
}
