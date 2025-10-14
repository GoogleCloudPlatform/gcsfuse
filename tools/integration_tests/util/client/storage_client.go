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

package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"cloud.google.com/go/storage/experimental"
	"github.com/googleapis/gax-go/v2"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	storagev1 "google.golang.org/api/storage/v1"
)

func CreateHttp1StorageClient(ctx context.Context) (*storage.Client, error) {
	defaultTokenSrc, err := google.DefaultTokenSource(ctx, storagev1.DevstorageFullControlScope)
	if err != nil {
		return nil, fmt.Errorf("unable to create default token source: %w", err)
	}

	httpClient := &http.Client{
		Transport: &oauth2.Transport{
			Base: &http.Transport{
				Proxy:               http.ProxyFromEnvironment,
				MaxIdleConns:        0, // No connection limit.
				MaxIdleConnsPerHost: 100,
				TLSNextProto:        make(map[string]func(authority string, c *tls.Conn) http.RoundTripper), // Disables HTTP/2 transport.
			},
			Source: defaultTokenSrc,
		},
		Timeout: 0, // No timeout.
	}

	return storage.NewClient(ctx, option.WithHTTPClient(httpClient))
}

func CreateStorageClient(ctx context.Context) (client *storage.Client, err error) {
	// Create new storage client.
	if setup.TestOnTPCEndPoint() {
		var ts oauth2.TokenSource
		// Set up the TPC endpoint and provide a token source for authentication.
		ts, err = getTokenSrc("/tmp/sa.key.json")
		if err != nil {
			return nil, fmt.Errorf("unable to fetch token-source for TPC: %w", err)
		}
		client, err = storage.NewClient(ctx, option.WithEndpoint("storage.apis-tpczero.goog:443"), option.WithTokenSource(ts))
	} else {
		if setup.IsZonalBucketRun() {
			client, err = storage.NewGRPCClient(ctx, experimental.WithGRPCBidiReads())
		} else {
			client, err = CreateHttp1StorageClient(ctx)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %w", err)
	}
	// RetryAlways causes all operations to be retried when the service returns
	// transient error, regardless of idempotency considerations. Since the
	// concurrent execution of our CI/CD tests (VMs, threads) doesn't share any
	// cloud-storage resources, hence it's safe to disregard idempotency.
	client.SetRetry(
		storage.WithBackoff(gax.Backoff{
			Max:        30 * time.Second,
			Multiplier: 2,
		}),
		storage.WithPolicy(storage.RetryAlways),
		storage.WithErrorFunc(storageutil.ShouldRetry),
		storage.WithMaxAttempts(5))
	return client, nil
}

func getTokenSrc(path string) (tokenSrc oauth2.TokenSource, err error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ReadFile(%q): %w", path, err)
	}

	// Create a config struct based on its contents.
	ts, err := google.JWTAccessTokenSourceWithScope(contents, storagev1.DevstorageFullControlScope)
	if err != nil {
		return nil, fmt.Errorf("JWTConfigFromJSON: %w", err)
	}
	return ts, err
}

// ReadObjectFromGCS downloads the object from GCS and returns the data.
func ReadObjectFromGCS(ctx context.Context, client *storage.Client, object string) (string, error) {
	bucket, object := setup.GetBucketAndObjectBasedOnTypeOfMount(object)

	if client == nil {
		return "", fmt.Errorf("client is nil")
	}
	// Create storage reader to read from GCS.
	rc, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return "", fmt.Errorf("Object(%q).NewReader: %w", object, err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("io.ReadAll failed: %v", err)
	}

	return string(content), nil
}

