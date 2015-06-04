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

	"github.com/googlecloudplatform/gcsfuse/mount"
)

type flagStorage struct {
	Help                               bool
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

	flag.BoolVar(
		&flags.Help,
		"help",
		false,
		"If set, print usage and exit successfully.")

	flag.Var(
		mount.OptionValue(flags.MountOptions),
		"o",
		"Additional system-specific mount options. Be careful!")

	flag.Int64Var(
		&flags.Uid,
		"uid",
		-1,
		"If non-negative, the UID that owns all inodes. The default is the UID of "+
			"the gcsfuse process.")

	flag.Int64Var(
		&flags.Gid,
		"gid",
		-1,
		"If non-negative, the GID that owns all inodes. The default is the GID of "+
			"the gcsfuse process.")

	flag.UintVar(
		&flags.FileMode,
		"file_mode",
		0644,
		"Permissions bits for files. Default is 0644.")

	flag.UintVar(
		&flags.DirMode,
		"dir_mode",
		0755,
		"Permissions bits for directories. Default is 0755.")

	flag.StringVar(
		&flags.TempDir,
		"temp_dir", "",
		"The temporary directory in which to store local copies of GCS objects. "+
			"If empty, the system default (probably /tmp) will be used.")

	flag.Int64Var(
		&flags.TempDirLimit,
		"temp_dir_bytes", 1<<31,
		"A desired limit on the number of bytes used in --temp_dir. May be "+
			"exceeded for dirty files that have not been flushed or closed.")

	flag.Uint64Var(
		&flags.GCSChunkSize,
		"gcs_chunk_size", 1<<24,
		"If set to a non-zero value N, split up GCS objects into multiple "+
			"chunks of size at most N when reading, and do not read or cache "+
			"unnecessary chunks.")

	flag.BoolVar(
		&flags.ImplicitDirs,
		"implicit_dirs",
		false,
		"Implicitly define directories based on their content. See "+
			"docs/semantics.md.")

	flag.DurationVar(
		&flags.StatCacheTTL,
		"stat_cache_ttl",
		time.Minute,
		"How long to cache StatObject results from GCS.")

	flag.DurationVar(
		&flags.TypeCacheTTL,
		"type_cache_ttl",
		time.Minute,
		"How long to cache name -> file/dir type mappings in directory inodes.")

	flag.Float64Var(
		&flags.OpRateLimitHz,
		"op_rate_limit_hz",
		-1,
		"If positive, a limit on the rate at which we send requests to GCS, "+
			"measured over a 30-second window.")

	flag.Float64Var(
		&flags.EgressBandwidthLimitBytesPerSecond,
		"egress_bandwidth_limit_bytes_per_second",
		-1,
		"If positive, a limit on the GCS -> gcsfuse bandwidth for reading "+
			"objects, measured over a 30-second window.")

	return
}
