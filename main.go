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
	"fmt"
	"log"
	"os"
	"os/signal"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jgeewax/cli"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func registerSIGINTHandler(mountPoint string) {
	// Register for SIGINT.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	// Start a goroutine that will unmount when the signal is received.
	go func() {
		for {
			<-signalChan
			log.Println("Received SIGINT, attempting to unmount...")

			err := fuse.Unmount(mountPoint)
			if err != nil {
				log.Printf("Failed to unmount in response to SIGINT: %v", err)
			} else {
				log.Printf("Successfully unmounted in response to SIGINT.")
				return
			}
		}
	}()
}

func getConn() (c gcs.Conn, err error) {
	// Create the oauth2 token source.
	const scope = gcs.Scope_FullControl
	tokenSrc, err := google.DefaultTokenSource(context.Background(), scope)
	if err != nil {
		return nil, err
	}

	// Create the connection.
	const userAgent = "gcsfuse/0.0"
	cfg := &gcs.ConnConfig{
		TokenSource: tokenSrc,
		UserAgent:   userAgent,
	}

	return gcs.NewConn(cfg)
}

////////////////////////////////////////////////////////////////////////
// main function
////////////////////////////////////////////////////////////////////////

func main() {

	// Make logging output better.
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	app := getApp()
	app.Action = func(c *cli.Context) {
		var err error

		// We should get two arguments exactly. Otherwise error out.
		if len(c.Args()) != 2 {
			fmt.Fprintf(
				os.Stderr,
				"Error: %s takes exactly two arguments.\n",
				app.Name)
				cli.ShowAppHelp(c)
				os.Exit(1)
			}

			// Populate and parse flags.
			bucketName := c.Args()[0]
			mountPoint := c.Args()[1]
			flags := populateFlags(c)

			// Grab the connection.
			conn, err := getConn()
			if err != nil {
				log.Fatalf("getConn: %v", err)
			}

			// Mount the file system.
			mfs, err := mount(
				context.Background(),
				bucketName,
				mountPoint,
				flags,
				conn)

				if err != nil {
					log.Fatalf("Mounting file system: %v", err)
				}

				log.Println("File system has been successfully mounted.")

				// Let the user unmount with Ctrl-C (SIGINT).
				registerSIGINTHandler(mfs.Dir())

				// Wait for the file system to be unmounted.
				err = mfs.Join(context.Background())
				if err != nil {
					err = fmt.Errorf("MountedFileSystem.Join: %v", err)
					return
				}

				log.Println("Successfully exiting.")
			}

			app.Run(os.Args)
		}
