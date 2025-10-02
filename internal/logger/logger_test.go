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
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	textLogPattern = `^time="[a-zA-Z0-9/:. ]{26}" severity=%s message="TestLogs: %s" mount_id=[a-zA-Z0-9-]+\s*$`
	jsonLogPattern = `^{\"timestamp\":{\"seconds\":\d{10},\"nanos\":\d{0,9}},\"severity\":\"%s\",\"message\":\"TestLogs: %s\",\"mount_id\":\"[a-zA-Z0-9-]+\"}\s*$`
)

type LoggerTest struct {
	suite.Suite
}

func expectedLogRegex(format, severity, message string) string {
	if format == "text" {
		return fmt.Sprintf(textLogPattern, severity, message)
	}
	if format == "json" {
		return fmt.Sprintf(jsonLogPattern, severity, message)
	}
	return ""
}

func TestLoggerSuite(t *testing.T) {
	suite.Run(t, new(LoggerTest))
}

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

func redirectLogsToGivenBuffer(buf *bytes.Buffer, level string) {
	var programLevel = new(slog.LevelVar)
	handler := defaultLoggerFactory.createJsonOrTextHandler(buf, programLevel, "TestLogs: ")
	handler = handler.WithAttrs([]slog.Attr{slog.String(mountLoggerIDKey,
		fmt.Sprintf("%s-%s", MountFSName(), MountInstanceID()))})
	defaultLogger = slog.New(handler)
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
	for i := range output {
		if expected[i] == "" {
			assert.Equal(t, expected[i], output[i])
		} else {
			assert.Regexp(t, expected[i], output[i])
		}
	}
}

