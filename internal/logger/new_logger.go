package logger

import (
	"log"
	"log/slog"
)

func (f *loggerFactory) NewLogger(prefix string, level slog.Level) *log.Logger {
	var programLevel = new(slog.LevelVar)
	logger := slog.NewLogLogger(f.handler(programLevel, prefix), level)
	setLoggingLevel(f.level, programLevel)
	return logger
}

func DefaultLoggerFactory() *loggerFactory {
	return defaultLoggerFactory
}
