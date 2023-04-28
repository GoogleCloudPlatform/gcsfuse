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
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	mountpkg "github.com/googlecloudplatform/gcsfuse/internal/mount"
	"github.com/urfave/cli"
)

// Defines the max value supported by sequential-read-size-mb flag.
const maxSequentialReadSizeMb = 1024

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
				Name:  "endpoint",
				Value: "https://storage.googleapis.com:443",
				Usage: "The endpoint to connect to.",
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
				Value: time.Minute,
				Usage: "The maximum duration allowed to sleep in a retry loop with exponential backoff " +
					"for failed requests to GCS backend. Once the backoff duration exceeds this limit, the retry stops." +
					" The default is 1 minute. A value of 0 disables retries.",
			},

			cli.IntFlag{
				Name:  "stat-cache-capacity",
				Value: 4096,
				Usage: "How many entries can the stat cache hold (impacts memory consumption)",
			},

			cli.DurationFlag{
				Name:  "stat-cache-ttl",
				Value: time.Minute,
				Usage: "How long to cache StatObject results and inode attributes.",
			},

			cli.DurationFlag{
				Name:  "type-cache-ttl",
				Value: time.Minute,
				Usage: "How long to cache name -> file/dir mappings in directory inodes.",
			},

			cli.DurationFlag{
				Name:  "http-client-timeout",
				Usage: "The time duration that http client will wait to get response from the server. The default value 0 indicates no timeout. ",
			},

			cli.DurationFlag{
				Name:  "max-retry-duration",
				Value: 30 * time.Second,
				Usage: "The operation will be retried till the value of max-retry-duration.",
			},

			cli.Float64Flag{
				Name:  "retry-multiplier",
				Value: 2,
				Usage: "Param for exponential backoff algorithm, which is used to increase waiting time b/w two consecutive retries.",
			},

			cli.BoolFlag{
				Name:  "experimental-local-file-cache",
				Usage: "Experimental: Cache GCS files on local disk for reads.",
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
					" flag is set, and flag type-cache-ttl is set to 10 minutes, then if we create the same file/node" +
					" in the meantime using the same mount, since we are not refreshing the cache, it will still return nil.",
			},

			/////////////////////////
			// Monitoring & Logging
			/////////////////////////

			cli.DurationFlag{
				Name:  "stackdriver-export-interval",
				Value: 0,
				Usage: "Experimental: Export metrics to stackdriver with this interval. The default value 0 indicates no exporting.",
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
				Name: "debug_http",
				Usage: "Dump HTTP requests and responses to/from GCS, " +
					"doesn't work when enable-storage-client-library flag is true.",
			},

			cli.BoolFlag{
				Name:  "debug_invariants",
				Usage: "Panic when internal invariants are violated.",
			},

			cli.BoolFlag{
				Name:  "debug_mutex",
				Usage: "Print debug messages when a mutex is held too long.",
			},

			/////////////////////////
			// Client
			/////////////////////////

			cli.BoolTFlag{
				Name:  "enable-storage-client-library",
				Usage: "If true, will use go storage client library otherwise jacobsa/gcloud",
			},
		},
	}

	return
}

type flagStorage struct {
	AppName    string
	Foreground bool

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
	Endpoint                           *url.URL
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
	StackdriverExportInterval time.Duration
	OtelCollectorAddress      string
	LogFile                   string
	LogFormat                 string
	DebugFuseErrors           bool

	// Debugging
	DebugFuse       bool
	DebugFS         bool
	DebugGCS        bool
	DebugHTTP       bool
	DebugInvariants bool
	DebugMutex      bool

	// client
	EnableStorageClientLibrary bool
}

const GCSFUSE_PARENT_PROCESS_DIR = "gcsfuse-parent-process-dir"

