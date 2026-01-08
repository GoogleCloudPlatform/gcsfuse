/*
 * Copyright 2024 Google LLC
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
	"reflect"
	"slices"
	"text/template" // NOLINT
	"unicode"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg/shared"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	outDir      = flag.String("outDir", "", "Output directory where the auto-generated files are to be placed")
	paramsFile  = flag.String("paramsFile", "", "Params YAML file")
	templateDir = flag.String("templateDir", ".", "Directory containing the template files")
)

type OptimizationRulesMap = map[string]shared.OptimizationRules

type templateData struct {
	TypeTemplateData      []typeTemplateData
	FlagTemplateData      []flagTemplateData
	MachineTypeToGroupMap map[string]string
	MachineTypeGroups     map[string][]string
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
func write(dataObj any, outputFile, templateFile string) (err error) {
	var outF *os.File
	if outF, err = os.Create(outputFile); err != nil {
		return err
	}
	defer func() {
		closeErr := outF.Close()
		if err == nil {
			err = closeErr
		}
	}()
	// Define the custom function map.
	funcMap := template.FuncMap{
		"formatValue": formatValue,
		"title":       cases.Title(language.English).String,
	}

	file := path.Base(templateFile)
	var tmpl *template.Template
	if tmpl, err = template.New(file).Funcs(funcMap).ParseFiles(templateFile); err != nil {
		return err
	}
	return tmpl.Execute(outF, dataObj)
}

// invertMachineTypeGroups takes the parsed map of group->machines
// and returns a map of machine->group.
func invertMachineTypeGroups(groups map[string][]string) map[string]string {
	inverted := make(map[string]string)
	for groupName, machineTypes := range groups {
		for _, machineType := range machineTypes {
			if alreadyMappedGroup, ok := inverted[machineType]; ok {
				panic(fmt.Sprintf("machine type %q mapped to multiple groups, %q and %q", machineType, alreadyMappedGroup, groupName))
			}
			inverted[machineType] = groupName
		}
	}
	return inverted
}

func main() {
	flag.Parse()
	err := validateFlags()
	if err != nil {
		panic(err)
	}

	paramsYAML, err := parseParamsYAML()
	if err != nil {
		panic(err)
	}

	td, err := constructTypeTemplateData(paramsYAML.Params)
	if err != nil {
		panic(err)
	}

	fd, err := computeFlagTemplateData(paramsYAML.Params)
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

	// Create a map from given machine type to all the machine type groups that it belongs to.
	machineTypeToGroupMap := invertMachineTypeGroups(paramsYAML.MachineTypeGroups)

	for _, rootFileName := range []string{"config", "config_test"} {
		generatedFilePath := path.Join(*outDir, rootFileName+".go")
		templateFilePath := path.Join(*templateDir, rootFileName+".tpl")
		err = write(templateData{
			FlagTemplateData:      fd,
			TypeTemplateData:      td,
			MachineTypeToGroupMap: machineTypeToGroupMap,
			MachineTypeGroups:     paramsYAML.MachineTypeGroups,
			Backticks:             "`",
		},
			generatedFilePath, templateFilePath)
		if err != nil {
			panic(fmt.Sprintf("failed to generate file %q: %v", generatedFilePath, err))
		}
	}
}

// formatValue is a custom template function that correctly formats values for Go code.
// It adds quotes to strings and leaves other types as-is.
// Special case: if a string looks like a function call (ends with ()), it's output as-is.
func formatValue(v any) string {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		s := v.(string)
		// Check if it looks like a function call - if so, output as-is without quotes
		if len(s) > 2 && s[len(s)-2:] == "()" && unicode.IsUpper(rune(s[0])) {
			return s
		}
		// Use %q to safely quote strings, e.g., "my-string"
		return fmt.Sprintf("%q", v)
	default:
		// Use %v for other types like int, bool, etc.
		return fmt.Sprintf("%v", v)
	}
}
