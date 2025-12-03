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

// TODO: remove these methods/file after slog support is added to jacobsa/fuse.

package logger

import (
	"log"
	"log/slog"
)

// NewLegacyLogger creates a new legacy logger. Avoid using this logger wherever possible and
// prioritise using the default logger methods like Infof(), Warnf(), Errorf(), etc
// present in logger package as this does not provide control to set log level for
// individual log messages.
// This method is created to support jacobsa/fuse loggers and will be removed
// after slog support is added.
func NewLegacyLogger(level slog.Level, prefix, fsName string) *log.Logger {
	handler := defaultLoggerFactory.handler(programLevel, prefix).WithAttrs(loggerAttr(fsName))
	logger := slog.NewLogLogger(handler, level)
	setLoggingLevel(defaultLoggerFactory.level)
	return logger
}
