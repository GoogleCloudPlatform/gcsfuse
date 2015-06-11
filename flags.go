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
	"flag"
	"time"

	mountpkg "github.com/googlecloudplatform/gcsfuse/mount"
)

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
}

// Add the flags accepted by run to the supplied flag set, returning the
// variables into which the flags will parse.
func populateFlagSet(fs *flag.FlagSet) (flags *flagStorage) {
	flags = new(flagStorage)

	flags.MountOptions = make(map[string]string)
	fs.Var(
		mountpkg.OptionValue(flags.MountOptions),
		"o",
		"Additional system-specific mount options. Be careful!")

	fs.Int64Var(
		&flags.Uid,
		"uid",
		-1,
		"If non-negative, the UID that owns all inodes. The default is the UID of "+
			"the gcsfuse process.")

	fs.Int64Var(
		&flags.Gid,
		"gid",
		-1,
		"If non-negative, the GID that owns all inodes. The default is the GID of "+
			"the gcsfuse process.")

	fs.UintVar(
		&flags.FileMode,
		"file-mode",
		0644,
		"Permissions bits for files. Default is 0644.")

	fs.UintVar(
		&flags.DirMode,
		"dir-mode",
		0755,
		"Permissions bits for directories. Default is 0755.")

	fs.StringVar(
		&flags.TempDir,
		"temp-dir", "",
		"The temporary directory in which to store local copies of GCS objects. "+
			"If empty, the system default (probably /tmp) will be used.")

	fs.Int64Var(
		&flags.TempDirLimit,
		"temp-dir-bytes", 1<<31,
		"A desired limit on the number of bytes used in --temp-dir. May be "+
			"exceeded for dirty files that have not been flushed or closed.")

	fs.Uint64Var(
		&flags.GCSChunkSize,
		"gcs-chunk-size", 1<<24,
		"If set to a non-zero value N, split up GCS objects into multiple "+
			"chunks of size at most N when reading, and do not read or cache "+
			"unnecessary chunks.")

	fs.BoolVar(
		&flags.ImplicitDirs,
		"implicit-dirs",
		false,
		"Implicitly define directories based on their content. See "+
			"docs/semantics.md.")

	fs.DurationVar(
		&flags.StatCacheTTL,
		"stat-cache-ttl",
		time.Minute,
		"How long to cache StatObject results from GCS.")

	fs.DurationVar(
		&flags.TypeCacheTTL,
		"type-cache-ttl",
		time.Minute,
		"How long to cache name -> file/dir type mappings in directory inodes.")

	fs.Float64Var(
		&flags.OpRateLimitHz,
		"limit-ops-per-sec",
		5.0,
		"If positive, a limit on the rate at which we send requests to GCS, "+
			"measured over a 30-second window.")

	fs.Float64Var(
		&flags.EgressBandwidthLimitBytesPerSecond,
		"limit-bytes-per-sec",
		-1,
		"If positive, a limit on the GCS -> gcsfuse bandwidth for reading "+
			"objects, measured over a 30-second window.")

	return
}