// ReadChunkFromGCS downloads the object chunk from GCS and returns the data.
func ReadChunkFromGCS(ctx context.Context, client *storage.Client, object string,
	offset, size int64) (string, error) {
	bucket, object := setup.GetBucketAndObjectBasedOnTypeOfMount(object)

	// Create storage reader to read from GCS.
	rc, err := client.Bucket(bucket).Object(object).NewRangeReader(ctx, offset, size)
	if err != nil {
		return "", fmt.Errorf("Object(%q).NewReader: %w", object, err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("io.ReadAll failed: %v", err)
	}

	return string(content), nil
}

// NewWriter is a wrapper over storage.NewWriter which
// extends support to zonal buckets.
func NewWriter(ctx context.Context, o *storage.ObjectHandle, client *storage.Client) (wc *storage.Writer, err error) {
	wc = o.NewWriter(ctx)
	wc.FinalizeOnClose = true

	// Changes specific to zonal bucket
	var attrs *storage.BucketAttrs
	attrs, err = client.Bucket(o.BucketName()).Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get attributes for bucket %q: %w", o.BucketName(), err)
	}
	if attrs.StorageClass == "RAPID" {
		if setup.IsZonalBucketRun() {
			// Zonal bucket writers require append-flag to be set.
			wc.Append = true
			// Zonal buckets with rapid appends should not finalize on close.
			wc.FinalizeOnClose = false
		} else {
			return nil, fmt.Errorf("found zonal bucket %q in non-zonal e2e test run (--zonal=false)", o.BucketName())
		}
	}

	return
}

func WriteToObject(ctx context.Context, client *storage.Client, object, content string, precondition storage.Conditions) error {
	bucket, object := setup.GetBucketAndObjectBasedOnTypeOfMount(object)

	o := client.Bucket(bucket).Object(object)
	if !reflect.DeepEqual(precondition, storage.Conditions{}) {
		o = o.If(precondition)
	}

	// Upload an object with storage.Writer.
	wc, err := NewWriter(ctx, o, client)
	if err != nil {
		return fmt.Errorf("Failed to open writer for object %q: %w", object, err)
	}
	if _, err := io.WriteString(wc, content); err != nil {
		return fmt.Errorf("io.WriteString failed for object %q: %w", object, err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close failed for object %q: %w", object, err)
	}
	operations.WaitForSizeUpdate(setup.IsZonalBucketRun(), operations.WaitDurationAfterCloseZB)

	return nil
}

// CreateObjectOnGCS creates an object with given name and content on GCS.
func CreateObjectOnGCS(ctx context.Context, client *storage.Client, object, content string) error {
	return WriteToObject(ctx, client, object, content, storage.Conditions{DoesNotExist: true})
}

func CreateFinalizedObjectOnGCS(ctx context.Context, client *storage.Client, object, content string) error {
	bucket, object := setup.GetBucketAndObjectBasedOnTypeOfMount(object)
	o := client.Bucket(bucket).Object(object)

	// Upload an object with storage.Writer with finalizeOnClose=true
	wc := o.NewWriter(ctx)
	wc.Append = true
	wc.FinalizeOnClose = true
	if _, err := io.WriteString(wc, content); err != nil {
		return fmt.Errorf("io.WriteString failed for object %q: %w", object, err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close failed for object %q: %w", object, err)
	}
	operations.WaitForSizeUpdate(setup.IsZonalBucketRun(), operations.WaitDurationAfterCloseZB)
	return nil
}

// CreateStorageClientWithCancel creates a new storage client with a cancelable context and returns a function that can be used to cancel the client's operations
func CreateStorageClientWithCancel(ctx *context.Context, storageClient **storage.Client) func() error {
	var err error
	var cancel context.CancelFunc
	*ctx, cancel = context.WithCancel(*ctx)
	*storageClient, err = CreateStorageClient(*ctx)
	if err != nil {
		log.Fatalf("client.CreateStorageClient: %v", err)
	}
	// Return func to close storage client and release resources.
	return func() error {
		err := (*storageClient).Close()
		if err != nil {
			return fmt.Errorf("failed to close storage client: %v", err)
		}
		defer cancel()
		return nil
	}
}

// DownloadObjectFromGCS downloads an object to a local file.
func DownloadObjectFromGCS(gcsFile string, destFileName string, t *testing.T) error {
	bucket, gcsFile := setup.GetBucketAndObjectBasedOnTypeOfMount(gcsFile)

	ctx := context.Background()
	var storageClient *storage.Client
	closeStorageClient := CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			t.Errorf("closeStorageClient failed: %v", err)
		}
	}()
	f := operations.CreateFile(destFileName, setup.FilePermission_0600, t)
	defer operations.CloseFileShouldNotThrowError(t, f)

	rc, err := storageClient.Bucket(bucket).Object(gcsFile).NewReader(ctx)
	if err != nil {
		return fmt.Errorf("Object(%q).NewReader: %w", gcsFile, err)
	}
	defer rc.Close()

	if _, err := io.Copy(f, rc); err != nil {
		return fmt.Errorf("io.Copy: %w", err)
	}

	return nil
}

