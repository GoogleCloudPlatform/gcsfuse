// Copyright 2023 Google LLC
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
	"bytes"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testFsName     = "testFS" // This is used in redirectLogsToGivenBuffer to construct the mount instance ID.
	textLogPattern = `^time="[a-zA-Z0-9/:. ]{26}" severity=%s message="TestLogs: %s" mount-id=testFS-[0-9a-f]{8}\s*$`
	jsonLogPattern = `^{"timestamp":{"seconds":\d{10},"nanos":\d{0,9}},"severity":"%s","message":"TestLogs: %s","mount-id":"testFS-[0-9a-f]{8}"}\s*$`
)

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func expectedLogRegex(t *testing.T, format, severity, message string) string {
	t.Helper()
	switch format {
	case "text":
		return fmt.Sprintf(textLogPattern, severity, message)
	case "json":
		return fmt.Sprintf(jsonLogPattern, severity, message)
	default:
		return ""
	}
}

func redirectLogsToGivenBuffer(buf *bytes.Buffer, level string) {
	handler := defaultLoggerFactory.createJsonOrTextHandler(buf, programLevel, "TestLogs: ")
	handler = handler.WithAttrs(loggerAttr(testFsName))
	defaultLogger = slog.New(handler)
	setLoggingLevel(level)
}

func getTestLoggingFunctions() []func() {
	return []func(){
		func() {
			Tracef("www.traceExample.com")
		},
		func() {
			Debugf("www.debugExample.com")
		},
		func() {
			Infof("www.infoExample.com")
		},
		func() {
			Warnf("www.warningExample.com")
		},
		func() {
			Errorf("www.errorExample.com")
		},
	}
}

// fetchAllLogLevelOutputsForSpecifiedSeverityLevel sets the log format and severity,
// executes standard logging functions, and returns their output.
func fetchAllLogLevelOutputsForSpecifiedSeverityLevel(t *testing.T, format, level string) []string {
	t.Helper()
	// set log format
	defaultLoggerFactory.format = format

	// create a logger that writes to buffer at configured level.
	var buf bytes.Buffer
	redirectLogsToGivenBuffer(&buf, level)

	var actualLogLines []string
	// run the functions provided.
	for _, f := range getTestLoggingFunctions() {
		f()
		actualLogLines = append(actualLogLines, buf.String())
		buf.Reset()
	}
	return actualLogLines
}

