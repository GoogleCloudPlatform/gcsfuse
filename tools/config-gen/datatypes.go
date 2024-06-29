// Copyright 2024 Google Inc. All Rights Reserved.
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

package main

import (
	"fmt"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type testCase struct {
	// Args represents the list of args that are to be passed to the command in
	// the test-case
	Args []string

	// Expected is the expected value that will be compared as-is with the
	// corresponding field in config for that set of args.
	Expected string
}

type datatypeSpec interface {
	// Returns the underlying param object.
	param() Param

	// The corresponding Golang type for the datatype e.g. for int flag types, it's int64
	goType() string

	// Returns how the default value in the flag declaration should be represented.
	// For instance, for int flags, the default value specified in the param definition
	// should be directly replaced in the flag declaration.
	// However, for a string flag, the string needs to be escaped properly.
	flagDefaultValue() string

	// The method that should be called on the flagset to declare the flag. It's
	// IntP to declare an int flag.
	flagFn() string

	// For custom types, the flag is declared and formatted as int. However,
	// for testing the default values, the assertion needs to be done against
	// the actual type. This method formats the default value in such a way that
	// it can be compared with the type of the corresponding field in Config.
	testDefaultValue() string

	// handleErrorInDefaultTest returns true if the method that parses the default
	// value into the applicable type returns error as the second return value.
	handleErrorInDefaultTest() bool

	// testCases returns the test-cases that are to be associated with the
	// datatype
	testCases() []testCase
}

type intDatatype struct {
	Param
}

func (d intDatatype) testCases() []testCase {
	return []testCase{
		{
			Args:     []string{fmt.Sprintf("--%s=11", d.Param.FlagName)},
			Expected: "int64(11)",
		},
		{
			Args:     []string{fmt.Sprintf("--%s", d.Param.FlagName), "3478923"},
			Expected: "int64(3478923)",
		},
		{
			Args:     []string{fmt.Sprintf("--%s", d.Param.FlagName), "-123"},
			Expected: "int64(-123)",
		},
	}
}

func (d intDatatype) param() Param {
	return d.Param
}

func (intDatatype) goType() string {
	return "int64"
}

func (d intDatatype) flagDefaultValue() string {
	if d.DefaultValue == "" {
		return "0"
	}
	return d.DefaultValue
}

func (intDatatype) flagFn() string {
	return "IntP"
}

func (d intDatatype) testDefaultValue() string {
	return fmt.Sprintf("%s", d.DefaultValue)
}

func (d intDatatype) handleErrorInDefaultTest() bool {
	return false
}

type float64Datatype struct {
	Param
}

func (d float64Datatype) param() Param {
	return d.Param
}

func (float64Datatype) goType() string {
	return "float64"
}

func (d float64Datatype) flagDefaultValue() string {
	if d.DefaultValue == "" {
		return "0.0"
	}
	return d.DefaultValue
}

func (float64Datatype) flagFn() string {
	return "Float64P"
}

func (d float64Datatype) testDefaultValue() string {
	return fmt.Sprintf("float64(%s)", d.flagDefaultValue())
}

func (d float64Datatype) handleErrorInDefaultTest() bool {
	return false
}

func (d float64Datatype) testCases() []testCase {
	return []testCase{
		{
			Args:     []string{fmt.Sprintf("--%s=2.5", d.Param.FlagName)},
			Expected: "2.5",
		},
		{
			Args:     []string{fmt.Sprintf("--%s", d.Param.FlagName), "3.5"},
			Expected: "3.5",
		},
	}
}

type boolDatatype struct {
	Param
}

func (d boolDatatype) param() Param {
	return d.Param
}

func (boolDatatype) goType() string {
	return "bool"
}

func (d boolDatatype) flagDefaultValue() string {
	if d.DefaultValue == "" {
		return "false"
	}
	return d.DefaultValue
}

func (boolDatatype) flagFn() string {
	return "BoolP"
}

func (d boolDatatype) testDefaultValue() string {
	return fmt.Sprintf("%s", d.DefaultValue)
}

func (d boolDatatype) handleErrorInDefaultTest() bool {
	return false
}

func (d boolDatatype) testCases() []testCase {
	return []testCase{
		{
			Args:     []string{fmt.Sprintf("--%s", d.Param.FlagName)},
			Expected: "true",
		},
		{
			Args:     []string{fmt.Sprintf("--%s=true", d.Param.FlagName)},
			Expected: "true",
		},
		{
			Args:     []string{fmt.Sprintf("--%s=false", d.Param.FlagName)},
			Expected: "false",
		},
	}
}

type durationDatatype struct {
	Param
}

func (d durationDatatype) param() Param {
	return d.Param
}

func (durationDatatype) goType() string {
	return "time.Duration"
}

func (d durationDatatype) flagDefaultValue() string {
	if d.DefaultValue == "" {
		return "0s"
	}
	if dur, err := time.ParseDuration(d.DefaultValue); err == nil {
		return fmt.Sprintf("%d * time.Nanosecond", dur.Nanoseconds())
	}
	return d.DefaultValue
}

func (durationDatatype) flagFn() string {
	return "DurationP"
}

func (d durationDatatype) testDefaultValue() string {
	return fmt.Sprintf("time.ParseDuration(%q)", d.DefaultValue)
}

func (d durationDatatype) handleErrorInDefaultTest() bool {
	return true
}

func (d durationDatatype) testCases() []testCase {
	return []testCase{
		{
			Args:     []string{fmt.Sprintf("--%s", d.Param.FlagName), "2h45m"},
			Expected: "2*time.Hour + 45*time.Minute",
		},
		{
			Args:     []string{fmt.Sprintf("--%s", d.Param.FlagName), "300ms"},
			Expected: "300 * time.Millisecond",
		},
		{
			Args:     []string{fmt.Sprintf("--%s", d.Param.FlagName), "1h49m12s11ms"},
			Expected: "1*time.Hour + 49*time.Minute + 12*time.Second + 11*time.Millisecond",
		},
		{
			Args:     []string{fmt.Sprintf("--%s=2h45m", d.Param.FlagName)},
			Expected: "2*time.Hour + 45*time.Minute",
		},
		{
			Args:     []string{fmt.Sprintf("--%s=300ms", d.Param.FlagName)},
			Expected: "300 * time.Millisecond",
		},
		{
			Args:     []string{fmt.Sprintf("--%s=25h49m12s", d.Param.FlagName)},
			Expected: "25*time.Hour + 49*time.Minute + 12*time.Second",
		},
	}
}

type octalDatatype struct {
	Param
}

func (d octalDatatype) param() Param {
	return d.Param
}

func (octalDatatype) goType() string {
	return "Octal"
}

func (d octalDatatype) flagDefaultValue() string {
	return fmt.Sprintf("%q", d.DefaultValue)
}

func (octalDatatype) flagFn() string {
	return "StringP"
}

func (d octalDatatype) testDefaultValue() string {
	return fmt.Sprintf("cfg.Octal(%s)", d.DefaultValue)
}

func (d octalDatatype) handleErrorInDefaultTest() bool {
	return false
}

func (d octalDatatype) testCases() []testCase {
	return []testCase{
		{
			Args:     []string{fmt.Sprintf("--%s", d.Param.FlagName), "764"},
			Expected: "cfg.Octal(0764)",
		},
	}
}

type urlDatatype struct {
	Param
}

func (d urlDatatype) param() Param {
	return d.Param
}

func (urlDatatype) goType() string {
	return "url.URL"
}

func (d urlDatatype) flagDefaultValue() string {
	return fmt.Sprintf("%q", d.DefaultValue)
}

func (urlDatatype) flagFn() string {
	return "StringP"
}

func (d urlDatatype) testDefaultValue() string {
	return fmt.Sprintf("cfg.ParseURL(%q)", d.DefaultValue)
}

func (d urlDatatype) handleErrorInDefaultTest() bool {
	return true
}

func (d urlDatatype) testCases() []testCase {
	return []testCase{}
}

type logSeverityDatatype struct {
	Param
}

func (d logSeverityDatatype) param() Param {
	return d.Param
}

func (logSeverityDatatype) goType() string {
	return "LogSeverity"
}

func (logSeverityDatatype) flagFn() string {
	return "StringP"
}

func (d logSeverityDatatype) flagDefaultValue() string {
	return fmt.Sprintf("%q", cases.Upper(language.English).String(d.DefaultValue))
}

func (d logSeverityDatatype) testDefaultValue() string {
	return fmt.Sprintf("cfg.LogSeverity(%s)", d.flagDefaultValue())
}

func (d logSeverityDatatype) handleErrorInDefaultTest() bool {
	return false
}

func (d logSeverityDatatype) testCases() []testCase {
	return []testCase{}
}

type protocolDatatype struct {
	Param
}

func (d protocolDatatype) param() Param {
	return d.Param
}

func (protocolDatatype) goType() string {
	return "Protocol"
}

func (d protocolDatatype) flagDefaultValue() string {
	return fmt.Sprintf("%q", d.DefaultValue)
}

func (protocolDatatype) flagFn() string {
	return "StringP"
}

func (d protocolDatatype) testDefaultValue() string {
	return fmt.Sprintf("cfg.Protocol(%s)", d.flagDefaultValue())
}

func (d protocolDatatype) handleErrorInDefaultTest() bool {
	return false
}

func (d protocolDatatype) testCases() []testCase {
	return []testCase{}
}

type resolvedPathDatatype struct {
	Param
}

func (d resolvedPathDatatype) param() Param {
	return d.Param
}

func (resolvedPathDatatype) goType() string {
	return "ResolvedPath"
}

func (d resolvedPathDatatype) flagDefaultValue() string {
	return fmt.Sprintf("%q", d.DefaultValue)
}

func (resolvedPathDatatype) flagFn() string {
	return "StringP"
}

func (d resolvedPathDatatype) testDefaultValue() string {
	return fmt.Sprintf("cfg.GetNewResolvedPath(%s)", d.flagDefaultValue())
}

func (d resolvedPathDatatype) handleErrorInDefaultTest() bool {
	return true
}

func (d resolvedPathDatatype) testCases() []testCase {
	return []testCase{}
}

type stringDatatype struct {
	Param
}

func (d stringDatatype) param() Param {
	return d.Param
}

func (stringDatatype) goType() string {
	return "string"
}

func (d stringDatatype) flagDefaultValue() string {
	return fmt.Sprintf("%q", d.DefaultValue)
}

func (stringDatatype) flagFn() string {
	return "StringP"
}

func (d stringDatatype) testDefaultValue() string {
	return d.flagDefaultValue()
}

func (d stringDatatype) handleErrorInDefaultTest() bool {
	return false
}

func (d stringDatatype) testCases() []testCase {
	return []testCase{}
}

type intSliceDatatype struct {
	Param
}

func (d intSliceDatatype) param() Param {
	return d.Param
}

func (intSliceDatatype) goType() string {
	return "[]int64"
}

func (d intSliceDatatype) flagDefaultValue() string {
	return fmt.Sprintf("[]int{%s}", d.DefaultValue)
}

func (intSliceDatatype) flagFn() string {
	return "IntSliceP"
}

func (d intSliceDatatype) testDefaultValue() string {
	return d.flagDefaultValue()
}

func (d intSliceDatatype) handleErrorInDefaultTest() bool {
	return false
}

func (d intSliceDatatype) testCases() []testCase {
	return []testCase{}
}

type stringSliceDatatype struct {
	Param
}

func (d stringSliceDatatype) param() Param {
	return d.Param
}

func (stringSliceDatatype) goType() string {
	return "[]string"
}

func (d stringSliceDatatype) flagDefaultValue() string {
	return fmt.Sprintf("[]string{%s}", d.DefaultValue)
}

func (stringSliceDatatype) flagFn() string {
	return "StringSliceP"
}

func (d stringSliceDatatype) testDefaultValue() string {
	return d.flagDefaultValue()
}

func (d stringSliceDatatype) handleErrorInDefaultTest() bool {
	return false
}

func (d stringSliceDatatype) testCases() []testCase {
	return []testCase{}
}
