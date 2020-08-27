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
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/googlecloudplatform/gcsfuse/internal/canned"
	"github.com/jacobsa/daemonize"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/syncutil"
	"github.com/kardianos/osext"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli"
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

func startMonitoringHTTPHandler(monitoringPort int) {
	fmt.Printf("Exporting metrics at localhost:%v/metrics\n", monitoringPort)
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(fmt.Sprintf(":%v", monitoringPort), nil)
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
		TokenSource:     tokenSrc,
		UserAgent:       userAgent,
		MaxBackoffSleep: flags.MaxRetrySleep,
	}

	// The default HTTP transport uses HTTP/2 with TCP multiplexing, which
	// does not create new TCP connections even when the idle connections
	// run out. To specify multiple connections per host, HTTP/2 is disabled
	// on purpose.
	if flags.DisableHTTP2 {
		cfg.Transport = &http.Transport{
			MaxConnsPerHost: flags.MaxConnsPerHost,
			// This disables HTTP/2 in the transport.
			TLSNextProto: make(
				map[string]func(string, *tls.Conn) http.RoundTripper,
			),
		}
	}

	if flags.DebugHTTP {
		cfg.HTTPDebugLogger = log.New(os.Stdout, "http: ", 0)
	}

	if flags.DebugGCS {
		cfg.GCSDebugLogger = log.New(os.Stdout, "gcs: ", log.Flags())
	}

	return gcs.NewConn(cfg)
}

////////////////////////////////////////////////////////////////////////
// main logic
////////////////////////////////////////////////////////////////////////

// Mount the file system according to arguments in the supplied context.
func mountWithArgs(
	bucketName string,
	mountPoint string,
	flags *flagStorage,
	mountStatus *log.Logger) (mfs *fuse.MountedFileSystem, err error) {
	// Enable invariant checking if requested.
	if flags.DebugInvariants {
		syncutil.EnableInvariantChecking()
	}

	// Grab the connection.
	//
	// Special case: if we're mounting the fake bucket, we don't need an actual
	// connection.
	var conn gcs.Conn
	if bucketName != canned.FakeBucketName {
		mountStatus.Println("Opening GCS connection...")

		conn, err = getConn(flags)
		if err != nil {
			err = fmt.Errorf("getConn: %v", err)
			return
		}
	}

	// Mount the file system.
	mfs, err = mountWithConn(
		context.Background(),
		bucketName,
		mountPoint,
		flags,
		conn,
		mountStatus)

	if err != nil {
		err = fmt.Errorf("mountWithConn: %v", err)
		return
	}

	return
}

func populateArgs(c *cli.Context) (
	bucketName string,
	mountPoint string,
	err error) {
	// Extract arguments.
	switch len(c.Args()) {
	case 1:
		bucketName = ""
		mountPoint = c.Args()[0]

	case 2:
		bucketName = c.Args()[0]
		mountPoint = c.Args()[1]

	default:
		err = fmt.Errorf(
			"%s takes one or two arguments. Run `%s --help` for more info.",
			path.Base(os.Args[0]),
			path.Base(os.Args[0]))

		return
	}

	// Canonicalize the mount point, making it absolute. This is important when
	// daemonizing below, since the daemon will change its working directory
	// before running this code again.
	mountPoint, err = filepath.Abs(mountPoint)
	if err != nil {
		err = fmt.Errorf("canonicalizing mount point: %v", err)
		return
	}
	return
}

func runCLIApp(c *cli.Context) (err error) {
	flags := populateFlags(c)

	var bucketName string
	var mountPoint string
	bucketName, mountPoint, err = populateArgs(c)
	if err != nil {
		return
	}

	fmt.Fprintf(os.Stdout, "Using mount point: %s\n", mountPoint)

	// If we haven't been asked to run in foreground mode, we should run a daemon
	// with the foreground flag set and wait for it to mount.
	if !flags.Foreground {
		// Find the executable.
		var path string
		path, err = osext.Executable()
		if err != nil {
			err = fmt.Errorf("osext.Executable: %v", err)
			return
		}

		// Set up arguments. Be sure to use foreground mode, and to send along the
		// potentially-modified mount point.
		args := append([]string{"--foreground"}, os.Args[1:]...)
		args[len(args)-1] = mountPoint

		// Pass along PATH so that the daemon can find fusermount on Linux.
		env := []string{
			fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		}

		// Pass along GOOGLE_APPLICATION_CREDENTIALS, since we document in
		// mounting.md that it can be used for specifying a key file.
		if p, ok := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS"); ok {
			env = append(env, fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s", p))
		}
		// Pass through the https_proxy/http_proxy environment variable,
		// in case the host requires a proxy server to reach the GCS endpoint.
		// http_proxy has precedence over http_proxy, in case both are set
		if p, ok := os.LookupEnv("https_proxy"); ok {
			env = append(env, fmt.Sprintf("https_proxy=%s", p))
			fmt.Fprintf(
				os.Stdout,
				"Added environment https_proxy: %s\n",
				p)
		} else if p, ok := os.LookupEnv("http_proxy"); ok {
			env = append(env, fmt.Sprintf("http_proxy=%s", p))
			fmt.Fprintf(
				os.Stdout,
				"Added environment http_proxy: %s\n",
				p)
		}

		// Run.
		err = daemonize.Run(path, args, env, os.Stdout)
		if err != nil {
			err = fmt.Errorf("daemonize.Run: %v", err)
			return
		}

		return
	}

	// Mount, writing information about our progress to the writer that package
	// daemonize gives us and telling it about the outcome.
	var mfs *fuse.MountedFileSystem
	{
		mountStatus := log.New(daemonize.StatusWriter, "", 0)
		mfs, err = mountWithArgs(bucketName, mountPoint, flags, mountStatus)

		if err == nil {
			mountStatus.Println("File system has been successfully mounted.")
			daemonize.SignalOutcome(nil)
		} else {
			err = fmt.Errorf("mountWithArgs: %v", err)
			daemonize.SignalOutcome(err)
			return
		}
	}

	// Open a port for exporting monitoring metrics
	if flags.MonitoringPort > 0 {
		startMonitoringHTTPHandler(flags.MonitoringPort)
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
}
