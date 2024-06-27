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

package cmd

import (
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

{{$assign := .Assign}}
func TestArgParsing(t *testing.T) {
	testcases := []struct {
		name     string
		args     []string
		actualFn func(config cfg.Config) any
		expected any
	}{
	{{range .Data}}
		{
			name:     "{{.TestName}}",
			args:     {{.Args}},
			actualFn: func(config cfg.Config) any { return config.{{.Accessor}} },
			expected: {{.Expected}},
		},
		{{end}}
	}
	for _, tc {{$assign}} range testcases {
		tc {{$assign}} tc
		t.Run(tc.name, func(t *testing.T) {
			var actual cfg.Config
			cmd, err := NewRootCmd(func(c cfg.Config) error {
				actual = c
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			if assert.Nil(t, cmd.Execute()) {
				assert.EqualValues(t, tc.expected, tc.actualFn(actual))
			}
		})
	}
}