func DeleteObjectOnGCS(ctx context.Context, client *storage.Client, objectName string) error {
	bucket, _ := setup.GetBucketAndObjectBasedOnTypeOfMount("")

	// Get handle to the object
	object := client.Bucket(bucket).Object(objectName)

	// Delete the object
	err := object.Delete(ctx)
	if err != nil {
		return err
	}
	return nil
}

// DeleteAllObjectsWithPrefix deletes all objects with the specified prefix in a GCS bucket.
// It concurrently iterates through objects with the given prefix and deletes them using multiple goroutines,
// leveraging the number of CPU cores for optimal performance.
func DeleteAllObjectsWithPrefix(ctx context.Context, client *storage.Client, prefix string) error {
	bucket, _ := setup.GetBucketAndObjectBasedOnTypeOfMount("")

	// Get an object iterator
	query := &storage.Query{Prefix: prefix}
	objectItr := client.Bucket(bucket).Objects(ctx, query)

	// Create a buffered channel to receive errors from goroutines
	errChan := make(chan error, 100)

	// Determine the number of concurrent goroutines using CPU cores
	numCores := runtime.NumCPU()
	sem := make(chan struct{}, numCores) // Semaphore to limit concurrency

	var wg sync.WaitGroup

	// Iterate through objects with the specified prefix
	for {
		attrs, err := objectItr.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("error iterating through objects: %w", err)
		}

		wg.Add(1)
		sem <- struct{}{} // Acquire a semaphore slot
		go func(attrs *storage.ObjectAttrs) {
			defer func() {
				<-sem // Release the semaphore slot
				wg.Done()
			}()
			if err := DeleteObjectOnGCS(ctx, client, attrs.Name); err != nil {
				errChan <- fmt.Errorf("error deleting object %s: %w", attrs.Name, err)
			}
		}(attrs)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// MustDeleteAllObjectsWithPrefix wraps over DeleteAllObjectsWithPrefix and
// panics instead of returning error.
func MustDeleteAllObjectsWithPrefix(ctx context.Context, client *storage.Client, prefix string) {
	err := DeleteAllObjectsWithPrefix(ctx, client, prefix)
	if err != nil {
		panic(fmt.Sprintf("DeleteAllObjectsWithPrefix(%q) failed: %v", prefix, err))
	}
}

func StatObject(ctx context.Context, client *storage.Client, object string) (*storage.ObjectAttrs, error) {
	bucket, object := setup.GetBucketAndObjectBasedOnTypeOfMount(object)

	attrs, err := client.Bucket(bucket).Object(object).Attrs(ctx)
	if err != nil {
		return nil, err
	}
	return attrs, nil
}

// UploadGcsObjectWithPreconditions uploads a local file to a specified GCS bucket and object with given preconditions.
// Handles gzip compression if requested.
func UploadGcsObjectWithPreconditions(ctx context.Context, client *storage.Client, localPath, bucketName, objectName string, uploadGzipEncoded bool, preconditions *storage.Conditions) error {
	// Create a writer to upload the object.
	obj := client.Bucket(bucketName).Object(objectName)
	if preconditions != nil {
		obj = obj.If(*preconditions)
	}
	w, err := NewWriter(ctx, obj, client)
	if err != nil {
		return fmt.Errorf("failed to open writer for GCS object gs://%s/%s: %w", bucketName, objectName, err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			log.Printf("Failed to close GCS object gs://%s/%s: %v", bucketName, objectName, err)
		}
		operations.WaitForSizeUpdate(setup.IsZonalBucketRun(), operations.WaitDurationAfterCloseZB)
	}()

	filePathToUpload := localPath
	// Set content encoding if gzip compression is needed.
	if uploadGzipEncoded {
		data, err := os.ReadFile(localPath)
		if err != nil {
			return err
		}

		content := string(data)
		if filePathToUpload, err = operations.CreateLocalTempFile(content, true); err != nil {
			return fmt.Errorf("failed to create local gzip file from %s for upload to bucket: %w", localPath, err)
		}
		defer func() {
			if removeErr := os.Remove(filePathToUpload); removeErr != nil {
				log.Printf("Error removing temporary gzip file %s: %v", filePathToUpload, removeErr)
			}
		}()
	}

	// Open the local file for reading.
	f, err := operations.OpenFileAsReadonly(filePathToUpload)
	if err != nil {
		return fmt.Errorf("failed to open local file %s: %w", filePathToUpload, err)
	}
	defer operations.CloseFile(f)

	// Copy the file contents to the object writer.
	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("failed to copy file %s to gs://%s/%s: %w", localPath, bucketName, objectName, err)
	}
	return nil
}

