// Copyright 2024 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"strings"
)

type flagParsingTestParamData struct {
	TestName string
	Args     string
	Accessor string
	Expected string
}

type flagParsingTestTemplate struct {
	Data   []flagParsingTestParamData
	Assign string
}

func computeFlagParsingTestTemplateData(dtype []datatypeSpec) flagParsingTestTemplate {
	tmplData := flagParsingTestTemplate{
		Data:   []flagParsingTestParamData{},
		Assign: ":=",
	}
	for _, d := range dtype {
		if d.param().ConfigPath == "" {
			continue
		}
		for idx, tc := range d.testCases() {
			var escapedArgs []string
			for _, s := range tc.Args {
				escapedArgs = append(escapedArgs, fmt.Sprintf("%q", s))
			}
			td := flagParsingTestParamData{
				TestName: fmt.Sprintf("Test flag: %s parsing #%d", d.param().FlagName, idx),
				Args:     fmt.Sprintf("[]string{\"gcsfuse\", \"abc\", %s}", strings.Join(escapedArgs, ", ")),
				Accessor: d.param().accessor(),
				Expected: tc.Expected,
			}
			tmplData.Data = append(tmplData.Data, td)
		}
	}
	return tmplData
}
