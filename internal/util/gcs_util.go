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
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

var (
	unsupportedObjectNameSubstrings = []string{"//", "/./", "/../"}
	unsupportedObjectNamePrefixes   = []string{"/", "./", "../"}
	unsupportedObjectNames          = []string{"", ".", ".."}
)

// isUnsupportedObjectName returns true if the passed
// string is a valid GCS object name or prefix,
// which is unsupported in GCSFuse.
func isUnsupportedObjectName(name string) bool {
	for _, substring := range unsupportedObjectNameSubstrings {
		if strings.Contains(name, substring) {
			return true
		}
	}
	for _, prefix := range unsupportedObjectNamePrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	for _, unsupportedObjectName := range unsupportedObjectNames {
		if name == unsupportedObjectName {
			return true
		}
	}
	return false
}

// RemoveUnsupportedObjectsFromListing is a utility to ignore unsupported
// GCS object names such as those containing `//` in their names.
// As an example, GCS can have two different objects a//b and a/b at the same time
// in the same bucket. In linux FS however, both paths are same as a/b.
// So, GCSFuse will ignore objects with names like a//b to avoid causing `input/output error` in
// linux FS.
func RemoveUnsupportedObjectsFromListing(listing *gcs.Listing) (newListing *gcs.Listing, removedListing *gcs.Listing) {
	newListing = &gcs.Listing{}
	removedListing = &gcs.Listing{}
	for _, collapsedRun := range listing.CollapsedRuns {
		if !isUnsupportedObjectName(collapsedRun) {
			newListing.CollapsedRuns = append(newListing.CollapsedRuns, collapsedRun)
		} else {
			removedListing.CollapsedRuns = append(removedListing.CollapsedRuns, collapsedRun)
		}
	}
	for _, object := range listing.Objects {
		if !isUnsupportedObjectName(object.Name) {
			newListing.Objects = append(newListing.Objects, object)
		} else {
			removedListing.Objects = append(removedListing.Objects, object)
		}
	}
	newListing.ContinuationToken = listing.ContinuationToken
	return newListing, removedListing
}
