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
	"bytes"
	"fmt"
	"os"
	"slices"

	"gopkg.in/yaml.v3"
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
	HideFlag           bool `yaml:"hide-flag"`
	HideShorthand      bool `yaml:"hide-shorthand"`
}

func parseParamsConfig() ([]Param, error) {
	buf, err := os.ReadFile(*paramsFile)
	if err != nil {
		return nil, err
	}
	var paramsConfig []Param
	dec := yaml.NewDecoder(bytes.NewReader(buf))
	dec.KnownFields(true)
	if err = dec.Decode(&paramsConfig); err != nil {
		return nil, err
	}
	if err = validateParams(paramsConfig); err != nil {
		return nil, err
	}
	return paramsConfig, nil
}

func validateParam(param Param) error {
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
			"[]string", "url", "logSeverity", "protocol", "resolvedPath"},
		func(dt string) bool {
			return dt == param.Type
		},
	)
	if idx == -1 {
		return fmt.Errorf("unsupported datatype: %s", param.Type)
	}

	return nil
}

func validateParams(params []Param) error {
	err := validateForDuplicates(params, func(param Param) string { return param.FlagName })
	if err != nil {
		return fmt.Errorf("duplicate flag names found: %w", err)
	}
	err = validateForDuplicates(params, func(param Param) string { return param.ConfigPath })
	if err != nil {
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
