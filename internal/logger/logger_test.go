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
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	. "github.com/jacobsa/ogletest"
)

const (
	textTraceString   = "severity=TRACE msg=\"TestLogs: www.traceExample.com\""
	textDebugString   = "severity=DEBUG msg=\"TestLogs: www.debugExample.com\""
	textInfoString    = "severity=INFO msg=\"TestLogs: www.infoExample.com\""
	textWarningString = "severity=WARNING msg=\"TestLogs: www.warningExample.com\""
	textErrorString   = "severity=ERROR msg=\"TestLogs: www.errorExample.com\""
	jsonTraceString   = "\"severity\":\"TRACE\",\"msg\":\"TestLogs: www.traceExample.com\"}"
	jsonDebugString   = "\"severity\":\"DEBUG\",\"msg\":\"TestLogs: www.debugExample.com\"}"
	jsonInfoString    = "\"severity\":\"INFO\",\"msg\":\"TestLogs: www.infoExample.com\"}"
	jsonWarningString = "\"severity\":\"WARNING\",\"msg\":\"TestLogs: www.warningExample.com\"}"
	jsonErrorString   = "\"severity\":\"ERROR\",\"msg\":\"TestLogs: www.errorExample.com\"}"
)

func TestLogger(t *testing.T) { RunTests(t) }

type LoggerTest struct {
}

func init() { RegisterTestSuite(&LoggerTest{}) }

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

// fetchLogOutputForSpecifiedSeverityLevel takes configured severity and
// functions that write logs as parameter and returns string array containing
// output from each function call.
func fetchLogOutputForSpecifiedSeverityLevel(level config.LogSeverity, functions []func()) []string {
	// create a logger that writes to buffer at configured level.
	var buf bytes.Buffer
	var programLevel = new(slog.LevelVar)
	logger := slog.New(
		defaultLoggerFactory.createJsonOrTextHandler(&buf, programLevel, "TestLogs: "),
	)
	setLoggingLevel(level, programLevel)

	// make the created logger default.
	defaultLogger = logger

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
			AssertTrue(strings.Contains(output[i], expected[i]))
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
