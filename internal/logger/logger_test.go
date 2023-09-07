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

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////
func (t *LoggerTest) TestTextFormatLogs_LogLevelOFF() {
	// set log format to text
	defaultLoggerFactory.format = "text"

	outputs := fetchLogOutputForSpecifiedSeverityLevel(config.OFF, getTestLoggingFunctions())

	// Assert that nothing is logged when log level is OFF.
	for _, output := range outputs {
		AssertEq("", output)
	}
}

func (t *LoggerTest) TestTextFormatLogs_LogLevelERROR() {
	// set log format to text
	defaultLoggerFactory.format = "text"

	output := fetchLogOutputForSpecifiedSeverityLevel(config.ERROR, getTestLoggingFunctions())

	// Assert only error logs are logged when log level is ERROR.
	var expected = []string{
		"", "", "", "", textErrorString,
	}
	validateOutput(expected, output)
}

func (t *LoggerTest) TestTextFormatLogs_LogLevelWARNING() {
	// set log format to text
	defaultLoggerFactory.format = "text"

	output := fetchLogOutputForSpecifiedSeverityLevel(config.WARNING, getTestLoggingFunctions())

	// Assert warning and error logs are logged when log level is WARNING.
	var expected = []string{
		"", "", "", textWarningString, textErrorString,
	}
	validateOutput(expected, output)
}

func (t *LoggerTest) TestTextFormatLogs_LogLevelINFO() {
	// set log format to text
	defaultLoggerFactory.format = "text"

	output := fetchLogOutputForSpecifiedSeverityLevel(config.INFO, getTestLoggingFunctions())

	// Assert info, warning & error logs are logged when log level is INFO.
	var expected = []string{
		"", "", textInfoString, textWarningString, textErrorString,
	}
	validateOutput(expected, output)
}

func (t *LoggerTest) TestTextFormatLogs_LogLevelDEBUG() {
	// set log format to text
	defaultLoggerFactory.format = "text"

	output := fetchLogOutputForSpecifiedSeverityLevel(config.DEBUG, getTestLoggingFunctions())

	// Assert debug, info, warning & error logs are logged when log level is DEBUG.
	var expected = []string{
		"", textDebugString, textInfoString, textWarningString, textErrorString,
	}
	validateOutput(expected, output)
}

func (t *LoggerTest) TestTextFormatLogs_LogLevelTRACE() {
	// set log format to text
	defaultLoggerFactory.format = "text"

	output := fetchLogOutputForSpecifiedSeverityLevel(config.TRACE, getTestLoggingFunctions())

	// Assert all logs are logged when log level is TRACE.
	var expected = []string{
		textTraceString, textDebugString, textInfoString, textWarningString, textErrorString,
	}
	validateOutput(expected, output)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelOFF() {
	// set log format to json
	defaultLoggerFactory.format = "json"

	outputs := fetchLogOutputForSpecifiedSeverityLevel(config.OFF, getTestLoggingFunctions())

	// Assert that nothing is logged when log level is OFF.
	for _, output := range outputs {
		AssertEq("", output)
	}
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelERROR() {
	// set log format to json
	defaultLoggerFactory.format = "json"

	output := fetchLogOutputForSpecifiedSeverityLevel(config.ERROR, getTestLoggingFunctions())

	// Assert only error logs are logged when log level is ERROR.
	var expected = []string{
		"", "", "", "", jsonErrorString,
	}
	validateOutput(expected, output)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelWARNING() {
	// set log format to json
	defaultLoggerFactory.format = "json"

	output := fetchLogOutputForSpecifiedSeverityLevel(config.WARNING, getTestLoggingFunctions())

	// Assert warning and error logs are logged when log level is WARNING.
	var expected = []string{
		"", "", "", jsonWarningString, jsonErrorString,
	}
	validateOutput(expected, output)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelINFO() {
	// set log format to json
	defaultLoggerFactory.format = "json"

	output := fetchLogOutputForSpecifiedSeverityLevel(config.INFO, getTestLoggingFunctions())

	// Assert info, warning & error logs are logged when log level is INFO.
	var expected = []string{
		"", "", jsonInfoString, jsonWarningString, jsonErrorString,
	}
	validateOutput(expected, output)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelDEBUG() {
	// set log format to json
	defaultLoggerFactory.format = "json"

	output := fetchLogOutputForSpecifiedSeverityLevel(config.DEBUG, getTestLoggingFunctions())

	// Assert debug, info, warning & error logs are logged when log level is DEBUG.
	var expected = []string{
		"", jsonDebugString, jsonInfoString, jsonWarningString, jsonErrorString,
	}
	validateOutput(expected, output)
}

func (t *LoggerTest) TestJSONFormatLogs_LogLevelTRACE() {
	// set log format to json
	defaultLoggerFactory.format = "json"

	output := fetchLogOutputForSpecifiedSeverityLevel(config.TRACE, getTestLoggingFunctions())

	// Assert all logs are logged when log level is TRACE.
	var expected = []string{
		jsonTraceString, jsonDebugString, jsonInfoString, jsonWarningString, jsonErrorString,
	}
	validateOutput(expected, output)
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
