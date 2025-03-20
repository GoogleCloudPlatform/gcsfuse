// Copyright 2025 Google LLC
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

package gcsx

// Feedback struct contains total read bytes, and boolean to signify if not read completely
// from the previous reader.
type Feedback struct {
	TotalReadBytes int64
	ReadCompletely bool
	LastOffsetRead int64
}

// ReadSizeProvider is an interface that provides the size of the next read request.
type ReadSizeProvider interface {
	// GetNextReadSize returns the size of the next read request, given the current offset.
	// It also returns an error if the offset is invalid.
	GetNextReadSize(offset int64) (size int64, err error)

	// ReadType returns the current 
	ReadType() string

	// Provide feedback of previous reader request.
	ProvideFeedback(f *Feedback)
}
