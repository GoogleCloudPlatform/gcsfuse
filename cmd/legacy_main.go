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
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"golang.org/x/sys/unix"
	"golang.org/x/term"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/canned"
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
	"golang.org/x/net/context"
)

const (
	SuccessfulMountMessage         = "File system has been successfully mounted."
	UnsuccessfulMountMessagePrefix = "Error while mounting gcsfuse"
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
			logger.Infof("Received %s, attempting to unmount...", sigName)

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

func getUserAgent(appName string, config string) string {
	gcsfuseMetadataImageType := os.Getenv("GCSFUSE_METADATA_IMAGE_TYPE")
	if len(gcsfuseMetadataImageType) > 0 {
		userAgent := fmt.Sprintf("gcsfuse/%s %s (GPN:gcsfuse-%s) (Cfg:%s)", common.GetVersion(), appName, gcsfuseMetadataImageType, config)
		return strings.Join(strings.Fields(userAgent), " ")
	} else if len(appName) > 0 {
		return fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-%s) (Cfg:%s)", common.GetVersion(), appName, config)
	} else {
		return fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse) (Cfg:%s)", common.GetVersion(), config)
	}
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
func createStorageHandle(newConfig *cfg.Config, userAgent string, metricHandle metrics.MetricHandle) (storageHandle storage.StorageHandle, err error) {
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
	}
	logger.Infof("UserAgent = %s\n", storageClientConfig.UserAgent)
	storageHandle, err = storage.NewStorageHandle(context.Background(), storageClientConfig, newConfig.GcsConnection.BillingProject)
	return
}

////////////////////////////////////////////////////////////////////////
// main logic
////////////////////////////////////////////////////////////////////////

// Mount the file system according to arguments in the supplied context.
func mountWithArgs(bucketName string, mountPoint string, newConfig *cfg.Config, metricHandle metrics.MetricHandle) (mfs *fuse.MountedFileSystem, err error) {
	// Enable invariant checking if requested.
	if newConfig.Debug.ExitOnInvariantViolation {
		locker.EnableInvariantsCheck()
	}
	if newConfig.Debug.LogMutex {
		locker.EnableDebugMessages()
	}

	// Grab the connection.
	//
	// Special case: if we're mounting the fake bucket, we don't need an actual
	// connection.
	var storageHandle storage.StorageHandle
	if bucketName != canned.FakeBucketName {
		userAgent := getUserAgent(newConfig.AppName, getConfigForUserAgent(newConfig))
		logger.Info("Creating Storage handle...")
		storageHandle, err = createStorageHandle(newConfig, userAgent, metricHandle)
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
		metricHandle)

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
	return env
}

