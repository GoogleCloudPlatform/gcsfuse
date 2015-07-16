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
	"time"

	mountpkg "github.com/googlecloudplatform/gcsfuse/mount"
	"github.com/jgeewax/cli"
)

func getApp() (app *cli.App) {
	app = cli.NewApp()
	app.Name = "gcsfuse"
	app.Usage = "Mount a GCS bucket locally"
	app.ArgumentUsage = "bucket mountpoint"
	app.HideHelp = true
	app.Version = "0.1.0"
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:        "dir-mode, d",
			Usage:       "Permissions bits for directories. (default: 0755)",
			HideDefault: true,
		},
		cli.IntFlag{
			Name:        "file-mode, f",
			Value:       0644,
			Usage:       "Permission bits for files (default: 0644)",
			HideDefault: true,
		},
		cli.IntFlag{
			Name:  "gcs-chunk-size",
			Value: 1 << 24,
			Usage: "Max chunk size for loading GCS objects.",
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
		cli.Float64Flag{
			Name:  "limit-bytes-per-sec",
			Value: -1,
			Usage: "Bandwidth limit for reading data, measured over a 30-second " +
				"window. (use -1 for no limit)",
		},
		cli.StringFlag{
			Name:        "key-file",
			Value:       "",
			HideDefault: true,
			Usage: "Path to JSON key file for use with GCS. " +
				"(default: none, Google application default credentials used)",
		},
		cli.Float64Flag{
			Name:  "limit-ops-per-sec",
			Value: 5.0,
			Usage: "Operations per second limit, measured over a 30-second window " +
				"(use -1 for no limit)",
		},
		cli.StringFlag{
			Name:        "mount-options, o",
			HideDefault: true,
			Usage:       "Additional system-specific mount options. Be careful!",
		},
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
		cli.StringFlag{
			Name:        "temp-dir, t",
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
		cli.IntFlag{
			Name:        "uid",
			Value:       -1,
			HideDefault: true,
			Usage:       "UID owner of all inodes.",
		},
		cli.BoolFlag{
			Name:  "help, h",
			Usage: "print this help text",
		},
	}
	return app
}

type flagStorage struct {
	MountOptions                       map[string]string
	Uid                                int64
	Gid                                int64
	FileMode                           uint
	DirMode                            uint
	TempDir                            string
	TempDirLimit                       int64
	GCSChunkSize                       uint64
	ImplicitDirs                       bool
	StatCacheTTL                       time.Duration
	TypeCacheTTL                       time.Duration
	OpRateLimitHz                      float64
	EgressBandwidthLimitBytesPerSecond float64
	KeyFile                            string
}

// Add the flags accepted by run to the supplied flag set, returning the
// variables into which the flags will parse.
func populateFlags(c *cli.Context) (flags *flagStorage) {
	flags = new(flagStorage)
	flags.MountOptions = make(map[string]string)
	mountpkg.OptionValue(flags.MountOptions).Set(c.String("mount-options"))
	flags.Uid = int64(c.Int("uid"))
	flags.Gid = int64(c.Int("gid"))
	flags.FileMode = uint(c.Int("file-mode"))
	flags.DirMode = uint(c.Int("dir-mode"))
	flags.TempDir = c.String("temp-dir")
	flags.TempDirLimit = int64(c.Int("temp-dir-bytes"))
	flags.GCSChunkSize = uint64(c.Int("gcs-chunk-size"))
	flags.ImplicitDirs = c.Bool("implicit-dirs")
	flags.StatCacheTTL = c.Duration("stat-cache-ttl")
	flags.TypeCacheTTL = c.Duration("type-cache-ttl")
	flags.OpRateLimitHz = c.Float64("limit-ops-per-sec")
	flags.EgressBandwidthLimitBytesPerSecond = c.Float64("limit-bytes-per-sec")
	flags.KeyFile = c.String("key-file")
	return
}
