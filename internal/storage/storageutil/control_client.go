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

package storageutil

import (
	"context"
	"fmt"
	"os"
	"time"

	control "cloud.google.com/go/storage/control/apiv2"
	"github.com/googleapis/gax-go/v2"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
)

// loggingRetryer wraps a gax.Retryer to add logging on each retry attempt.
// It implements the gax.Retryer interface. It invokes the original
// retryer to determine if a retry is needed and for how long to wait.
// If a retry is scheduled, it logs the operation-name, error and the delay.
type loggingRetryer struct {
	originalRetryer gax.Retryer
	operationName   string
}

func (r *loggingRetryer) Retry(err error) (time.Duration, bool) {
	// Get the delay and retry decision from the original retryer.
	delay, shouldRetry := r.originalRetryer.Retry(err)

	// If a retry is happening, log it.
	if shouldRetry {
		// Using the logger from gcsfuse/internal/logger is recommended
		// to maintain consistent logging.
		logger.Tracef("Retrying operation %q due to error: %v. Waiting for %v before next attempt.", r.operationName, err, delay)
	}

	return delay, shouldRetry
}

// storageControlClientRetryOptions configures the retry behavior for the
// StorageControlClient. It now uses the custom loggingRetryer to ensure
// each retry attempt is logged.
func storageControlClientRetryOptions(clientConfig *StorageClientConfig, operationName string) []gax.CallOption {
	// originalRetryerFactory creates an instance of the default retryer provided by gcsfuse.
	// This is what we will wrap with our logging logic.
	originalRetryerFactory := func() gax.Retryer {
		return gax.OnCodes([]codes.Code{
			codes.ResourceExhausted,
			codes.Unavailable,
			codes.DeadlineExceeded,
			codes.Internal,
			codes.Unknown,
		}, gax.Backoff{
			Max:        10 * time.Second,
			Multiplier: clientConfig.RetryMultiplier,
		})
	}

	return []gax.CallOption{
		gax.WithTimeout(300000 * time.Millisecond),
		// WithRetry accepts a function that returns a Retryer.
		// We use this to inject our custom loggingRetryer.
		gax.WithRetry(func() gax.Retryer {
			return &loggingRetryer{
				originalRetryer: originalRetryerFactory(),
				operationName:   operationName,
			}
		}),
	}
}

func setRetryConfigForFolderAPIs(sc *control.StorageControlClient, clientConfig *StorageClientConfig) {
	sc.CallOptions.RenameFolder = storageControlClientRetryOptions(clientConfig, "RenameFolder")
	sc.CallOptions.GetFolder = storageControlClientRetryOptions(clientConfig, "GetFolder")
	sc.CallOptions.GetStorageLayout = storageControlClientRetryOptions(clientConfig, "GetStorageLayout")
	sc.CallOptions.CreateFolder = storageControlClientRetryOptions(clientConfig, "CreateFolder")
	sc.CallOptions.DeleteFolder = storageControlClientRetryOptions(clientConfig, "DeleteFolder")
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