// validateLogOutputs compares the captured log line output with the expected log regex patterns.
func validateLogOutputs(t *testing.T, expectedLogLineRegexes, actualLogLines []string) {
	t.Helper()
	require.Equal(t, len(expectedLogLineRegexes), len(actualLogLines))
	for i := range actualLogLines {
		assert.Regexp(t, expectedLogLineRegexes[i], actualLogLines[i])
	}
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////
func TestTextFormatLogs_LogLevelOFF(t *testing.T) {
	var expectedLogLineRegexes = []string{
		"", // TRACE
		"", // DEBUG
		"", // INFO
		"", // WARNING
		"", // ERROR
	}

	actualLogLines := fetchAllLogLevelOutputsForSpecifiedSeverityLevel(t, "text", cfg.OFF)

	validateLogOutputs(t, expectedLogLineRegexes, actualLogLines)
}

func TestTextFormatLogs_LogLevelERROR(t *testing.T) {
	var expectedLogLineRegexes = []string{
		"", // TRACE
		"", // DEBUG
		"", // INFO
		"", // WARNING
		expectedLogRegex(t, "text", "ERROR", "www.errorExample.com"),
	}

	actualLogLines := fetchAllLogLevelOutputsForSpecifiedSeverityLevel(t, "text", cfg.ERROR)

	validateLogOutputs(t, expectedLogLineRegexes, actualLogLines)
}

func TestTextFormatLogs_LogLevelWARNING(t *testing.T) {
	expectedLogLineRegexes := []string{
		"", // TRACE
		"", // DEBUG
		"", // INFO
		expectedLogRegex(t, "text", "WARNING", "www.warningExample.com"),
		expectedLogRegex(t, "text", "ERROR", "www.errorExample.com"),
	}

	actualLogLines := fetchAllLogLevelOutputsForSpecifiedSeverityLevel(t, "text", cfg.WARNING)

	validateLogOutputs(t, expectedLogLineRegexes, actualLogLines)
}

func TestTextFormatLogs_LogLevelINFO(t *testing.T) {
	expectedLogLineRegexes := []string{
		"", // TRACE
		"", // DEBUG
		expectedLogRegex(t, "text", "INFO", "www.infoExample.com"),
		expectedLogRegex(t, "text", "WARNING", "www.warningExample.com"),
		expectedLogRegex(t, "text", "ERROR", "www.errorExample.com"),
	}

	actualLogLines := fetchAllLogLevelOutputsForSpecifiedSeverityLevel(t, "text", cfg.INFO)

	validateLogOutputs(t, expectedLogLineRegexes, actualLogLines)
}

func TestTextFormatLogs_LogLevelDEBUG(t *testing.T) {
	expectedLogLineRegexes := []string{
		"", // TRACE
		expectedLogRegex(t, "text", "DEBUG", "www.debugExample.com"),
		expectedLogRegex(t, "text", "INFO", "www.infoExample.com"),
		expectedLogRegex(t, "text", "WARNING", "www.warningExample.com"),
		expectedLogRegex(t, "text", "ERROR", "www.errorExample.com"),
	}

	actualLogLines := fetchAllLogLevelOutputsForSpecifiedSeverityLevel(t, "text", cfg.DEBUG)

	validateLogOutputs(t, expectedLogLineRegexes, actualLogLines)
}

func TestTextFormatLogs_LogLevelTRACE(t *testing.T) {
	expectedLogLineRegexes := []string{
		expectedLogRegex(t, "text", "TRACE", "www.traceExample.com"),
		expectedLogRegex(t, "text", "DEBUG", "www.debugExample.com"),
		expectedLogRegex(t, "text", "INFO", "www.infoExample.com"),
		expectedLogRegex(t, "text", "WARNING", "www.warningExample.com"),
		expectedLogRegex(t, "text", "ERROR", "www.errorExample.com"),
	}

	actualLogLines := fetchAllLogLevelOutputsForSpecifiedSeverityLevel(t, "text", cfg.TRACE)

	validateLogOutputs(t, expectedLogLineRegexes, actualLogLines)
}

func TestJSONFormatLogs_LogLevelOFF(t *testing.T) {
	expectedLogLineRegexes := []string{
		"", // TRACE
		"", // DEBUG
		"", // INFO
		"", // WARNING
		"", // ERROR
	}

	actualLogLines := fetchAllLogLevelOutputsForSpecifiedSeverityLevel(t, "json", cfg.OFF)

	validateLogOutputs(t, expectedLogLineRegexes, actualLogLines)
}

func TestJSONFormatLogs_LogLevelERROR(t *testing.T) {
	expectedLogLineRegexes := []string{
		"", // TRACE
		"", // DEBUG
		"", // INFO
		"", // WARNING
		expectedLogRegex(t, "json", "ERROR", "www.errorExample.com"),
	}

	actualLogLines := fetchAllLogLevelOutputsForSpecifiedSeverityLevel(t, "json", cfg.ERROR)

	validateLogOutputs(t, expectedLogLineRegexes, actualLogLines)
}

func TestJSONFormatLogs_LogLevelWARNING(t *testing.T) {
	expectedLogLineRegexes := []string{
		"", // TRACE
		"", // DEBUG
		"", // INFO
		expectedLogRegex(t, "json", "WARNING", "www.warningExample.com"),
		expectedLogRegex(t, "json", "ERROR", "www.errorExample.com"),
	}

	actualLogLines := fetchAllLogLevelOutputsForSpecifiedSeverityLevel(t, "json", cfg.WARNING)

	validateLogOutputs(t, expectedLogLineRegexes, actualLogLines)
}

func TestJSONFormatLogs_LogLevelINFO(t *testing.T) {
	expectedLogLineRegexes := []string{
		"", // TRACE
		"", // DEBUG
		expectedLogRegex(t, "json", "INFO", "www.infoExample.com"),
		expectedLogRegex(t, "json", "WARNING", "www.warningExample.com"),
		expectedLogRegex(t, "json", "ERROR", "www.errorExample.com"),
	}

	actualLogLines := fetchAllLogLevelOutputsForSpecifiedSeverityLevel(t, "json", cfg.INFO)

	validateLogOutputs(t, expectedLogLineRegexes, actualLogLines)
}

func TestJSONFormatLogs_LogLevelDEBUG(t *testing.T) {
	expectedLogLineRegexes := []string{
		"", // TRACE
		expectedLogRegex(t, "json", "DEBUG", "www.debugExample.com"),
		expectedLogRegex(t, "json", "INFO", "www.infoExample.com"),
		expectedLogRegex(t, "json", "WARNING", "www.warningExample.com"),
		expectedLogRegex(t, "json", "ERROR", "www.errorExample.com"),
	}

	actualLogLines := fetchAllLogLevelOutputsForSpecifiedSeverityLevel(t, "json", cfg.DEBUG)

	validateLogOutputs(t, expectedLogLineRegexes, actualLogLines)
}

func TestJSONFormatLogs_LogLevelTRACE(t *testing.T) {
	expectedLogLineRegexes := []string{
		expectedLogRegex(t, "json", "TRACE", "www.traceExample.com"),
		expectedLogRegex(t, "json", "DEBUG", "www.debugExample.com"),
		expectedLogRegex(t, "json", "INFO", "www.infoExample.com"),
		expectedLogRegex(t, "json", "WARNING", "www.warningExample.com"),
		expectedLogRegex(t, "json", "ERROR", "www.errorExample.com"),
	}

	actualLogLines := fetchAllLogLevelOutputsForSpecifiedSeverityLevel(t, "json", cfg.TRACE)

	validateLogOutputs(t, expectedLogLineRegexes, actualLogLines)
}

func TestSetLoggingLevel(t *testing.T) {
	testCases := []struct {
		name                 string
		inputLevel           string
		expectedProgramLevel slog.Level
	}{
		{
			name:                 "TRACE",
			inputLevel:           cfg.TRACE,
			expectedProgramLevel: LevelTrace,
		},
		{
			name:                 "DEBUG",
			inputLevel:           cfg.DEBUG,
			expectedProgramLevel: LevelDebug,
		},
		{
			name:                 "WARNING",
			inputLevel:           cfg.WARNING,
			expectedProgramLevel: LevelWarn,
		},
		{
			name:                 "ERROR",
			inputLevel:           cfg.ERROR,
			expectedProgramLevel: LevelError,
		},
		{
			name:                 "OFF",
			inputLevel:           cfg.OFF,
			expectedProgramLevel: LevelOff,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setLoggingLevel(tc.inputLevel)

			assert.Equal(t, tc.expectedProgramLevel, programLevel.Level())
		})
	}
}

