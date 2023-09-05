package logger

import (
	"log/slog"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
)

const (
	LevelTrace = slog.Level(-8)
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
	LevelOff   = slog.Level(12)
)

func setLoggingLevel(level config.LogSeverity, programLevel *slog.LevelVar) {
	switch level {
	// logs having severity >= the configured value will be logged.
	case config.TRACE:
		// Setting severity to -8, so that all the other levels are logged.
		programLevel.Set(LevelTrace)
		break
	case config.DEBUG:
		programLevel.Set(LevelDebug)
		break
	case config.INFO:
		programLevel.Set(LevelInfo)
		break
	case config.WARNING:
		programLevel.Set(LevelWarn)
		break
	case config.ERROR:
		programLevel.Set(LevelError)
		break
	case config.OFF:
		// Setting severity to 12, so that nothing is logged.
		programLevel.Set(LevelOff)
		break
	}
}

func GetHandlerOptions(levelVar *slog.LevelVar, prefix string) *slog.HandlerOptions {
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

			if a.Key == slog.MessageKey {
				message := a.Value.Any().(string)
				var sb strings.Builder
				sb.WriteString(prefix)
				sb.WriteString(message)
				a.Value = slog.StringValue(sb.String())
			}
			return a
		},
	}
}
