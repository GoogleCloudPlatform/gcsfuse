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
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/grpc/codes"
)

func storageControlClientGaxRetryOptions(clientConfig *StorageClientConfig) gax.Retryer {
	return gax.OnCodes([]codes.Code{codes.Internal, codes.Unavailable}, gax.Backoff{
		Max:        clientConfig.MaxRetrySleep,
		Multiplier: clientConfig.RetryMultiplier,
	})
}
