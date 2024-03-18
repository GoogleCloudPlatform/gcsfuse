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

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
)

const (
	// LevelTrace value is set to -8, so that all the other log levels are logged.
	LevelTrace = slog.Level(-8)
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
	// LevelOff value is set to 12, so that nothing is logged.
	LevelOff     = slog.Level(12)
	messageKey   = "message"
	timestampKey = "timestamp"
	secondsKey   = "seconds"
	nanosKey     = "nanos"
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

// CustomiseLevels changes the name of the level key to "severity" and the value to
// it's corresponding level like DEBUG, INFO, ERROR, etc.
func customiseLevels(a *slog.Attr) {
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

// AddPrefixToMessage adds the prefix to the log message.
func addPrefixToMessage(a *slog.Attr, prefix string) {
	// Change key name to "message" so that it is compatible with fluentD.
	a.Key = messageKey
	// Add prefix to the message.
	message := a.Value.Any().(string)
	var sb strings.Builder
	sb.WriteString(prefix)
	sb.WriteString(message)
	a.Value = slog.StringValue(sb.String())
}

// CustomiseTimeFormat converts the time to below specified format:
// 1. for json logs:
// "timestamp":{"seconds":1704697907,"nanos":553918512}
// 2. for text logs:
// time="08/09/2023 09:24:54.437193"
func customiseTimeFormat(a *slog.Attr, format string) {
	currTime := a.Value.Any().(time.Time).Round(0)
	if format == textFormat {
		a.Value = slog.StringValue(currTime.Round(0).Format("02/01/2006 03:04:05.000000"))
	} else {
		*a = slog.Group(timestampKey, secondsKey, currTime.Unix(), nanosKey, currTime.Nanosecond())
	}
}

func getHandlerOptions(levelVar *slog.LevelVar, prefix string, format string) *slog.HandlerOptions {
	return &slog.HandlerOptions{
		// Set log level to configured value.
		Level: levelVar,

		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				customiseLevels(&a)
			}

			// Add prefix to the message value.
			if a.Key == slog.MessageKey {
				addPrefixToMessage(&a, prefix)
			}

			if a.Key == slog.TimeKey {
				customiseTimeFormat(&a, format)
			}
			return a
		},
	}
}
