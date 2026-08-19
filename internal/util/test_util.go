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

import "math/rand"

var seedPattern = func() []byte {
	b := make([]byte, 64*1024)
	for i := range b {
		b[i] = byte(rand.Intn(26) + 65)
	}
	return b
}()

func GenerateRandomBytes(length int) []byte {
	if length <= 0 {
		return nil
	}
	randBytes := make([]byte, length)
	chunk := copy(randBytes, seedPattern)
	for chunk < length {
		chunk += copy(randBytes[chunk:], randBytes[:chunk])
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
