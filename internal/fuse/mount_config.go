// Copyright 2023 Google LLC
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

package fuse

import (
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/mount"
)

type MountConfig struct {
	// The name of the file system, for use in statistics and logging.
	FSName string

	// The subtype of the file system, for use in "mount -t" on Linux.
	Subtype string

	// The volume name, for use in the UI of OS X.
	VolumeName string

	// Extra options to be passed to mount(2), keyed by option name.
	//
	// For options without a value, the value may be empty.
	Options map[string]string
}

func getFuseMountConfig(fsName string, newConfig *cfg.Config) *MountConfig {
	// Handle the repeated "-o" flag.
	parsedOptions := make(map[string]string)
	for _, o := range newConfig.FileSystem.FuseOptions {
		mount.ParseOptions(parsedOptions, o)
	}

	mountCfg := &MountConfig{
		FSName:     fsName,
		Subtype:    "gcsfuse",
		VolumeName: "gcsfuse",
		Options:    parsedOptions,
	}

	return mountCfg
}
