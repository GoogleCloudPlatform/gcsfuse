// Copyright 2024 Google LLC
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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"golang.org/x/sys/unix"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/canned"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/kernelparams"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/monitor"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/mount"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/profiler"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/jacobsa/daemonize"
	"github.com/jacobsa/fuse"
	"github.com/kardianos/osext"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
)

const (
	SuccessfulMountMessage         = "File system has been successfully mounted."
	UnsuccessfulMountMessagePrefix = "Error while mounting gcsfuse"
	MountSlownessMessage           = "Mount slowness detected: mount time %v exceeded threshold %v"
	DynamicMountFSName             = "gcsfuse"
	WaitTimeOnSignalReceive        = 30 * time.Second
	MountTimeThreshold             = 8 * time.Second
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func registerTerminatingSignalHandler(mountPoint string) {
	// Register for SIGINT.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, unix.SIGTERM)

	// Start a goroutine that will unmount when the signal is received.
	go func() {
		for {
			sig := <-signalChan
			sigName := "undefined"
			switch sig {
			case unix.SIGTERM:
				sigName = "SIGTERM"
			case os.Interrupt:
				sigName = "SIGINT"
			}

			//On signal receive wait in background and give 30 second for unmount to finish
			//and then exit, so application is closed.
			go func() {
				logger.Warnf("Received %s, waiting for %s to let system gracefully unmount before killing the process", sigName, WaitTimeOnSignalReceive)
				time.Sleep(WaitTimeOnSignalReceive)
				logger.Warnf("killing goroutines and exit")
				//Forcefully exit to 0 so that caller get success on forcefull exit also.
				os.Exit(0)
			}()

			logger.Warnf("Received %s, attempting to unmount...", sigName)
			err := fuse.Unmount(mountPoint)
			if err != nil {
				if errors.Is(err, fuse.ErrExternallyManagedMountPoint) {
					logger.Infof("Mount point %s is externally managed; gcsfuse will not unmount it.", mountPoint)
					return
				}
				logger.Errorf("Failed to unmount in response to %s: %v", sigName, err)
			} else {
				logger.Infof("Successfully unmounted in response to %s.", sigName)
				return
			}
		}
	}()
}

func getUserAgent(appName, config, mountInstanceID string) string {
	var userAgent string
	gcsfuseMetadataImageType := os.Getenv("GCSFUSE_METADATA_IMAGE_TYPE")
	if len(gcsfuseMetadataImageType) > 0 {
		userAgent = fmt.Sprintf("gcsfuse/%s %s (GPN:gcsfuse-%s) (Cfg:%s)", common.GetVersion(), appName, gcsfuseMetadataImageType, config)
		userAgent = strings.Join(strings.Fields(userAgent), " ")
	} else if len(appName) > 0 {
		userAgent = fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-%s) (Cfg:%s)", common.GetVersion(), appName, config)
	} else {
		userAgent = fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse) (Cfg:%s)", common.GetVersion(), config)
	}
	return fmt.Sprintf("%s (mount-id:%s)", userAgent, mountInstanceID)
}

