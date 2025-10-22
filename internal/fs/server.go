// Copyright 2021 Google LLC
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

//go:build libfuse
// +build libfuse

package fs

import (
	"fmt"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
)

func getFuseMountConfig(
	fsName string,
	newConfig *cfg.Config) (mountCfg *fuse.MountConfig) {
	// Create a mounting config.
	mountCfg = &fuse.MountConfig{
		FSName:   fsName,
		Subtype:  "gcsfuse",
		VolumeName: "gcsfuse",
		Options:  make(map[string]string),
	}
	if newConfig.Write.Debug {
		mountCfg.DebugLogger = logger.NewDebug("fuse: ")
	}
	if newConfig.Foreground {
		mountCfg.ErrorLogger = logger.NewError("fuse: ")
	}
	for k, v := range newConfig.MountOptions {
		mountCfg.Options[k] = v
	}
	// Let the user override the file system name.
	fsName, ok := mountCfg.Options["fsname"]
	if ok {
		mountCfg.FSName = fsName
	}
	return
}
