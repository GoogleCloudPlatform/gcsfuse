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

package helpers

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func setBucketAndObjectBasedOnTypeOfMount(bucket, object *string) {
	*bucket = setup.TestBucket()
	if setup.DynamicBucketMounted() != "" {
		*bucket = setup.DynamicBucketMounted()
	}
	if setup.OnlyDirMounted() != "" {
		*object = path.Join(setup.OnlyDirMounted(), *object)
	}
}

// ReadObjectFromGCS downloads the object from GCS and returns the data.
func ReadObjectFromGCS(object string) (string, error) {
	var bucket string
	setBucketAndObjectBasedOnTypeOfMount(&bucket, &object)

	// Create new storage client.
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("storage.NewClient: %w", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	// Create storage reader to read from GCS.
	rc, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return "", fmt.Errorf("Object(%q).NewReader: %w", object, err)
	}
	defer rc.Close()

	// Variable buf will contain the output from reader.
	buf := make([]byte, 1024)
	_, err = rc.Read(buf)
	if err != nil && !strings.Contains(err.Error(), "EOF") {
		return "", fmt.Errorf("rc.Read: %w", err)
	}

	// Remove any extra null characters from buf before returning.
	return strings.Trim(string(buf), "\x00"), nil
}

// CreateObjectOnGCS creates an object with given name and content on GCS.
func CreateObjectOnGCS(object, content string) error {
	var bucket string
	setBucketAndObjectBasedOnTypeOfMount(&bucket, &object)

	// Create new storage client.
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %w", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	o := client.Bucket(bucket).Object(object)
	o = o.If(storage.Conditions{DoesNotExist: true})

	// Upload an object with storage.Writer.
	wc := o.NewWriter(ctx)
	if _, err = io.WriteString(wc, content); err != nil {
		return fmt.Errorf("io.WriteSTring: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %w", err)
	}

	return nil
}
