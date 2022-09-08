// Copyright 2022 Google Inc. All Rights Reserved.
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

package storage

import (
	"context"
	"time"

	"cloud.google.com/go/storage"
	"github.com/fsouza/fake-gcs-server/fakestorage"
)

const port uint16 = 8081
const host string = "127.0.0.1"

// Todo: Please complete it with correct value
func GetDefaultObject() fakestorage.Object {
	return fakestorage.Object{}
}

// Call this inside storage_handle_test and store it instead of server.
func CreateFakeStorageClient() (client *storage.Client, err error) {
	fakeServer, err := fakestorage.NewServerWithOptions(fakestorage.Options{
		Host: host,
		Port: port,
	})
	if err != nil {
		return
	}

	client = fakeServer.Client()
	return
}

// This might not required.
func AddBucketToClient(client *storage.Client, bucketName string) (bucket *storage.BucketHandle, err error) {
	bucket = client.Bucket(bucketName)
	_, err = bucket.Attrs(context.Background())

	if err != nil {
		err = bucket.Create(context.Background(), "test_bucket", &storage.BucketAttrs{
			Name:                       "",
			ACL:                        nil,
			BucketPolicyOnly:           storage.BucketPolicyOnly{},
			UniformBucketLevelAccess:   storage.UniformBucketLevelAccess{},
			PublicAccessPrevention:     0,
			DefaultObjectACL:           nil,
			DefaultEventBasedHold:      false,
			PredefinedACL:              "",
			PredefinedDefaultObjectACL: "",
			Location:                   "",
			CustomPlacementConfig:      nil,
			MetaGeneration:             0,
			StorageClass:               "",
			Created:                    time.Time{},
			VersioningEnabled:          false,
			Labels:                     nil,
			RequesterPays:              false,
			Lifecycle:                  storage.Lifecycle{},
			RetentionPolicy:            nil,
			CORS:                       nil,
			Encryption:                 nil,
			Logging:                    nil,
			Website:                    nil,
			Etag:                       "",
			LocationType:               "",
			ProjectNumber:              0,
			RPO:                        0,
		})
	}
	return
}

// We need server instance to add the object.
