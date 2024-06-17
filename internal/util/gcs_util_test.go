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

package util

import (
	"fmt"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
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
	suite.Run(t, new(UtilTest))
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////
func (ts *GcsUtilTest) TestIsUnsupportedObjectName() {
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
			name:          "foo/./bar",
			isUnsupported: true,
		},
		{
			name:          "foo/../bar",
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
			name:          "abc/./",
			isUnsupported: true,
		},
		{
			name:          "abc/../",
			isUnsupported: true,
		},
		{
			name:          "/foo",
			isUnsupported: true,
		},
		{
			name:          "./foo",
			isUnsupported: true,
		},
		{
			name:          "../foo",
			isUnsupported: true,
		},
		{
			name:          "/",
			isUnsupported: true,
		},
		{
			name:          ".",
			isUnsupported: true,
		},
		{
			name:          "..",
			isUnsupported: true,
		},
	}

	for _, tc := range cases {
		ts.Run(fmt.Sprintf("name=%s", tc.name), func() {
			assert.Equal(ts.T(), tc.isUnsupported, isUnsupportedObjectName(tc.name))
		})
	}
}

func (t *GcsUtilTest) Test_RemoveUnsupportedObjectsFromListing() {
	createObject := func(name string) *gcs.Object {
		return &gcs.Object{Name: name}
	}

	origGcsListing := &gcs.Listing{
		CollapsedRuns:     []string{"/", "a/", "b//", "c/d/", "e//f/", "g/h//"},
		Objects:           []*gcs.Object{createObject("a"), createObject("/b"), createObject("c/d"), createObject("e//f"), createObject("g/h//i")},
		ContinuationToken: "hfdwefo",
	}
	expectedNewGcsListing := &gcs.Listing{
		CollapsedRuns:     []string{"a/", "c/d/"},
		Objects:           []*gcs.Object{createObject("a"), createObject("c/d")},
		ContinuationToken: "hfdwefo",
	}
	expectedRemovedGcsListing := &gcs.Listing{
		CollapsedRuns:     []string{"/", "b//", "e//f/", "g/h//"},
		Objects:           []*gcs.Object{createObject("/b"), createObject("e//f"), createObject("g/h//i")},
		ContinuationToken: "hfdwefo",
	}

	newGcsListing, removedGcsListing := RemoveUnsupportedObjectsFromListing(origGcsListing)
	assert.Equal(t.T(), *expectedNewGcsListing, *newGcsListing)
	assert.Equal(t.T(), *expectedRemovedGcsListing, *removedGcsListing)
}
