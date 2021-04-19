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
	"os"
	"strconv"
	"time"

	mountpkg "github.com/googlecloudplatform/gcsfuse/internal/mount"
	"github.com/urfave/cli"
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

			cli.BoolFlag{
				Name:  "foreground",
				Usage: "Stay in the foreground after mounting.",
			},

			/////////////////////////
			// File system
			/////////////////////////

			cli.StringSliceFlag{
				Name:  "o",
				Usage: "Additional system-specific mount options. Be careful!",
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
				Name: "implicit-dirs",
				Usage: "Implicitly define directories based on content. See " +
					"docs/semantics.md",
			},

			cli.StringFlag{
				Name:  "only-dir",
				Usage: "Mount only the given directory, relative to the bucket root.",
			},

			cli.IntFlag{
				Name:  "rename-dir-limit",
				Value: 0,
				Usage: "Allow renaming a directory containing fewer files than " +
					"this limit in a non-atomic way.",
			},

			/////////////////////////
			// GCS
			/////////////////////////

			cli.StringFlag{
				Name:  "billing-project",
				Value: "",
				Usage: "Project to use for billing when accessing requester pays buckets. " +
					"(default: none)",
			},

			cli.StringFlag{
				Name:  "key-file",
				Value: "",
				Usage: "Absolute path to JSON key file for use with GCS. " +
					"(default: none, Google application default credentials used)",
			},

			cli.StringFlag{
				Name:  "token-url",
				Value: "",
				Usage: "An url for getting an access token when key-file is absent.",
			},

			cli.Float64Flag{
				Name:  "limit-bytes-per-sec",
				Value: -1,
				Usage: "Bandwidth limit for reading data, measured over a 30-second " +
					"window. (use -1 for no limit)",
			},

			cli.Float64Flag{
				Name:  "limit-ops-per-sec",
				Value: 5.0,
				Usage: "Operations per second limit, measured over a 30-second window " +
					"(use -1 for no limit)",
			},

			/////////////////////////
			// Tuning
			/////////////////////////

			cli.DurationFlag{
				Name:  "max-retry-sleep",
				Value: time.Minute,
				Usage: "The maximum duration allowed to sleep in a retry loop with " +
					"exponential backoff for failed requests to GCS backend. Once the " +
					"backoff duration exceeds this limit, the retry stops. The default " +
					"is 1 minute. A value of 0 disables retries.",
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
				Usage: "How long to cache name -> file/dir mappings in directory " +
					"inodes.",
			},

			cli.BoolFlag{
				Name:  "local-file-cache",
				Usage: "Cache GCS files on local disk for reads.",
			},

			cli.StringFlag{
				Name:  "temp-dir",
				Value: "",
				Usage: "Absolute path to temporary directory for local GCS object " +
					"copies. (default: system default, likely /tmp)",
			},

			cli.BoolFlag{
				Name: "disable-http2",
				Usage: "Once set, the protocol used for communicating with " +
					"GCS backend would be HTTP/1.1, instead of the default HTTP/2.",
			},

			cli.IntFlag{
				Name:  "max-conns-per-host",
				Value: 10,
				Usage: "The max number of TCP connections allowed per server. " +
					"This is effective when --disable-http2 is set.",
			},

			/////////////////////////
			// Monitoring & Logging
			/////////////////////////

			cli.IntFlag{
				Name:  "monitoring-port",
				Value: 0,
				Usage: "The port used to export prometheus metrics for monitoring. " +
					"The default value 0 indicates no monitoring metrics.",
			},

			cli.StringFlag{
				Name:  "log-file",
				Value: "",
				Usage: "The file for storing logs that can be parsed by " +
					"fluentd. When not provided, plain text logs are printed to " +
					"stdout.",
			},

			cli.StringFlag{
				Name:  "log-format",
				Value: "json",
				Usage: "The format of the log file: 'text' or 'json'.",
			},

			/////////////////////////
			// Debugging
			/////////////////////////

			cli.BoolFlag{
				Name:  "debug_fuse",
				Usage: "Enable fuse-related debugging output.",
			},

			cli.BoolFlag{
				Name:  "debug_fs",
				Usage: "Enable file system debugging output.",
			},

			cli.BoolFlag{
				Name:  "debug_gcs",
				Usage: "Print GCS request and timing information.",
			},

			cli.BoolFlag{
				Name:  "debug_http",
				Usage: "Dump HTTP requests and responses to/from GCS.",
			},

			cli.BoolFlag{
				Name:  "debug_invariants",
				Usage: "Panic when internal invariants are violated.",
			},
		},
	}

	return
}

type flagStorage struct {
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
	BillingProject                     string
	KeyFile                            string
	TokenUrl                           string
	EgressBandwidthLimitBytesPerSecond float64
	OpRateLimitHz                      float64

	// Tuning
	MaxRetrySleep     time.Duration
	StatCacheCapacity int
	StatCacheTTL      time.Duration
	TypeCacheTTL      time.Duration
	LocalFileCache    bool
	TempDir           string
	DisableHTTP2      bool
	MaxConnsPerHost   int

	// Monitoring
	MonitoringPort int
	LogFile        string
	LogFormat      string

	// Debugging
	DebugFuse       bool
	DebugFS         bool
	DebugGCS        bool
	DebugHTTP       bool
	DebugInvariants bool
}

// Add the flags accepted by run to the supplied flag set, returning the
// variables into which the flags will parse.
func populateFlags(c *cli.Context) (flags *flagStorage) {
	flags = &flagStorage{
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
		BillingProject:                     c.String("billing-project"),
		KeyFile:                            c.String("key-file"),
		TokenUrl:                           c.String("token-url"),
		EgressBandwidthLimitBytesPerSecond: c.Float64("limit-bytes-per-sec"),
		OpRateLimitHz:                      c.Float64("limit-ops-per-sec"),

		// Tuning,
		MaxRetrySleep:     c.Duration("max-retry-sleep"),
		StatCacheCapacity: c.Int("stat-cache-capacity"),
		StatCacheTTL:      c.Duration("stat-cache-ttl"),
		TypeCacheTTL:      c.Duration("type-cache-ttl"),
		LocalFileCache:    c.Bool("local-file-cache"),
		TempDir:           c.String("temp-dir"),
		DisableHTTP2:      c.Bool("disable-http2"),
		MaxConnsPerHost:   c.Int("max-conns-per-host"),

		// Monitoring
		MonitoringPort: c.Int("monitoring-port"),
		LogFile:        c.String("log-file"),
		LogFormat:      c.String("log-format"),

		// Debugging,
		DebugFuse:       c.Bool("debug_fuse"),
		DebugGCS:        c.Bool("debug_gcs"),
		DebugFS:         c.Bool("debug_fs"),
		DebugHTTP:       c.Bool("debug_http"),
		DebugInvariants: c.Bool("debug_invariants"),
	}

	// Handle the repeated "-o" flag.
	for _, o := range c.StringSlice("o") {
		mountpkg.ParseOptions(flags.MountOptions, o)
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
