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

package cfg

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/stretchr/testify/assert"
)

func TestOctalTypeInConfigStringify(t *testing.T) {
	c := Config{
		FileSystem: FileSystemConfig{
			DirMode: 0755,
		},
	}

	str, err := util.Stringify(&c)

	if assert.NoError(t, err) {
		assert.Equal(t, "file-system:\n    dir-mode: \"755\"\n", str)
	}
}
func TestOctalStringify(t *testing.T) {
	o := Octal(0765)

	str, err := util.Stringify(&o)

	if assert.NoError(t, err) {
		assert.Equal(t, "\"765\"\n", str)
	}
}
