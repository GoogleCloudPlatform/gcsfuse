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

// ReadFeedback provides feedback on a completed read operation.  This information
// is used to adapt future read sizes.
type ReadFeedback struct {
	TotalBytesRead int64 `json:"totalBytesRead"` // Total bytes read since the last feedback.
	ReadComplete   bool  `json:"readComplete"`   // True if the entire requested range was read.
	LastOffset     int64 `json:"lastOffset"`     // The last byte offset read.
}

// ReadSizeProvider determines the optimal size for subsequent read requests
// based on the observed read patterns.  This helps optimize performance for
// both sequential and random access patterns.
type ReadSizeProvider interface {
	// GetNextReadSize determines the optimal size for the next read request,
	// given the current offset.  It returns an error if the offset is invalid
	// or if an internal error occurs.
	GetNextReadSize(offset int64) (size int64, err error)

	// GetReadType returns the currently detected read type ("sequential" or "random").
	GetReadType() string

	// ProvideFeedback updates the read size provider with feedback from a completed
	// read operation.  This allows the provider to adapt its read size strategy
	// based on the observed patterns.
	ProvideFeedback(feedback *ReadFeedback)
}
