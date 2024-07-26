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
	"path"
	"slices"
	"text/template" // NOLINT
)

var (
	outDir      = flag.String("outDir", "", "Output directory where the auto-generated files are to be placed.")
	paramsFile  = flag.String("paramsFile", "", "Params YAML file.")
	templateDir = flag.String("templateDir", ".", "Directory containing the template files.")
)

type templateData struct {
	TypeTemplateData []typeTemplateData
	FlagTemplateData []flagTemplateData
	// Back-ticks are not supported in templates. So, passing as a parameter.
	Backticks string
}

func validateFlags() error {
	if *paramsFile == "" {
		return fmt.Errorf("params filename cannot be empty")
	}
	if *outDir == "" {
		return fmt.Errorf("output directory cannot be empty")
	}
	if *templateDir == "" {
		return fmt.Errorf("template directory cannot be empty")
	}
	return nil
}

// write applies the dataObj which could be an object of any type, to the
// templateFile and writes the generated text into the outputFile.
func write(dataObj any, outputFile, templateFile string) error {
	o, err := os.Create(outputFile)
	if err != nil {
		defer func() { _ = o.Close() }()
	}
	if err != nil {
		return err
	}
	_, file := path.Split(templateFile)
	tmpl, err := template.New(file).ParseFiles(templateFile)
	if err != nil {
		return err
	}
	err = tmpl.Execute(o, dataObj)
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
	}, path.Join(*outDir, "config.go"), path.Join(*templateDir, "config.tpl"))
	if err != nil {
		panic(err)
	}

}
