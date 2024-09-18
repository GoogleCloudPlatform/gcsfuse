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
)

var (
	gcsObjectsToBeDeletedEventually []string
	storageClient                   *storage.Client
)

func setup_testdata(m *testing.M, ctx context.Context) error {
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
func createContentOfSize(contentSize int) (string, error) {
	if contentSize <= 0 {
		return "", fmt.Errorf("unsupported fileSize: %d", contentSize)
	}
	const tempStr = "This is a test file\n"
	iter := (contentSize + len(tempStr) - 1) / len(tempStr)
	str := strings.Repeat(tempStr, iter)
	return str[:contentSize], nil
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	var err error
	ctx := context.Background()
	if storageClient, err = client.CreateStorageClient(ctx); err != nil {
		log.Fatalf("Error creating storage client: %v\n", err)
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
		log.Fatal("Please pass the name of bucket mounted at mountedDirectory to --testBucket flag.")
	}

	err = setup_testdata(m, ctx)
	if err != nil {
		log.Fatalf("Failed to setup test data: %v", err)
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