// 1. Returns the same filepath in case of absolute path or empty filename.
// 2. For child process, it resolves relative path like, ./test.txt, test.txt
// ../test.txt etc, with respect to GCSFUSE_PARENT_PROCESS_DIR
// because we execute the child process from different directory and input
// files are provided with respect to GCSFUSE_PARENT_PROCESS_DIR.
// 3. For relative path starting with ~, it resolves with respect to home dir.
func getResolvedPath(filePath string) (resolvedPath string, err error) {
	if filePath == "" || path.IsAbs(filePath) {
		resolvedPath = filePath
		return
	}

	// Relative path starting with tilda (~)
	if strings.HasPrefix(filePath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("fetch home dir: %w", err)
		}
		return filepath.Join(homeDir, filePath[2:]), err
	}

	// We reach here, when relative path starts with . or .. or other than (/ or ~)
	gcsfuseParentProcessDir, _ := os.LookupEnv(GCSFUSE_PARENT_PROCESS_DIR)
	gcsfuseParentProcessDir = strings.TrimSpace(gcsfuseParentProcessDir)
	if gcsfuseParentProcessDir == "" {
		return filepath.Abs(filePath)
	} else {
		return filepath.Join(gcsfuseParentProcessDir, filePath), err
	}
}

// This method resolves path in the context dictionary.
func resolvePathForTheFlagInContext(flagKey string, c *cli.Context) (err error) {
	flagValue := c.String(flagKey)
	resolvedPath, err := getResolvedPath(flagValue)
	if flagValue == resolvedPath || err != nil {
		return
	}

	logger.Infof("Value of [%s] resolved from [%s] to [%s]\n",
		flagKey, flagValue, resolvedPath)
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

	return
}

// Add the flags accepted by run to the supplied flag set, returning the
// variables into which the flags will parse.
func populateFlags(c *cli.Context) (flags *flagStorage, err error) {
	endpoint, err := url.Parse(c.String("endpoint"))
	if err != nil {
		fmt.Printf("Could not parse endpoint")
		return
	}
	clientProtocolString := strings.ToLower(c.String("client-protocol"))
	clientProtocol := mountpkg.ClientProtocol(clientProtocolString)
	flags = &flagStorage{
		AppName:    c.String("app-name"),
		Foreground: c.Bool("foreground"),

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
		Endpoint:                           endpoint,
		BillingProject:                     c.String("billing-project"),
		KeyFile:                            c.String("key-file"),
		TokenUrl:                           c.String("token-url"),
		ReuseTokenFromUrl:                  c.BoolT("reuse-token-from-url"),
		EgressBandwidthLimitBytesPerSecond: c.Float64("limit-bytes-per-sec"),
		OpRateLimitHz:                      c.Float64("limit-ops-per-sec"),
		SequentialReadSizeMb:               int32(c.Int("sequential-read-size-mb")),

		// Tuning,
		MaxRetrySleep:              c.Duration("max-retry-sleep"),
		StatCacheCapacity:          c.Int("stat-cache-capacity"),
		StatCacheTTL:               c.Duration("stat-cache-ttl"),
		TypeCacheTTL:               c.Duration("type-cache-ttl"),
		HttpClientTimeout:          c.Duration("http-client-timeout"),
		MaxRetryDuration:           c.Duration("max-retry-duration"),
		RetryMultiplier:            c.Float64("retry-multiplier"),
		LocalFileCache:             c.Bool("experimental-local-file-cache"),
		TempDir:                    c.String("temp-dir"),
		ClientProtocol:             clientProtocol,
		MaxConnsPerHost:            c.Int("max-conns-per-host"),
		MaxIdleConnsPerHost:        c.Int("max-idle-conns-per-host"),
		EnableNonexistentTypeCache: c.Bool("enable-nonexistent-type-cache"),

		// Monitoring & Logging
		StackdriverExportInterval: c.Duration("stackdriver-export-interval"),
		OtelCollectorAddress:      c.String("experimental-opentelemetry-collector-address"),
		LogFile:                   c.String("log-file"),
		LogFormat:                 c.String("log-format"),

		// Debugging,
		DebugFuseErrors: c.BoolT("debug_fuse_errors"),
		DebugFuse:       c.Bool("debug_fuse"),
		DebugGCS:        c.Bool("debug_gcs"),
		DebugFS:         c.Bool("debug_fs"),
		DebugHTTP:       c.Bool("debug_http"),
		DebugInvariants: c.Bool("debug_invariants"),
		DebugMutex:      c.Bool("debug_mutex"),

		// Client,
		EnableStorageClientLibrary: c.Bool("enable-storage-client-library"),
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
