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

// A fuse file system for Google Cloud Storage buckets.
//
// Usage:
//
//     gcsfuse [flags] bucket mount_point
//
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/codegangsta/cli"
	"github.com/jacobsa/daemonize"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/syncutil"
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

func handleCPUProfileSignals() {
	profileOnce := func(duration time.Duration, path string) (err error) {
		// Set up the file.
		var f *os.File
		f, err = os.Create(path)
		if err != nil {
			err = fmt.Errorf("Create: %v", err)
			return
		}

		defer func() {
			closeErr := f.Close()
			if err == nil {
				err = closeErr
			}
		}()

		// Profile.
		pprof.StartCPUProfile(f)
		time.Sleep(duration)
		pprof.StopCPUProfile()
		return
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR1)
	for range c {
		const path = "/tmp/cpu.pprof"
		const duration = 10 * time.Second

		log.Printf("Writing %v CPU profile to %s...", duration, path)

		err := profileOnce(duration, path)
		if err == nil {
			log.Printf("Done writing CPU profile to %s.", path)
		} else {
			log.Printf("Error writing CPU profile: %v", err)
		}
	}
}

func handleMemoryProfileSignals() {
	profileOnce := func(path string) (err error) {
		// Trigger a garbage collection to get up to date information (cf.
		// https://goo.gl/aXVQfL).
		runtime.GC()

		// Open the file.
		var f *os.File
		f, err = os.Create(path)
		if err != nil {
			err = fmt.Errorf("Create: %v", err)
			return
		}

		defer func() {
			closeErr := f.Close()
			if err == nil {
				err = closeErr
			}
		}()

		// Dump to the file.
		err = pprof.Lookup("heap").WriteTo(f, 0)
		if err != nil {
			err = fmt.Errorf("WriteTo: %v", err)
			return
		}

		return
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR2)
	for range c {
		const path = "/tmp/mem.pprof"

		err := profileOnce(path)
		if err == nil {
			log.Printf("Wrote memory profile to %s.", path)
		} else {
			log.Printf("Error writing memory profile: %v", err)
		}
	}
}

// Create token source from the JSON file at the supplide path.
func newTokenSourceFromPath(
	path string,
	scope string) (ts oauth2.TokenSource, err error) {
	// Read the file.
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		err = fmt.Errorf("ReadFile(%q): %v", path, err)
		return
	}

	// Create a config struct based on its contents.
	jwtConfig, err := google.JWTConfigFromJSON(contents, scope)
	if err != nil {
		err = fmt.Errorf("JWTConfigFromJSON: %v", err)
		return
	}

	// Create the token source.
	ts = jwtConfig.TokenSource(context.Background())

	return
}

func getConn(flags *flagStorage) (c gcs.Conn, err error) {
	// Create the oauth2 token source.
	const scope = gcs.Scope_FullControl

	var tokenSrc oauth2.TokenSource
	if flags.KeyFile != "" {
		tokenSrc, err = newTokenSourceFromPath(flags.KeyFile, scope)
		if err != nil {
			err = fmt.Errorf("newTokenSourceFromPath: %v", err)
			return
		}
	} else {
		tokenSrc, err = google.DefaultTokenSource(context.Background(), scope)
		if err != nil {
			err = fmt.Errorf("DefaultTokenSource: %v", err)
			return
		}
	}

	// Create the connection.
	const userAgent = "gcsfuse/0.0"
	cfg := &gcs.ConnConfig{
		TokenSource: tokenSrc,
		UserAgent:   userAgent,
	}

	if flags.DebugHTTP {
		cfg.HTTPDebugLogger = log.New(os.Stderr, "http: ", 0)
	}

	if flags.DebugGCS {
		cfg.GCSDebugLogger = log.New(os.Stderr, "gcs: ", 0)
	}

	return gcs.NewConn(cfg)
}

////////////////////////////////////////////////////////////////////////
// main logic
////////////////////////////////////////////////////////////////////////

// Mount the file system according to arguments in the supplied context.
func mountFromContext(
	c *cli.Context,
	mountStatus *log.Logger) (mfs *fuse.MountedFileSystem, err error) {
	// Extract arguments.
	if len(c.Args()) != 2 {
		err = fmt.Errorf(
			"Error: %s takes exactly two arguments. Run `%s --help` for more info.",
			path.Base(os.Args[0]),
			path.Base(os.Args[0]))

		return
	}

	bucketName := c.Args()[0]
	mountPoint := c.Args()[1]

	// Populate and parse flags.
	flags := populateFlags(c)

	// Enable invariant checking if requested.
	if flags.DebugInvariants {
		syncutil.EnableInvariantChecking()
	}

	// Grab the connection.
	mountStatus.Println("Opening GCS connection...")

	conn, err := getConn(flags)
	if err != nil {
		err = fmt.Errorf("getConn: %v", err)
		return
	}

	// Mount the file system.
	mfs, err = mount(
		context.Background(),
		bucketName,
		mountPoint,
		flags,
		conn,
		mountStatus)

	if err != nil {
		err = fmt.Errorf("mount: %v", err)
		return
	}

	return
}

func runCLIApp(c *cli.Context) (err error) {
	// Mount, writing information about our progress to the writer that package
	// daemonize gives us and telling it about the outcome.
	var mfs *fuse.MountedFileSystem
	{
		mountStatus := log.New(daemonize.StatusWriter, "", 0)
		mfs, err = mountFromContext(c, mountStatus)

		if err == nil {
			mountStatus.Println("File system has been successfully mounted.")
			daemonize.SignalOutcome(nil)
		} else {
			err = fmt.Errorf("mountFromContext: %v", err)
			daemonize.SignalOutcome(err)
			return
		}
	}

	// Let the user unmount with Ctrl-C (SIGINT).
	registerSIGINTHandler(mfs.Dir())

	// Wait for the file system to be unmounted.
	err = mfs.Join(context.Background())
	if err != nil {
		err = fmt.Errorf("MountedFileSystem.Join: %v", err)
		return
	}

	return
}

func run() (err error) {
	// Set up the app.
	app := newApp()

	var appErr error
	app.Action = func(c *cli.Context) {
		appErr = runCLIApp(c)
	}

	// Run it.
	err = app.Run(os.Args)
	if err != nil {
		return
	}

	err = appErr
	return
}

func main() {
	// Make logging output better.
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Set up profiling handlers.
	go handleCPUProfileSignals()
	go handleMemoryProfileSignals()

	// Run.
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	log.Println("Successfully exiting.")
}
