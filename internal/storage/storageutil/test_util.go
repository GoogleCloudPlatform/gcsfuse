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

package storageutil

import (
	"time"

	"github.com/vipnydav/gcsfuse/v3/cfg"
)

const CustomEndpoint = "https://localhost:9000"
const CustomTokenUrl = "http://custom-token-url"

// GetDefaultStorageClientConfig is only for test.
func GetDefaultStorageClientConfig(keyFile string) (clientConfig StorageClientConfig) {
	return StorageClientConfig{
		ClientProtocol:             cfg.HTTP1,
		MaxConnsPerHost:            10,
		MaxIdleConnsPerHost:        100,
		HttpClientTimeout:          800 * time.Millisecond,
		MaxRetrySleep:              time.Minute,
		MaxRetryAttempts:           0,
		RetryMultiplier:            2,
		UserAgent:                  "gcsfuse/unknown (Go version go1.20-pre3 cl/474093167 +a813be86df) (GCP:gcsfuse)",
		CustomEndpoint:             "",
		KeyFile:                    keyFile,
		TokenUrl:                   "",
		ReuseTokenFromUrl:          true,
		ExperimentalEnableJsonRead: false,
		AnonymousAccess:            false,
		EnableHNS:                  true,
		EnableGoogleLibAuth:        true,
		ReadStallRetryConfig: cfg.ReadStallGcsRetriesConfig{
			Enable:              false,
			InitialReqTimeout:   20 * time.Second,
			MaxReqTimeout:       1200 * time.Second,
			MinReqTimeout:       500 * time.Millisecond,
			ReqIncreaseRate:     15,
			ReqTargetPercentile: 0.99,
		},
	}
}
