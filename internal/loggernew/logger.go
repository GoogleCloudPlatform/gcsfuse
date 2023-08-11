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

package loggernew

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"log/syslog"
	"os"
	"runtime"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Syslog file contains logs from all different programmes running on the VM.
// ProgrammeName is prefixed to all the logs written to syslog. This constant is
// used to filter the logs from syslog and write it to respective log files -
// gcsfuse.log in case of GCSFuse.
const ProgrammeName string = "gcsfuse"
const GCSFuseInBackgroundMode string = "GCSFUSE_IN_BACKGROUND_MODE"

var (
	defaultLoggerFactory *loggerFactory
	defaultLogger        *slog.Logger
)

// InitLogFile initializes the logger factory to create loggers that print to
// a log file.
// In case of empty file, it starts writing the log to syslog file, which
// is eventually filtered and redirected to a fixed location using syslog
// config.
// Here, background true means, this InitLogFile has been called for the
// background daemon.
func InitLogFile(filename string, format string, level string) error {
	var f *os.File
	var sysWriter *syslog.Writer
	var err error
	if filename != "" {
		f, err = os.OpenFile(
			filename,
			os.O_WRONLY|os.O_CREATE|os.O_APPEND,
			0644,
		)
		if err != nil {
			return err
		}
	} else {
		if _, ok := os.LookupEnv(GCSFuseInBackgroundMode); ok {
			// Priority consist of facility and severity, here facility to specify the
			// type of system that is logging the message to syslog and severity is log-level.
			// User applications are allowed to take facility value between LOG_LOCAL0
			// to LOG_LOCAL7. We are using LOG_LOCAL7 as facility and LOG_DEBUG to write
			// debug messages.

			// Suppressing the error while creating the syslog, although logger will
			// be initialised with stdout/err, log will be printed anywhere. Because,
			// in this case gcsfuse will be running as daemon.
			sysWriter, _ = syslog.New(syslog.LOG_LOCAL7|syslog.LOG_DEBUG, ProgrammeName)
		}
	}

	defaultLoggerFactory = &loggerFactory{
		file:      f,
		sysWriter: sysWriter,
		flag:      0,
		format:    format,
		level:     level,
	}
	defaultLogger = defaultLoggerFactory.newLogger(level)

	return nil
}

func GetDefaultLoggerFactory() *loggerFactory {
	return defaultLoggerFactory
}

// init initializes the logger factory to use stdout and stderr.
func init() {
	defaultLoggerFactory = &loggerFactory{
		file:  nil,
		flag:  log.Ldate | log.Lmicroseconds,
		level: "INFO",
	}
	defaultLogger = defaultLoggerFactory.newLogger("INFO")

}

// Close closes the log file when necessary.
func Close() {
	if f := defaultLoggerFactory.file; f != nil {
		f.Close()
		defaultLoggerFactory.file = nil
	}
}

// Errorf calls the default error logger to print the message using Printf.
func Errorf(format string, v ...interface{}) {
	defaultLogger.Error(fmt.Sprintf(format, v...))
}

// Info calls the default info logger to print the message using Printf.
func Infof(format string, v ...interface{}) {
	defaultLogger.Info(fmt.Sprintf(format, v...))
}

// Info calls the default info logger to print the message using Println.
func Info(v ...interface{}) {
	defaultLogger.Info(fmt.Sprintln(v...))
}

// Fatal calls the default info logger to call the Fatal function of go-src-logs.
// https://github.com/golang/go/blob/master/src/log/log.go#L282
func Fatal(v ...interface{}) {
	Info(v...)
	os.Exit(1)
}

type loggerFactory struct {
	// If nil, log to stdout or stderr. Otherwise, log to this file.
	file      *os.File
	sysWriter *syslog.Writer
	flag      int
	format    string
	level     string
}

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

func setLoggingLevel(level string, programLevel *slog.LevelVar) {
	switch level {
	case "INFO":
	case "info":
		programLevel.Set(slog.LevelInfo)
		break
	case "DEBUG":
	case "debug":
		programLevel.Set(slog.LevelDebug)
		break
	case "ERROR":
	case "error":
		programLevel.Set(slog.LevelError)
		break
	}
}

func (f *loggerFactory) NewStandardDebugLogger(prefix string) *log.Logger {
	var programLevel = new(slog.LevelVar) // Info by default
	//logger := slog.NewLogLogger(f.handler(programLevel), slog.LevelDebug)
	logger := log.New(&handlerWriter{f.handler(programLevel), slog.LevelDebug, true}, prefix, 0)
	setLoggingLevel(f.level, programLevel)
	return logger
}

func (f *loggerFactory) newLogger(level string) *slog.Logger {
	var programLevel = new(slog.LevelVar) // Info by default
	logger := slog.New(f.handler(programLevel))
	slog.SetDefault(logger)
	setLoggingLevel(level, programLevel)
	return logger
}

func (f *loggerFactory) handler(levelVar *slog.LevelVar) slog.Handler {
	if f.file != nil {
		fileWriter := &lumberjack.Logger{
			Filename:   f.file.Name(),
			MaxSize:    1, // megabytes
			MaxBackups: 3,
			MaxAge:     28,   //days
			Compress:   true, // disabled by default
		}
		return f.createJsonOrTextHandler(fileWriter, levelVar)
	}

	if f.sysWriter != nil {
		return f.createJsonOrTextHandler(f.sysWriter, levelVar)
	}

	return f.createJsonOrTextHandler(os.Stdout, levelVar)

	//switch level {
	//case "ERROR":
	//	return f.createJsonOrTextHandler(os.Stderr)
	//default:
	//	return f.createJsonOrTextHandler(os.Stdout)
	//}
}

func (f *loggerFactory) createJsonOrTextHandler(writer io.Writer, levelVar *slog.LevelVar) slog.Handler {
	if f.format == "json" {
		return slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: levelVar})
	}
	return slog.NewTextHandler(writer, &slog.HandlerOptions{Level: levelVar})
}
