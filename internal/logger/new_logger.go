package logger

import (
	"log"
	"log/slog"
)

func (f *loggerFactory) NewStandardDebugLogger(prefix string) *log.Logger {
	var programLevel = new(slog.LevelVar)
	logger := log.New(&handlerWriter{f.handler(programLevel), slog.LevelDebug, true}, prefix, 0)
	setLoggingLevel(f.level, programLevel)
	return logger
}

func DefaultLoggerFactory() *loggerFactory {
	return defaultLoggerFactory
}
