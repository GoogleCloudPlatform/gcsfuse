// Copyright 2024 Google LLC
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
	"fmt"
	"log"
	"strings"
	"time"

	control "cloud.google.com/go/storage/control/apiv2"
	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
)

func CreateControlClient(ctx context.Context) (client *control.StorageControlClient, err error) {
	client, err = control.NewStorageControlClient(ctx)
	clientConfig := &storageutil.StorageClientConfig{MaxRetrySleep: 30 * time.Second, RetryMultiplier: 2}

	client.CallOptions.RenameFolder = storageutil.StorageControlClientRetryOptions(clientConfig)
	client.CallOptions.GetFolder = storageutil.StorageControlClientRetryOptions(clientConfig)
	client.CallOptions.GetStorageLayout = storageutil.StorageControlClientRetryOptions(clientConfig)
	client.CallOptions.CreateFolder = storageutil.StorageControlClientRetryOptions(clientConfig)
	client.CallOptions.DeleteFolder = storageutil.StorageControlClientRetryOptions(clientConfig)
	client.CallOptions.CreateManagedFolder = storageutil.StorageControlClientRetryOptions(clientConfig)
	client.CallOptions.DeleteManagedFolder = storageutil.StorageControlClientRetryOptions(clientConfig)
	client.CallOptions.ListManagedFolders = storageutil.StorageControlClientRetryOptions(clientConfig)
	client.CallOptions.GetManagedFolder = storageutil.StorageControlClientRetryOptions(clientConfig)

	if err != nil {
		return nil, fmt.Errorf("control.NewStorageControlClient: #{err}")
	}
	return client, nil
}

func CreateControlClientWithCancel(ctx *context.Context, controlClient **control.StorageControlClient) func() error {
	var err error
	var cancel context.CancelFunc
	*ctx, cancel = context.WithCancel(*ctx)
	*controlClient, err = CreateControlClient(*ctx)
	if err != nil {
		log.Fatalf("client.CreateControlClient: %v", err)
	}
	// Return func to close storage client and release resources.
	return func() error {
		err := (*controlClient).Close()
		if err != nil {
			return fmt.Errorf("failed to close control client: %v", err)
		}
		defer cancel()
		return nil
	}
}

func DeleteManagedFoldersInBucket(ctx context.Context, client *control.StorageControlClient, managedFolderPath, bucket string) {
	folderPath := fmt.Sprintf("projects/_/buckets/%v/managedFolders/%v/", bucket, managedFolderPath)
	req := &controlpb.DeleteManagedFolderRequest{
		Name:          folderPath,
		AllowNonEmpty: true,
	}
	if err := client.DeleteManagedFolder(ctx, req); err != nil && !strings.Contains(err.Error(), "The following URLs matched no objects or files") {
		log.Fatalf("Error while deleting managed folder: %v", err)
	}
}

func CreateManagedFoldersInBucket(ctx context.Context, client *control.StorageControlClient, managedFolderPath, bucket string) {
	mf := &controlpb.ManagedFolder{}
	req := &controlpb.CreateManagedFolderRequest{
		Parent:          fmt.Sprintf("projects/_/buckets/%v", bucket),
		ManagedFolder:   mf,
		ManagedFolderId: managedFolderPath,
	}
	if _, err := client.CreateManagedFolder(ctx, req); err != nil && !strings.Contains(err.Error(), "The specified managed folder already exists") {
		log.Fatalf("Error while creating managed folder: %v", err)
	}
}
