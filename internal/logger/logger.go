// Copyright 2020 Google LLC
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
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Syslog file contains logs from all different programs running on the VM.
// ProgramName is prefixed to all the logs written to syslog. This constant is
// used to filter the logs from syslog and write it to respective log files -
// gcsfuse.log in case of GCSFuse.
const (
	ProgramName             = "gcsfuse"
	GCSFuseInBackgroundMode = "GCSFUSE_IN_BACKGROUND_MODE"
	MountUUIDEnvKey         = "GCSFUSE_MOUNT_UUID"
	MountIDKey              = "mount-id" // Combination of fsName and GCSFUSE_MOUNT_UUID
	textFormat              = "text"
	// Max possible length can be 32 as UUID has 32 characters excluding 4 hyphens.
	mountUUIDLength = 8
)

var (
	defaultLoggerFactory *loggerFactory
	defaultLogger        *slog.Logger
	mountUUID            string
	setupMountUUIDOnce   sync.Once
)

// InitLogFile initializes the logger factory to create loggers that print to
// a log file, with MountInstanceID set as a custom attribute.
// In case of empty file, it starts writing the log to syslog file, which
// is eventually filtered and redirected to a fixed location using syslog
// config.
// Here, background true means, this InitLogFile has been called for the
// background daemon.
func InitLogFile(newLogConfig cfg.LoggingConfig, fsName string) error {
	var f *os.File
	var sysWriter *syslog.Writer
	var fileWriter *lumberjack.Logger
	var err error
	if newLogConfig.FilePath != "" {
		f, err = os.OpenFile(
			string(newLogConfig.FilePath),
			os.O_WRONLY|os.O_CREATE|os.O_APPEND,
			0644,
		)
		if err != nil {
			return err
		}
		fileWriter = &lumberjack.Logger{
			Filename:   f.Name(),
			MaxSize:    int(newLogConfig.LogRotate.MaxFileSizeMb),
			MaxBackups: int(newLogConfig.LogRotate.BackupFileCount),
			Compress:   newLogConfig.LogRotate.Compress,
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
			sysWriter, _ = syslog.New(syslog.LOG_LOCAL7|syslog.LOG_DEBUG, ProgramName)
		}
	}

	defaultLoggerFactory = &loggerFactory{
		file:       f,
		sysWriter:  sysWriter,
		fileWriter: fileWriter,
		format:     newLogConfig.Format,
		level:      string(newLogConfig.Severity),
		logRotate:  newLogConfig.LogRotate,
	}
	defaultLogger = defaultLoggerFactory.newLoggerWithMountInstanceID(string(newLogConfig.Severity), fsName)

	return nil
}

// init initializes the logger factory to use stdout and stderr.
func init() {
	logConfig := cfg.DefaultLoggingConfig()
	defaultLoggerFactory = &loggerFactory{
		file:      nil,
		format:    logConfig.Format,
		level:     string(logConfig.Severity), // setting log level to INFO by default
		logRotate: logConfig.LogRotate,
	}
	defaultLogger = defaultLoggerFactory.newLogger(cfg.INFO)
}

// generateMountUUID generates a random string of size from UUID.
func generateMountUUID(size int) (string, error) {
	if size <= 0 {
		return "", fmt.Errorf("requested size for MountUUID must be positive, but got %d", size)
	}
	uuid := uuid.New()
	uuidStr := strings.ReplaceAll(uuid.String(), "-", "")
	if size > len(uuidStr) {
		return "", fmt.Errorf("UUID is smaller than requested size %d for MountUUID, UUID: %s", size, uuidStr)
	}
	return uuidStr[:size], nil
}

// setupMountUUID handles the retrieval of mountUUID if GCSFuse is in
// background mode or generates one if running in foreground mode.
func setupMountUUID() {
	if _, ok := os.LookupEnv(GCSFuseInBackgroundMode); ok {
		// If GCSFuse is in background mode then look for the GCSFUSE_MOUNT_UUID in env which was set by the caller of demonize run.
		if mountUUID, ok = os.LookupEnv(MountUUIDEnvKey); !ok || mountUUID == "" {
			Fatal("Could not retrieve %s env variable or it's empty.", MountUUIDEnvKey)
		}
		return
	}
	// If GCSFuse is not running in the background mode then generate a random UUID.
	var err error
	if mountUUID, err = generateMountUUID(mountUUIDLength); err != nil {
		Fatal("Could not generate MountUUID of length %d, err: %v", mountUUIDLength, err)
	}
}

