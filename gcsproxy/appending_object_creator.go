// Copyright 2015 Google Inc. All Rights Reserved.
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

package gcsproxy

import "github.com/jacobsa/gcloud/gcs"

// Create an objectCreator that accepts a source object and the contents that
// should be "appended" to it, storing temporary objects using the supplied
// prefix.
func newAppendObjectCreator(
	prefix string,
	bucket gcs.Bucket) (oc objectCreator) {
	// TODO(jacobsa): Add units tests that use a mock bucket.
	//
	// TODO(jacobsa): Figure out what to do about garbage collection. Probably
	// hoist that all the way to main.go.
	panic("TODO")
}