func TestInitLogFile(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "log.txt")
	format := "text"
	fileSize := int64(100)
	backupFileCount := int64(2)
	newLogConfig := cfg.LoggingConfig{
		FilePath: cfg.ResolvedPath(filePath),
		Severity: "DEBUG",
		Format:   format,
		LogRotate: cfg.LogRotateLoggingConfig{
			MaxFileSizeMb:   fileSize,
			BackupFileCount: backupFileCount,
			Compress:        true,
		},
	}

	err := InitLogFile(newLogConfig, testFsName)

	require.NoError(t, err)
	require.NotNil(t, defaultLoggerFactory.file)
	t.Cleanup(func() {
		defaultLoggerFactory.file.Close() // Close file handle to release resources.
	})
	assert.Equal(t, filePath, defaultLoggerFactory.file.Name())
	assert.Nil(t, defaultLoggerFactory.sysWriter)
	assert.Equal(t, format, defaultLoggerFactory.format)
	assert.Equal(t, cfg.DEBUG, defaultLoggerFactory.level)
	assert.Equal(t, fileSize, defaultLoggerFactory.logRotate.MaxFileSizeMb)
	assert.Equal(t, backupFileCount, defaultLoggerFactory.logRotate.BackupFileCount)
	assert.True(t, defaultLoggerFactory.logRotate.Compress)
}

