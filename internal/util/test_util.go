// Copyright 2023 Google LLC
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

// Contains common utility methods for test packages. Methods in this file
// are not tested.

package util

import "crypto/rand"

func GenerateRandomBytes(length int) []byte {
	if length <= 0 {
		return nil
	}
	randBytes := make([]byte, length)
	chunk := 64 * 1024
	if chunk > length {
		chunk = length
	}
	_, _ = rand.Read(randBytes[:chunk])
	for i := range chunk {
		randBytes[i] = 'A' + (randBytes[i] % 26)
	}
	for copied := chunk; copied < length; {
		copied += copy(randBytes[copied:], randBytes[:copied])
	}
	return randBytes
}

// ConvertReadResponseToBytes concatenates the data slices from a read response into a single byte slice.
func ConvertReadResponseToBytes(data [][]byte, size int) []byte {
	buf := make([]byte, size)
	bytesCopied := 0
	for _, dataSlice := range data {
		bytesCopied += copy(buf[bytesCopied:], dataSlice)
	}
	return buf
}
