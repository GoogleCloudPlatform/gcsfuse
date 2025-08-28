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
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/mount"
)

func getFuseMountConfig(fsName string, newConfig *cfg.Config) *MountConfig {
	// Handle the repeated "-o" flag.
	parsedOptions := make(map[string]string)
	for _, o := range newConfig.FileSystem.FuseOptions {
		mount.ParseOptions(parsedOptions, o)
	}

	mountCfg := &fuse.MountConfig{
		FSName:                      fsName,
		Subtype:                     "gcsfuse",
		VolumeName:                  "gcsfuse",
		Options:                     parsedOptions,
		EnableParallelDirOps:        !(newConfig.FileSystem.DisableParallelDirops),
		DisableWritebackCaching:     newConfig.Write.EnableStreamingWrites,
		EnableReaddirplus:           newConfig.FileSystem.ExperimentalEnableReaddirplus,
		EnableNoOpenSupport:         !newConfig.FileSystem.EnableNoOpen,
		EnableNoOpendirSupport:      !newConfig.FileSystem.EnableNoOpendir,
		EnableAsyncRead:             newConfig.FileSystem.EnableAsyncRead,
		EnableFusexattr:             newConfig.FileSystem.EnableFusexattr,
		IgnoreSecurityLabels:        newConfig.FileSystem.IgnoreSecurityLabels,
		KernelListCacheTTL:          cfg.ListCacheTTLSecsToDuration(newConfig.FileSystem.KernelListCacheTtlSecs),
		MaxBackgroundActiveRequests: newConfig.FileSystem.MaxBackgroundActiveRequests,
	}

	// GCSFuse to Jacobsa Fuse Log Level mapping:
	// OFF           OFF
	// ERROR         ERROR
	// WARNING       ERROR
	// INFO          ERROR
	// DEBUG         ERROR
	// TRACE         TRACE
	if newConfig.Logging.Severity.Rank() <= cfg.ErrorLogSeverity.Rank() {
		mountCfg.ErrorLogger = logger.NewLegacyLogger(logger.LevelError, "fuse: ")
	}
	if newConfig.Logging.Severity.Rank() <= cfg.TraceLogSeverity.Rank() {
		mountCfg.DebugLogger = logger.NewLegacyLogger(logger.LevelTrace, "fuse_debug: ")
	}
	return mountCfg
}
