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

// FallbackToAnotherReaderError is returned when data could not be retrieved
// from the current reader, indicating that the caller should attempt to fall back
// to an alternative reader.
type FallbackToAnotherReaderError struct{}

func (e *FallbackToAnotherReaderError) Error() string {
	return "fallback to another reader is required."
}

var FallbackToAnotherReader = &FallbackToAnotherReaderError{}

// GCSReaderReq represents the request parameters needed to read a data from a GCS object.
type GCSReaderReq struct {
	// Buffer is provided by jacobsa/fuse and should be filled with data from the object.
	Buffer []byte

	// Offset specifies the starting position in the object from where data should be read.
	Offset int64

	// The end offset that needs to be fetched from GCS.
	EndOffset int64
}

// ObjectData represents the response returned as part of a ReadAt call.
// It includes the actual data read and its size.
type ObjectData struct {
	// DataBuf contains the bytes read from the object.
	DataBuf []byte

	// Size indicates how many bytes were read into DataBuf.
	Size int
}
