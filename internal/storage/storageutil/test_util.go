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

package storageutil

import (
	"net/url"
	"time"

	mountpkg "github.com/googlecloudplatform/gcsfuse/internal/mount"
)

const CustomEndpoint = "https://localhost:9000"
const DummyKeyFile = "test/test_creds.json"
const CustomTokenUrl = "http://custom-token-url"

// GetDefaultStorageClientConfig is only for test, making the default endpoint
// non-nil, so that we can create dummy tokenSource while unit test.
func GetDefaultStorageClientConfig() (clientConfig StorageClientConfig) {
	return StorageClientConfig{
		ClientProtocol:             mountpkg.HTTP1,
		MaxConnsPerHost:            10,
		MaxIdleConnsPerHost:        100,
		HttpClientTimeout:          800 * time.Millisecond,
		MaxRetrySleep:              time.Minute,
		RetryMultiplier:            2,
		UserAgent:                  "gcsfuse/unknown (Go version go1.20-pre3 cl/474093167 +a813be86df) (GCP:gcsfuse)",
		CustomEndpoint:             &url.URL{},
		KeyFile:                    DummyKeyFile,
		TokenUrl:                   "",
		ReuseTokenFromUrl:          true,
		ExperimentalEnableJsonRead: false,
	}
}
