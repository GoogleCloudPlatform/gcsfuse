package logger

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

func setLoggingLevel(level string, programLevel *slog.LevelVar) {
	switch level {
	// logs having severity >= the configured value will be logged.
	case "TRACE":
		// Setting severity to -8, so that all the other levels are logged.
		programLevel.Set(-8)
		break
	case "DEBUG":
		programLevel.Set(slog.LevelDebug)
		break
	case "INFO":
		programLevel.Set(slog.LevelInfo)
		break
	case "WARNING":
		programLevel.Set(slog.LevelWarn)
		break
	case "ERROR":
		programLevel.Set(slog.LevelError)
		break
	case "OFF":
		// Setting severity to 12, so that nothing is logged.
		programLevel.Set(12)
		break
	}
}

// this code has been copied from https://github.com/golang/go/blob/master/src/log/slog/logger.go#L46
// inorder to provide support for appending prefix string to new loggers.
type handlerWriter struct {
	h         slog.Handler
	level     slog.Level
	capturePC bool
}

func (w *handlerWriter) Write(buf []byte) (int, error) {
	if !w.h.Enabled(context.Background(), w.level) {
		return 0, nil
	}
	var pc uintptr
	if w.capturePC {
		// skip [runtime.Callers, w.Write, Logger.Output, log.Print]
		var pcs [1]uintptr
		runtime.Callers(4, pcs[:])
		pc = pcs[0]
	}

	// Remove final newline.
	origLen := len(buf) // Report that the entire buf was written.
	if len(buf) > 0 && buf[len(buf)-1] == '\n' {
		buf = buf[:len(buf)-1]
	}
	r := slog.NewRecord(time.Now(), w.level, string(buf), pc)
	return origLen, w.h.Handle(context.Background(), r)
}
