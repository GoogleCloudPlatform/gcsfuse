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
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/internal/mount"
	mountpkg "github.com/googlecloudplatform/gcsfuse/internal/mount"
	"github.com/googlecloudplatform/gcsfuse/internal/util"
	"github.com/urfave/cli"
)

// Defines the max value supported by sequential-read-size-mb flag.
const (
	// maxSequentialReadSizeMb is the max value supported by sequential-read-size-mb flag.
	maxSequentialReadSizeMb = 1024
)

// Set up custom help text for gcsfuse; in particular the usage section.
func init() {
	cli.AppHelpTemplate = `NAME:
   {{.Name}} - {{.Usage}}

USAGE:
   {{.Name}} {{if .Flags}}[global options]{{end}} [bucket] mountpoint
   {{if .Version}}
VERSION:
   {{.Version}}
   {{end}}{{if len .Authors}}
AUTHOR(S):
   {{range .Authors}}{{ . }}{{end}}
   {{end}}{{if .Flags}}
GLOBAL OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}{{end}}{{if .Copyright }}
COPYRIGHT:
   {{.Copyright}}
   {{end}}
`
}

func newApp() (app *cli.App) {
	dirModeValue := new(OctalInt)
	*dirModeValue = 0755

	fileModeValue := new(OctalInt)
	*fileModeValue = 0644

	app = &cli.App{
		Name:    "gcsfuse",
		Version: getVersion(),
		Usage:   "Mount a specified GCS bucket or all accessible buckets locally",
		Writer:  os.Stderr,
		Flags: []cli.Flag{

			cli.StringFlag{
				Name:  "app-name",
				Value: "",
				Usage: "The application name of this mount.",
			},

			cli.BoolFlag{
				Name:  "foreground",
				Usage: "Stay in the foreground after mounting.",
			},

			cli.StringFlag{
				Name:  "config-file",
				Value: "",
				Usage: "The path to the config file where all gcsfuse related config needs to be specified. " +
					"Refer to 'https://cloud.google.com/storage/docs/gcsfuse-cli#config-file' for possible configurations.",
			},

			/////////////////////////
			// File system
			/////////////////////////

			cli.StringSliceFlag{
				Name:  "o",
				Usage: "Additional system-specific mount options. Multiple options can be passed as comma separated. For readonly, use --o ro",
			},

			cli.GenericFlag{
				Name:  "dir-mode",
				Value: dirModeValue,
				Usage: "Permissions bits for directories, in octal.",
			},

			cli.GenericFlag{
				Name:  "file-mode",
				Value: fileModeValue,
				Usage: "Permission bits for files, in octal.",
			},

			cli.IntFlag{
				Name:  "uid",
				Value: -1,
				Usage: "UID owner of all inodes.",
			},

			cli.IntFlag{
				Name:  "gid",
				Value: -1,
				Usage: "GID owner of all inodes.",
			},

			cli.BoolFlag{
				Name:  "implicit-dirs",
				Usage: "Implicitly define directories based on content. See files and directories in docs/semantics for more information",
			},

			cli.StringFlag{
				Name:  "only-dir",
				Usage: "Mount only a specific directory within the bucket. See docs/mounting for more information",
			},

			cli.IntFlag{
				Name:  "rename-dir-limit",
				Value: 0,
				Usage: "Allow rename a directory containing fewer descendants than this limit.",
			},

			/////////////////////////
			// GCS
			/////////////////////////

			cli.StringFlag{
				Name: "custom-endpoint",
				Usage: "Specifies an alternative custom endpoint for fetching data. Should only be used for testing. " +
					"The custom endpoint must support the equivalent resources and operations as the GCS " +
					"JSON endpoint, https://storage.googleapis.com/storage/v1. If a custom endpoint is not specified, " +
					"GCSFuse uses the global GCS JSON API endpoint, https://storage.googleapis.com/storage/v1. " +
					"If a custom endpoint is specified, authentication is disabled on the endpoint.",
			},

			cli.StringFlag{
				Name:  "billing-project",
				Value: "",
				Usage: "Project to use for billing when accessing a bucket enabled with “Requester Pays” (default: none)",
			},

			cli.StringFlag{
				Name:  "key-file",
				Value: "",
				Usage: "Absolute path to JSON key file for use with GCS. (default: none, Google application default credentials used)",
			},

			cli.StringFlag{
				Name:  "token-url",
				Value: "",
				Usage: "A url for getting an access token when the key-file is absent.",
			},

			cli.BoolTFlag{
				Name:  "reuse-token-from-url",
				Usage: "If false, the token acquired from token-url is not reused.",
			},

			cli.Float64Flag{
				Name:  "limit-bytes-per-sec",
				Value: -1,
				Usage: "Bandwidth limit for reading data, measured over a 30-second window. (use -1 for no limit)",
			},

			cli.Float64Flag{
				Name:  "limit-ops-per-sec",
				Value: -1,
				Usage: "Operations per second limit, measured over a 30-second window (use -1 for no limit)",
			},

			cli.IntFlag{
				Name:  "sequential-read-size-mb",
				Value: 200,
				Usage: "File chunk size to read from GCS in one call. Need to specify the value in MB. ChunkSize less than 1MB is not supported",
			},

			/////////////////////////
			// Tuning
			/////////////////////////

			cli.DurationFlag{
				Name:  "max-retry-sleep",
				Value: 30 * time.Second,
				Usage: "The maximum duration allowed to sleep in a retry loop with exponential backoff for failed requests to GCS backend." +
					" Once the backoff duration exceeds this limit, the retry continues with this specified maximum value.",
			},

			cli.IntFlag{
				Name:  "stat-cache-capacity",
				Value: mount.DefaultStatCacheCapacity,
				Usage: "How many entries can the stat-cache hold (impacts memory consumption). This flag has been deprecated (starting v2.0) and in its place only metadata-cache:stat-cache-max-size-mb in the gcsfuse config-file will be supported. For now, the value of stat-cache-capacity will be translated to the next higher corresponding value of metadata-cache:stat-cache-max-size-mb (assuming stat-cache entry-size ~= 1640 bytes, including 1400 for positive entry and 240 for corresponding negative entry), when metadata-cache:stat-cache-max-size-mb is not set.",
			},

			cli.DurationFlag{
				Name:  "stat-cache-ttl",
				Value: mount.DefaultStatOrTypeCacheTTL,
				Usage: "How long to cache StatObject results and inode attributes. This flag has been deprecated (starting v2.0) and in its place only metadata-cache:ttl-secs in the gcsfuse config-file will be supported. For now, the minimum of stat-cache-ttl and type-cache-ttl values, rounded up to the next higher multiple of a second, is used as ttl for both stat-cache and type-cache, when metadata-cache:ttl-secs is not set.",
			},

			cli.DurationFlag{
				Name:  "type-cache-ttl",
				Value: mount.DefaultStatOrTypeCacheTTL,
				Usage: "How long to cache name -> file/dir mappings in directory inodes. This flag has been deprecated (starting v2.0) and in its place only metadata-cache:ttl-secs in the gcsfuse config-file will be supported. For now, the minimum of stat-cache-ttl and type-cache-ttl values, rounded up to the next higher multiple of a second, is used as ttl for both stat-cache and type-cache, when metadata-cache:ttl-secs is not set.",
			},

			cli.DurationFlag{
				Name:  "http-client-timeout",
				Usage: "The time duration that http client will wait to get response from the server. The default value 0 indicates no timeout. ",
			},

			cli.DurationFlag{
				Name:  "max-retry-duration",
				Value: -1 * time.Second,
				Usage: "This flag is currently unused.",
			},

			cli.Float64Flag{
				Name:  "retry-multiplier",
				Value: 2,
				Usage: "Param for exponential backoff algorithm, which is used to increase waiting time b/w two consecutive retries.",
			},

			cli.StringFlag{
				Name:  "temp-dir",
				Value: "",
				Usage: "Path to the temporary directory where writes are staged prior to" +
					" upload to Cloud Storage. (default: system default, likely /tmp)",
			},

			cli.StringFlag{
				Name:  "client-protocol",
				Value: string(mountpkg.HTTP1),
				Usage: "The protocol used for communicating with the GCS backend. " +
					"Value can be 'http1' (HTTP/1.1) or 'http2' (HTTP/2).",
			},

			cli.IntFlag{
				Name:  "max-conns-per-host",
				Value: 100,
				Usage: "The max number of TCP connections allowed per server. This is " +
					"effective when --client-protocol is set to 'http1'.",
			},

			cli.IntFlag{
				Name:  "max-idle-conns-per-host",
				Value: 100,
				Usage: "The number of maximum idle connections allowed per server.",
			},

			cli.BoolFlag{
				Name: "enable-nonexistent-type-cache",
				Usage: "Once set, if an inode is not found in GCS, a type cache entry with type NonexistentType" +
					" will be created. This also means new file/dir created might not be seen. For example, if this" +
					" flag is set, and metadata-cache:ttl-secs (in config-file) or flag type-cache-ttl are set, then if we create the same file/node" +
					" in the meantime using the same mount, since we are not refreshing the cache, it will still return nil.",
			},

			/////////////////////////
			// Monitoring & Logging
			/////////////////////////

			cli.DurationFlag{
				Name:  "stackdriver-export-interval",
				Value: 0,
				Usage: "Export metrics to stackdriver with this interval. The default value 0 indicates no exporting.",
			},

			cli.StringFlag{
				Name:  "experimental-opentelemetry-collector-address",
				Value: "",
				Usage: "Experimental: Export metrics to the OpenTelemetry collector at this address.",
			},

			cli.StringFlag{
				Name:  "log-file",
				Value: "",
				Usage: "The file for storing logs that can be parsed by fluentd. When not provided," +
					" plain text logs are printed to stdout.",
			},

			cli.StringFlag{
				Name:  "log-format",
				Value: "json",
				Usage: "The format of the log file: 'text' or 'json'.",
			},

			cli.BoolFlag{
				Name: "experimental-enable-json-read",
				Usage: "By default, GCSFuse uses the GCS XML API to get and read objects. " +
					"When this flag is specified, GCSFuse uses the GCS JSON API instead.",
			},

			/////////////////////////
			// Debugging
			/////////////////////////

			cli.BoolTFlag{
				Name: "debug_fuse_errors",
				Usage: "If false, fuse errors will not be logged to the console (in case of --foreground) " +
					"or the log-file (if specified)",
			},

			cli.BoolFlag{
				Name:  "debug_fuse",
				Usage: "Enable fuse-related debugging output.",
			},

			cli.BoolFlag{
				Name:  "debug_fs",
				Usage: "This flag is currently unused.",
			},

			cli.BoolFlag{
				Name:  "debug_gcs",
				Usage: "Print GCS request and timing information.",
			},

			cli.BoolFlag{
				Name:  "debug_http",
				Usage: "This flag is currently unused.",
			},

			cli.BoolFlag{
				Name:  "debug_invariants",
				Usage: "Panic when internal invariants are violated.",
			},

			cli.BoolFlag{
				Name:  "debug_mutex",
				Usage: "Print debug messages when a mutex is held too long.",
			},
		},
	}

	return
}

