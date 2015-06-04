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
	"fmt"
	"log"
	"os"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/gcloud/gcs"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func handleSIGINT(mountPoint string) {
	log.Println("Received SIGINT, attempting to unmount...")

	err := fuse.Unmount(mountPoint)
	if err != nil {
		log.Printf("Failed to unmount in response to SIGINT: %v", err)
	} else {
		log.Printf("Successfully unmounted in response to SIGINT.")
		return
	}
}

func getConn() (c gcs.Conn, err error) {
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

////////////////////////////////////////////////////////////////////////
// main function
////////////////////////////////////////////////////////////////////////

func main() {
	// Make logging output better.
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Grab the connection.
	conn, err := getConn()
	if err != nil {
		log.Fatalf("getConn: %v", err)
	}

	// Run.
	err = run(
		os.Args[1:],
		flag.CommandLine,
		conn,
		handleSIGINT)

	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}

	log.Println("Successfully exiting.")
}
