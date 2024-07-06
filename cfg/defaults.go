// Copyright 2024 Google Inc. All Rights Reserved.
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

package cfg

// GetDefaultLoggingConfig returns the default configuration that is to be used
// during the application startup - when the provided configuration hasn't been
// parsed yet.
func GetDefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		Severity: "INFO",
		LogRotate: LogRotateLoggingConfig{
			BackupFileCount: 10,
			Compress:        true,
			MaxFileSizeMb:   512,
		},
	}
}
