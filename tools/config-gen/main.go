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
	outFile      = flag.String("outFile", "", "Output file")
	paramsFile   = flag.String("paramsFile", "", "Params YAML file")
	templateFile = flag.String("templateFile", "", "Template file")
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
	if *outFile == "" {
		return fmt.Errorf("output filename cannot be empty")
	}
	if *templateFile == "" {
		return fmt.Errorf("template filename cannot be empty")
	}
	return nil
}

func write(data templateData) error {
	outputFile, err := os.Create(*outFile)
	if err != nil {
		defer outputFile.Close()
	}
	if err != nil {
		return err
	}
	tmpl, err := template.New("config.tpl").ParseFiles(*templateFile)
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

	td, err := constructTypeTemplateData(paramsConfig)
	if err != nil {
		panic(err)
	}

	fd, err := computeFlagTemplateData(paramsConfig)
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
	})
	if err != nil {
		panic(err)
	}

}
