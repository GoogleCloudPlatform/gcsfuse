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

	"github.com/codegangsta/cli"
	mountpkg "github.com/googlecloudplatform/gcsfuse/internal/mount"
)

// Set up custom help text for gcsfuse; in particular the usage section.
func init() {
	cli.AppHelpTemplate = `NAME:
   {{.Name}} - {{.Usage}}

USAGE:
   {{.Name}} {{if .Flags}}[global options]{{end}} bucket mountpoint
   {{if .Version}}
VERSION:
   {{.Version}}
   {{end}}{{if len .Authors}}
AUTHOR(S):
   {{range .Authors}}{{ . }}{{end}}
   {{end}}{{if .Commands}}
COMMANDS:
   {{range .Commands}}{{join .Names ", "}}{{ "\t" }}{{.Usage}}
   {{end}}{{end}}{{if .Flags}}
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
		Name:     "gcsfuse",
		Version:  getVersion(),
		Usage:    "Mount a GCS bucket locally",
		HideHelp: true,
		Writer:   os.Stderr,
		Flags: []cli.Flag{

			cli.BoolFlag{
				Name:  "help, h",
				Usage: "Print this help text and exit successfuly.",
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
				Usage: "Implicitly define directories based on content. See" +
					"docs/semantics.md",
			},

			cli.StringFlag{
				Name:  "only-dir",
				Usage: "Mount only the given directory, relative to the bucket root.",
			},

			/////////////////////////
			// GCS
			/////////////////////////

			cli.StringFlag{
				Name:  "key-file",
				Value: "",
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
				Usage: "How long to cache StatObject results and inode attributes.",
			},

			cli.DurationFlag{
				Name:  "type-cache-ttl",
				Value: time.Minute,
				Usage: "How long to cache name -> file/dir mappings in directory " +
					"inodes.",
			},

			cli.StringFlag{
				Name:  "temp-dir",
				Value: "",
				Usage: "Temporary directory for local GCS object copies. " +
					"(default: system default, likely /tmp)",
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
	OnlyDir      string

	// GCS
	KeyFile                            string
	EgressBandwidthLimitBytesPerSecond float64
	OpRateLimitHz                      float64

	// Tuning
	StatCacheTTL time.Duration
	TypeCacheTTL time.Duration
	TempDir      string

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
		DirMode:      os.FileMode(*c.Generic("dir-mode").(*OctalInt)),
		FileMode:     os.FileMode(*c.Generic("file-mode").(*OctalInt)),
		Uid:          int64(c.Int("uid")),
		Gid:          int64(c.Int("gid")),
		ImplicitDirs: c.Bool("implicit-dirs"),
		OnlyDir:      c.String("only-dir"),

		// GCS,
		KeyFile: c.String("key-file"),
		EgressBandwidthLimitBytesPerSecond: c.Float64("limit-bytes-per-sec"),
		OpRateLimitHz:                      c.Float64("limit-ops-per-sec"),

		// Tuning,
		StatCacheTTL: c.Duration("stat-cache-ttl"),
		TypeCacheTTL: c.Duration("type-cache-ttl"),
		TempDir:      c.String("temp-dir"),

		// Debugging,
		DebugFuse:       c.Bool("debug_fuse"),
		DebugGCS:        c.Bool("debug_gcs"),
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
		err = fmt.Errorf("Parsing as octal: %v", err)
		return
	}

	*oi = OctalInt(tmp)
	return
}

func (oi OctalInt) String() string {
	return fmt.Sprintf("%o", oi)
}
