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
)

const (
	textLogPattern = `^time="[a-zA-Z0-9/:. ]{26}" severity=%s message="TestLogs: %s"\s*$`
	jsonLogPattern = `^{\"timestamp\":{\"seconds\":\d{10},\"nanos\":\d{0,9}},\"severity\":\"%s\",\"message\":\"TestLogs: %s\"}\s*$`
)

// //////////////////////////////////////////////////////////////////////
// Helper
// //////////////////////////////////////////////////////////////////////

func expectedLogRegex(format, severity, message string) string {
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
	var programLevel = new(slog.LevelVar)
	defaultLogger = slog.New(
		defaultLoggerFactory.createJsonOrTextHandler(buf, programLevel, "TestLogs: "),
	)
	setLoggingLevel(level, programLevel)
}

// fetchLogOutputForSpecifiedSeverityLevel takes configured severity and
// functions that write logs as parameter and returns string array containing
// output from each function call.
func fetchLogOutputForSpecifiedSeverityLevel(level string, functions []func()) []string {
	// create a logger that writes to buffer at configured level.
	var buf bytes.Buffer
	redirectLogsToGivenBuffer(&buf, level)

	var output []string
	// run the functions provided.
	for _, f := range functions {
		f()
		output = append(output, buf.String())
		buf.Reset()
	}
	return output
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

func validateOutput(t *testing.T, expected []string, output []string) {
	t.Helper()
	for i := range output {
		if expected[i] == "" {
			assert.Equal(t, expected[i], output[i])
		} else {
			assert.Regexp(t, expected[i], output[i])
		}
	}
}

func validateLogOutputAtSpecifiedFormatAndSeverity(t *testing.T, format string, level string, expectedOutput []string) {
	t.Helper()
	// set log format
	defaultLoggerFactory.format = format

	output := fetchLogOutputForSpecifiedSeverityLevel(level, getTestLoggingFunctions())

	validateOutput(t, expectedOutput, output)
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////
func TestTextFormatLogs_LogLevelOFF(t *testing.T) {
	var expected = []string{
		"", "", "", "", "",
	}

	// Assert that nothing is logged when log level is OFF.
	validateLogOutputAtSpecifiedFormatAndSeverity(t, "json", cfg.OFF, expected)
}

func TestTextFormatLogs_LogLevelERROR(t *testing.T) {
	var expected = []string{
		"", "", "", "", expectedLogRegex("text", "ERROR", "www.errorExample.com"),
	}

	// Assert only error logs are logged when log level is ERROR.
	validateLogOutputAtSpecifiedFormatAndSeverity(t, "text", cfg.ERROR, expected)
}

func TestTextFormatLogs_LogLevelWARNING(t *testing.T) {
	expected := []string{
		"", // TRACE
		"", // DEBUG
		"", // INFO
		expectedLogRegex("text", "WARNING", "www.warningExample.com"),
		expectedLogRegex("text", "ERROR", "www.errorExample.com"),
	}

	// Assert warning and error logs are logged when log level is WARNING.
	validateLogOutputAtSpecifiedFormatAndSeverity(t, "text", cfg.WARNING, expected)
}

func TestTextFormatLogs_LogLevelINFO(t *testing.T) {
	expected := []string{
		"", // TRACE
		"", // DEBUG
		expectedLogRegex("text", "INFO", "www.infoExample.com"),
		expectedLogRegex("text", "WARNING", "www.warningExample.com"),
		expectedLogRegex("text", "ERROR", "www.errorExample.com"),
	}

	// Assert info, warning & error logs are logged when log level is INFO.
	validateLogOutputAtSpecifiedFormatAndSeverity(t, "text", cfg.INFO, expected)
}

func TestTextFormatLogs_LogLevelDEBUG(t *testing.T) {
	expected := []string{
		"", // TRACE
		expectedLogRegex("text", "DEBUG", "www.debugExample.com"),
		expectedLogRegex("text", "INFO", "www.infoExample.com"),
		expectedLogRegex("text", "WARNING", "www.warningExample.com"),
		expectedLogRegex("text", "ERROR", "www.errorExample.com"),
	}

	// Assert debug, info, warning & error logs are logged when log level is DEBUG.
	validateLogOutputAtSpecifiedFormatAndSeverity(t, "text", cfg.DEBUG, expected)
}

func TestTextFormatLogs_LogLevelTRACE(t *testing.T) {
	expected := []string{
		expectedLogRegex("text", "TRACE", "www.traceExample.com"),
		expectedLogRegex("text", "DEBUG", "www.debugExample.com"),
		expectedLogRegex("text", "INFO", "www.infoExample.com"),
		expectedLogRegex("text", "WARNING", "www.warningExample.com"),
		expectedLogRegex("text", "ERROR", "www.errorExample.com"),
	}

	// Assert all logs are logged when log level is TRACE.
	validateLogOutputAtSpecifiedFormatAndSeverity(t, "text", cfg.TRACE, expected)
}

func TestJSONFormatLogs_LogLevelOFF(t *testing.T) {
	expected := []string{
		"", // TRACE
		"", // DEBUG
		"", // INFO
		"", // WARNING
		"", // ERROR
	}

	// Assert that nothing is logged when log level is OFF.
	validateLogOutputAtSpecifiedFormatAndSeverity(t, "json", cfg.OFF, expected)
}

func TestJSONFormatLogs_LogLevelERROR(t *testing.T) {
	expected := []string{
		"", // TRACE
		"", // DEBUG
		"", // INFO
		"", // WARNING
		expectedLogRegex("json", "ERROR", "www.errorExample.com"),
	}

	// Assert only error logs are logged when log level is ERROR.
	validateLogOutputAtSpecifiedFormatAndSeverity(t, "json", cfg.ERROR, expected)
}

func TestJSONFormatLogs_LogLevelWARNING(t *testing.T) {
	expected := []string{
		"", // TRACE
		"", // DEBUG
		"", // INFO
		expectedLogRegex("json", "WARNING", "www.warningExample.com"),
		expectedLogRegex("json", "ERROR", "www.errorExample.com"),
	}

	// Assert warning and error logs are logged when log level is WARNING.
	validateLogOutputAtSpecifiedFormatAndSeverity(t, "json", cfg.WARNING, expected)
}

func TestJSONFormatLogs_LogLevelINFO(t *testing.T) {
	expected := []string{
		"", // TRACE
		"", // DEBUG
		expectedLogRegex("json", "INFO", "www.infoExample.com"),
		expectedLogRegex("json", "WARNING", "www.warningExample.com"),
		expectedLogRegex("json", "ERROR", "www.errorExample.com"),
	}

	// Assert info, warning & error logs are logged when log level is INFO.
	validateLogOutputAtSpecifiedFormatAndSeverity(t, "json", cfg.INFO, expected)
}

func TestJSONFormatLogs_LogLevelDEBUG(t *testing.T) {
	expected := []string{
		"", // TRACE
		expectedLogRegex("json", "DEBUG", "www.debugExample.com"),
		expectedLogRegex("json", "INFO", "www.infoExample.com"),
		expectedLogRegex("json", "WARNING", "www.warningExample.com"),
		expectedLogRegex("json", "ERROR", "www.errorExample.com"),
	}

	// Assert debug, info, warning & error logs are logged when log level is DEBUG.
	validateLogOutputAtSpecifiedFormatAndSeverity(t, "json", cfg.DEBUG, expected)
}

func TestJSONFormatLogs_LogLevelTRACE(t *testing.T) {
	expected := []string{
		expectedLogRegex("json", "TRACE", "www.traceExample.com"),
		expectedLogRegex("json", "DEBUG", "www.debugExample.com"),
		expectedLogRegex("json", "INFO", "www.infoExample.com"),
		expectedLogRegex("json", "WARNING", "www.warningExample.com"),
		expectedLogRegex("json", "ERROR", "www.errorExample.com"),
	}

	// Assert all logs are logged when log level is TRACE.
	validateLogOutputAtSpecifiedFormatAndSeverity(t, "json", cfg.TRACE, expected)
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
			programLevel := new(slog.LevelVar)

			setLoggingLevel(tc.inputLevel, programLevel)

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

	err := InitLogFile(newLogConfig)
	t.Cleanup(func() {
		defaultLoggerFactory.file.Close() // Close file handle to release resources.
	})

	assert.NoError(t, err)
	assert.Equal(t, filePath, defaultLoggerFactory.file.Name())
	assert.Nil(t, defaultLoggerFactory.sysWriter)
	assert.Equal(t, format, defaultLoggerFactory.format)
	assert.Equal(t, cfg.DEBUG, defaultLoggerFactory.level)
	assert.Equal(t, fileSize, defaultLoggerFactory.logRotate.MaxFileSizeMb)
	assert.Equal(t, backupFileCount, defaultLoggerFactory.logRotate.BackupFileCount)
	assert.True(t, defaultLoggerFactory.logRotate.Compress)
}

func TestSetLogFormat(t *testing.T) {
	testCases := []struct {
		name           string
		format         string
		expectedOutput string
	}{
		{
			name:           "TextFormat",
			format:         "text",
			expectedOutput: expectedLogRegex("text", "INFO", "www.infoExample.com"),
		},
		{
			name:           "JsonFormat",
			format:         "json",
			expectedOutput: expectedLogRegex("json", "INFO", "www.infoExample.com"),
		},
		{
			name:           "EmptyFormatDefaultsToJson",
			format:         "",
			expectedOutput: expectedLogRegex("json", "INFO", "www.infoExample.com"),
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

			SetLogFormat(tc.format)

			var buf bytes.Buffer
			redirectLogsToGivenBuffer(&buf, defaultLoggerFactory.level)
			Infof("www.infoExample.com")

			assert.Regexp(t, tc.expectedOutput, buf.String())
		})
	}
}

func TestGenerateMountInstanceID_Success(t *testing.T) {
	testCases := []struct {
		name                         string
		size                         int
		expectedMountInstanceIDRegex string
	}{
		{
			name:                         "TenChars",
			size:                         10,
			expectedMountInstanceIDRegex: "^[0-9a-f]{10}$",
		},
		{
			name:                         "ThirtyTwoChars",
			size:                         32,
			expectedMountInstanceIDRegex: "^[0-9a-f]{32}$",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mountInstanceID, err := generateMountInstanceID(tc.size)

			assert.NoError(t, err)
			assert.Regexp(t, tc.expectedMountInstanceIDRegex, mountInstanceID)
		})
	}
}

