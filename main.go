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
//	gcsfuse [flags] bucket mount_point
package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/storage"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"github.com/googlecloudplatform/gcsfuse/internal/auth"
	"github.com/googlecloudplatform/gcsfuse/internal/canned"
	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/internal/monitor"
	"github.com/googlecloudplatform/gcsfuse/internal/perf"
	"github.com/jacobsa/daemonize"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/kardianos/osext"
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
			logger.Info("Received SIGINT, attempting to unmount...")

			err := fuse.Unmount(mountPoint)
			if err != nil {
				logger.Infof("Failed to unmount in response to SIGINT: %v", err)
			} else {
				logger.Infof("Successfully unmounted in response to SIGINT.")
				return
			}
		}
	}()
}

func getConn(flags *flagStorage) (c *gcsx.Connection, err error) {
	var tokenSrc oauth2.TokenSource
	if flags.Endpoint.Hostname() == "storage.googleapis.com" {
		tokenSrc, err = auth.GetTokenSource(
			context.Background(),
			flags.KeyFile,
			flags.TokenUrl,
			flags.ReuseTokenFromUrl,
		)
		if err != nil {
			err = fmt.Errorf("GetTokenSource: %w", err)
			return
		}
	} else {
		// Do not use OAuth with non-Google hosts.
		tokenSrc = oauth2.StaticTokenSource(&oauth2.Token{})
	}

	// Create the connection.
	cfg := &gcs.ConnConfig{
		Url:             flags.Endpoint,
		TokenSource:     tokenSrc,
		UserAgent:       fmt.Sprintf("gcsfuse/%s %s", getVersion(), flags.AppName),
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
		cfg.HTTPDebugLogger = logger.NewDebug("http: ")
	}

	if flags.DebugGCS {
		cfg.GCSDebugLogger = logger.NewDebug("gcs: ")
	}

	return gcsx.NewConnection(cfg)
}

func getConnWithRetry(flags *flagStorage) (c *gcsx.Connection, err error) {
	c, err = getConn(flags)
	for delay := 1 * time.Second; delay <= flags.MaxRetrySleep && err != nil; delay = delay/2 + delay {
		logger.Infof("Waiting for connection: %v\n", err)
		time.Sleep(delay)
		c, err = getConn(flags)
	}
	return
}

////////////////////////////////////////////////////////////////////////
// main logic
////////////////////////////////////////////////////////////////////////

// Mount the file system according to arguments in the supplied context.
func createStorageHandle(flags *flagStorage) (storageHandle storage.StorageHandle, err error) {
	tokenSrc, err := auth.GetTokenSource(context.Background(), flags.KeyFile, flags.TokenUrl, true)
	if err != nil {
		err = fmt.Errorf("get token source: %w", err)
		return
	}
	storageClientConfig := storage.StorageClientConfig{
		DisableHTTP2:        flags.DisableHTTP2,
		MaxConnsPerHost:     flags.MaxConnsPerHost,
		MaxIdleConnsPerHost: flags.MaxIdleConnsPerHost,
		TokenSrc:            tokenSrc,
		HttpClientTimeout:   flags.HttpClientTimeout,
		MaxRetryDuration:    flags.MaxRetryDuration,
		RetryMultiplier:     flags.RetryMultiplier,
	}

	storageHandle, err = storage.NewStorageHandle(context.Background(), storageClientConfig)
	return
}

