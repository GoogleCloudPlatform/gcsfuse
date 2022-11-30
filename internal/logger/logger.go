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
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jacobsa/daemonize"
)

var (
	defaultLoggerFactory *loggerFactory
	defaultInfoLogger    *log.Logger
	listOfLoggers        []*log.Logger

	fileNumber     int
	base_file_name string
	extension      string
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
	listOfLoggers = make([]*log.Logger, 0, 15)
	fileNumber = 0

	base_name := filepath.Base(defaultLoggerFactory.file.Name())
	dot_pos := strings.Index(base_name, ".")
	if dot_pos == -1 {
		return fmt.Errorf("There should be some extension in the base name")
	}
	base_file_name = base_name[:dot_pos]
	extension = base_name[(dot_pos + 1):]
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

func UpdateTheWriterInAllLoggers() error {
	for _, lg := range listOfLoggers {
		lg.SetOutput(defaultLoggerFactory.writer(lg.Prefix()))
	}
	return nil
}

func HandleReInitLogFile() error {
	for {
		time.Sleep(time.Minute / 6)
		base_name := filepath.Base(defaultLoggerFactory.file.Name())
		dot_pos := strings.Index(base_name, ".")

		if dot_pos == -1 {
			return fmt.Errorf("There should be some extension in the base name")
		}

		rel_file_name := base_file_name + strconv.Itoa(fileNumber) + "." + extension
		file_name := filepath.Dir(defaultLoggerFactory.file.Name()) + "/" + rel_file_name
		ReInitLogFile(file_name)
	}
}

func ReInitLogFile(filename string) error {
	f, err := os.OpenFile(
		filename,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		0644,
	)
	if err != nil {
		return err
	}
	previousLoggerFactory := defaultLoggerFactory
	defaultLoggerFactory = &loggerFactory{
		file:   f,
		flag:   0,
		format: previousLoggerFactory.format,
	}
	defaultInfoLogger = NewInfo("")
	UpdateTheWriterInAllLoggers()

	fileNumber++
	if fp := previousLoggerFactory.file; fp != nil {
		fp.Close()
		previousLoggerFactory.file = nil
	}
	return nil
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
	tmp := defaultLoggerFactory.newLogger("NOTICE", prefix)
	listOfLoggers = append(listOfLoggers, tmp)
	return tmp
}

// NewDebug returns a new logger for logging debug messages with given prefix
// to the log file or stdout.
func NewDebug(prefix string) *log.Logger {
	tmp := defaultLoggerFactory.newLogger("DEBUG", prefix)
	listOfLoggers = append(listOfLoggers, tmp)
	return tmp
}

// NewInfo returns a new logger for logging info with given prefix to the log
// file or stdout.
func NewInfo(prefix string) *log.Logger {
	tmp := defaultLoggerFactory.newLogger("INFO", prefix)
	listOfLoggers = append(listOfLoggers, tmp)
	return tmp
}

// NewError returns a new logger for logging errors with given prefix to the log
// file or stderr.
func NewError(prefix string) *log.Logger {
	tmp := defaultLoggerFactory.newLogger("ERROR", prefix)
	listOfLoggers = append(listOfLoggers, tmp)
	return tmp
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
