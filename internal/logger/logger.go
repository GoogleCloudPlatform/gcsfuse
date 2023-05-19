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
	"io"
	"log"
	"log/syslog"
	"os"

	"github.com/jacobsa/daemonize"
)

// Syslog file contains logs from all different programmes running on the VM.
// ProgrammeName is prefixed to all the logs written to syslog. This constant is
// used to filter the logs from syslog and write it to respective log files -
// gcsfuse.log in case of GCSFuse.
const ProgrammeName string = "gcsfuse"
const GCSFuseInBackgroundMode string = "GCSFUSE_IN_BACKGROUND_MODE"

var (
	defaultLoggerFactory *loggerFactory
	defaultInfoLogger    *log.Logger
)

// InitLogFile initializes the logger factory to create loggers that print to
// a log file.
// In case of empty file, it starts writing the log to syslog file, which
// is eventually filtered and redirected to a fixed location using syslog
// config.
// Here, background true means, this InitLogFile has been called for the
// background daemon.
func InitLogFile(filename string, format string) error {
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
	}
	defaultInfoLogger = NewInfo("")

	return nil
}

// init initializes the logger factory to use stdout and stderr.
func init() {
	defaultLoggerFactory = &loggerFactory{
		file: nil,
		flag: log.Ldate | log.Ltime | log.Lmicroseconds,
	}
	defaultInfoLogger = NewInfo("")
}

// Close closes the log file when necessary.
func Close() {
	if f := defaultLoggerFactory.file; f != nil {
		f.Close()
		defaultLoggerFactory.file = nil
	}
}

// NewNotice returns a new logger for logging notice with given prefix to
// the log file or the status writer which forwards the notices to the invoker
// from the daemon.
func NewNotice(prefix string) *log.Logger {
	return defaultLoggerFactory.newLogger("NOTICE", prefix)
}

// NewDebug returns a new logger for logging debug messages with given prefix
// to the log file or stdout.
func NewDebug(prefix string) *log.Logger {
	return defaultLoggerFactory.newLogger("DEBUG", prefix)
}

// NewInfo returns a new logger for logging info with given prefix to the log
// file or stdout.
func NewInfo(prefix string) *log.Logger {
	return defaultLoggerFactory.newLogger("INFO", prefix)
}

// NewError returns a new logger for logging errors with given prefix to the log
// file or stderr.
func NewError(prefix string) *log.Logger {
	return defaultLoggerFactory.newLogger("ERROR", prefix)
}

// Info calls the default info logger to print the message using Printf.
func Infof(format string, v ...interface{}) {
	defaultInfoLogger.Printf(format, v...)
}

// Info calls the default info logger to print the message using Println.
func Info(v ...interface{}) {
	defaultInfoLogger.Println(v...)
}

// Fatal calls the default info logger to call the Fatal function of go-src-logs.
// https://github.com/golang/go/blob/master/src/log/log.go#L282
func Fatal(v ...interface{}) {
	defaultInfoLogger.Fatal(v...)
}

type loggerFactory struct {
	// If nil, log to stdout or stderr. Otherwise, log to this file.
	file      *os.File
	sysWriter *syslog.Writer
	flag      int
	format    string
}

func (f *loggerFactory) newLogger(level, prefix string) *log.Logger {
	return log.New(f.writer(level), prefix, f.flag)
}

func (f *loggerFactory) createJsonOrTextWriter(level string, writer io.Writer) io.Writer {
	if f.format == "json" {
		return &jsonWriter{
			w:     writer,
			level: level,
		}
	}

	return &textWriter{
		w:     writer,
		level: level,
	}
}

func (f *loggerFactory) writer(level string) io.Writer {
	if f.file != nil {
		return f.createJsonOrTextWriter(level, f.file)
	}

	if f.sysWriter != nil {
		return f.createJsonOrTextWriter(level, f.sysWriter)
	}

	switch level {
	case "NOTICE":
		return f.createJsonOrTextWriter(level, daemonize.StatusWriter)
	case "ERROR":
		return f.createJsonOrTextWriter(level, os.Stderr)
	default:
		return f.createJsonOrTextWriter(level, os.Stdout)
	}
}