type flagStorage struct {
	AppName    string
	Foreground bool
	ConfigFile string

	// File system
	MountOptions   map[string]string
	DirMode        os.FileMode
	FileMode       os.FileMode
	Uid            int64
	Gid            int64
	ImplicitDirs   bool
	OnlyDir        string
	RenameDirLimit int64

	// GCS
	CustomEndpoint                     *url.URL
	BillingProject                     string
	KeyFile                            string
	TokenUrl                           string
	ReuseTokenFromUrl                  bool
	EgressBandwidthLimitBytesPerSecond float64
	OpRateLimitHz                      float64
	SequentialReadSizeMb               int32

	// Tuning
	MaxRetrySleep              time.Duration
	StatCacheCapacity          int
	StatCacheTTL               time.Duration
	TypeCacheTTL               time.Duration
	HttpClientTimeout          time.Duration
	MaxRetryDuration           time.Duration
	RetryMultiplier            float64
	LocalFileCache             bool
	TempDir                    string
	ClientProtocol             mountpkg.ClientProtocol
	MaxConnsPerHost            int
	MaxIdleConnsPerHost        int
	EnableNonexistentTypeCache bool

	// Monitoring & Logging
	StackdriverExportInterval  time.Duration
	OtelCollectorAddress       string
	LogFile                    string
	LogFormat                  string
	ExperimentalEnableJsonRead  bool
	DebugFuseErrors             bool

	// Debugging
	DebugFuse       bool
	DebugFS         bool
	DebugGCS        bool
	DebugHTTP       bool
	DebugInvariants bool
	DebugMutex      bool
}

