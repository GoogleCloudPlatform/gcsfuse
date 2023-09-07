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

package logger

import (
	"log/slog"
	"strings"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
)

const (
	// LevelTrace value is set to -8, so that all the other log levels are logged.
	LevelTrace = slog.Level(-8)
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
	// LevelOff value is set to 12, so that nothing is logged.
	LevelOff = slog.Level(12)
)

func setLoggingLevel(level config.LogSeverity, programLevel *slog.LevelVar) {
	switch level {
	// logs having severity >= the configured value will be logged.
	case config.TRACE:
		programLevel.Set(LevelTrace)
	case config.DEBUG:
		programLevel.Set(LevelDebug)
	case config.INFO:
		programLevel.Set(LevelInfo)
	case config.WARNING:
		programLevel.Set(LevelWarn)
	case config.ERROR:
		programLevel.Set(LevelError)
	case config.OFF:
		programLevel.Set(LevelOff)
	}
}

func getHandlerOptions(levelVar *slog.LevelVar, prefix string) *slog.HandlerOptions {
	return &slog.HandlerOptions{
		// Set log level to configured value.
		Level: levelVar,

		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize the name of the level key and the output string, including
			// custom level values.
			if a.Key == slog.LevelKey {
				// Rename the level key from "level" to "sev".
				a.Key = "severity"

				// Handle custom level values.
				level := a.Value.Any().(slog.Level)
				switch {
				case level == LevelTrace:
					a.Value = slog.StringValue(string(config.TRACE))
				case level == LevelDebug:
					a.Value = slog.StringValue(string(config.DEBUG))
				case level == LevelInfo:
					a.Value = slog.StringValue(string(config.INFO))
				case level == LevelWarn:
					a.Value = slog.StringValue(string(config.WARNING))
				case level == LevelError:
					a.Value = slog.StringValue(string(config.ERROR))
				case level == LevelOff:
					a.Value = slog.StringValue(string(config.OFF))
				default:
					a.Value = slog.StringValue(string(config.INFO))
				}
			}

			// Add prefix to the message key.
			if a.Key == slog.MessageKey {
				message := a.Value.Any().(string)
				var sb strings.Builder
				sb.WriteString(prefix)
				sb.WriteString(message)
				a.Value = slog.StringValue(sb.String())
			}

			if a.Key == slog.TimeKey {
				currTime := a.Value.Any().(time.Time)
				a.Value = slog.StringValue(currTime.Round(0).Format("02/01/2006 03:04:05.000000"))
			}
			return a
		},
	}
}
