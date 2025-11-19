// Copyright 2024 Google LLC
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

package storageutil_test

import (
	"fmt"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type GcsUtilTest struct {
	suite.Suite
}

func TestGcsUtil(t *testing.T) {
	suite.Run(t, new(GcsUtilTest))
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////
func (ts *GcsUtilTest) TestIsUnsupportedPathName() {
	cases := []struct {
		name          string
		isUnsupported bool
	}{
		{
			name:          "foo",
			isUnsupported: false,
		},
		{
			name:          "foo/bar",
			isUnsupported: false,
		},
		{
			name:          "foo//bar",
			isUnsupported: true,
		},
		{
			name:          "abc/",
			isUnsupported: false,
		},
		{
			name:          "abc//",
			isUnsupported: true,
		},
		{
			name:          "/foo",
			isUnsupported: true,
		},
		{
			name:          "/",
			isUnsupported: true,
		},
		{
			name:          "",
			isUnsupported: true,
		},
		{
			name:          "foo/.",
			isUnsupported: true,
		},
		{
			name:          "foo/..",
			isUnsupported: true,
		},
		{
			name:          "foo/./",
			isUnsupported: true,
		},
		{
			name:          "foo/../",
			isUnsupported: true,
		},
		{
			name:          "foo/.config",
			isUnsupported: false,
		},
		{
			name:          "foo/c..d",
			isUnsupported: false,
		},
	}

	for _, tc := range cases {
		ts.Run(fmt.Sprintf("name=%s", tc.name), func() {
			assert.Equal(ts.T(), tc.isUnsupported, IsUnsupportedPath(tc.name))
		})
	}
}
