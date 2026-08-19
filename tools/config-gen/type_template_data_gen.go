/*
 * Copyright 2026 Google LLC
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
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	cfgSegmentRegex = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9\-]*`)
)

type fieldInfo struct {
	TypeName   string
	FieldName  string
	DataType   string
	ConfigPath string
}

type typeTemplateData struct {
	// Name of the type
	TypeName string
	// Fields that are to be included in the type.
	Fields []fieldInfo
}

type protoFieldTemplateData struct {
	ProtoType      string
	ProtoFieldName string
	ProtoTag       int
}

func computeProtoFields(params []Param) []protoFieldTemplateData {
	var fields []protoFieldTemplateData
	for _, p := range params {
		if p.ConfigPath == "" || p.ProtoFieldName == "" || p.ProtoType == "" {
			continue
		}
		fields = append(fields, protoFieldTemplateData{
			ProtoType:      p.ProtoType,
			ProtoFieldName: p.ProtoFieldName,
			ProtoTag:       p.ProtoTag,
		})
	}
	slices.SortFunc(fields, func(a, b protoFieldTemplateData) int {
		return cmp.Compare(a.ProtoTag, b.ProtoTag)
	})
	return fields
}

func formatReservedTags(tags []int) string {
	if len(tags) == 0 {
		return ""
	}
	sorted := slices.Clone(tags)
	slices.Sort(sorted)
	var strTags []string
	for _, t := range sorted {
		strTags = append(strTags, strconv.Itoa(t))
	}
	return strings.Join(strTags, ", ")
}

func capitalizeIdentifier(name string) (string, error) {
	if !cfgSegmentRegex.MatchString(name) {
		return "", fmt.Errorf("%s is not a supported name", name)
	}

	// For the purposes of capitalization, both "." and "-" are equivalent.
	name = strings.ReplaceAll(name, ".", "-")
	var buf strings.Builder
	for w := range strings.SplitSeq(name, "-") {
		// Capitalize the first letter and concatenate.
		buf.WriteString(cases.Title(language.English).String(w))
	}
	return buf.String(), nil
}

func getGoDataType(dt string) string {
	switch dt {
	case "octal":
		return "Octal"
	case "logSeverity":
		return "LogSeverity"
	case "protocol":
		return "Protocol"
	case "resolvedPath":
		return "ResolvedPath"
	case "duration":
		return "time.Duration"
	case "int":
		return "int64"
	case "[]int":
		return "[]int64"
	case "directPathStrategy":
		return "DirectPathStrategy"
	default:
		return dt
	}
}

// Returns a flat list with one entry for each field that needs to be created and the corresponding type.
// A config path of x.y.z for a param of type int would return the following entries
// 1. {TypeName: Config, FieldName: X, DataType: XConfig, ConfigPath: x}
// 2. {TypeName: XConfig, FieldName: Y, DataType: YXConfig, ConfigPath: y}
// 3. {TypeName: YXConfig, FieldName: Z, DataType: int, ConfigPath: z}
func computeFields(param Param) ([]fieldInfo, error) {
	segments := strings.Split(param.ConfigPath, ".")
	fieldInfos := make([]fieldInfo, 0, len(segments))
	typeName := "Config"
	for idx, s := range segments {
		fld, err := capitalizeIdentifier(s)
		if err != nil {
			return nil, err
		}

		var dt string
		if idx == len(segments)-1 {
			// Dealing with leaf field here.
			dt = getGoDataType(param.Type)
		} else {
			// Not a leaf field.
			tn, err := capitalizeIdentifier(s)
			if err != nil {
				return nil, err
			}
			dt = tn + typeName
		}
		fieldInfos = append(fieldInfos, fieldInfo{
			TypeName:   typeName,
			FieldName:  fld,
			DataType:   dt,
			ConfigPath: s,
		})
		typeName = dt
	}

	return fieldInfos, nil
}

func constructTypeTemplateData(paramsConfig []Param) ([]typeTemplateData, error) {
	var fields []fieldInfo
	for _, p := range paramsConfig {
		// ConfigPath can be empty for deprecated flags.
		if p.ConfigPath == "" {
			continue
		}
		f, err := computeFields(p)
		if err != nil {
			return nil, err
		}

		fields = append(fields, f...)
	}

	ttf := make(map[string][]fieldInfo)
	for _, f := range fields {
		ttf[f.TypeName] = append(ttf[f.TypeName], f)
	}

	var ttd []typeTemplateData
	for k, v := range ttf {
		// Sort field names for reliable ordering.
		slices.SortFunc(v, func(i, j fieldInfo) int {
			return cmp.Compare(i.FieldName, j.FieldName)
		})

		// Remove duplicates.
		compacted := slices.Compact(v)

		ttd = append(ttd, typeTemplateData{
			TypeName: k,
			Fields:   compacted,
		})
	}
	// Sort type names for reliable ordering.
	slices.SortFunc(ttd, func(i, j typeTemplateData) int {
		return cmp.Compare(i.TypeName, j.TypeName)
	})
	return ttd, nil
}
