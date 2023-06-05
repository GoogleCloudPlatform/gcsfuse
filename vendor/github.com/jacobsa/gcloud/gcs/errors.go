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

package gcs

import "fmt"

// A *NotFoundError value is an error that indicates an object name or a
// particular generation for that name were not found.
type NotFoundError struct {
	Err error
}

func (nfe *NotFoundError) Error() string {
	return fmt.Sprintf("gcs.NotFoundError: %v", nfe.Err)
}

// A *PreconditionError value is an error that indicates a precondition failed.
type PreconditionError struct {
	Err error
}

// Returns pe.Err.Error().
func (pe *PreconditionError) Error() string {
	return fmt.Sprintf("gcs.PreconditionError: %v", pe.Err)
}
