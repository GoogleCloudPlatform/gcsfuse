// Copyright 2023 Google Inc. All Rights Reserved.
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
	"log/slog"
	"os"
	"regexp"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	. "github.com/jacobsa/ogletest"
)

const (
	textTraceString   = "^time=\"[a-zA-Z0-9/:. ]{26}\" severity=TRACE message=\"TestLogs: www.traceExample.com\""
	textDebugString   = "^time=\"[a-zA-Z0-9/:. ]{26}\" severity=DEBUG message=\"TestLogs: www.debugExample.com\""
	textInfoString    = "^time=\"[a-zA-Z0-9/:. ]{26}\" severity=INFO message=\"TestLogs: www.infoExample.com\""
	textWarningString = "^time=\"[a-zA-Z0-9/:. ]{26}\" severity=WARNING message=\"TestLogs: www.warningExample.com\""
	textErrorString   = "^time=\"[a-zA-Z0-9/:. ]{26}\" severity=ERROR message=\"TestLogs: www.errorExample.com\""

	jsonTraceString   = "^{\"timestamp\":{\"seconds\":\\d{10},\"nanos\":\\d{0,9}},\"severity\":\"TRACE\",\"message\":\"TestLogs: www.traceExample.com\"}"
	jsonDebugString   = "^{\"timestamp\":{\"seconds\":\\d{10},\"nanos\":\\d{0,9}},\"severity\":\"DEBUG\",\"message\":\"TestLogs: www.debugExample.com\"}"
	jsonInfoString    = "^{\"timestamp\":{\"seconds\":\\d{10},\"nanos\":\\d{0,9}},\"severity\":\"INFO\",\"message\":\"TestLogs: www.infoExample.com\"}"
	jsonWarningString = "^{\"timestamp\":{\"seconds\":\\d{10},\"nanos\":\\d{0,9}},\"severity\":\"WARNING\",\"message\":\"TestLogs: www.warningExample.com\"}"
	jsonErrorString   = "^{\"timestamp\":{\"seconds\":\\d{10},\"nanos\":\\d{0,9}},\"severity\":\"ERROR\",\"message\":\"TestLogs: www.errorExample.com\"}"
)

func TestLogger(t *testing.T) { RunTests(t) }

type LoggerTest struct {
}

func init() { RegisterTestSuite(&LoggerTest{}) }

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

func redirectLogsToGivenBuffer(buf *bytes.Buffer, level config.LogSeverity) {
	var programLevel = new(slog.LevelVar)
	defaultLogger = slog.New(
		defaultLoggerFactory.createJsonOrTextHandler(buf, programLevel, "TestLogs: "),
	)
	setLoggingLevel(level, programLevel)
}

