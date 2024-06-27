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

type defaultsTestParamData struct {
	DefVal      string
	FieldName   string
	HandleError bool
}

type defaultsTestTemplateData struct {
	Assign string
	Data   []defaultsTestParamData
}

func computeDefaultsTestTemplateData(dt []datatype) defaultsTestTemplateData {
	td := make([]defaultsTestParamData, 0, len(dt))
	for _, d := range dt {
		if d.param().ConfigPath == "" {
			continue
		}
		td = append(td, defaultsTestParamData{
			DefVal:      d.testDefaultValue(),
			FieldName:   d.param().accessor(),
			HandleError: d.handleErrorInDefaultTest(),
		})
	}
	return defaultsTestTemplateData{
		Data:   td,
		Assign: ":=",
	}
}