func Mount(newConfig *cfg.Config, bucketName, mountPoint string) (err error) {
	// If bug-report-path is not set, just run gcsfuse normally.
	if newConfig.BugReportPath == "" {
		return runMountProcess(newConfig, bucketName, mountPoint)
	}

	// If we are in the child process, run gcsfuse normally.
	if os.Getenv("GCSFUSE_BUG_REPORT_CHILD") == "true" {
		return runMountProcess(newConfig, bucketName, mountPoint)
	}

	// This is the parent process. It will monitor the child and generate the bug report.
	tempDir, err := os.MkdirTemp("", "gcsfuse-bug-report")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory for bug report: %w", err)
	}
	defer os.RemoveAll(tempDir)
	logger.Infof("Bug report temporary directory: %s", tempDir)

	// If log-file is not set, create one in the temporary directory.
	if newConfig.Logging.FilePath == "" {
		newConfig.Logging.FilePath = cfg.ResolvedPath(filepath.Join(tempDir, "gcsfuse.log"))
	}

	// Start collecting dmesg output.
	dmesgLogFile := filepath.Join(tempDir, "dmesg.log")
	dmesgScript := fmt.Sprintf("dmesg -T -w >> %s", dmesgLogFile)
	if !checkDmesgPermissions() {
		// SECURITY: This is a security risk. We are prompting for a password and
		// passing it to sudo on the command line. This can expose the password
		// in the system's process list. The user has insisted on this feature.
		fmt.Println("GCSFuse needs superuser permission to collect dmesg logs for the bug report.")
		fmt.Print("Please enter your password to proceed: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		password := string(bytePassword)
		fmt.Println()

		dmesgScript = fmt.Sprintf("echo %s | sudo -S -p '' -- %s", password, dmesgScript)
	}
	dmesgCmd, err := startCollectionScript(dmesgScript, dmesgLogFile)
	if err != nil {
		return fmt.Errorf("failed to start dmesg collection: %w", err)
	}
	logger.Infof("Started dmesg collection.")

	// Start collecting system stats.
	statsLogFile := filepath.Join(tempDir, "stats.log")
	statsScript := fmt.Sprintf("vmstat 1 >> %s", statsLogFile)
	statsCmd, err := startCollectionScript(statsScript, statsLogFile)
	if err != nil {
		return fmt.Errorf("failed to start stats collection: %w", err)
	}
	logger.Infof("Started stats collection.")

	// Re-exec gcsfuse without the bug-report-path flag and with required flags.
	var newArgs []string
	hasForeground := false
	hasLogFile := false
	hasLogSeverity := false
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if strings.HasPrefix(arg, "--bug-report-path") {
			if !strings.Contains(arg, "=") {
				i++
			}
			continue
		}
		if arg == "--foreground" {
			hasForeground = true
		}
		if strings.HasPrefix(arg, "--log-file") {
			hasLogFile = true
		}
		if strings.HasPrefix(arg, "--log-severity") {
			hasLogSeverity = true
		}
		newArgs = append(newArgs, arg)
	}
	if !hasForeground {
		newArgs = append(newArgs, "--foreground")
	}
	if !hasLogFile {
		newArgs = append(newArgs, "--log-file="+string(newConfig.Logging.FilePath))
	}
	if !hasLogSeverity {
		newArgs = append(newArgs, "--log-severity=TRACE")
	}

	cmd := exec.Command(os.Args[0], newArgs...)
	cmd.Env = append(os.Environ(), "GCSFUSE_BUG_REPORT_CHILD=true")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the child process.
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start gcsfuse child process: %w", err)
	}

	// Set up a channel to catch signals.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Start a goroutine to forward signals to the child process.
	go func() {
		for sig := range signalChan {
			logger.Infof("Parent process received signal: %v. Forwarding to child process.", sig)
			if err := cmd.Process.Signal(sig); err != nil {
				logger.Errorf("Failed to forward signal to child process: %v", err)
			}
		}
	}()

	// Wait for the child process to exit.
	runErr := cmd.Wait()

	// Stop the collection scripts.
	stopCollectionScript(dmesgCmd)
	logger.Infof("Stopped dmesg collection.")
	stopCollectionScript(statsCmd)
	logger.Infof("Stopped stats collection.")

	// Create the tarball.
	tarballPath := filepath.Join(string(newConfig.BugReportPath), fmt.Sprintf("gcsfuse-bug-report-%s.tar.gz", time.Now().Format("2006-01-02-15-04-05")))
	if err := createTarball(tarballPath, dmesgLogFile, statsLogFile, string(newConfig.Logging.FilePath)); err != nil {
		logger.Errorf("failed to create bug report tarball: %v", err)
	} else {
		logger.Infof("Bug report saved to %s", tarballPath)
	}

	return runErr
}

