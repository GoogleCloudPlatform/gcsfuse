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
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	textTraceString   = "^time=\"[a-zA-Z0-9/:. ]{26}\" severity=TRACE message=\"TestLogs: www.traceExample.com\" mount_id=[a-zA-Z0-9-]+\\s*$"
	textDebugString   = "^time=\"[a-zA-Z0-9/:. ]{26}\" severity=DEBUG message=\"TestLogs: www.debugExample.com\" mount_id=[a-zA-Z0-9-]+\\s*$"
	textInfoString    = "^time=\"[a-zA-Z0-9/:. ]{26}\" severity=INFO message=\"TestLogs: www.infoExample.com\" mount_id=[a-zA-Z0-9-]+\\s*$"
	textWarningString = "^time=\"[a-zA-Z0-9/:. ]{26}\" severity=WARNING message=\"TestLogs: www.warningExample.com\" mount_id=[a-zA-Z0-9-]+\\s*$"
	textErrorString   = "^time=\"[a-zA-Z0-9/:. ]{26}\" severity=ERROR message=\"TestLogs: www.errorExample.com\" mount_id=[a-zA-Z0-9-]+\\s*$"

	jsonTraceString   = "^{\"timestamp\":{\"seconds\":\\d{10},\"nanos\":\\d{0,9}},\"severity\":\"TRACE\",\"message\":\"TestLogs: www.traceExample.com\",\"mount_id\":\"[a-zA-Z0-9-]+\"}\\s*$"
	jsonDebugString   = "^{\"timestamp\":{\"seconds\":\\d{10},\"nanos\":\\d{0,9}},\"severity\":\"DEBUG\",\"message\":\"TestLogs: www.debugExample.com\",\"mount_id\":\"[a-zA-Z0-9-]+\"}\\s*$"
	jsonInfoString    = "^{\"timestamp\":{\"seconds\":\\d{10},\"nanos\":\\d{0,9}},\"severity\":\"INFO\",\"message\":\"TestLogs: www.infoExample.com\",\"mount_id\":\"[a-zA-Z0-9-]+\"}\\s*$"
	jsonWarningString = "^{\"timestamp\":{\"seconds\":\\d{10},\"nanos\":\\d{0,9}},\"severity\":\"WARNING\",\"message\":\"TestLogs: www.warningExample.com\",\"mount_id\":\"[a-zA-Z0-9-]+\"}\\s*$"
	jsonErrorString   = "^{\"timestamp\":{\"seconds\":\\d{10},\"nanos\":\\d{0,9}},\"severity\":\"ERROR\",\"message\":\"TestLogs: www.errorExample.com\",\"mount_id\":\"[a-zA-Z0-9-]+\"}\\s*$"
)

type LoggerTest struct {
	suite.Suite
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
			expectedRegexp := regexp.MustCompile(expected[i])
			assert.True(t, expectedRegexp.MatchString(output[i]))
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
		"", "", "", "", textErrorString,
	}

	// Assert only error logs are logged when log level is ERROR.
	validateLogOutputAtSpecifiedFormatAndSeverity(t.T(), "text", cfg.ERROR, expected)
}

func (t *LoggerTest) TestTextFormatLogs_LogLevelWARNING() {
	var expected = []string{
		"", "", "", textWarningString, textErrorString,
	}

	// Assert warning and error logs are logged when log level is WARNING.
	validateLogOutputAtSpecifiedFormatAndSeverity(t.T(), "text", cfg.WARNING, expected)
}

func (t *LoggerTest) TestTextFormatLogs_LogLevelINFO() {
	var expected = []string{
		"", "", textInfoString, textWarningString, textErrorString,
	}

	// Assert info, warning & error logs are logged when log level is INFO.
	validateLogOutputAtSpecifiedFormatAndSeverity(t.T(), "text", cfg.INFO, expected)
}

func (t *LoggerTest) TestTextFormatLogs_LogLevelDEBUG() {
	var expected = []string{
		"", textDebugString, textInfoString, textWarningString, textErrorString,
	}

	// Assert debug, info, warning & error logs are logged when log level is DEBUG.
	validateLogOutputAtSpecifiedFormatAndSeverity(t.T(), "text", cfg.DEBUG, expected)
}

func (t *LoggerTest) TestTextFormatLogs_LogLevelTRACE() {
	var expected = []string{
		textTraceString, textDebugString, textInfoString, textWarningString, textErrorString,
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
		"", "", "", "", jsonErrorString,
	}

	// Assert only error logs are logged when log level is ERROR.
	validateLogOutputAtSpecifiedFormatAndSeverity(t.T(), "json", cfg.ERROR, expected)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelWARNING() {
	var expected = []string{
		"", "", "", jsonWarningString, jsonErrorString,
	}

	// Assert warning and error logs are logged when log level is WARNING.
	validateLogOutputAtSpecifiedFormatAndSeverity(t.T(), "json", cfg.WARNING, expected)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelINFO() {
	var expected = []string{
		"", "", jsonInfoString, jsonWarningString, jsonErrorString,
	}

	// Assert info, warning & error logs are logged when log level is INFO.
	validateLogOutputAtSpecifiedFormatAndSeverity(t.T(), "json", cfg.INFO, expected)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelDEBUG() {
	var expected = []string{
		"", jsonDebugString, jsonInfoString, jsonWarningString, jsonErrorString,
	}

	// Assert debug, info, warning & error logs are logged when log level is DEBUG.
	validateLogOutputAtSpecifiedFormatAndSeverity(t.T(), "json", cfg.DEBUG, expected)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelTRACE() {
	var expected = []string{
		jsonTraceString, jsonDebugString, jsonInfoString, jsonWarningString, jsonErrorString,
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
			textInfoString,
		},
		{
			"json",
			"my-bucket",
			jsonInfoString,
		},
		{
			"",
			"my-other-bucket",
			jsonInfoString,
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
		expectedRegexp := regexp.MustCompile(test.expectedOutput)
		assert.True(t.T(), expectedRegexp.MatchString(output))
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
			jsonInfoString,
		},
		{
			"MountInstanceIDNotSet",
			false,
			defaultMountInstanceID,
			"gcsfuse",
			"text",
			textInfoString,
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
			expectedRegexp := regexp.MustCompile(test.expectedOutput)
			assert.True(t, expectedRegexp.MatchString(output))
			assert.True(t, strings.Contains(output, fmt.Sprintf("%s-%s", test.fsName, test.expectedInstanceID)))
		})
	}
}