func resolveFilePath(filePath string, configKey string) (resolvedPath string, err error) {
	resolvedPath, err = util.GetResolvedPath(filePath)
	if filePath == resolvedPath || err != nil {
		return
	}

	logger.Infof("Value of [%s] resolved from [%s] to [%s]\n", configKey, filePath, resolvedPath)
	return resolvedPath, nil
}

// This method resolves path in the context dictionary.
func resolvePathForTheFlagInContext(flagKey string, c *cli.Context) (err error) {
	flagValue := c.String(flagKey)
	resolvedPath, err := resolveFilePath(flagValue, flagKey)
	if err != nil {
		return
	}

	err = c.Set(flagKey, resolvedPath)
	return
}

// For parent process: it only resolves the path with respect to home folder.
// For child process: it resolves the path relative to both home directory and
// GCSFUSE_PARENT_PROCESS_DIR. Child process is spawned when --foreground flag
// is disabled.
func resolvePathForTheFlagsInContext(c *cli.Context) (err error) {
	err = resolvePathForTheFlagInContext("log-file", c)
	if err != nil {
		return fmt.Errorf("resolving for log-file: %w", err)
	}

	err = resolvePathForTheFlagInContext("key-file", c)
	if err != nil {
		return fmt.Errorf("resolving for key-file: %w", err)
	}

	err = resolvePathForTheFlagInContext("config-file", c)
	if err != nil {
		return fmt.Errorf("resolving for config-file: %w", err)
	}

	return
}

