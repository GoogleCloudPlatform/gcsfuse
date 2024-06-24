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
	"fmt"
	"strings"
	"time"
)

type flagTemplateData struct {
	Param
	// The pFlag function to invoke in order to add the flag.
	Fn string
	// The leaf config name for the flag - to be used as a key to bind cobra flag
	// with viper config (using BindPFlag).
	ConfigName string
}

func computeFlagTemplateData(paramsConfig []Param) ([]flagTemplateData, error) {
	var flgTemplate []flagTemplateData
	for _, p := range paramsConfig {
		td, err := computeFlagTemplateDataForParam(p)
		if err != nil {
			return nil, err
		}
		flgTemplate = append(flgTemplate, td)
	}
	return flgTemplate, nil
}

func computeFlagTemplateDataForParam(p Param) (flagTemplateData, error) {
	var defaultValue string
	var fn string
	switch p.Type {
	case "int":
		if p.DefaultValue == "" {
			defaultValue = "0"
		} else {
			defaultValue = p.DefaultValue
		}
		fn = "IntP"
	case "float64":
		if p.DefaultValue == "" {
			defaultValue = "0.0"
		} else {
			defaultValue = p.DefaultValue
		}
		fn = "Float64P"
	case "bool":
		if p.DefaultValue == "" {
			defaultValue = "false"
		} else {
			defaultValue = p.DefaultValue
		}
		fn = "BoolP"
	case "duration":
		if p.DefaultValue == "" {
			defaultValue = "0s"
		} else {
			defaultValue = p.DefaultValue
		}
		dur, err := time.ParseDuration(defaultValue)
		if err != nil {
			return flagTemplateData{}, err
		}
		defaultValue = fmt.Sprintf("%d * time.Nanosecond", dur.Nanoseconds())
		fn = "DurationP"
	case "octal", "url", "logSeverity", "protocol", "resolvedPath":
		fallthrough
	case "string":
		defaultValue = fmt.Sprintf("%q", p.DefaultValue)
		fn = "StringP"
	case "[]int":
		defaultValue = fmt.Sprintf("[]int{%s}", p.DefaultValue)
		fn = "IntSliceP"
	case "[]string":
		defaultValue = fmt.Sprintf("[]string{%s}", p.DefaultValue)
		fn = "StringSliceP"
	default:
		return flagTemplateData{}, fmt.Errorf("unhandled type: %s", p.Type)
	}
	p.DefaultValue = defaultValue
	// Usage string safely escaped with Go syntax.
	p.Usage = fmt.Sprintf("%q", p.Usage)
	configParts := strings.Split(p.ConfigPath, ".")
	return flagTemplateData{
		Param:      p,
		Fn:         fn,
		ConfigName: configParts[len(configParts)-1],
	}, nil
}
