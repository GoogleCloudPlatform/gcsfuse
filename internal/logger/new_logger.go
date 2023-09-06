// Copyright 2020 Google Inc. All Rights Reserved.
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

package logger

import (
	"log"
	"log/slog"
)

// NewLogger creates a new logger. Avoid using this logger wherever possible and
// prioritise using the default logger methods like Info(), Warn(), Errorf(), etc
// present in logger package as this does not provide control to set log level for
// individual log messages.
func (f *loggerFactory) NewLogger(level slog.Level, prefix string) *log.Logger {
	var programLevel = new(slog.LevelVar)
	logger := slog.NewLogLogger(f.handler(programLevel, prefix), level)
	setLoggingLevel(f.level, programLevel)
	return logger
}

// DefaultLoggerFactory returns the defaultLoggerFactory of the logger package.
func DefaultLoggerFactory() *loggerFactory {
	return defaultLoggerFactory
}
