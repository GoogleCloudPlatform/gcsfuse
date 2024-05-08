// Copyright 2024 Google Inc. All Rights Reserved.
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

package storageutil

import (
	"context"
	"fmt"
	"os"

	control "cloud.google.com/go/storage/control/apiv2"
	"github.com/googleapis/gax-go/v2"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"google.golang.org/api/option"
)

func storageControlClientRetryOptions(clientConfig *StorageClientConfig) []gax.CallOption {
	return []gax.CallOption{gax.WithRetry(func() gax.Retryer {
		return gax.OnErrorFunc(gax.Backoff{
			Max:        clientConfig.MaxRetrySleep,
			Multiplier: clientConfig.RetryMultiplier,
		}, ShouldRetry)
	})}
}

func setRetryConfigForFolderAPIs(sc *control.StorageControlClient, clientConfig *StorageClientConfig) {
	sc.CallOptions.CreateFolder = storageControlClientRetryOptions(clientConfig)
	sc.CallOptions.DeleteFolder = storageControlClientRetryOptions(clientConfig)
	sc.CallOptions.RenameFolder = storageControlClientRetryOptions(clientConfig)
	sc.CallOptions.GetFolder = storageControlClientRetryOptions(clientConfig)
	sc.CallOptions.GetStorageLayout = storageControlClientRetryOptions(clientConfig)
}

func CreateGRPCControlClient(ctx context.Context, clientOpts []option.ClientOption, clientConfig *StorageClientConfig) (controlClient *control.StorageControlClient, err error) {
	if err := os.Setenv("GOOGLE_CLOUD_ENABLE_DIRECT_PATH_XDS", "true"); err != nil {
		logger.Fatal("error setting direct path env var: %v", err)
	}

	controlClient, err = control.NewStorageControlClient(ctx, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("NewStorageControlClient: %w", err)
	}

	// Set retries for control client.
	setRetryConfigForFolderAPIs(controlClient, clientConfig)

	// Unset the environment variable, since it's used only while creation of grpc client.
	if err := os.Unsetenv("GOOGLE_CLOUD_ENABLE_DIRECT_PATH_XDS"); err != nil {
		logger.Fatal("error while unsetting direct path env var: %v", err)
	}

	return controlClient, err
}
