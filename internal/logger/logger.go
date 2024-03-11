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
	"context"
	"fmt"
	"io"
	"log/slog"
	"log/syslog"
	"os"
	"runtime/debug"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Syslog file contains logs from all different programmes running on the VM.
// ProgrammeName is prefixed to all the logs written to syslog. This constant is
// used to filter the logs from syslog and write it to respective log files -
// gcsfuse.log in case of GCSFuse.
const (
	ProgrammeName           string = "gcsfuse"
	GCSFuseInBackgroundMode string = "GCSFUSE_IN_BACKGROUND_MODE"
	jsonFormat              string = "json"
	textFormat              string = "text"
	defaultFormat           string = jsonFormat
)

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
func InitLogFile(logConfig config.LogConfig) error {
	var f *os.File
	var sysWriter *syslog.Writer
	var fileWriter *lumberjack.Logger
	var err error
	if logConfig.FilePath != "" {
		f, err = os.OpenFile(
			logConfig.FilePath,
			os.O_WRONLY|os.O_CREATE|os.O_APPEND,
			0644,
		)
		if err != nil {
			return err
		}
		fileWriter = &lumberjack.Logger{
			Filename:   f.Name(),
			MaxSize:    logConfig.LogRotateConfig.MaxFileSizeMB,
			MaxBackups: logConfig.LogRotateConfig.BackupFileCount,
			Compress:   logConfig.LogRotateConfig.Compress,
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
		file:            f,
		sysWriter:       sysWriter,
		fileWriter:      fileWriter,
		format:          logConfig.Format,
		level:           logConfig.Severity,
		logRotateConfig: logConfig.LogRotateConfig,
	}
	defaultLogger = defaultLoggerFactory.newLogger(logConfig.Severity)

	return nil
}

// init initializes the logger factory to use stdout and stderr.
func init() {
	defaultLoggerFactory = &loggerFactory{
		file:            nil,
		format:          defaultFormat,
		level:           config.INFO, // setting log level to INFO by default
		logRotateConfig: config.DefaultLogRotateConfig(),
	}
	defaultLogger = defaultLoggerFactory.newLogger(config.INFO)
}

// SetLogFormat updates the log format of default logger.
func SetLogFormat(format string) {
	if format == defaultLoggerFactory.format {
		return
	}
	defaultLoggerFactory.format = format
	defaultLogger = defaultLoggerFactory.newLogger(defaultLoggerFactory.level)
}

// Close closes the log file when necessary.
func Close() {
	if f := defaultLoggerFactory.file; f != nil {
		f.Close()
		defaultLoggerFactory.file = nil
	}
}

// Tracef prints the message with TRACE severity in the specified format.
func Tracef(format string, v ...interface{}) {
	defaultLogger.Log(context.Background(), LevelTrace, fmt.Sprintf(format, v...))
}

// Debugf prints the message with DEBUG severity in the specified format.
func Debugf(format string, v ...interface{}) {
	defaultLogger.Debug(fmt.Sprintf(format, v...))
}

// Infof prints the message with INFO severity in the specified format.
func Infof(format string, v ...interface{}) {
	defaultLogger.Info(fmt.Sprintf(format, v...))
}

// Info prints the message with info severity.
func Info(v ...interface{}) {
	defaultLogger.Info(fmt.Sprint(v...))
}

// Warnf prints the message with WARNING severity in the specified format.
func Warnf(format string, v ...interface{}) {
	defaultLogger.Warn(fmt.Sprintf(format, v...))
}

// Errorf prints the message with ERROR severity in the specified format.
func Errorf(format string, v ...interface{}) {
	defaultLogger.Error(fmt.Sprintf(format, v...))
}

// Fatal prints an error log and exits with non-zero exit code.
func Fatal(format string, v ...interface{}) {
	Errorf(format, v...)
	Errorf(string(debug.Stack()))
	os.Exit(1)
}

type loggerFactory struct {
	// If nil, log to stdout or stderr. Otherwise, log to this file.
	file            *os.File
	sysWriter       *syslog.Writer
	format          string
	level           config.LogSeverity
	logRotateConfig config.LogRotateConfig
	fileWriter      *lumberjack.Logger
}

func (f *loggerFactory) newLogger(level config.LogSeverity) *slog.Logger {
	// create a new logger
	var programLevel = new(slog.LevelVar)
	logger := slog.New(f.handler(programLevel, ""))
	slog.SetDefault(logger)
	setLoggingLevel(level, programLevel)
	return logger
}

func (f *loggerFactory) createJsonOrTextHandler(writer io.Writer, levelVar *slog.LevelVar, prefix string) slog.Handler {
	if f.format == textFormat {
		return slog.NewTextHandler(writer, getHandlerOptions(levelVar, prefix, f.format))
	}
	return slog.NewJSONHandler(writer, getHandlerOptions(levelVar, prefix, f.format))
}

func (f *loggerFactory) handler(levelVar *slog.LevelVar, prefix string) slog.Handler {
	if f.fileWriter != nil {
		return f.createJsonOrTextHandler(f.fileWriter, levelVar, prefix)
	}

	if f.sysWriter != nil {
		return f.createJsonOrTextHandler(f.sysWriter, levelVar, prefix)
	}
	return f.createJsonOrTextHandler(os.Stdout, levelVar, prefix)
}
