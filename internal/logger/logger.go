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
	"os"

	"github.com/jacobsa/daemonize"
)

var (
	defaultLoggerFactory *loggerFactory
	defaultInfoLogger    *log.Logger
)

// InitLogFile initializes the logger factory to create loggers that print to
// a log file.
func InitLogFile(filename string, format string) error {
	f, err := os.OpenFile(
		filename,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		0644,
	)
	if err != nil {
		return err
	}

	defaultLoggerFactory = &loggerFactory{
		file:   f,
		flag:   0,
		format: format,
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

type loggerFactory struct {
	// If nil, log to stdout or stderr. Otherwise, log to this file.
	file   *os.File
	flag   int
	format string
}

func (f *loggerFactory) newLogger(level, prefix string) *log.Logger {
	return log.New(f.writer(level), prefix, f.flag)
}

func (f *loggerFactory) writer(level string) io.Writer {
	if f.file != nil {
		switch f.format {
		case "json":
			return &jsonWriter{
				w:     f.file,
				level: level,
			}
		case "text":
			return &textWriter{
				w:     f.file,
				level: level,
			}
		}
	}

	switch level {
	case "NOTICE":
		return daemonize.StatusWriter
	case "ERROR":
		return os.Stderr
	default:
		return os.Stdout
	}
}
