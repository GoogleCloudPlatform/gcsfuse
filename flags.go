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
	"os"
	"time"

	mountpkg "github.com/googlecloudplatform/gcsfuse/mount"
	"github.com/jgeewax/cli"
)

func newApp() (app *cli.App) {
	app = &cli.App{
		Name:          "gcsfuse",
		Usage:         "Mount a GCS bucket locally",
		ArgumentUsage: "bucket mountpoint",
		HideHelp:      true,
		HideVersion:   true,
		Writer:        os.Stderr,
		Flags: []cli.Flag{

			cli.BoolFlag{
				Name:  "help, h",
				Usage: "Print this help text and exit successfuly.",
			},

			/////////////////////////
			// File system
			/////////////////////////

			cli.StringFlag{
				Name:        "o",
				HideDefault: true,
				Usage:       "Additional system-specific mount options. Be careful!",
			},

			cli.IntFlag{
				Name:        "dir-mode",
				Value:       0755,
				Usage:       "Permissions bits for directories. (default: 0755)",
				HideDefault: true,
			},

			cli.IntFlag{
				Name:        "file-mode",
				Value:       0644,
				Usage:       "Permission bits for files (default: 0644)",
				HideDefault: true,
			},

			cli.IntFlag{
				Name:        "uid",
				Value:       -1,
				HideDefault: true,
				Usage:       "UID owner of all inodes.",
			},

			cli.IntFlag{
				Name:        "gid",
				Value:       -1,
				HideDefault: true,
				Usage:       "GID owner of all inodes.",
			},

			cli.BoolFlag{
				Name: "implicit-dirs",
				Usage: "Implicitly define directories based on content. See" +
					"docs/semantics.md",
			},

			/////////////////////////
			// GCS
			/////////////////////////

			cli.StringFlag{
				Name:        "key-file",
				Value:       "",
				HideDefault: true,
				Usage: "Path to JSON key file for use with GCS. " +
					"(default: none, Google application default credentials used)",
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
				Name:  "stat-cache-ttl",
				Value: time.Minute,
				Usage: "How long to cache StatObject results from GCS.",
			},

			cli.DurationFlag{
				Name:  "type-cache-ttl",
				Value: time.Minute,
				Usage: "How long to cache name -> file/dir mappings in directory " +
					"inodes.",
			},

			cli.IntFlag{
				Name:  "gcs-chunk-size",
				Value: 1 << 24,
				Usage: "Max chunk size for loading GCS objects.",
			},

			cli.StringFlag{
				Name:        "temp-dir",
				Value:       "",
				HideDefault: true,
				Usage: "Temporary directory for local GCS object copies. " +
					"(default: system default, likely /tmp)",
			},

			cli.IntFlag{
				Name:  "temp-dir-bytes",
				Value: 1 << 31,
				Usage: "Size limit of the temporary directory.",
			},

			/////////////////////////
			// Debugging
			/////////////////////////

			cli.BoolFlag{
				Name:  "debug_fuse",
				Usage: "Enable fuse-related debugging output.",
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
	// File system
	MountOptions map[string]string
	DirMode      os.FileMode
	FileMode     os.FileMode
	Uid          int64
	Gid          int64
	ImplicitDirs bool

	// GCS
	KeyFile                            string
	EgressBandwidthLimitBytesPerSecond float64
	OpRateLimitHz                      float64

	// Tuning
	StatCacheTTL time.Duration
	TypeCacheTTL time.Duration
	GCSChunkSize uint64
	TempDir      string
	TempDirLimit int64

	// Debugging
	DebugFuse       bool
	DebugGCS        bool
	DebugHTTP       bool
	DebugInvariants bool
}

// Add the flags accepted by run to the supplied flag set, returning the
// variables into which the flags will parse.
func populateFlags(c *cli.Context) (flags *flagStorage) {
	flags = &flagStorage{
		// File system
		MountOptions: make(map[string]string),
		DirMode:      os.FileMode(c.Int("dir-mode")),
		FileMode:     os.FileMode(c.Int("file-mode")),
		Uid:          int64(c.Int("uid")),
		Gid:          int64(c.Int("gid")),

		// GCS,
		KeyFile: c.String("key-file"),
		EgressBandwidthLimitBytesPerSecond: c.Float64("limit-bytes-per-sec"),
		OpRateLimitHz:                      c.Float64("limit-ops-per-sec"),

		// Tuning,
		StatCacheTTL: c.Duration("stat-cache-ttl"),
		TypeCacheTTL: c.Duration("type-cache-ttl"),
		GCSChunkSize: uint64(c.Int("gcs-chunk-size")),
		TempDir:      c.String("temp-dir"),
		TempDirLimit: int64(c.Int("temp-dir-bytes")),
		ImplicitDirs: c.Bool("implicit-dirs"),

		// Debugging,
		DebugFuse:       c.Bool("debug_fuse"),
		DebugGCS:        c.Bool("debug_gcs"),
		DebugHTTP:       c.Bool("debug_http"),
		DebugInvariants: c.Bool("debug_invariants"),
	}

	mountpkg.OptionValue(flags.MountOptions).Set(c.String("mount-options"))

	return
}