// fetchLogOutputForSpecifiedSeverityLevel takes configured severity and
// functions that write logs as parameter and returns string array containing
// output from each function call.
func fetchLogOutputForSpecifiedSeverityLevel(level config.LogSeverity, functions []func()) []string {
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

func validateOutput(expected []string, output []string) {
	for i := range output {
		if expected[i] == "" {
			AssertEq(expected[i], output[i])
		} else {
			expectedRegexp := regexp.MustCompile(expected[i])
			AssertTrue(expectedRegexp.MatchString(output[i]))
		}
	}
}

func validateLogOutputAtSpecifiedFormatAndSeverity(format string, level config.LogSeverity, expectedOutput []string) {
	// set log format
	defaultLoggerFactory.format = format

	output := fetchLogOutputForSpecifiedSeverityLevel(level, getTestLoggingFunctions())

	validateOutput(expectedOutput, output)
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////
func (t *LoggerTest) TestTextFormatLogs_LogLevelOFF() {
	var expected = []string{
		"", "", "", "", "",
	}

	// Assert that nothing is logged when log level is OFF.
	validateLogOutputAtSpecifiedFormatAndSeverity("json", config.OFF, expected)
}

func (t *LoggerTest) TestTextFormatLogs_LogLevelERROR() {
	var expected = []string{
		"", "", "", "", textErrorString,
	}

	// Assert only error logs are logged when log level is ERROR.
	validateLogOutputAtSpecifiedFormatAndSeverity("text", config.ERROR, expected)
}

func (t *LoggerTest) TestTextFormatLogs_LogLevelWARNING() {
	var expected = []string{
		"", "", "", textWarningString, textErrorString,
	}

	// Assert warning and error logs are logged when log level is WARNING.
	validateLogOutputAtSpecifiedFormatAndSeverity("text", config.WARNING, expected)
}

func (t *LoggerTest) TestTextFormatLogs_LogLevelINFO() {
	var expected = []string{
		"", "", textInfoString, textWarningString, textErrorString,
	}

	// Assert info, warning & error logs are logged when log level is INFO.
	validateLogOutputAtSpecifiedFormatAndSeverity("text", config.INFO, expected)
}

func (t *LoggerTest) TestTextFormatLogs_LogLevelDEBUG() {
	var expected = []string{
		"", textDebugString, textInfoString, textWarningString, textErrorString,
	}

	// Assert debug, info, warning & error logs are logged when log level is DEBUG.
	validateLogOutputAtSpecifiedFormatAndSeverity("text", config.DEBUG, expected)
}

func (t *LoggerTest) TestTextFormatLogs_LogLevelTRACE() {
	var expected = []string{
		textTraceString, textDebugString, textInfoString, textWarningString, textErrorString,
	}

	// Assert all logs are logged when log level is TRACE.
	validateLogOutputAtSpecifiedFormatAndSeverity("text", config.TRACE, expected)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelOFF() {
	var expected = []string{
		"", "", "", "", "",
	}

	// Assert that nothing is logged when log level is OFF.
	validateLogOutputAtSpecifiedFormatAndSeverity("json", config.OFF, expected)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelERROR() {
	var expected = []string{
		"", "", "", "", jsonErrorString,
	}

	// Assert only error logs are logged when log level is ERROR.
	validateLogOutputAtSpecifiedFormatAndSeverity("json", config.ERROR, expected)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelWARNING() {
	var expected = []string{
		"", "", "", jsonWarningString, jsonErrorString,
	}

	// Assert warning and error logs are logged when log level is WARNING.
	validateLogOutputAtSpecifiedFormatAndSeverity("json", config.WARNING, expected)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelINFO() {
	var expected = []string{
		"", "", jsonInfoString, jsonWarningString, jsonErrorString,
	}

	// Assert info, warning & error logs are logged when log level is INFO.
	validateLogOutputAtSpecifiedFormatAndSeverity("json", config.INFO, expected)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelDEBUG() {
	var expected = []string{
		"", jsonDebugString, jsonInfoString, jsonWarningString, jsonErrorString,
	}

	// Assert debug, info, warning & error logs are logged when log level is DEBUG.
	validateLogOutputAtSpecifiedFormatAndSeverity("json", config.DEBUG, expected)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelTRACE() {
	var expected = []string{
		jsonTraceString, jsonDebugString, jsonInfoString, jsonWarningString, jsonErrorString,
	}

	// Assert all logs are logged when log level is TRACE.
	validateLogOutputAtSpecifiedFormatAndSeverity("json", config.TRACE, expected)
}

func (t *LoggerTest) TestSetLoggingLevel() {
	testData := []struct {
		inputLevel           config.LogSeverity
		programLevel         *slog.LevelVar
		expectedProgramLevel slog.Level
	}{
		{
			config.TRACE,
			new(slog.LevelVar),
			LevelTrace,
		},
		{
			config.DEBUG,
			new(slog.LevelVar),
			LevelDebug,
		},
		{
			config.WARNING,
			new(slog.LevelVar),
			LevelWarn,
		},
		{
			config.ERROR,
			new(slog.LevelVar),
			LevelError,
		},
		{
			config.OFF,
			new(slog.LevelVar),
			LevelOff,
		},
	}

	for _, test := range testData {
		setLoggingLevel(test.inputLevel, test.programLevel)
		AssertEq(test.programLevel.Level(), test.expectedProgramLevel)
	}
}

func (t *LoggerTest) TestInitLogFile() {
	format := "text"
	filePath, _ := os.UserHomeDir()
	filePath += "/log.txt"
	fileSize := 100
	backupFileCount := 2
	logConfig := config.LogConfig{
		Severity: config.DEBUG,
		Format:   format,
		FilePath: filePath,
		LogRotateConfig: config.LogRotateConfig{
			MaxFileSizeMB:   fileSize,
			BackupFileCount: backupFileCount,
			Compress:        true,
		},
	}

	err := InitLogFile(logConfig)

	AssertEq(nil, err)
	ExpectEq(filePath, defaultLoggerFactory.file.Name())
	ExpectEq(nil, defaultLoggerFactory.sysWriter)
	ExpectEq(format, defaultLoggerFactory.format)
	ExpectEq(config.DEBUG, defaultLoggerFactory.level)
	ExpectEq(fileSize, defaultLoggerFactory.logRotateConfig.MaxFileSizeMB)
	ExpectEq(backupFileCount, defaultLoggerFactory.logRotateConfig.BackupFileCount)
	ExpectEq(true, defaultLoggerFactory.logRotateConfig.Compress)
}

func (t *LoggerTest) TestSetLogFormatToText() {
	defaultLoggerFactory = &loggerFactory{
		file:            nil,
		level:           config.INFO, // setting log level to INFO by default
		logRotateConfig: config.DefaultLogRotateConfig(),
	}

	testData := []struct {
		format         string
		expectedOutput string
	}{
		{
			"text",
			textInfoString,
		},
		{
			"json",
			jsonInfoString,
		},
		{
			"",
			jsonInfoString,
		},
	}

	for _, test := range testData {
		SetLogFormat(test.format)

		AssertNe(nil, defaultLoggerFactory)
		AssertNe(nil, defaultLogger)
		ExpectEq(defaultLoggerFactory.format, test.format)
		// Create a logger using defaultLoggerFactory that writes to buffer.
		var buf bytes.Buffer
		redirectLogsToGivenBuffer(&buf, defaultLoggerFactory.level)
		Infof("www.infoExample.com")
		output := buf.String()
		// Compare expected and actual log.
		expectedRegexp := regexp.MustCompile(test.expectedOutput)
		ExpectTrue(expectedRegexp.MatchString(output))
	}
}
