// Copyright 2023 Google Inc. All Rights Reserved.
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

package config

import "github.com/googlecloudplatform/gcsfuse/internal/flag"

// OverrideWithLoggingFlags overwrites the configs with the flag values if the
// config values are empty.
func OverrideWithLoggingFlags(mountConfig *MountConfig, flag *flag.FlagStorage) {
	// If log file is not set in config file, override it with flag value.
	if mountConfig.LogConfig.FilePath == "" {
		mountConfig.LogConfig.FilePath = flag.LogFile
	}
	// If log format is not set in config file, override it with flag value.
	if mountConfig.LogConfig.Format == "" {
		mountConfig.LogConfig.Format = flag.LogFormat
	}
	// If debug_fuse, debug_gcsfuse or debug_mutex flag is set, override log
	// severity to TRACE.
	if flag.DebugFuse || flag.DebugGCS || flag.DebugMutex {
		mountConfig.LogConfig.Severity = TRACE
	}
}