func validateLogOutputAtSpecifiedFormatAndSeverity(t *testing.T, format string, level string, expectedOutput []string) {
	// set log format
	defaultLoggerFactory.format = format

	output := fetchLogOutputForSpecifiedSeverityLevel(level, getTestLoggingFunctions())
	validateOutput(t, expectedOutput, output)
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////
func (t *LoggerTest) TestTextFormatLogs_LogLevelOFF() {
	var expected = []string{
		"", "", "", "", "",
	}

	// Assert that nothing is logged when log level is OFF.
	validateLogOutputAtSpecifiedFormatAndSeverity(t.T(), "json", cfg.OFF, expected)
}

func (t *LoggerTest) TestTextFormatLogs_LogLevelERROR() {
	var expected = []string{
		"", "", "", "", expectedLogRegex("text", "ERROR", "www.errorExample.com"),
	}

	// Assert only error logs are logged when log level is ERROR.
	validateLogOutputAtSpecifiedFormatAndSeverity(t.T(), "text", cfg.ERROR, expected)
}

func (t *LoggerTest) TestTextFormatLogs_LogLevelWARNING() {
	var expected = []string{
		"", "", "", expectedLogRegex("text", "WARNING", "www.warningExample.com"), expectedLogRegex("text", "ERROR", "www.errorExample.com"),
	}

	// Assert warning and error logs are logged when log level is WARNING.
	validateLogOutputAtSpecifiedFormatAndSeverity(t.T(), "text", cfg.WARNING, expected)
}

func (t *LoggerTest) TestTextFormatLogs_LogLevelINFO() {
	var expected = []string{
		"", "", expectedLogRegex("text", "INFO", "www.infoExample.com"), expectedLogRegex("text", "WARNING", "www.warningExample.com"), expectedLogRegex("text", "ERROR", "www.errorExample.com"),
	}

	// Assert info, warning & error logs are logged when log level is INFO.
	validateLogOutputAtSpecifiedFormatAndSeverity(t.T(), "text", cfg.INFO, expected)
}

func (t *LoggerTest) TestTextFormatLogs_LogLevelDEBUG() {
	var expected = []string{
		"", expectedLogRegex("text", "DEBUG", "www.debugExample.com"), expectedLogRegex("text", "INFO", "www.infoExample.com"), expectedLogRegex("text", "WARNING", "www.warningExample.com"), expectedLogRegex("text", "ERROR", "www.errorExample.com"),
	}

	// Assert debug, info, warning & error logs are logged when log level is DEBUG.
	validateLogOutputAtSpecifiedFormatAndSeverity(t.T(), "text", cfg.DEBUG, expected)
}

func (t *LoggerTest) TestTextFormatLogs_LogLevelTRACE() {
	var expected = []string{
		expectedLogRegex("text", "TRACE", "www.traceExample.com"), expectedLogRegex("text", "DEBUG", "www.debugExample.com"), expectedLogRegex("text", "INFO", "www.infoExample.com"), expectedLogRegex("text", "WARNING", "www.warningExample.com"), expectedLogRegex("text", "ERROR", "www.errorExample.com"),
	}

	// Assert all logs are logged when log level is TRACE.
	validateLogOutputAtSpecifiedFormatAndSeverity(t.T(), "text", cfg.TRACE, expected)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelOFF() {
	var expected = []string{
		"", "", "", "", "",
	}

	// Assert that nothing is logged when log level is OFF.
	validateLogOutputAtSpecifiedFormatAndSeverity(t.T(), "json", cfg.OFF, expected)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelERROR() {
	var expected = []string{
		"", "", "", "", expectedLogRegex("json", "ERROR", "www.errorExample.com"),
	}

	// Assert only error logs are logged when log level is ERROR.
	validateLogOutputAtSpecifiedFormatAndSeverity(t.T(), "json", cfg.ERROR, expected)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelWARNING() {
	var expected = []string{
		"", "", "", expectedLogRegex("json", "WARNING", "www.warningExample.com"), expectedLogRegex("json", "ERROR", "www.errorExample.com"),
	}

	// Assert warning and error logs are logged when log level is WARNING.
	validateLogOutputAtSpecifiedFormatAndSeverity(t.T(), "json", cfg.WARNING, expected)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelINFO() {
	var expected = []string{
		"", "", expectedLogRegex("json", "INFO", "www.infoExample.com"), expectedLogRegex("json", "WARNING", "www.warningExample.com"), expectedLogRegex("json", "ERROR", "www.errorExample.com"),
	}

	// Assert info, warning & error logs are logged when log level is INFO.
	validateLogOutputAtSpecifiedFormatAndSeverity(t.T(), "json", cfg.INFO, expected)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelDEBUG() {
	var expected = []string{
		"", expectedLogRegex("json", "DEBUG", "www.debugExample.com"), expectedLogRegex("json", "INFO", "www.infoExample.com"), expectedLogRegex("json", "WARNING", "www.warningExample.com"), expectedLogRegex("json", "ERROR", "www.errorExample.com"),
	}

	// Assert debug, info, warning & error logs are logged when log level is DEBUG.
	validateLogOutputAtSpecifiedFormatAndSeverity(t.T(), "json", cfg.DEBUG, expected)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelTRACE() {
	var expected = []string{
		expectedLogRegex("json", "TRACE", "www.traceExample.com"), expectedLogRegex("json", "DEBUG", "www.debugExample.com"), expectedLogRegex("json", "INFO", "www.infoExample.com"), expectedLogRegex("json", "WARNING", "www.warningExample.com"), expectedLogRegex("json", "ERROR", "www.errorExample.com"),
	}

	// Assert all logs are logged when log level is TRACE.
	validateLogOutputAtSpecifiedFormatAndSeverity(t.T(), "json", cfg.TRACE, expected)
}

func (t *LoggerTest) TestSetLoggingLevel() {
	testData := []struct {
		inputLevel           string
		programLevel         *slog.LevelVar
		expectedProgramLevel slog.Level
	}{
		{
			cfg.TRACE,
			new(slog.LevelVar),
			LevelTrace,
		},
		{
			cfg.DEBUG,
			new(slog.LevelVar),
			LevelDebug,
		},
		{
			cfg.WARNING,
			new(slog.LevelVar),
			LevelWarn,
		},
		{
			cfg.ERROR,
			new(slog.LevelVar),
			LevelError,
		},
		{
			cfg.OFF,
			new(slog.LevelVar),
			LevelOff,
		},
	}

	for _, test := range testData {
		setLoggingLevel(test.inputLevel, test.programLevel)
		assert.Equal(t.T(), test.programLevel.Level(), test.expectedProgramLevel)
	}
}

func (t *LoggerTest) TestInitLogFile() {
	format := "text"
	filePath, _ := os.UserHomeDir()
	filePath += "/log.txt"
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

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), filePath, defaultLoggerFactory.file.Name())
	assert.Nil(t.T(), defaultLoggerFactory.sysWriter)
	assert.Equal(t.T(), format, defaultLoggerFactory.format)
	assert.Equal(t.T(), cfg.DEBUG, defaultLoggerFactory.level)
	assert.Equal(t.T(), fileSize, defaultLoggerFactory.logRotate.MaxFileSizeMb)
	assert.Equal(t.T(), backupFileCount, defaultLoggerFactory.logRotate.BackupFileCount)
	assert.True(t.T(), defaultLoggerFactory.logRotate.Compress)
}

func (t *LoggerTest) TestSetLogFormatAndFSName() {
	logConfig := cfg.DefaultLoggingConfig()
	defaultLoggerFactory = &loggerFactory{
		file:      nil,
		level:     string(logConfig.Severity), // setting log level to INFO by default
		logRotate: logConfig.LogRotate,
	}

	testData := []struct {
		format         string
		fsName         string
		expectedOutput string
	}{
		{
			"text",
			"gcsfuse",
			expectedLogRegex("text", "INFO", "www.infoExample.com"),
		},
		{
			"json",
			"my-bucket",
			expectedLogRegex("json", "INFO", "www.infoExample.com"),
		},
		{
			"",
			"my-other-bucket",
			expectedLogRegex("json", "INFO", "www.infoExample.com"),
		},
	}

	for _, test := range testData {
		SetLogFormatAndFsName(test.format, test.fsName)

		assert.NotNil(t.T(), defaultLoggerFactory)
		assert.NotNil(t.T(), defaultLogger)
		assert.Equal(t.T(), defaultLoggerFactory.format, test.format)
		// Create a logger using defaultLoggerFactory that writes to buffer.
		var buf bytes.Buffer
		redirectLogsToGivenBuffer(&buf, defaultLoggerFactory.level)
		Infof("www.infoExample.com")
		output := buf.String()
		// Compare expected and actual log.
		assert.Regexp(t.T(), test.expectedOutput, output)
		assert.True(t.T(), strings.Contains(output, test.fsName))
	}
}

func (t *LoggerTest) TestSetLogFormatAndFsNameWithBackgroundMode() {
	logConfig := cfg.DefaultLoggingConfig()
	defaultLoggerFactory = &loggerFactory{
		file:      nil,
		level:     string(logConfig.Severity),
		logRotate: logConfig.LogRotate,
	}

	testData := []struct {
		name               string
		instanceIDSet      bool
		expectedInstanceID string
		fsName             string
		format             string
		expectedOutput     string
	}{
		{
			"MountInstanceIDSet",
			true,
			"12121212",
			"my-bucket",
			"json",
			expectedLogRegex("json", "INFO", "www.infoExample.com"),
		},
		{
			"MountInstanceIDNotSet",
			false,
			defaultMountInstanceID,
			"gcsfuse",
			"text",
			expectedLogRegex("text", "INFO", "www.infoExample.com"),
		},
	}

	for _, test := range testData {
		t.T().Run(test.name, func(t *testing.T) {
			t.Setenv(GCSFuseInBackgroundMode, "true")
			if test.instanceIDSet {
				t.Setenv(GCSFuseMountInstanceIDEnvKey, test.expectedInstanceID)
			}
			setupMountInstanceID()
			initializeDefaultLogger()
			SetLogFormatAndFsName(test.format, test.fsName)
			assert.NotNil(t, defaultLoggerFactory)
			assert.NotNil(t, defaultLogger)

			// Create a logger using defaultLoggerFactory that writes to buffer.
			var buf bytes.Buffer
			redirectLogsToGivenBuffer(&buf, defaultLoggerFactory.level)
			Info("www.infoExample.com")
			output := buf.String()

			// Compare expected and actual log.
			assert.Regexp(t, test.expectedOutput, output)
			assert.True(t, strings.Contains(output, fmt.Sprintf("%s-%s", test.fsName, test.expectedInstanceID)))
		})
	}
}

func (t *LoggerTest) TestGenerateMountInstanceID_Success() {
	id := generateMountInstanceID()

	assert.Len(t.T(), id, mountInstanceIDLength)
	assert.NotEqual(t.T(), defaultMountInstanceID, id)
	assert.Regexp(t.T(), fmt.Sprintf("^[0-9a-f]{%d}$", mountInstanceIDLength), id)
}

func (t *LoggerTest) TestGenerateMountInstanceID_Failure() {
	originalNewRandom := newRandomUUID
	defer func() { newRandomUUID = originalNewRandom }()

	newRandomUUID = func() (uuid.UUID, error) {
		return uuid.UUID{}, errors.New("uuid generation error")
	}

	id := generateMountInstanceID()

	assert.Equal(t.T(), defaultMountInstanceID, id)
}

func (t *LoggerTest) TestSetupMountInstanceID() {
	testCases := []struct {
		name               string
		inBackgroundMode   bool
		mountInstanceIDEnv string
		expectDefaultID    bool
		expectedID         string
	}{
		{
			name:             "ForegroundMode",
			inBackgroundMode: false,
			expectDefaultID:  false,
		},
		{
			name:               "BackgroundModeWithInstanceID",
			inBackgroundMode:   true,
			mountInstanceIDEnv: "12345678",
			expectDefaultID:    false,
			expectedID:         "12345678",
		},
		{
			name:             "BackgroundModeWithoutInstanceID",
			inBackgroundMode: true,
			expectDefaultID:  true,
		},
		{
			name:               "BackgroundModeWithEmptyInstanceID",
			inBackgroundMode:   true,
			mountInstanceIDEnv: "",
			expectDefaultID:    true,
		},
	}

	for _, tc := range testCases {
		t.T().Run(tc.name, func(t *testing.T) {
			if tc.inBackgroundMode {
				t.Setenv(GCSFuseInBackgroundMode, "true")
				t.Setenv(GCSFuseMountInstanceIDEnvKey, tc.mountInstanceIDEnv)
			}

			setupMountInstanceID()

			if tc.expectDefaultID {
				assert.Equal(t, defaultMountInstanceID, mountInstanceID)
			} else if tc.expectedID != "" {
				assert.Equal(t, tc.expectedID, mountInstanceID)
			} else {
				assert.NotEqual(t, defaultMountInstanceID, mountInstanceID)
				assert.Len(t, mountInstanceID, mountInstanceIDLength)
				assert.Regexp(t, fmt.Sprintf("^[0-9a-f]{%d}$", mountInstanceIDLength), mountInstanceID)
			}
		})
	}
}
