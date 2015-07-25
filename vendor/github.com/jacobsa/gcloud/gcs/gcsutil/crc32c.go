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

package gcsutil

import "hash/crc32"

var crc32cTable = crc32.MakeTable(crc32.Castagnoli)

// Return a value appropriate for placing in CreateObjectRequest.CRC32C for the
// given object contents.
func CRC32C(contents []byte) *uint32 {
	checksum := crc32.Checksum(contents, crc32cTable)
	return &checksum
}