// Get the object size of the GCS object.
func GetGcsObjectSize(ctx context.Context, client *storage.Client, object string) (int64, error) {
	attrs, err := StatObject(ctx, client, object)
	if err != nil {
		return -1, err
	}
	return attrs.Size, nil
}

// Clears cache-control attributes on given GCS object.
// Fails if the object doesn't exist or permission to modify object's metadata is not
// available.
func ClearCacheControlOnGcsObject(ctx context.Context, client *storage.Client, object string) error {
	attrs, err := StatObject(ctx, client, object)
	if err != nil {
		return err
	}
	attrs.CacheControl = ""

	return nil
}

// UploadGcsObject uploads a local file to a specified GCS bucket and object without any preconditions.
// Handles gzip compression if requested.
func UploadGcsObject(ctx context.Context, client *storage.Client, localPath, bucketName, objectName string, uploadGzipEncoded bool) error {
	return UploadGcsObjectWithPreconditions(ctx, client, localPath, bucketName, objectName, uploadGzipEncoded, nil)
}

func CopyFileInBucket(ctx context.Context, storageClient *storage.Client, srcfilePath, destFilePath, bucket string) {
	err := UploadGcsObject(ctx, storageClient, srcfilePath, bucket, destFilePath, false)
	if err != nil {
		log.Fatalf("Error while copying file %q to GCS object \"gs://%s/%s\" : %v", srcfilePath, bucket, destFilePath, err)
	}
}

func CopyFileInBucketWithPreconditions(ctx context.Context, storageClient *storage.Client, srcfilePath, destFilePath, bucket string, preconditions *storage.Conditions) {
	err := UploadGcsObjectWithPreconditions(ctx, storageClient, srcfilePath, bucket, destFilePath, false, preconditions)
	if err != nil {
		log.Fatalf("Error while copying file %q to GCS object \"gs://%s/%s\" : %v", srcfilePath, bucket, destFilePath, err)
	}
}

func DeleteBucket(ctx context.Context, client *storage.Client, bucketName string) error {
	bucket := client.Bucket(bucketName)

	// Iterate through objects and delete them
	query := &storage.Query{}
	it := bucket.Objects(ctx, query)
	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break // No more objects
		}
		if err != nil {
			log.Fatalf("Error iterating through objects: %v", err)
		}

		obj := bucket.Object(objAttrs.Name)
		err = obj.Delete(ctx)
		if err != nil {
			log.Fatalf("Failed to delete object %s: %v", objAttrs.Name, err)
		}
	}

	if err := bucket.Delete(ctx); err != nil {
		log.Printf("Bucket(%q).Delete: %v", bucketName, err)
		return err
	}
	return nil
}

func AppendableWriter(ctx context.Context, client *storage.Client, object string, precondition storage.Conditions) (*storage.Writer, error) {
	bucket, object := setup.GetBucketAndObjectBasedOnTypeOfMount(object)

	o := client.Bucket(bucket).Object(object)
	if !reflect.DeepEqual(precondition, storage.Conditions{}) {
		o = o.If(precondition)
	}

	// Upload an object with storage.Writer.
	wc, err := NewWriter(ctx, o, client)
	if err != nil {
		return nil, fmt.Errorf("failed to open writer for object %q: %w", o.ObjectName(), err)
	}
	return wc, nil
}

// CreateGcsDir creates a GCS object with trailing slash "/" to simulate a directory.
func CreateGcsDir(ctx context.Context, client *storage.Client, dirName, bucketName, objectName string) error {
	// Combine objectName and dirName to form the full GCS object path
	fullObjectPath := path.Join(objectName, dirName)

	// Ensure fullObjectPath ends with a "/"
	if !strings.HasSuffix(fullObjectPath, "/") {
		fullObjectPath += "/"
	}

	// Create an empty object with the directory path
	err := WriteToObject(ctx, client, fullObjectPath, "", storage.Conditions{})
	if err != nil {
		return fmt.Errorf("failed to create GCS directory object %q in bucket %q: %w", fullObjectPath, bucketName, err)
	}

	return nil
}
