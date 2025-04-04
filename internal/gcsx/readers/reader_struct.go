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

package readers

type FallbackToAnotherReaderError struct{}

func (e *FallbackToAnotherReaderError) Error() string {
	return "fallback to another reader is not allowed"
}

// Usage
var FallbackToAnotherReader = &FallbackToAnotherReaderError{}

type GCSReaderReq struct {
	Buffer      []byte
	Offset      int64
	EndPosition int64
}

// ObjectData specifies the response returned as part of ReadAt call.
type ObjectData struct {
	// Byte array populated with the requested data.
	DataBuf []byte
	// Size of the data returned.
	Size int
}