// MountUUID returns a unique ID for the current GCSFuse mount,
// ensuring the ID is initialized only once. On the first call, it either
// generates a random ID (foreground mode) or retrieves it from the
// GCSFUSE_MOUNT_UUID environment variable (background mode).
// Subsequent calls return the same cached ID.
func MountUUID() string {
	setupMountUUIDOnce.Do(setupMountUUID)
	return mountUUID
}

// MountInstanceID returns the InstanceID of current gcsfuse mount.
// This is combination of `fsName` + MountUUID.
// Note: fsName is passed here explicitly, as logger package doesn't know about fsName
// when MountInstanceID method is invoked.
func MountInstanceID(fsName string) string {
	return fmt.Sprintf("%s-%s", fsName, MountUUID())
}

// UpdateDefaultLogger updates the log format and creates a new logger with MountInstanceID set as custom attribute.
func UpdateDefaultLogger(format, fsName string) {
	defaultLoggerFactory.format = format
	defaultLogger = defaultLoggerFactory.newLoggerWithMountInstanceID(defaultLoggerFactory.level, fsName)
}

// Tracef prints the message with TRACE severity in the specified format.
func Tracef(format string, v ...any) {
	defaultLogger.Log(context.Background(), LevelTrace, fmt.Sprintf(format, v...))
}

// Debugf prints the message with DEBUG severity in the specified format.
func Debugf(format string, v ...any) {
	defaultLogger.Debug(fmt.Sprintf(format, v...))
}

// Infof prints the message with INFO severity in the specified format.
func Infof(format string, v ...any) {
	defaultLogger.Info(fmt.Sprintf(format, v...))
}

// Info prints the message with info severity.
func Info(message string, args ...any) {
	defaultLogger.Info(message, args...)
}

// Warnf prints the message with WARNING severity in the specified format.
func Warnf(format string, v ...any) {
	defaultLogger.Warn(fmt.Sprintf(format, v...))
}

// Errorf prints the message with ERROR severity in the specified format.
func Errorf(format string, v ...any) {
	defaultLogger.Error(fmt.Sprintf(format, v...))
}

// Error prints the message with ERROR severity.
func Error(error string) {
	defaultLogger.Error(error)
}

// Fatal prints an error log and exits with non-zero exit code.
func Fatal(format string, v ...any) {
	Errorf(format, v...)
	Error(string(debug.Stack()))
	os.Exit(1)
}

type loggerFactory struct {
	// If nil, log to stdout or stderr. Otherwise, log to this file.
	file       *os.File
	sysWriter  *syslog.Writer
	format     string
	level      string
	logRotate  cfg.LogRotateLoggingConfig
	fileWriter *lumberjack.Logger
}

func (f *loggerFactory) newLogger(level string) *slog.Logger {
	// create a new logger
	var programLevel = new(slog.LevelVar)
	logger := slog.New(f.handler(programLevel, ""))
	slog.SetDefault(logger)
	setLoggingLevel(level, programLevel)
	return logger
}

func loggerAttr(fsName string) []slog.Attr {
	return []slog.Attr{slog.String(MountIDKey, MountInstanceID(fsName))}
}

// create a new logger with mountInstanceID set as custom attribute on logger.
func (f *loggerFactory) newLoggerWithMountInstanceID(level, fsName string) *slog.Logger {
	var programLevel = new(slog.LevelVar)
	logger := slog.New(f.handler(programLevel, "").WithAttrs(loggerAttr(fsName)))
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

var (
	// latencies stores read latencies in microseconds.
	latencies []int64
)

// AddLatency adds a duration to the global latencies slice in microseconds.
func AddLatency(duration time.Duration) {
	latencies = append(latencies, duration.Microseconds())
}

func LogLatencies() {
	if len(latencies) == 0 {
		Infof("No latencies to log")
		return
	}
	for i := 0; i < len(latencies); i++ {
		if latencies[i] > 50 {
			Infof("Latency at %d readFile operation: %d microseconds", i, latencies[i])
		}
	}
}
