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

package workerpool

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyTask struct {
	executed bool
}

func (d *dummyTask) Execute() {
	d.executed = true
}

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
			pool, err := NewStaticWorkerPool(uint32(tc.priorityWorker), uint32(tc.normalWorker), 100)

			assert.NoError(t, err)
			assert.NotNil(t, pool)
			assert.Equal(t, tc.priorityWorker, pool.priorityWorker)
			assert.Equal(t, tc.normalWorker, pool.normalWorker)
			pool.Stop() // Clean up
		})
	}
}

func TestNewStaticWorkerPool_Failure(t *testing.T) {
	pool, err := NewStaticWorkerPool(0, 0, 0)

	assert.Error(t, err)
	assert.Nil(t, pool)
	assert.Panics(t, pool.Stop, "Stop should panic if pool is nil")
}

func TestStaticWorkerPool_Start(t *testing.T) {
	pool, err := NewStaticWorkerPool(2, 3, 5)
	require.NoError(t, err)
	require.NotNil(t, pool)

	pool.Start()
	defer pool.Stop()

	// Add a task in the channel and later see, that channel will be empty after execution.
	dt := &dummyTask{}
	pool.priorityCh <- dt
	// Wait for the task to be executed.
	assert.Eventually(t, func() bool {
		return dt.executed
	}, 100*time.Millisecond, time.Millisecond, "Task was not executed in time.")
	assert.Equal(t, 0, len(pool.priorityCh), "Priority channel should be empty after task execution.")
}

func TestStaticWorkerPool_SchedulePriorityTask(t *testing.T) {
	pool, err := NewStaticWorkerPool(2, 3, 5)
	require.NoError(t, err)
	require.NotNil(t, pool)
	pool.Start()
	defer pool.Stop()

	dt := &dummyTask{}
	pool.Schedule(true, dt)

	// Wait for the task to be executed.
	assert.Eventually(t, func() bool {
		return dt.executed
	}, 100*time.Millisecond, time.Millisecond, "Task was not executed in time.")
}

func TestStaticWorkerPool_ScheduleNormalTask(t *testing.T) {
	pool, err := NewStaticWorkerPool(2, 3, 5)
	require.NoError(t, err)
	require.NotNil(t, pool)
	pool.Start()
	defer pool.Stop()

	dt := &dummyTask{}
	pool.Schedule(false, dt)

	// Wait for the task to be executed.
	require.Eventually(t, func() bool {
		return dt.executed
	}, 100*time.Millisecond, time.Millisecond, "Priority task was not executed in time.")
}

func TestStaticWorkerPool_HighNumberOfTasks(t *testing.T) {
	pool, err := NewStaticWorkerPool(5, 10, 15)
	require.NoError(t, err)
	require.NotNil(t, pool)
	pool.Start()
	defer pool.Stop()

	// Schedule a large number of tasks.
	for i := 0; i < 100; i++ {
		dt := &dummyTask{}
		pool.Schedule(i%2 == 0, dt) // Alternate between priority and normal tasks
	}

	// Wait for all tasks to be executed.
	assert.Eventually(t, func() bool {
		return len(pool.priorityCh) == 0 && len(pool.normalCh) == 0
	}, 500*time.Millisecond, 10*time.Millisecond, "Not all tasks were executed in time.")
}

func TestStaticWorkerPool_ScheduleAfterStop(t *testing.T) {
	pool, err := NewStaticWorkerPool(2, 3, 5)
	require.NoError(t, err)
	require.NotNil(t, pool)
	pool.Start()

	pool.Stop()

	assert.Panics(t, func() { pool.Schedule(true, &dummyTask{}) }, "Should panic when scheduling after cancel.")
}

func TestStaticWorkerPool_Stop(t *testing.T) {
	pool, err := NewStaticWorkerPool(2, 3, 5)
	require.NoError(t, err)
	require.NotNil(t, pool)
	pool.Start()

	// Stop the pool and check if channels are closed.
	pool.Stop()

	assert.Panics(t, func() { pool.stop <- true }, "normalCh channel is not closed.")
	assert.Panics(t, func() { pool.normalCh <- &dummyTask{} }, "normalCh channel is not closed.")
	assert.Panics(t, func() { pool.priorityCh <- &dummyTask{} }, "priorityCh channel is not closed.")
}

func TestNewStaticWorkerPoolForCurrentCPU(t *testing.T) {
	readGlobalMaxBlocks := int64(100)

	pool, err := NewStaticWorkerPoolForCurrentCPU(readGlobalMaxBlocks)

	require.NoError(t, err)
	require.NotNil(t, pool)
	defer pool.Stop()
	staticPool, ok := pool.(*staticWorkerPool)
	require.True(t, ok, "The returned pool should be of type *staticWorkerPool")
	// Re-calculate the expected number of workers based on the real CPU count
	// to verify the logic.
	totalWorkers := 3 * runtime.NumCPU()
	if cappedWorkers := (11*readGlobalMaxBlocks + 9) / 10; int64(totalWorkers) > cappedWorkers {
		totalWorkers = int(cappedWorkers)
	}
	expectedPriorityWorkers := (totalWorkers + 9) / 10
	expectedNormalWorkers := totalWorkers - expectedPriorityWorkers
	dt := &dummyTask{}
	pool.Schedule(true, dt)
	assert.Equal(t, uint32(expectedPriorityWorkers), staticPool.priorityWorker)
	assert.Equal(t, uint32(expectedNormalWorkers), staticPool.normalWorker)
	// Verify that the pool is functional.
	assert.Eventually(t, func() bool { return dt.executed }, 100*time.Millisecond, time.Millisecond, "Task was not executed in time.")
}

func Test_newStaticWorkerPoolForCurrentCPU(t *testing.T) {
	testCases := []struct {
		name                    string
		readGlobalMaxBlocks     int64
		mockNumCPU              func() int
		expectedPriorityWorkers uint32
		expectedNormalWorkers   uint32
	}{
		{
			name:                "low CPU count, workers not capped",
			readGlobalMaxBlocks: 100,
			mockNumCPU:          func() int { return 2 },
			// totalWorkers = 3*2=6. priority=ceil(0.1*6)=1, normal=5.
			expectedPriorityWorkers: 1,
			expectedNormalWorkers:   5,
		},
		{
			name:                "high CPU count, workers capped by max blocks",
			readGlobalMaxBlocks: 50,
			mockNumCPU:          func() int { return 100 },
			// totalWorkers = 3*100=300, capped to ceil(1.1*50)=55. priority=ceil(0.1*55)=6, normal=49.
			expectedPriorityWorkers: 6,
			expectedNormalWorkers:   49,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := newStaticWorkerPoolForCurrentCPU(tc.readGlobalMaxBlocks, tc.mockNumCPU)

			require.NoError(t, err)
			require.NotNil(t, pool)
			defer pool.Stop()
			staticPool, ok := pool.(*staticWorkerPool)
			require.True(t, ok, "The returned pool should be of type *staticWorkerPool")
			assert.Equal(t, tc.expectedPriorityWorkers, staticPool.priorityWorker)
			assert.Equal(t, tc.expectedNormalWorkers, staticPool.normalWorker)
		})
	}
}
