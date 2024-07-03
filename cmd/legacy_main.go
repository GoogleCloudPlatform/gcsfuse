// Copyright 2024 Google Inc. All Rights Reserved.
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
package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/canned"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/monitor"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/mount"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/perf"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/jacobsa/daemonize"
	"github.com/jacobsa/fuse"
	"github.com/kardianos/osext"
	"github.com/urfave/cli"
	"golang.org/x/net/context"
)

const (
	SuccessfulMountMessage         = "File system has been successfully mounted."
	UnsuccessfulMountMessagePrefix = "Error while mounting gcsfuse"
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

func getUserAgent(appName string, config string) string {
	gcsfuseMetadataImageType := os.Getenv("GCSFUSE_METADATA_IMAGE_TYPE")
	if len(gcsfuseMetadataImageType) > 0 {
		userAgent := fmt.Sprintf("gcsfuse/%s %s (GPN:gcsfuse-%s) (Cfg:%s)", getVersion(), appName, gcsfuseMetadataImageType, config)
		return strings.Join(strings.Fields(userAgent), " ")
	} else if len(appName) > 0 {
		return fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-%s) (Cfg:%s)", getVersion(), appName, config)
	} else {
		return fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse) (Cfg:%s)", getVersion(), config)
	}
}

func getConfigForUserAgent(mountConfig *config.MountConfig) string {
	// Minimum configuration details created in a bitset fashion. Right now, its restricted only to File Cache Settings.
	isFileCacheEnabled := "0"
	if config.IsFileCacheEnabled(mountConfig) {
		isFileCacheEnabled = "1"
	}
	isFileCacheForRangeReadEnabled := "0"
	if mountConfig.FileCacheConfig.CacheFileForRangeRead {
		isFileCacheForRangeReadEnabled = "1"
	}
	return fmt.Sprintf("%s:%s", isFileCacheEnabled, isFileCacheForRangeReadEnabled)
}
func createStorageHandle(newConfig *cfg.Config, flags *flagStorage, mountConfig *config.MountConfig, userAgent string) (storageHandle storage.StorageHandle, err error) {
	storageClientConfig := storageutil.StorageClientConfig{
		ClientProtocol:             newConfig.GcsConnection.ClientProtocol,
		MaxConnsPerHost:            flags.MaxConnsPerHost,
		MaxIdleConnsPerHost:        flags.MaxIdleConnsPerHost,
		HttpClientTimeout:          flags.HttpClientTimeout,
		MaxRetrySleep:              flags.MaxRetrySleep,
		RetryMultiplier:            flags.RetryMultiplier,
		UserAgent:                  userAgent,
		CustomEndpoint:             flags.CustomEndpoint,
		KeyFile:                    string(newConfig.GcsAuth.KeyFile),
		AnonymousAccess:            mountConfig.GCSAuth.AnonymousAccess,
		TokenUrl:                   newConfig.GcsAuth.TokenUrl,
		ReuseTokenFromUrl:          newConfig.GcsAuth.ReuseTokenFromUrl,
		ExperimentalEnableJsonRead: newConfig.GcsConnection.ExperimentalEnableJsonRead,
		GrpcConnPoolSize:           mountConfig.GCSConnection.GRPCConnPoolSize,
		EnableHNS:                  mountConfig.EnableHNS,
	}
	logger.Infof("UserAgent = %s\n", storageClientConfig.UserAgent)
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
	mountConfig *config.MountConfig,
	newConfig *cfg.Config) (mfs *fuse.MountedFileSystem, err error) {
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
		userAgent := getUserAgent(newConfig.AppName, getConfigForUserAgent(mountConfig))
		logger.Info("Creating Storage handle...")
		storageHandle, err = createStorageHandle(newConfig, flags, mountConfig, userAgent)
		if err != nil {
			err = fmt.Errorf("failed to create storage handle using createStorageHandle: %w", err)
			return
		}
	}

	// Mount the file system.
	logger.Infof("Creating a mount at %q\n", mountPoint)
	mfs, err = mountWithStorageHandle(
		context.Background(),
		bucketName,
		mountPoint,
		newConfig,
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

func callListRecursive(mountPoint string) (err error) {
	logger.Debugf("Started recursive metadata-prefetch of directory: \"%s\" ...", mountPoint)
	numItems := 0
	err = filepath.WalkDir(mountPoint, func(path string, d fs.DirEntry, err error) error {
		if err == nil {
			numItems++
			return err
		}
		if d == nil {
			return fmt.Errorf("got error walking: path=\"%s\" does not exist, error = %w", path, err)
		}
		return fmt.Errorf("got error walking: path=\"%s\", dentry=\"%s\", isDir=%v, error = %w", path, d.Name(), d.IsDir(), err)
	})

	if err != nil {
		return fmt.Errorf("failed in recursive metadata-prefetch of directory: \"%s\"; error = %w", mountPoint, err)
	}

	logger.Debugf("... Completed recursive metadata-prefetch of directory: \"%s\". Number of items discovered: %v", mountPoint, numItems)

	return nil
}

func isDynamicMount(bucketName string) bool {
	return bucketName == "" || bucketName == "_"
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

	newConfig, err := PopulateNewConfigFromLegacyFlagsAndConfig(c, flags, mountConfig)
	if err != nil {
		return fmt.Errorf("error resolving flags and configs: %w", err)
	}

	cfg.OverrideWithLoggingFlags(newConfig, newConfig.Debug.Fuse, newConfig.Debug.Gcs, newConfig.Debug.LogMutex)
	config.OverrideWithIgnoreInterruptsFlag(c, mountConfig, flags.IgnoreInterrupts)
	config.OverrideWithAnonymousAccessFlag(c, mountConfig, flags.AnonymousAccess)
	config.OverrideWithKernelListCacheTtlFlag(c, mountConfig, flags.KernelListCacheTtlSeconds)

	// Ideally this call to SetLogFormat (which internally creates a new defaultLogger)
	// should be set as an else to the 'if flags.Foreground' check below, but currently
	// that means the logs generated by resolveConfigFilePaths below don't honour
	// the user-provided log-format.
	logger.SetLogFormat(newConfig.Logging.Format)

	err = resolveConfigFilePaths(mountConfig)
	if err != nil {
		return fmt.Errorf("Resolving path: %w", err)
	}

	if newConfig.Foreground {
		err = logger.InitLogFile(mountConfig.LogConfig, newConfig.Logging)
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

	logger.Infof("Start gcsfuse/%s for app %q using mount point: %s\n", getVersion(), newConfig.AppName, mountPoint)

	// Log mount-config and the CLI flags in the log-file.
	// If there is no log-file, then log these to stdout.
	// Do not log these in stdout in case of daemonized run
	// if these are already being logged into a log-file, otherwise
	// there will be duplicate logs for these in both places (stdout and log-file).
	if newConfig.Foreground || newConfig.Logging.FilePath == "" {
		flagsStringified, err := util.Stringify(*flags)
		if err != nil {
			logger.Warnf("failed to stringify cli flags: %v", err)
		} else {
			logger.Infof("GCSFuse mount command flags: %s", flagsStringified)
		}

		mountConfigStringified, err := util.Stringify(*mountConfig)
		if err != nil {
			logger.Warnf("failed to stringify config-file: %v", err)
		} else {
			logger.Infof("GCSFuse mount config flags: %s", mountConfigStringified)
		}
	}

	// The following will not warn if the user explicitly passed the default value for StatCacheCapacity.
	if flags.StatCacheCapacity != mount.DefaultStatCacheCapacity {
		logger.Warnf("Deprecated flag stat-cache-capacity used! Please switch to config parameter 'metadata-cache: stat-cache-max-size-mb'.")
	}

	// The following will not warn if the user explicitly passed the default value for StatCacheTTL or TypeCacheTTL.
	if flags.StatCacheTTL != mount.DefaultStatOrTypeCacheTTL || flags.TypeCacheTTL != mount.DefaultStatOrTypeCacheTTL {
		logger.Warnf("Deprecated flag stat-cache-ttl and/or type-cache-ttl used! Please switch to config parameter 'metadata-cache: ttl-secs' .")
	}

	// If we haven't been asked to run in foreground mode, we should run a daemon
	// with the foreground flag set and wait for it to mount.
	if !newConfig.Foreground {
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
			return fmt.Errorf("daemonize.Run: %w", err)
		}
		logger.Infof(SuccessfulMountMessage)
		return err
	}

	// The returned error is ignored as we do not enforce monitoring exporters
	_ = monitor.EnableStackdriverExporter(newConfig.Metrics.StackdriverExportInterval)
	_ = monitor.EnableOpenTelemetryCollectorExporter(newConfig.Monitoring.ExperimentalOpentelemetryCollectorAddress)

	// Mount, writing information about our progress to the writer that package
	// daemonize gives us and telling it about the outcome.
	var mfs *fuse.MountedFileSystem
	{
		mfs, err = mountWithArgs(bucketName, mountPoint, flags, mountConfig, newConfig)

		// This utility is to absorb the error
		// returned by daemonize.SignalOutcome calls by simply
		// logging them as error logs.
		callDaemonizeSignalOutcome := func(err error) {
			if err2 := daemonize.SignalOutcome(err); err2 != nil {
				logger.Errorf("Failed to signal error to parent-process from daemon: %v", err2)
			}
		}

		markSuccessfulMount := func() {
			// Print the success message in the log-file/stdout depending on what the logger is set to.
			logger.Info(SuccessfulMountMessage)
			callDaemonizeSignalOutcome(nil)
		}

		markMountFailure := func(err error) {
			// Printing via mountStatus will have duplicate logs on the console while
			// mounting gcsfuse in foreground mode. But this is important to avoid
			// losing error logs when run in the background mode.
			logger.Errorf("%s: %v\n", UnsuccessfulMountMessagePrefix, err)
			err = fmt.Errorf("%s: mountWithArgs: %w", UnsuccessfulMountMessagePrefix, err)
			callDaemonizeSignalOutcome(err)
		}

		if err != nil {
			markMountFailure(err)
			return err
		}
		if !isDynamicMount(bucketName) {
			switch flags.ExperimentalMetadataPrefetchOnMount {
			case config.ExperimentalMetadataPrefetchOnMountSynchronous:
				if err = callListRecursive(mountPoint); err != nil {
					markMountFailure(err)
					return err
				}
			case config.ExperimentalMetadataPrefetchOnMountAsynchronous:
				go func() {
					if err := callListRecursive(mountPoint); err != nil {
						logger.Errorf("Metadata-prefetch failed: %v", err)
					}
				}()
			}
		}
		markSuccessfulMount()
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

func ExecuteLegacyMain() {
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
