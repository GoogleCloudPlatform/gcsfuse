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
	"bufio"
	"cmp"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	cfgSegmentRegex   = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9\-]*`)
	protoFieldRegex   = regexp.MustCompile(`^\s*(?:repeated\s+)?[\w\.]+\s+(\w+)\s*=\s*(\d+)\s*;`)
	protoMessageRegex = regexp.MustCompile(`^\s*message\s+(\w+)`)
)

func parseExistingProtoTags(protoFilePath string) (tags map[string]map[string]int, err error) {
	var file *os.File
	file, err = os.Open(protoFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]map[string]int), nil
		}
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close file: %w", closeErr)
		}
	}()

	tags = make(map[string]map[string]int)
	var currentMessage string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if match := protoMessageRegex.FindStringSubmatch(line); match != nil {
			currentMessage = match[1]
			tags[currentMessage] = make(map[string]int)
			continue
		}

		if currentMessage != "" {
			if match := protoFieldRegex.FindStringSubmatch(line); match != nil {
				fieldName := match[1]
				tag, _ := strconv.Atoi(match[2])
				tags[currentMessage][fieldName] = tag
			} else if strings.Contains(line, "}") {
				currentMessage = ""
			}
		}
	}

	err = scanner.Err()
	return tags, err
}

type fieldInfo struct {
	TypeName       string
	FieldName      string
	DataType       string
	ConfigPath     string
	ProtoType      string
	ProtoFieldName string
	ProtoTag       int
}

type typeTemplateData struct {
	// Name of the type
	TypeName string
	// Fields that are to be included in the type.
	Fields []fieldInfo
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
// A config path of x.y.z for a param of type int would return the follow entries
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

		var dt, protoType, protoFieldName string
		if idx == len(segments)-1 {
			// Dealing with leaf field here.
			dt = getGoDataType(param.Type)
			protoType = param.ProtoType
			protoFieldName = param.ProtoFieldName
		} else {
			// Not a leaf field.
			tn, err := capitalizeIdentifier(s)
			if err != nil {
				return nil, err
			}

			dt = tn + typeName
			protoType = dt
			protoFieldName = strings.ReplaceAll(s, "-", "_")
		}
		fieldInfos = append(fieldInfos, fieldInfo{
			TypeName:       typeName,
			FieldName:      fld,
			DataType:       dt,
			ConfigPath:     s,
			ProtoType:      protoType,
			ProtoFieldName: protoFieldName,
		})
		typeName = dt
	}

	return fieldInfos, nil
}

func constructTypeTemplateData(paramsConfig []Param, existingTags map[string]map[string]int) ([]typeTemplateData, error) {
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

		// Assign tags from existingTags, or maxTag+1
		messageTags := existingTags[k]
		if messageTags == nil {
			messageTags = make(map[string]int)
		}

		maxTag := 0
		for _, tag := range messageTags {
			if tag > maxTag {
				maxTag = tag
			}
		}

		for i := range compacted {
			fName := compacted[i].ProtoFieldName
			if fName == "" {
				continue
			}
			if tag, exists := messageTags[fName]; exists {
				compacted[i].ProtoTag = tag
			} else {
				maxTag++
				compacted[i].ProtoTag = maxTag
				messageTags[fName] = maxTag
			}
		}

		ttd = append(ttd, typeTemplateData{
			TypeName: k,
			Fields:   compacted,
		},
		)
	}
	// Sort type names for reliable ordering.
	slices.SortFunc(ttd, func(i, j typeTemplateData) int {
		return cmp.Compare(i.TypeName, j.TypeName)
	})
	return ttd, nil
}
