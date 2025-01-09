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

func GenerateRandomBytes(length int) []byte {
	randBytes := make([]byte, length)
	for i := 0; i < length; i++ {
		randBytes[i] = byte(rand.Intn(26) + 65)
	}
	return randBytes
}
