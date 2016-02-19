// Copyright 2016 Google Inc. All Rights Reserved.
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

package gcsx_test

import (
	"strings"
	"testing"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
)

func TestContentTypeBucket_CreateObject(t *testing.T) {
	testCases := []struct {
		name     string
		request  string // ContentType in request
		expected string // Expected final type
	}{
		/////////////////
		// No extension
		/////////////////

		0: {
			name:     "foo/bar",
			request:  "",
			expected: "",
		},

		1: {
			name:     "foo/bar",
			request:  "image/jpeg",
			expected: "image/jpeg",
		},

		//////////////////////
		// Unknown extension
		//////////////////////

		2: {
			name:     "foo/bar.asdf",
			request:  "",
			expected: "",
		},

		3: {
			name:     "foo/bar.asdf",
			request:  "image/jpeg",
			expected: "image/jpeg",
		},

		//////////////////////
		// Known extension
		//////////////////////

		4: {
			name:     "foo/bar.jpg",
			request:  "",
			expected: "image/jpeg",
		},

		5: {
			name:     "foo/bar.jpg",
			request:  "text/plain",
			expected: "text/plain",
		},
	}

	for i, tc := range testCases {
		// Set up a bucket.
		bucket := gcsfake.NewFakeBucket(timeutil.RealClock(), "")

		// Create the object.
		req := &gcs.CreateObjectRequest{
			Name:        tc.name,
			ContentType: tc.request,
			Contents:    strings.NewReader(""),
		}

		o, err := bucket.CreateObject(context.Background(), req)
		if err != nil {
			t.Fatalf("Test case %d: CreateObject: %v", i, err)
		}

		// Check the content type.
		if got, want := o.ContentType, tc.expected; got != want {
			t.Errorf("Test case %d: o.ContentType is %q, want %q", i, got, want)
		}
	}
}

func TestContentTypeBucket_ComposeObjects(t *testing.T) {
	t.Fatal("TODO")
}
