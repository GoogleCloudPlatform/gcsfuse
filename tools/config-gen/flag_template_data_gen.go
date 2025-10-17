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
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type flagTemplateData struct {
	Param
	// The pFlag function to invoke in order to add the flag.
	Fn string
	// The path to the field in the Go Config struct, e.g. "MetadataCache.TtlSecs"
	GoPath string
	// The Go type of the field, e.g. "int64"
	GoType string
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

func capitalize(name string) string {
	var buf strings.Builder
	for w := range strings.SplitSeq(name, "-") {
		buf.WriteString(cases.Title(language.English).String(w))
	}
	return buf.String()
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
	case "octal", "logSeverity", "protocol", "resolvedPath":
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
	// Compute the Go field path, e.g., "metadata-cache.ttl-secs" -> "MetadataCache.TtlSecs"
	var goPathParts []string
	for part := range strings.SplitSeq(p.ConfigPath, ".") {
		goPathParts = append(goPathParts, capitalize(part))
	}
	goPath := strings.Join(goPathParts, ".")
	return flagTemplateData{
		Param:  p,
		Fn:     fn,
		GoPath: goPath,
		GoType: getGoDataType(p.Type), // Assumes getGoDataType is available
	}, nil
}
