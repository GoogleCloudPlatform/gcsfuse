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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
			pool, err := NewStaticWorkerPool(uint32(tc.priorityWorker), uint32(tc.normalWorker))

			assert.NoError(t, err)
			assert.NotNil(t, pool)
			assert.Equal(t, tc.priorityWorker, pool.priorityWorker)
			assert.Equal(t, tc.normalWorker, pool.normalWorker)
			pool.Stop() // Clean up
		})
	}
}

func TestNewStaticWorkerPool_Failure(t *testing.T) {
	pool, err := NewStaticWorkerPool(0, 0)

	assert.Error(t, err)
	assert.Nil(t, pool)
	assert.Panics(t, pool.Stop, "Stop should panic if pool is nil")
}

func TestStaticWorkerPool_Start(t *testing.T) {
	pool, err := NewStaticWorkerPool(2, 3)
	assert.NoError(t, err)
	assert.NotNil(t, pool)

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
	pool, err := NewStaticWorkerPool(2, 3)
	assert.NoError(t, err)
	assert.NotNil(t, pool)
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
	pool, err := NewStaticWorkerPool(2, 3)
	assert.NoError(t, err)
	assert.NotNil(t, pool)
	pool.Start()
	defer pool.Stop()

	dt := &dummyTask{}
	pool.Schedule(false, dt)

	// Wait for the task to be executed.
	assert.Eventually(t, func() bool {
		return dt.executed
	}, 100*time.Millisecond, time.Millisecond, "Priority task was not executed in time.")
}

func TestStaticWorkerPool_HighNumberOfTasks(t *testing.T) {
	pool, err := NewStaticWorkerPool(5, 10)
	assert.NoError(t, err)
	assert.NotNil(t, pool)
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
	pool, err := NewStaticWorkerPool(2, 3)
	assert.NoError(t, err)
	assert.NotNil(t, pool)
	pool.Start()

	pool.Stop()

	assert.Panics(t, func() { pool.Schedule(true, &dummyTask{}) }, "Should panic when scheduling after cancel.")
	assert.NotNil(t, pool.ctx.Err(), "Context should be cancelled.")
}

func TestStaticWorkerPool_Stop(t *testing.T) {
	pool, err := NewStaticWorkerPool(2, 3)
	assert.NoError(t, err)
	assert.NotNil(t, pool)
	pool.Start()

	// Stop the pool and check if channels are closed.
	pool.Stop()

	assert.NotNil(t, pool.ctx.Err())
	assert.Panics(t, func() { pool.normalCh <- &dummyTask{} }, "normalCh channel is not closed.")
	assert.Panics(t, func() { pool.priorityCh <- &dummyTask{} }, "priorityCh channel is not closed.")
}
