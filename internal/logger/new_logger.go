package logger

import (
	"log"
	"log/slog"
)

func (f *loggerFactory) NewDebugLogger(prefix string) *log.Logger {
	var programLevel = new(slog.LevelVar)
	logger := log.New(&handlerWriter{f.handler(programLevel), slog.LevelDebug, true}, prefix, 0)
	setLoggingLevel(f.level, programLevel)
	return logger
}

func (f *loggerFactory) NewErrorLogger(prefix string) *log.Logger {
	var programLevel = new(slog.LevelVar)
	logger := log.New(&handlerWriter{f.handler(programLevel), slog.LevelError, true}, prefix, 0)
	setLoggingLevel(f.level, programLevel)
	return logger
}

func DefaultLoggerFactory() *loggerFactory {
	return defaultLoggerFactory
}
