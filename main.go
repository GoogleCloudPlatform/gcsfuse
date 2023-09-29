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
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/internal/canned"
	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"github.com/googlecloudplatform/gcsfuse/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/internal/monitor"
	"github.com/googlecloudplatform/gcsfuse/internal/perf"
	"github.com/googlecloudplatform/gcsfuse/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/internal/util"
	"github.com/jacobsa/daemonize"
	"github.com/jacobsa/fuse"
	"github.com/kardianos/osext"
	"github.com/urfave/cli"
	"golang.org/x/net/context"
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
				logger.Errorf("Failed to unmount in response to SIGINT: %v", err)
			} else {
				logger.Infof("Successfully unmounted in response to SIGINT.")
				return
			}
		}
	}()
}

func getUserAgent(appName string) string {
	gcsfuseMetadataImageType := os.Getenv("GCSFUSE_METADATA_IMAGE_TYPE")
	if len(gcsfuseMetadataImageType) > 0 {
		userAgent := fmt.Sprintf("gcsfuse/%s %s (GPN:gcsfuse-%s)", getVersion(), appName, gcsfuseMetadataImageType)
		return strings.Join(strings.Fields(userAgent), " ")
	} else if len(appName) > 0 {
		return fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-%s)", getVersion(), appName)
	} else {
		return fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse)", getVersion())
	}
}

func createStorageHandle(flags *flagStorage) (storageHandle storage.StorageHandle, err error) {
	storageClientConfig := storageutil.StorageClientConfig{
		ClientProtocol:             flags.ClientProtocol,
		MaxConnsPerHost:            flags.MaxConnsPerHost,
		MaxIdleConnsPerHost:        flags.MaxIdleConnsPerHost,
		HttpClientTimeout:          flags.HttpClientTimeout,
		MaxRetryDuration:           flags.MaxRetryDuration,
		RetryMultiplier:            flags.RetryMultiplier,
		UserAgent:                  getUserAgent(flags.AppName),
		CustomEndpoint:             flags.CustomEndpoint,
		KeyFile:                    flags.KeyFile,
		TokenUrl:                   flags.TokenUrl,
		ReuseTokenFromUrl:          flags.ReuseTokenFromUrl,
		ExperimentalEnableJsonRead: flags.ExperimentalEnableJsonRead,
	}

	storageHandle, err = storage.NewStorageHandle(context.Background(), storageClientConfig)
	return
}

////////////////////////////////////////////////////////////////////////
// main logic
////////////////////////////////////////////////////////////////////////

// Mount the file system according to arguments in the supplied context.
func mountWithArgs(
	bucketName string,
	mountPoint string,
	flags *flagStorage,
	mountConfig *config.MountConfig) (mfs *fuse.MountedFileSystem, err error) {
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
	var storageHandle storage.StorageHandle
	if bucketName != canned.FakeBucketName {
		logger.Info("Creating Storage handle...")
		storageHandle, err = createStorageHandle(flags)
		if err != nil {
			err = fmt.Errorf("Failed to create storage handle using createStorageHandle: %w", err)
			return
		}
	}

	// Mount the file system.
	logger.Infof("Creating a mount at %q\n", mountPoint)
	mfs, err = mountWithStorageHandle(
		context.Background(),
		bucketName,
		mountPoint,
		flags,
		mountConfig,
		storageHandle)

	if err != nil {
		err = fmt.Errorf("mountWithStorageHandle: %w", err)
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
	mountPoint, err = util.GetResolvedPath(mountPoint)
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

	mountConfig, err := config.ParseConfigFile(flags.ConfigFile)
	if err != nil {
		return fmt.Errorf("parsing config file failed: %w", err)
	}

	config.OverrideWithLoggingFlags(mountConfig, flags.LogFile, flags.LogFormat,
		flags.DebugFuse, flags.DebugGCS, flags.DebugMutex)

	err = util.ResolveConfigFilePaths(mountConfig)
	if err != nil {
		return fmt.Errorf("Resolving path: %w", err)
	}

	if flags.Foreground {
		err = logger.InitLogFile(mountConfig.LogConfig.FilePath, mountConfig.LogConfig.Format, mountConfig.LogConfig.Severity)
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
			env = append(env, fmt.Sprintf("%s=%s", util.GCSFUSE_PARENT_PROCESS_DIR,
				parentProcessExecutionDir))
		}

		// Here, parent process doesn't pass the $HOME to child process implicitly,
		// hence we need to pass it explicitly.
		if homeDir, _ := os.UserHomeDir(); err == nil {
			env = append(env, fmt.Sprintf("HOME=%s", homeDir))
		}

		// This environment variable will be helpful to distinguish b/w the main
		// process and daemon process. If this environment variable set that means
		// programme is running as daemon process.
		env = append(env, fmt.Sprintf("%s=true", logger.GCSFuseInBackgroundMode))

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
		mfs, err = mountWithArgs(bucketName, mountPoint, flags, mountConfig)

		if err == nil {
			logger.Info("File system has been successfully mounted.")
			daemonize.SignalOutcome(nil)
		} else {
			// Printing via mountStatus will have duplicate logs on the console while
			// mounting gcsfuse in foreground mode. But this is important to avoid
			// losing error logs when run in the background mode.
			logger.Errorf("Error while mounting gcsfuse: %v\n", err)
			err = fmt.Errorf("mountWithArgs: %w", err)
			daemonize.SignalOutcome(err)
			return
		}
	}

	// Let the user unmount with Ctrl-C (SIGINT).
	registerSIGINTHandler(mfs.Dir())

	// Wait for the file system to be unmounted.
	err = mfs.Join(context.Background())
	if err != nil {
		return fmt.Errorf("failed MountedFileSystem.Join: %w", err)
	}

	monitor.CloseStackdriverExporter()

	if err := monitor.CloseOpenTelemetryCollectorExporter(); err != nil {
		return fmt.Errorf("failed to close open-telemetry collector exporter: %w", err)
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

// Handle panic if the crash occurs during mounting.
func handlePanicWhileMounting() {
	// Detect if panic happens in main go routine.
	a := recover()
	if a != nil {
		logger.Fatal("Panic: %v", a)
	}
}

func main() {
	defer handlePanicWhileMounting()

	// Make logging output better.
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Set up profiling handlers.
	go perf.HandleCPUProfileSignals()
	go perf.HandleMemoryProfileSignals()

	// Run.
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
