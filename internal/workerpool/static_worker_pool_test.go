// Copyright 2024 Google LLC
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

package workerpool

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStaticWorkerPool_Success(t *testing.T) {
	tests := []struct {
		name           string
		priorityWorker uint32
		normalWorker   uint32
	}{
		{"valid_workers", 5, 10},
		{"zero_normal_worker", 1, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := NewStaticWorkerPool(uint32(tc.priorityWorker), uint32(tc.normalWorker))

			assert.NoError(t, err)
			assert.NotNil(t, pool)
			pool.Stop() // Clean up
		})
	}
}

func TestNewStaticWorkerPool_Failure(t *testing.T) {
	pool, err := NewStaticWorkerPool(0, 0)

	assert.Error(t, err)
	assert.Nil(t, pool)
	pool.Stop() // Clean up
}
