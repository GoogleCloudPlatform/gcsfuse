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

package main

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/oauth2/google"
)

const (
	vendorClientReader   string = "github.com/jacobsa/gcloud/gcs/read.go"
	officialClientReader string = "cloud.google/com/go/storage/reader.go"
)

type readerFactory interface {
	NewReader(objectName string) io.ReadCloser
}

func newReaderFactory(
	transport *http.Transport,
	readerType string,
	bucketName string) (rf readerFactory) {
	switch readerType {
	case vendorClientReader:
		rf = newVendorReaderFactory(transport, bucketName)
	case officialClientReader:
		rf = newOfficialReaderFactory(transport, bucketName)
	default:
		panic(fmt.Errorf("Unknown reader type: %q", readerType))
	}
	return
}

// Vendor reader depends on "github.com/jacobsa/gcloud/gcs"
type vendorReaderFactory struct {
	bucket gcs.Bucket
}

func newVendorReaderFactory(
	transport *http.Transport,
	bucketName string) (rf readerFactory) {
	tokenSrc, err := google.DefaultTokenSource(
		context.Background(),
		gcs.Scope_FullControl,
	)
	if err != nil {
		panic(fmt.Errorf("Cannot get token source: %w", err))
	}
	config := &gcs.ConnConfig{
		TokenSource: tokenSrc,
		UserAgent:   "gcsfuse/0.0",
		Transport:   transport,
	}
	conn, err := gcs.NewConn(config)
	if err != nil {
		panic(fmt.Errorf("Cannot create conn: %w", err))
	}
	bucket, err := conn.OpenBucket(
		context.Background(),
		&gcs.OpenBucketOptions{
			Name: bucketName,
		},
	)
	if err != nil {
		panic(fmt.Errorf("Cannot open bucket %q: %w", bucketName, err))
	}
	rf = &vendorReaderFactory{
		bucket: bucket,
	}
	return
}

func (rf *vendorReaderFactory) NewReader(
	objectName string) io.ReadCloser {
	r, err := rf.bucket.NewReader(
		context.Background(),
		&gcs.ReadObjectRequest{
			Name: objectName,
		},
	)
	if err != nil {
		panic(fmt.Errorf("Cannot read %q: %w", objectName, err))
	}
	return r
}

// Official reader depends on "cloud.google.com/go/storage"
type officialReaderFactory struct {
	bucket *storage.BucketHandle
}

func newOfficialReaderFactory(
	transport *http.Transport,
	bucketName string) (rf readerFactory) {
	client, err := storage.NewClient(context.Background())
	if err != nil {
		panic(fmt.Errorf("NewClient: %w", err))
	}
	bucket := client.Bucket(bucketName)
	return &officialReaderFactory{
		bucket: bucket,
	}
}

func (rf *officialReaderFactory) NewReader(
	objectName string) io.ReadCloser {
	object := rf.bucket.Object(objectName)
	reader, err := object.NewReader(context.Background())
	if err != nil {
		panic(fmt.Errorf("NewReader: %w", err))
	}
	return reader
}