// resolveConfigFilePaths resolves the config file paths specified in the config file.
func resolveConfigFilePaths(mountConfig *config.MountConfig) (err error) {
	mountConfig.LogConfig.FilePath, err = resolveFilePath(mountConfig.LogConfig.FilePath, "logging: file")
	if err != nil {
		return
	}

	// Resolve cache-dir path
	resolvedPath, err := resolveFilePath(string(mountConfig.CacheDir), "cache-dir")
	mountConfig.CacheDir = config.CacheDir(resolvedPath)
	if err != nil {
		return
	}

	return
}

// Add the flags accepted by run to the supplied flag set, returning the
// variables into which the flags will parse.
func populateFlags(c *cli.Context) (flags *flagStorage, err error) {
	customEndpointStr := c.String("custom-endpoint")
	var customEndpoint *url.URL

	if customEndpointStr == "" {
		customEndpoint = nil
	} else {
		customEndpoint, err = url.Parse(customEndpointStr)
		if err != nil {
			err = fmt.Errorf("could not parse custom-endpoint: %w", err)
			return
		}
	}

	clientProtocolString := strings.ToLower(c.String("client-protocol"))
	clientProtocol := mountpkg.ClientProtocol(clientProtocolString)
	flags = &flagStorage{
		AppName:    c.String("app-name"),
		Foreground: c.Bool("foreground"),
		ConfigFile: c.String("config-file"),

		// File system
		MountOptions:   make(map[string]string),
		DirMode:        os.FileMode(*c.Generic("dir-mode").(*OctalInt)),
		FileMode:       os.FileMode(*c.Generic("file-mode").(*OctalInt)),
		Uid:            int64(c.Int("uid")),
		Gid:            int64(c.Int("gid")),
		ImplicitDirs:   c.Bool("implicit-dirs"),
		OnlyDir:        c.String("only-dir"),
		RenameDirLimit: int64(c.Int("rename-dir-limit")),

		// GCS,
		CustomEndpoint:                     customEndpoint,
		BillingProject:                     c.String("billing-project"),
		KeyFile:                            c.String("key-file"),
		TokenUrl:                           c.String("token-url"),
		ReuseTokenFromUrl:                  c.BoolT("reuse-token-from-url"),
		EgressBandwidthLimitBytesPerSecond: c.Float64("limit-bytes-per-sec"),
		OpRateLimitHz:                      c.Float64("limit-ops-per-sec"),
		SequentialReadSizeMb:               int32(c.Int("sequential-read-size-mb")),

		// Tuning,
		MaxRetrySleep:     c.Duration("max-retry-sleep"),
		StatCacheCapacity: c.Int("stat-cache-capacity"),
		StatCacheTTL:      c.Duration("stat-cache-ttl"),
		TypeCacheTTL:      c.Duration("type-cache-ttl"),
		HttpClientTimeout: c.Duration("http-client-timeout"),
		MaxRetryDuration:  c.Duration("max-retry-duration"),
		RetryMultiplier:   c.Float64("retry-multiplier"),
		// This flag is deprecated and we have plans to remove the implementation related to this flag in next release.
		LocalFileCache:             false,
		TempDir:                    c.String("temp-dir"),
		ClientProtocol:             clientProtocol,
		MaxConnsPerHost:            c.Int("max-conns-per-host"),
		MaxIdleConnsPerHost:        c.Int("max-idle-conns-per-host"),
		EnableNonexistentTypeCache: c.Bool("enable-nonexistent-type-cache"),

		// Monitoring & Logging
		StackdriverExportInterval:  c.Duration("stackdriver-export-interval"),
		OtelCollectorAddress:       c.String("experimental-opentelemetry-collector-address"),
		LogFile:                    c.String("log-file"),
		LogFormat:                  c.String("log-format"),
		ExperimentalEnableJsonRead: c.Bool("experimental-enable-json-read"),

		// Debugging,
		DebugFuseErrors: c.BoolT("debug_fuse_errors"),
		DebugFuse:       c.Bool("debug_fuse"),
		DebugGCS:        c.Bool("debug_gcs"),
		DebugHTTP:       c.Bool("debug_http"),
		DebugFS:         c.Bool("debug_fs"),
		DebugInvariants: c.Bool("debug_invariants"),
		DebugMutex:      c.Bool("debug_mutex"),
	}

	// Handle the repeated "-o" flag.
	for _, o := range c.StringSlice("o") {
		mountpkg.ParseOptions(flags.MountOptions, o)
	}

	err = validateFlags(flags)

	return
}

func validateFlags(flags *flagStorage) (err error) {
	if flags.SequentialReadSizeMb < 1 || flags.SequentialReadSizeMb > maxSequentialReadSizeMb {
		err = fmt.Errorf("SequentialReadSizeMb should be less than %d", maxSequentialReadSizeMb)
		return
	}

	if !flags.ClientProtocol.IsValid() {
		err = fmt.Errorf("client protocol: %s is not valid", flags.ClientProtocol)
	}

	return
}

// A cli.Generic that can be used with cli.GenericFlag to obtain an int flag
// that is parsed in octal.
type OctalInt int

var _ cli.Generic = (*OctalInt)(nil)

func (oi *OctalInt) Set(value string) (err error) {
	tmp, err := strconv.ParseInt(value, 8, 32)
	if err != nil {
		err = fmt.Errorf("Parsing as octal: %w", err)
		return
	}

	*oi = OctalInt(tmp)
	return
}

func (oi OctalInt) String() string {
	return fmt.Sprintf("%o", oi)
}