func TestGenerateMountInstanceID_FailureDueToUUIDSize(t *testing.T) {
	mountInstanceID, err := generateMountInstanceID(999)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "UUID is smaller than requested size")
	assert.Equal(t, "", mountInstanceID)
}

func TestGenerateMountInstanceID_FailureDueToNegativeSize(t *testing.T) {
	mountInstanceID, err := generateMountInstanceID(0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MountInstanceID must be positive")
	assert.Equal(t, "", mountInstanceID)
}

func TestSetupMountInstanceID_Success(t *testing.T) {
	testCases := []struct {
		name                         string
		inBackgroundMode             bool
		mountInstanceIDEnv           string
		expectedID                   string
		expectedMountInstanceIDRegex string
	}{
		{
			name:                         "ForegroundMode",
			inBackgroundMode:             false,
			expectedID:                   "",
			expectedMountInstanceIDRegex: "^[0-9a-f]{8}$", // default size for MountInstanceID is 8.
		},
		{
			name:               "BackgroundModeWithInstanceID",
			inBackgroundMode:   true,
			mountInstanceIDEnv: "12345678",
			expectedID:         "12345678",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() {
				// Initialize package level variables for each subtest to ensure isolation.
				mountInstanceID = ""
				setupMountInstanceIDOnce = sync.Once{}
			})
			if tc.inBackgroundMode {
				t.Setenv(GCSFuseInBackgroundMode, "true")
				t.Setenv(GCSFuseMountInstanceIDEnvKey, tc.mountInstanceIDEnv)
			}

			setupMountInstanceID()

			if tc.inBackgroundMode {
				assert.Equal(t, tc.expectedID, mountInstanceID)
			} else {
				assert.Len(t, mountInstanceID, mountInstanceIDLength)
				assert.Regexp(t, tc.expectedMountInstanceIDRegex, mountInstanceID)
			}
		})
	}
}
