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
	"flag"
	"log"
	"net/http"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/oauthutil"
	storagev1 "google.golang.org/api/storage/v1"
)

var fKeyFile = flag.String("key_file", "", "Path to a JSON key for a service account created on the Google Developers Console.")

// Return an HTTP client configured with OAuth credentials from command-line
// flags. May block on network traffic.
func getAuthenticatedHttpClient() (*http.Client, error) {
	if *fKeyFile == "" {
		log.Fatalln("You must set --key_file.")
	}

	const scope = storagev1.DevstorageRead_writeScope
	return oauthutil.NewJWTHttpClient(*fKeyFile, []string{scope})
}

// Return a GCS connection pre-bound with authentication parameters derived
// from command-line flags. May block on network traffic.
func getConn() (gcs.Conn, error) {
	// TODO(jacobsa): A project ID is apparently only needed for creating and
	// listing buckets, presumably since a bucket ID already maps to a unique
	// project ID (cf. http://goo.gl/Plh3rb). So do we need this at all for our
	// use case? Probably not.
	const projectId = "fixme"

	// Create the HTTP client.
	httpClient, err := getAuthenticatedHttpClient()
	if err != nil {
		return nil, err
	}

	// Create the connection.
	return gcs.NewConn(projectId, httpClient)
}
