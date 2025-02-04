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

package gcs

import "io"

// An interface to generalize the MultiRangeDownloader
// structure in go-storage module to ease our testing.
type MultiRangeDownloader interface {
	Add(output io.Writer, offset, length int64, callback func(int64, int64, error))
	Close() error
	Wait()
}
