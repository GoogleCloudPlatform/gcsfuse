/*
 * Copyright 2025 Google LLC
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
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg/shared"
	"gopkg.in/yaml.v3"
)

const (
	machineTypeGroupRegexPattern = `^[a-z]+(-[a-z0-9]+)*$`
)

var (
	machineTypeGroupRegex = regexp.MustCompile(machineTypeGroupRegexPattern)
)

type Param struct {
	FlagName           string `yaml:"flag-name"`
	Shorthand          string
	Type               string
	DefaultValue       string `yaml:"default"`
	ConfigPath         string `yaml:"config-path"`
	IsDeprecated       bool   `yaml:"deprecated"`
	DeprecationWarning string `yaml:"deprecation-warning"`
	Usage              string
	HideFlag           bool                      `yaml:"hide-flag"`
	HideShorthand      bool                      `yaml:"hide-shorthand"`
	Optimizations      *shared.OptimizationRules `yaml:"optimizations,omitempty"`
}

// ParamsYAML mirrors the params.yaml file itself.
type ParamsYAML struct {
	Params            []Param             `yaml:"params"`
	MachineTypeGroups map[string][]string `yaml:"machine-type-groups"`
}

func parseParamsYAMLStr(paramsYAMLStr string) (paramsYAML ParamsYAML, err error) {
	dec := yaml.NewDecoder(strings.NewReader(paramsYAMLStr))
	dec.KnownFields(true)
	if err = dec.Decode(&paramsYAML); err != nil {
		return ParamsYAML{}, err
	}
	if err = validateParams(paramsYAML.Params); err != nil {
		return ParamsYAML{}, err
	}
	if err = validateMachineTypeGroups(paramsYAML.MachineTypeGroups); err != nil {
		return ParamsYAML{}, err
	}
	return paramsYAML, nil
}

func parseParamsYAML() (ParamsYAML, error) {
	buf, err := os.ReadFile(*paramsFile)
	if err != nil {
		return ParamsYAML{}, err
	}
	return parseParamsYAMLStr(string(buf))
}

func checkFlagName(name string) error {
	if name == "" {
		return fmt.Errorf("flag-name cannot be empty")
	}

	// A valid name should contain only lower-case characters with hyphens as
	// separators. It must start and end with an alphabet.
	regex := `^[a-z]+([-_][a-z]+)*$`
	if matched, _ := regexp.MatchString(regex, name); !matched {
		return fmt.Errorf("flag-name %q does not conform to the regex: %s", name, regex)
	}
	return nil
}

func validateParam(param Param) error {
	if err := checkFlagName(param.FlagName); err != nil {
		return err
	}
	if param.IsDeprecated && param.DeprecationWarning == "" {
		return fmt.Errorf("param %s is marked deprecated but deprecation-warning is not set", param.FlagName)
	}

	if param.ConfigPath == "" && !param.IsDeprecated {
		return fmt.Errorf("config-path is empty for flag-name: %s", param.FlagName)
	}
	for k, v := range map[string]string{
		"flag-name": param.FlagName,
		"usage":     param.Usage,
		"type":      param.Type,
	} {
		if v == "" {
			return fmt.Errorf("%s is empty for flag-name: %s", k, param.FlagName)
		}
	}

	// Validate the data type.
	idx := slices.IndexFunc(
		[]string{"int", "float64", "bool", "string", "duration", "octal", "[]int",
			"[]string", "logSeverity", "protocol", "resolvedPath"},
		func(dt string) bool {
			return dt == param.Type
		},
	)
	if idx == -1 {
		return fmt.Errorf("unsupported datatype: %s", param.Type)
	}

	// Validate bucket-based optimizations if present.
	if param.Optimizations != nil {
		validBucketTypes := []string{"zonal", "hierarchical", "flat"}
		for _, bto := range param.Optimizations.BucketTypeOptimization {
			if !slices.Contains(validBucketTypes, bto.BucketType) {
				return fmt.Errorf("invalid bucket-type %q for flag %s; must be one of: %v",
					bto.BucketType, param.FlagName, validBucketTypes)
			}
		}
	}

	return nil
}

func isSorted(params []Param) error {
	if len(params) == 0 {
		return nil
	}
	prev := params[0]
	for _, next := range params[1:] {
		if (next.ConfigPath != "" && (prev.ConfigPath == "" || prev.ConfigPath > next.ConfigPath)) ||
			(next.ConfigPath == "" && prev.ConfigPath == "" && prev.FlagName > next.FlagName) {
			return fmt.Errorf("params.yaml is not sorted - flag: %s is at an incorrect position. Please refer to the documentation in params.yaml to know how to sort", next.FlagName)
		}
		prev = next
	}
	return nil
}

func validateParams(params []Param) error {
	if err := isSorted(params); err != nil {
		return fmt.Errorf("incorrect sorting order detected: %w", err)
	}
	if err := validateForDuplicates(params, func(param Param) string { return param.FlagName }); err != nil {
		return fmt.Errorf("duplicate flag names found: %w", err)
	}
	if err := validateForDuplicates(params, func(param Param) string { return param.ConfigPath }); err != nil {
		return fmt.Errorf("duplicate config-paths found: %w", err)
	}
	for _, param := range params {
		err := validateParam(param)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateForDuplicates(params []Param, fn func(param Param) string) error {
	lookup := make(map[string]bool)
	for _, p := range params {
		value := fn(p)
		if value == "" {
			continue
		}
		if _, ok := lookup[value]; ok {
			return fmt.Errorf("%s is present more than once", value)
		}
		lookup[value] = true
	}
	return nil
}

// validateMachineTypeGroups validates the machine-type-groups map.
func validateMachineTypeGroups(groups map[string][]string) error {
	// Temporary map to check if a machine-type belong to multiple groups.
	machineTypeToGroupMap := make(map[string]string)
	// Note: We can't easily validate that the group names themselves are sorted
	// in the YAML file because Go maps do not preserve insertion order. This
	// should be enforced through code reviews or a linter.
	for groupName, machineTypes := range groups {
		// 1. Validate group name format (e.g., kebab-case).
		if !machineTypeGroupRegex.MatchString(groupName) {
			return fmt.Errorf("group name %q does not conform to machineTypeGroupRegexPattern: %q", groupName, machineTypeGroupRegexPattern)
		}

		if len(machineTypes) == 0 {
			return fmt.Errorf("group %q must contain at least one machine type", groupName)
		}

		// 2. Validate machine types within the group are sorted and unique.
		if !slices.IsSorted(machineTypes) {
			return fmt.Errorf("machine types in group %q are not sorted alphabetically", groupName)
		}
		if err := validateForDuplicatesInSortedSlice(machineTypes); err != nil {
			return fmt.Errorf("duplicate machine type found in group %q: %w", groupName, err)
		}
		// Check for cross-group uniqueness for each machine type.
		for _, machineType := range machineTypes {
			if existingGroup, ok := machineTypeToGroupMap[machineType]; ok {
				return fmt.Errorf(
					"machine type %q cannot be in multiple groups; it is in both %q and %q",
					machineType,
					existingGroup,
					groupName,
				)
			}
			machineTypeToGroupMap[machineType] = groupName
		}
	}

	return nil
}

// validateForDuplicatesInSortedSlice is a helper to check for duplicates in an already sorted string slice.
func validateForDuplicatesInSortedSlice(items []string) error {
	for i, item := range items {
		if item == "" {
			return fmt.Errorf("item cannot be an empty string")
		}
		if i > 0 && item == items[i-1] {
			return fmt.Errorf("%q is present more than once", item)
		}
	}
	return nil
}
