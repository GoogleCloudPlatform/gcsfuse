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

	"github.com/googlecloudplatform/gcsfuse/v3/cfg/shared"
)

var (
	outDir      = flag.String("outDir", "", "Output directory where the auto-generated files are to be placed")
	paramsFile  = flag.String("paramsFile", "", "Params YAML file")
	templateDir = flag.String("templateDir", ".", "Directory containing the template files")
)

type OptimizationRulesMap = map[string]shared.OptimizationRules

type templateData struct {
	TypeTemplateData       []typeTemplateData
	FlagTemplateData       []flagTemplateData
	MachineTypeToGroupsMap map[string][]string
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
	}

	file := path.Base(templateFile)
	var tmpl *template.Template
	if tmpl, err = template.New(file).Funcs(funcMap).ParseFiles(templateFile); err != nil {
		return err
	}
	return tmpl.Execute(outF, dataObj)
}

// invertMachineTypeGroups takes the parsed map of group->machines
// and returns a map of machine->groups.
func invertMachineTypeGroups(groups map[string][]string) map[string][]string {
	inverted := make(map[string][]string)
	for groupName, machineTypes := range groups {
		for _, machineType := range machineTypes {
			inverted[machineType] = append(inverted[machineType], groupName)
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
	machineTypeToGroupsMap := invertMachineTypeGroups(paramsYAML.MachineTypeGroups)

	err = write(templateData{
		FlagTemplateData:       fd,
		TypeTemplateData:       td,
		MachineTypeToGroupsMap: machineTypeToGroupsMap,
		Backticks:              "`",
	},
		path.Join(*outDir, "config.go"),
		path.Join(*templateDir, "config.tpl"))
	if err != nil {
		panic(err)
	}
}

// formatValue is a custom template function that correctly formats values for Go code.
// It adds quotes to strings and leaves other types as-is.
func formatValue(v any) string {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		// Use %q to safely quote strings, e.g., "my-string"
		return fmt.Sprintf("%q", v)
	default:
		// Use %v for other types like int, bool, etc.
		return fmt.Sprintf("%v", v)
	}
}
