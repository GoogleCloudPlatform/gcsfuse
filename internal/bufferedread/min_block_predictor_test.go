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

func TestDefaultMinBlockToStartBufferedReadSuccess(t *testing.T) {
	tests := []struct {
		name       string
		blockSize  int64
		objectSize uint64
		expected   uint
	}{
		{"Small block size", 2 * util.MiB, 20 * util.MiB, 6},
		{"Medium block size", 4 * util.MiB, 40 * util.MiB, 6},
		{"Large block size", 8 * util.MiB, 100 * util.MiB, 4},
		{"Very large block size", 16 * util.MiB, 1000 * util.MiB, 2},
		{"Object size smaller than block size", 4 * util.MiB, 2 * util.MiB, 1},
		{"Object size equal to block size", 4 * util.MiB, 4 * util.MiB, 1},
		{"Object size less than two blocks", 4 * util.MiB, 6 * util.MiB, 2},
		{"Object size exactly two blocks", 4 * util.MiB, 8 * util.MiB, 2},
		{"Object size larger than two blocks", 4 * util.MiB, 20 * util.MiB, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaultPredictor := &defaultMinBlockPredictor{}

			result, err := defaultPredictor.PredictMinBlockCount(tt.blockSize, tt.objectSize)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)

		})
	}
}

func TestDefaultMinBlockToStartBufferedReadFailure(t *testing.T) {
	tests := []struct {
		name       string
		blockSize  int64
		objectSize uint64
	}{
		{"Zero block size", 0, 10 * util.MiB},
		{"Negative block size", -1, 10 * util.MiB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaultPredictor := &defaultMinBlockPredictor{}

			_, err := defaultPredictor.PredictMinBlockCount(tt.blockSize, tt.objectSize)

			assert.Error(t, err)
		})
	}
}

func TestStaticMinBlockToStartBufferedReadSuccess(t *testing.T) {
	tests := []struct {
		name           string
		staticBlockCnt uint
		blockSize      int64
		objectSize     uint64
		expected       uint
	}{
		{"Static block count 2", 2, 4 * util.MiB, 20 * util.MiB, 2},
		{"Static block count 4", 4, 8 * util.MiB, 40 * util.MiB, 4},
		{"Static block count 6", 6, 2 * util.MiB, 60 * util.MiB, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaultPredictor := &staticMinBlockPredictor{minBlockCount: tt.staticBlockCnt}

			result, err := defaultPredictor.PredictMinBlockCount(tt.blockSize, tt.objectSize)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)

		})
	}
}

func TestStaticMinBlockToStartBufferedReadFailure(t *testing.T) {
	tests := []struct {
		name           string
		staticBlockCnt uint
		blockSize      int64
		objectSize     uint64
	}{
		{"Zero block-size", 0, 4 * util.MiB, 20 * util.MiB},
		{"Negative block-size", 0, -1, 10 * util.MiB},
		{"Static block count not set", 0, 4 * util.MiB, 20 * util.MiB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaultPredictor := &staticMinBlockPredictor{minBlockCount: tt.staticBlockCnt}

			_, err := defaultPredictor.PredictMinBlockCount(tt.blockSize, tt.objectSize)

			assert.Error(t, err)

		})
	}
}