func mountWithArgs(
	bucketName string,
	mountPoint string,
	flags *flagStorage,
	mountStatus *log.Logger) (mfs *fuse.MountedFileSystem, err error) {
	// Enable invariant checking if requested.
	if flags.DebugInvariants {
		locker.EnableInvariantsCheck()
	}
	if flags.DebugMutex {
		locker.EnableDebugMessages()
	}

	// Grab the connection.
	//
	// Special case: if we're mounting the fake bucket, we don't need an actual
	// connection.
	var conn *gcsx.Connection
	var storageHandle storage.StorageHandle
	if bucketName != canned.FakeBucketName {
		mountStatus.Println("Opening GCS connection...")

		if flags.EnableStorageClientLibrary {
			storageHandle, err = createStorageHandle(flags)
		} else {
			conn, err = getConnWithRetry(flags)
		}
		if err != nil {
			mountStatus.Printf("Failed to open connection: %v\n", err)
			err = fmt.Errorf("getConnWithRetry: %w", err)
			return
		}
	}

	// Mount the file system.
	logger.Infof("Creating a mount at %q\n", mountPoint)
	mfs, err = mountWithConn(
		context.Background(),
		bucketName,
		mountPoint,
		flags,
		conn,
		storageHandle,
		mountStatus)

	if err != nil {
		err = fmt.Errorf("mountWithConn: %w", err)
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
	mountPoint, err = getResolvedPath(mountPoint)
	if err != nil {
		err = fmt.Errorf("canonicalizing mount point: %w", err)
		return
	}
	return
}

func runCLIApp(c *cli.Context) (err error) {
	err = resolvePathForTheFlagsInContext(c)
	if err != nil {
		return fmt.Errorf("Resolving path: %w", err)
	}

	flags, err := populateFlags(c)
	if err != nil {
		return fmt.Errorf("parsing flags failed: %w", err)
	}

	if flags.Foreground && flags.LogFile != "" {
		err = logger.InitLogFile(flags.LogFile, flags.LogFormat)
		if err != nil {
			return fmt.Errorf("init log file: %w", err)
		}
	}

	var bucketName string
	var mountPoint string
	bucketName, mountPoint, err = populateArgs(c)
	if err != nil {
		return
	}

	logger.Infof("Start gcsfuse/%s for app %q using mount point: %s\n", getVersion(), flags.AppName, mountPoint)

	// If we haven't been asked to run in foreground mode, we should run a daemon
	// with the foreground flag set and wait for it to mount.
	if !flags.Foreground {
		// Find the executable.
		var path string
		path, err = osext.Executable()
		if err != nil {
			err = fmt.Errorf("osext.Executable: %w", err)
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
		// https_proxy has precedence over http_proxy, in case both are set
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
		// Pass through the no_proxy enviroment variable. Whenever
		// using the http(s)_proxy environment variables. This should
		// also be included to know for which hosts the use of proxies
		// should be ignored.
		if p, ok := os.LookupEnv("no_proxy"); ok {
			env = append(env, fmt.Sprintf("no_proxy=%s", p))
			fmt.Fprintf(
				os.Stdout,
				"Added environment no_proxy: %s\n",
				p)
		}

		// Pass the parent process working directory to child process via
		// environment variable. This variable will be used to resolve relative paths.
		if parentProcessExecutionDir, err := os.Getwd(); err == nil {
			env = append(env, fmt.Sprintf("%s=%s", GCSFUSE_PARENT_PROCESS_DIR,
				parentProcessExecutionDir))
		}

		// Here, parent process doesn't pass the $HOME to child process implicitly,
		// hence we need to pass it explicitly.
		if homeDir, _ := os.UserHomeDir(); err == nil {
			env = append(env, fmt.Sprintf("HOME=%s", homeDir))
		}

		// Run.
		err = daemonize.Run(path, args, env, os.Stdout)
		if err != nil {
			err = fmt.Errorf("daemonize.Run: %w", err)
			return
		}

		return
	}

	// The returned error is ignored as we do not enforce monitoring exporters
	monitor.EnableStackdriverExporter(flags.StackdriverExportInterval)
	monitor.EnableOpenTelemetryCollectorExporter(flags.OtelCollectorAddress)

	// Mount, writing information about our progress to the writer that package
	// daemonize gives us and telling it about the outcome.
	var mfs *fuse.MountedFileSystem
	{
		mountStatus := logger.NewNotice("")
		mfs, err = mountWithArgs(bucketName, mountPoint, flags, mountStatus)

		if err == nil {
			mountStatus.Println("File system has been successfully mounted.")
			daemonize.SignalOutcome(nil)
		} else {
			err = fmt.Errorf("mountWithArgs: %w", err)
			daemonize.SignalOutcome(err)
			return
		}
	}

	// Let the user unmount with Ctrl-C (SIGINT).
	registerSIGINTHandler(mfs.Dir())

	// Wait for the file system to be unmounted.
	err = mfs.Join(context.Background())

	monitor.CloseStackdriverExporter()
	monitor.CloseOpenTelemetryCollectorExporter()

	if err != nil {
		err = fmt.Errorf("MountedFileSystem.Join: %w", err)
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
	go perf.HandleCPUProfileSignals()
	go perf.HandleMemoryProfileSignals()
	go logger.HandleReInitLogFile()

	// Run.
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
