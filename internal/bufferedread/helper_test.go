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

package bufferedread

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/stretchr/testify/assert"
)

func TestMinBlockToStartBufferedReadSuccess(t *testing.T) {
	tests := []struct {
		name      string
		blockSize int64
		expected  int
	}{
		{"Small block size", 1 * util.MiB, 6},
		{"Medium block size", 4 * util.MiB, 6},
		{"Large block size", 8 * util.MiB, 4},
		{"Very large block size", 16 * util.MiB, 2},
		{"Huge block size", 32 * util.MiB, 2},
		{"Gigantic block size", 64 * util.MiB, 2},
		{"Excessive block size", 128 * util.MiB, 2},
		{"Massive block size", 256 * util.MiB, 2},
		{"Enormous block size", 512 * util.MiB, 2},
		{"Colossal block size", 1024 * util.MiB, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := minBlockToStartBufferedRead(tt.blockSize)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)

		})
	}
}

func TestMinBlockToStartBufferedReadFailure(t *testing.T) {
	tests := []struct {
		name      string
		blockSize int64
	}{
		{"Zero block size", 0},
		{"Negative block size", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := minBlockToStartBufferedRead(tt.blockSize)

			assert.Error(t, err)
		})
	}
}