func boolToBin(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func getConfigForUserAgent(mountConfig *cfg.Config) string {
	// Minimum configuration details created in a bitset fashion.
	parts := []string{
		boolToBin(cfg.IsFileCacheEnabled(mountConfig)),
		boolToBin(mountConfig.FileCache.CacheFileForRangeRead),
		boolToBin(cfg.IsParallelDownloadsEnabled(mountConfig)),
		boolToBin(mountConfig.Write.EnableStreamingWrites),
		boolToBin(mountConfig.Read.EnableBufferedRead),
		boolToBin(mountConfig.Profile != ""),
	}
	return strings.Join(parts, ":")
}
func createStorageHandle(newConfig *cfg.Config, userAgent string, metricHandle metrics.MetricHandle, isGKE bool) (storageHandle storage.StorageHandle, err error) {
	storageClientConfig := storageutil.StorageClientConfig{
		ClientProtocol:             newConfig.GcsConnection.ClientProtocol,
		MaxConnsPerHost:            int(newConfig.GcsConnection.MaxConnsPerHost),
		MaxIdleConnsPerHost:        int(newConfig.GcsConnection.MaxIdleConnsPerHost),
		HttpClientTimeout:          newConfig.GcsConnection.HttpClientTimeout,
		MaxRetrySleep:              newConfig.GcsRetries.MaxRetrySleep,
		MaxRetryAttempts:           int(newConfig.GcsRetries.MaxRetryAttempts),
		RetryMultiplier:            newConfig.GcsRetries.Multiplier,
		UserAgent:                  userAgent,
		CustomEndpoint:             newConfig.GcsConnection.CustomEndpoint,
		KeyFile:                    string(newConfig.GcsAuth.KeyFile),
		AnonymousAccess:            newConfig.GcsAuth.AnonymousAccess,
		TokenUrl:                   newConfig.GcsAuth.TokenUrl,
		ReuseTokenFromUrl:          newConfig.GcsAuth.ReuseTokenFromUrl,
		ExperimentalEnableJsonRead: newConfig.GcsConnection.ExperimentalEnableJsonRead,
		GrpcConnPoolSize:           int(newConfig.GcsConnection.GrpcConnPoolSize),
		EnableHNS:                  newConfig.EnableHns,
		EnableGoogleLibAuth:        newConfig.EnableGoogleLibAuth,
		ReadStallRetryConfig:       newConfig.GcsRetries.ReadStall,
		MetricHandle:               metricHandle,
		TracingEnabled:             cfg.IsTracingEnabled(newConfig),
		EnableHTTPDNSCache:         newConfig.GcsConnection.EnableHttpDnsCache,
		LocalSocketAddress:         newConfig.GcsConnection.ExperimentalLocalSocketAddress,
		EnableGrpcMetrics:          newConfig.Metrics.ExperimentalEnableGrpcMetrics,
		IsGKE:                      isGKE,
	}
	logger.Infof("UserAgent = %s\n", storageClientConfig.UserAgent)
	storageHandle, err = storage.NewStorageHandle(context.Background(), storageClientConfig, newConfig.GcsConnection.BillingProject)
	return
}

////////////////////////////////////////////////////////////////////////
// main logic
////////////////////////////////////////////////////////////////////////

// Mount the file system according to arguments in the supplied context.
func mountWithArgs(bucketName string, mountPoint string, newConfig *cfg.Config, metricHandle metrics.MetricHandle, traceHandle tracing.TraceHandle, viperConfig *viper.Viper) (mfs *fuse.MountedFileSystem, err error) {
	// Enable invariant checking if requested.
	if newConfig.Debug.ExitOnInvariantViolation {
		locker.EnableInvariantsCheck()
	}
	if newConfig.Debug.LogMutex {
		locker.EnableDebugMessages()
	}
	// Parse the mountPoint string and detect whether or not in GKE environment
	isGKE := cfg.IsGKEEnvironment(mountPoint)

	// Grab the connection.
	//
	// Special case: if we're mounting the fake bucket, we don't need an actual
	// connection.
	var storageHandle storage.StorageHandle
	if bucketName != canned.FakeBucketName {
		userAgent := getUserAgent(newConfig.AppName, getConfigForUserAgent(newConfig), logger.MountInstanceID(fsName(bucketName)))
		logger.Info("Creating Storage handle...")
		storageHandle, err = createStorageHandle(newConfig, userAgent, metricHandle, isGKE)
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
		storageHandle,
		metricHandle,
		traceHandle,
		viperConfig)

	if err != nil {
		err = fmt.Errorf("mountWithStorageHandle: %w", err)
		return
	}

	return
}

func populateArgs(args []string) (
	bucketName string,
	mountPoint string,
	err error) {
	// Extract arguments.
	switch len(args) {
	case 1:
		bucketName = ""
		mountPoint = args[0]

	case 2:
		bucketName = args[0]
		mountPoint = args[1]

	default:
		err = fmt.Errorf(
			"%s takes one or two arguments. Run `%s --help` for more info",
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

func fsName(bucketName string) string {
	if isDynamicMount(bucketName) {
		return DynamicMountFSName
	}
	return bucketName
}

// forwardedEnvVars collects and returns all the environment
// variables which should be sent to the gcsfuse daemon
// process in case of background run.
func forwardedEnvVars() []string {
	// Pass along PATH so that the daemon can find fusermount on Linux.
	env := []string{
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
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

	// Forward GOOGLE_APPLICATION_CREDENTIALS, since we document in
	// mounting.md that it can be used for specifying a key file.
	// Forward the no_proxy environment variable. Whenever
	// using the http(s)_proxy environment variables. This should
	// also be included to know for which hosts the use of proxies
	// should be ignored.
	// Forward GCE_METADATA_HOST, GCE_METADATA_ROOT, GCE_METADATA_IP as these are used for mocked metadata services.
	// Forward GRPC_GO_LOG_VERBOSITY_LEVEL and GRPC_GO_LOG_SEVERITY_LEVEL as these are used to enable grpc debug logs.
	for _, envvar := range []string{"GOOGLE_APPLICATION_CREDENTIALS", "no_proxy", "GCE_METADATA_HOST", "GCE_METADATA_ROOT", "GCE_METADATA_IP", "GRPC_GO_LOG_VERBOSITY_LEVEL", "GRPC_GO_LOG_SEVERITY_LEVEL"} {
		if envval, ok := os.LookupEnv(envvar); ok {
			env = append(env, fmt.Sprintf("%s=%s", envvar, envval))
			fmt.Fprintf(
				os.Stdout,
				"Added environment %s: %s\n",
				envvar, envval)
		}
	}

	// Pass the parent process working directory to child process via
	// environment variable. This variable will be used to resolve relative paths.
	if parentProcessExecutionDir, err := os.Getwd(); err == nil {
		env = append(env, fmt.Sprintf("%s=%s", util.GCSFUSE_PARENT_PROCESS_DIR,
			parentProcessExecutionDir))
	}

	// Here, parent process doesn't pass the $HOME to child process implicitly,
	// hence we need to pass it explicitly.
	if homeDir, err := os.UserHomeDir(); err == nil {
		env = append(env, fmt.Sprintf("HOME=%s", homeDir))
	}

	// This environment variable will be helpful to distinguish b/w the main
	// process and daemon process. If this environment variable set that means
	// programme is running as daemon process.
	env = append(env, fmt.Sprintf("%s=true", logger.GCSFuseInBackgroundMode))

	// This environment variable is used to enhance gcsfuse logging by using unique
	// MountUUID to identify logs from different mounts.
	// MountUUID is used here instead of the MountInstanceID for unified logic
	// in callers of MountInstaceID in both background and foreground mode.
	env = append(env, fmt.Sprintf("%s=%s", logger.MountUUIDEnvKey, logger.MountUUID()))
	return env
}

// logGCSFuseMountInformation logs the CLI flags, config file flags and the resolved config.
func logGCSFuseMountInformation(mountInfo *mountInfo) {
	logger.Info("GCSFuse Config", "CLI Flags", mountInfo.cliFlags)
	if mountInfo.configFileFlags != nil {
		logger.Info("GCSFuse Config", "ConfigFile Flags", mountInfo.configFileFlags)
	}
	if len(mountInfo.optimizedFlags) > 0 {
		logger.Info("GCSFuse Config", "Optimized Flags", mountInfo.optimizedFlags)
	}
	logger.Info("GCSFuse Config", "Full Config", mountInfo.config)
}

func Mount(mountInfo *mountInfo, bucketName, mountPoint string) (err error) {
	newConfig := mountInfo.config
	// Ideally this call to UpdateDefaultLogger (which internally creates a
	// new defaultLogger with user provided log-format and custom attribute 'fsName-MountInstanceID')
	// should be set as an else to the 'if flags.Foreground' check below, but currently
	// that means the logs generated by resolveConfigFilePaths below don't honour
	// the user-provided log-format.
	logger.UpdateDefaultLogger(newConfig.Logging.Format, fsName(bucketName))

	if newConfig.Foreground {
		err = logger.InitLogFile(newConfig.Logging, fsName(bucketName))
		if err != nil {
			return fmt.Errorf("init log file: %w", err)
		}
	}

	logger.Infof("Start gcsfuse/%s for app %q using mount point: %s\n", common.GetVersion(), newConfig.AppName, mountPoint)

	// Log mount-config and the CLI flags in the log-file.
	// If there is no log-file, then log these to stdout.
	// Do not log these in stdout in case of daemonized run
	// if these are already being logged into a log-file, otherwise
	// there will be duplicate logs for these in both places (stdout and log-file).
	if newConfig.Foreground || newConfig.Logging.FilePath == "" {
		logGCSFuseMountInformation(mountInfo)
	}

	// The following will not warn if the user explicitly passed the default value for StatCacheCapacity.
	if newConfig.MetadataCache.DeprecatedStatCacheCapacity != mount.DefaultStatCacheCapacity {
		logger.Warnf("Deprecated flag stat-cache-capacity used! Please switch to config parameter 'metadata-cache: stat-cache-max-size-mb'.")
	}

	// The following will not warn if the user explicitly passed the default value for StatCacheTTL or TypeCacheTTL.
	if newConfig.MetadataCache.DeprecatedStatCacheTtl != mount.DefaultStatOrTypeCacheTTL || newConfig.MetadataCache.DeprecatedTypeCacheTtl != mount.DefaultStatOrTypeCacheTTL {
		logger.Warnf("Deprecated flag stat-cache-ttl and/or type-cache-ttl used! Please switch to config parameter 'metadata-cache: ttl-secs' .")
	}

	if newConfig.EnableTypeCacheDeprecation && (newConfig.MetadataCache.TypeCacheMaxSizeMb != mount.DefaultTypeCacheSizeMB || newConfig.MetadataCache.DeprecatedTypeCacheTtl != mount.DefaultStatOrTypeCacheTTL) {
		logger.Warnf("Type cache is deprecated. The flags 'type-cache-max-size-mb' and 'type-cache-ttl' will be ignored.")
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

		env := forwardedEnvVars()

		// logfile.stderr will capture the standard error (stderr) output of the gcsfuse background process.
		var stderrFile *os.File
		if newConfig.Logging.FilePath != "" {
			stderrFileName := string(newConfig.Logging.FilePath) + ".stderr"
			if stderrFile, err = os.OpenFile(stderrFileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err != nil {
				return err
			}
		}
		// Run.
		startTime := time.Now()
		err = daemonize.Run(path, args, env, os.Stdout, stderrFile)
		if err != nil {
			return fmt.Errorf("daemonize.Run: %w", err)
		}
		mountDuration := time.Since(startTime)
		if mountDuration > MountTimeThreshold {
			logger.Warnf(MountSlownessMessage, mountDuration, MountTimeThreshold)
		}
		logger.Infof(SuccessfulMountMessage)
		return err
	}

	ctx := context.Background()
	var metricExporterShutdownFn common.ShutdownFn
	metricHandle := metrics.NewNoopMetrics()
	if cfg.IsMetricsEnabled(&newConfig.Metrics) {
		metricExporterShutdownFn = monitor.SetupOTelMetricExporters(ctx, newConfig, logger.MountInstanceID(fsName(bucketName)))
		if metricHandle, err = metrics.NewOTelMetrics(ctx, int(newConfig.Metrics.Workers), int(newConfig.Metrics.BufferSize)); err != nil {
			metricHandle = metrics.NewNoopMetrics()
		}
	}
	shutdownTracingFn := monitor.SetupTracing(ctx, newConfig, logger.MountInstanceID(fsName(bucketName)))
	traceHandle := tracing.NewNoopTracer()
	if cfg.IsTracingEnabled(newConfig) {
		traceHandle = tracing.NewOTELTracer()
	}

	shutdownFn := common.JoinShutdownFunc(metricExporterShutdownFn, shutdownTracingFn)

	// No-op if profiler is disabled.
	if err := profiler.SetupCloudProfiler(&newConfig.CloudProfiler); err != nil {
		logger.Warnf("Failed to setup cloud profiler: %v", err)
	}

	// Mount, writing information about our progress to the writer that package
	// daemonize gives us and telling it about the outcome.
	var mfs *fuse.MountedFileSystem
	{
		startTime := time.Now()
		mfs, err = mountWithArgs(bucketName, mountPoint, newConfig, metricHandle, traceHandle, mountInfo.viperConfig)

		// This utility is to absorb the error
		// returned by daemonize.SignalOutcome calls by simply
		// logging them as error logs.
		callDaemonizeSignalOutcome := func(err error) {
			if err2 := daemonize.SignalOutcome(err); err2 != nil {
				logger.Errorf("Failed to signal error to parent-process from daemon: %v", err2)
			}
		}

		markSuccessfulMount := func() {
			mountDuration := time.Since(startTime)
			if mountDuration > MountTimeThreshold {
				logger.Warnf(MountSlownessMessage, mountDuration, MountTimeThreshold)
			}
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
			switch newConfig.MetadataCache.ExperimentalMetadataPrefetchOnMount {
			case cfg.ExperimentalMetadataPrefetchOnMountSynchronous:
				if err = callListRecursive(mountPoint); err != nil {
					markMountFailure(err)
					return err
				}
			case cfg.ExperimentalMetadataPrefetchOnMountAsynchronous:
				go func() {
					if err := callListRecursive(mountPoint); err != nil {
						logger.Errorf("Metadata-prefetch failed: %v", err)
					}
				}()
			}
		}
		markSuccessfulMount()

		// Apply post mount kernel settings in non-GKE environments for non dynamic mounts when kernel reader is enabled.
		if !isDynamicMount(bucketName) && !cfg.IsGKEEnvironment(mountPoint) && newConfig.FileSystem.EnableKernelReader {
			kernelparams := kernelparams.NewKernelParamsManager()
			kernelparams.SetReadAheadKb(int(newConfig.FileSystem.MaxReadAheadKb))
			kernelparams.SetCongestionWindowThreshold(int(newConfig.FileSystem.CongestionThreshold))
			kernelparams.SetMaxBackgroundRequests(int(newConfig.FileSystem.MaxBackground))
			kernelparams.ApplyNonGKE(mountPoint)
		}
	}

	// Let the user unmount with Ctrl-C (SIGINT).
	registerTerminatingSignalHandler(mfs.Dir())

	// Wait for the file system to be unmounted.
	if err = mfs.Join(ctx); err != nil {
		err = fmt.Errorf("MountedFileSystem.Join: %w", err)
	}

	if shutdownFn != nil {
		if shutdownErr := shutdownFn(ctx); shutdownErr != nil {
			logger.Errorf("Error while shutting down dependencies: %v", shutdownErr)
		}
	}

	return err
}