func TestUpdateDefaultLogger(t *testing.T) {
	testCases := []struct {
		name          string
		format        string
		expectedRegex string
	}{
		{
			name:          "TextFormat",
			format:        "text",
			expectedRegex: expectedLogRegex(t, "text", "INFO", "www.infoExample.com"),
		},
		{
			name:          "JsonFormat",
			format:        "json",
			expectedRegex: expectedLogRegex(t, "json", "INFO", "www.infoExample.com"),
		},
		{
			name:          "EmptyFormatDefaultsToJson",
			format:        "",
			expectedRegex: expectedLogRegex(t, "json", "INFO", "www.infoExample.com"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Initialize logger factory for each subtest to ensure isolation.
			logConfig := cfg.DefaultLoggingConfig()
			defaultLoggerFactory = &loggerFactory{
				file:      nil,
				level:     string(logConfig.Severity), // setting log level to INFO by default
				logRotate: logConfig.LogRotate,
			}

			UpdateDefaultLogger(tc.format, testFsName)
			var buf bytes.Buffer
			redirectLogsToGivenBuffer(&buf, defaultLoggerFactory.level)
			Infof("www.infoExample.com")

			assert.Regexp(t, tc.expectedRegex, buf.String())
		})
	}
}

func TestGenerateMountUUID_Success(t *testing.T) {
	testCases := []struct {
		name                   string
		size                   int
		expectedMountUUIDRegex string
	}{
		{
			name:                   "TenChars",
			size:                   10,
			expectedMountUUIDRegex: "^[0-9a-f]{10}$",
		},
		{
			name:                   "ThirtyTwoChars",
			size:                   32,
			expectedMountUUIDRegex: "^[0-9a-f]{32}$",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mountUUID, err := generateMountUUID(tc.size)

			require.NoError(t, err)
			assert.Regexp(t, tc.expectedMountUUIDRegex, mountUUID)
		})
	}
}

func TestGenerateMountUUID_FailureDueToUUIDSize(t *testing.T) {
	mountUUID, err := generateMountUUID(999)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "UUID is smaller than requested size")
	assert.Equal(t, "", mountUUID)
}

func TestGenerateMountUUID_FailureDueToNegativeSize(t *testing.T) {
	mountUUID, err := generateMountUUID(0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "MountUUID must be positive")
	assert.Equal(t, "", mountUUID)
}

func TestSetupMountUUID_Success(t *testing.T) {
	testCases := []struct {
		name                   string
		inBackgroundMode       bool
		mountUUIDEnv           string
		expectedID             string
		expectedMountUUIDRegex string
	}{
		{
			name:                   "ForegroundMode",
			inBackgroundMode:       false,
			expectedID:             "",
			expectedMountUUIDRegex: "^[0-9a-f]{8}$", // default size for MountUUID is 8.
		},
		{
			name:             "BackgroundModeWithInstanceID",
			inBackgroundMode: true,
			mountUUIDEnv:     "12345678",
			expectedID:       "12345678",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() {
				// Initialize package level variables for each subtest to ensure isolation.
				mountUUID = ""
				setupMountUUIDOnce = sync.Once{}
			})
			if tc.inBackgroundMode {
				t.Setenv(GCSFuseInBackgroundMode, "true")
				t.Setenv(MountUUIDEnvKey, tc.mountUUIDEnv)
			}

			setupMountUUID()

			if tc.inBackgroundMode {
				assert.Equal(t, tc.expectedID, mountUUID)
			} else {
				assert.Len(t, mountUUID, mountUUIDLength)
				assert.Regexp(t, tc.expectedMountUUIDRegex, mountUUID)
			}
		})
	}
}
