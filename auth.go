// Copyright 2015 Google Inc. All Rights Reserved.
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
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"

	"github.com/jacobsa/gcloud/gcs"
)

// Return a GCS connection pre-bound with authentication parameters derived
// from command-line flags. May block on network traffic.
func getConn() (gcs.Conn, error) {
	// Create the authenticated HTTP client.
	const scope = gcs.Scope_FullControl
	httpClient, err := google.DefaultClient(context.Background(), scope)
	if err != nil {
		return nil, err
	}

	// Create the connection.
	const userAgent = "gcsfuse/0.0"
	cfg := &gcs.ConnConfig{
		HTTPClient: httpClient,
		UserAgent:  userAgent,
	}

	return gcs.NewConn(cfg)
}
