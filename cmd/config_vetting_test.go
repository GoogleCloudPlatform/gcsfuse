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

	"github.com/stretchr/testify/assert"
)

func TestEnableEmptyManagedFoldersResolution(t *testing.T) {
	testcases := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "enable-hns set to true",
			args:     []string{"--enable-hns"},
			expected: true,
		},
		{
			name:     "enable-hns set to true and enable-empty-managed-folders set to false",
			args:     []string{"--enable-hns", "--enable-empty-managed-folders=false"},
			expected: true,
		},
		{
			name:     "enable-hns set to false and enable-empty-managed-folders set to true",
			args:     []string{"--enable-hns=false", "--enable-empty-managed-folders=true"},
			expected: true,
		},
		{
			name:     "both enable-hns and enable-empty-managed-folders set to false",
			args:     []string{"--enable-hns=false", "--enable-empty-managed-folders=false"},
			expected: false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := getConfigObject(t, tc.args)

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expected, c.List.EnableEmptyManagedFolders)
			}
		})
	}
}
