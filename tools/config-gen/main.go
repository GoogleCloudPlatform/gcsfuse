/*
 * Copyright 2024 Google Inc. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"cmp"
	"flag"
	"fmt"
	"os"
	"slices"
	"text/template" // NOLINT
)

var (
	outFile                  = flag.String("outFile", "", "Output file")
	paramsFile               = flag.String("paramsFile", "", "Params YAML file")
	templateFile             = flag.String("templateFile", "", "Template file")
	testDefaultsTemplateFile = flag.String("testDefaultsTemplateFile", "", "Template for the test defaults file")
	testDefaultsOutFile      = flag.String("testDefaultsOutFile", "", "Output path for emitting the defaults test")

	testFlagParsingOutFile      = flag.String("testFlagParsingOutFile", "", "Output path for emitting the flag parsing test")
	testFlagParsingTemplateFile = flag.String("testFlagParsingTemplateFile", "", "Path to the template file for testing CLI flag parsing")
)

type templateData struct {
	TypeTemplateData []typeTemplateData
	FlagTemplateData []flagTemplateData
	// Back-ticks are not supported in templates. So, passing as a parameter.
	Backticks string
}

func validateFlags() error {
	if *paramsFile == "" {
		return fmt.Errorf("input filename cannot be empty")
	}
	return nil
}

func write(data any, outF string, tmplF string) error {
	outputFile, err := os.Create(outF)
	if err != nil {
		defer func() { _ = outputFile.Close() }()
	}
	if err != nil {
		return err
	}
	tmpl, err := template.New(tmplF).ParseFiles(tmplF)
	if err != nil {
		return err
	}
	err = tmpl.Execute(outputFile, data)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	flag.Parse()
	err := validateFlags()
	if err != nil {
		panic(err)
	}

	paramsConfig, err := parseParamsConfig()
	if err != nil {
		panic(err)
	}

	d, err := dataTypesFromParams(paramsConfig)
	if err != nil {
		panic(err)
	}

	if *outFile != "" && *templateFile != "" {
		td, err := constructTypeTemplateData(d)
		if err != nil {
			panic(err)
		}

		fd, err := computeFlagTemplateData(d)
		if err != nil {
			panic(err)
		}

		// Sort to have reliable ordering.
		slices.SortFunc(td, func(i, j typeTemplateData) int {
			return cmp.Compare(i.TypeName, j.TypeName)
		})
		slices.SortFunc(fd, func(i, j flagTemplateData) int {
			return cmp.Compare(i.FlagName, j.FlagName)
		})
		err = write(templateData{
			FlagTemplateData: fd,
			TypeTemplateData: td,
			Backticks:        "`",
		}, *outFile, *templateFile)
		if err != nil {
			panic(err)
		}
	}
	if *testDefaultsOutFile != "" && *testDefaultsTemplateFile != "" {
		defTest := computeDefaultsTestTemplateData(d)
		if err = write(defTest, *testDefaultsOutFile, *testDefaultsTemplateFile); err != nil {
			panic(err)
		}
	}
	if *testFlagParsingOutFile != "" && *testFlagParsingTemplateFile != "" {
		flagParsingTest := computeFlagParsingTestTemplateData(d)
		err = write(flagParsingTest, *testFlagParsingOutFile, *testFlagParsingTemplateFile)
	}
}

func resolveDataType(p Param) (datatype, error) {
	switch p.Type {
	case "int":
		return intDatatype{Param: p}, nil
	case "float64":
		return float64Datatype{Param: p}, nil
	case "bool":
		return boolDatatype{Param: p}, nil
	case "duration":
		return durationDatatype{Param: p}, nil
	case "octal":
		return octalDatatype{Param: p}, nil
	case "url":
		return urlDatatype{Param: p}, nil
	case "logSeverity":
		return logSeverityDatatype{Param: p}, nil
	case "protocol":
		return protocolDatatype{Param: p}, nil
	case "resolvedPath":
		return resolvedPathDatatype{Param: p}, nil
	case "string":
		return stringDatatype{Param: p}, nil
	case "[]int":
		return intSliceDatatype{Param: p}, nil
	case "[]string":
		return stringSliceDatatype{Param: p}, nil
	default:
		return nil, fmt.Errorf("unsupported data type: %s", p.Type)
	}
}

func dataTypesFromParams(params []Param) ([]datatype, error) {
	d := make([]datatype, 0, len(params))
	for _, p := range params {
		dt, err := resolveDataType(p)
		if err != nil {
			return nil, err
		}
		d = append(d, dt)
	}
	return d, nil
}