func runMountProcess(newConfig *cfg.Config, bucketName, mountPoint string) (err error) {
	// Ideally this call to SetLogFormat (which internally creates a new defaultLogger)
	// should be set as an else to the 'if flags.Foreground' check below, but currently
	// that means the logs generated by resolveConfigFilePaths below don't honour
	// the user-provided log-format.
	logger.SetLogFormat(newConfig.Logging.Format)

	if newConfig.Foreground {
		err = logger.InitLogFile(newConfig.Logging)
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
		logger.Info("GCSFuse config", "config", newConfig)
	}

	// The following will not warn if the user explicitly passed the default value for StatCacheCapacity.
	if newConfig.MetadataCache.DeprecatedStatCacheCapacity != mount.DefaultStatCacheCapacity {
		logger.Warnf("Deprecated flag stat-cache-capacity used! Please switch to config parameter 'metadata-cache: stat-cache-max-size-mb'.")
	}

	// The following will not warn if the user explicitly passed the default value for StatCacheTTL or TypeCacheTTL.
	if newConfig.MetadataCache.DeprecatedStatCacheTtl != mount.DefaultStatOrTypeCacheTTL || newConfig.MetadataCache.DeprecatedTypeCacheTtl != mount.DefaultStatOrTypeCacheTTL {
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
		err = daemonize.Run(path, args, env, os.Stdout, stderrFile)
		if err != nil {
			return fmt.Errorf("daemonize.Run: %w", err)
		}
		logger.Infof(SuccessfulMountMessage)
		return err
	}

	ctx := context.Background()
	var metricExporterShutdownFn common.ShutdownFn
	metricHandle := metrics.NewNoopMetrics()
	if cfg.IsMetricsEnabled(&newConfig.Metrics) {
		metricExporterShutdownFn = monitor.SetupOTelMetricExporters(ctx, newConfig)
		if metricHandle, err = metrics.NewOTelMetrics(ctx, int(newConfig.Metrics.Workers), int(newConfig.Metrics.BufferSize)); err != nil {
			metricHandle = metrics.NewNoopMetrics()
		}
	}
	shutdownTracingFn := monitor.SetupTracing(ctx, newConfig)
	shutdownFn := common.JoinShutdownFunc(metricExporterShutdownFn, shutdownTracingFn)

	// No-op if profiler is disabled.
	if err := profiler.SetupCloudProfiler(&newConfig.CloudProfiler); err != nil {
		logger.Warnf("Failed to setup cloud profiler: %v", err)
	}

	// Mount, writing information about our progress to the writer that package
	// daemonize gives us and telling it about the outcome.
	var mfs *fuse.MountedFileSystem
	{
		mfs, err = mountWithArgs(bucketName, mountPoint, newConfig, metricHandle)

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
	}

	// Let the user unmount with Ctrl-C (SIGINT).
	registerTerminatingSignalHandler(mfs.Dir())

	// Wait for the file system to be unmounted.
	if err = mfs.Join(ctx); err != nil {
		err = fmt.Errorf("MountedFileSystem.Join: %w", err)
	}

	if shutdownFn != nil {
		if shutdownErr := shutdownFn(ctx); shutdownErr != nil {
			logger.Errorf("Error while shutting down trace exporter: %v", shutdownErr)
		}
	}

	return err
}

func checkDmesgPermissions() bool {
	cmd := exec.Command("dmesg", "-T")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func startCollectionScript(script, logFile string) (*exec.Cmd, error) {
	// We need to create the log file before starting the script.
	// Otherwise, the script will fail if the directory doesn't exist.
	f, err := os.Create(logFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file %s: %w", logFile, err)
	}
	f.Close()

	cmd := exec.Command("bash", "-c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start script: %w", err)
	}
	return cmd, nil
}

func stopCollectionScript(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		// If the process is already dead, we'll get an error.
		// In that case, we can just ignore it.
		return
	}
	if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
		// If the process group is already dead, we'll get an error.
		// In that case, we can just ignore it.
	}
}

func createTarball(tarballPath, dmesgLog, statsLog, gcsfuseLog string) error {
	// Create the tarball file.
	tarball, err := os.Create(tarballPath)
	if err != nil {
		return fmt.Errorf("failed to create tarball file: %w", err)
	}
	defer tarball.Close()

	// Create a new gzip writer.
	gz := gzip.NewWriter(tarball)
	defer gz.Close()

	// Create a new tar writer.
	tw := tar.NewWriter(gz)
	defer tw.Close()

	// Add files to the tarball.
	files := []struct {
		Name   string
		Source string
	}{
		{"dmesg.log", dmesgLog},
		{"stats.log", statsLog},
	}

	if gcsfuseLog != "" {
		files = append(files, struct {
			Name   string
			Source string
		}{"gcsfuse.log", gcsfuseLog})
	}

	for _, file := range files {
		if file.Source == "" {
			continue
		}
		if err := addFileToTar(tw, file.Source, file.Name); err != nil {
			logger.Warnf("failed to add %s to tarball: %v", file.Source, err)
		}
	}

	return nil
}

func addFileToTar(tw *tar.Writer, path, name string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name:    name,
		Mode:    int64(stat.Mode()),
		Size:    stat.Size(),
		ModTime: stat.ModTime(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	if _, err := io.Copy(tw, file); err != nil {
		return err
	}

	return nil
}
