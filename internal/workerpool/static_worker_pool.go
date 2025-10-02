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
	"fmt"
	"runtime"
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

// staticWorkerPool starts all the workers (goroutines) on startup and keeps them running.
// It keep two types of workers - priority and normal. Priority workers will only
// execute tasks that are marked as urgent while scheduling. Normal workers will
// execute both urgent and normal tasks, but gives precedence to urgent task.
type staticWorkerPool struct {
	priorityWorker uint32 // Number of priority workers in this pool.
	normalWorker   uint32 // Number of normal workers in this pool.

	// Stop channel to notify all the workers to stop.
	stop chan bool

	// Wait group to wait for all workers to finish.
	wg sync.WaitGroup

	// Channels for normal and priority tasks.
	priorityCh chan Task
	normalCh   chan Task
}

// NewStaticWorkerPool creates a new thread pool
func NewStaticWorkerPool(priorityWorker uint32, normalWorker uint32, readGlobalMaxBlocks int64) (*staticWorkerPool, error) {
	totalWorkers := priorityWorker + normalWorker
	if totalWorkers == 0 {
		return nil, fmt.Errorf("staticWorkerPool: can't create with 0 workers, priority: %d, normal: %d", priorityWorker, normalWorker)
	}

	logger.Infof("staticWorkerPool: creating with %d normal, and %d priority workers.", normalWorker, priorityWorker)

	// The channel capacity is set to the minimum of a worker-based buffer size
	// and a global cap. This prevents creating overly large channels, which can be
	// slow to initialize and consume unnecessary memory. The cap is based on
	// `readGlobalMaxBlocks` because we can't schedule more download tasks than this at once.
	priorityChSize := min(int(priorityWorker)*200, int(2*readGlobalMaxBlocks))
	normalChSize := min(int(normalWorker)*5000, int(2*readGlobalMaxBlocks))
	return &staticWorkerPool{
		priorityWorker: priorityWorker,
		normalWorker:   normalWorker,
		stop:           make(chan bool),
		priorityCh:     make(chan Task, priorityChSize),
		normalCh:       make(chan Task, normalChSize),
	}, nil
}

// NewStaticWorkerPoolForCurrentCPU creates and starts a new worker pool. The
// number of workers is determined based on the number of available CPUs and
// the provided readGlobalMaxBlocks.
func NewStaticWorkerPoolForCurrentCPU(readGlobalMaxBlocks int64) (WorkerPool, error) {
	return newStaticWorkerPoolForCurrentCPU(readGlobalMaxBlocks, runtime.NumCPU)
}

// newStaticWorkerPoolForCurrentCPU is an unexported helper for testing.
func newStaticWorkerPoolForCurrentCPU(readGlobalMaxBlocks int64, numCPU func() int) (WorkerPool, error) {
	// It's a general heuristic to use 2-3 times the number of CPUs for I/O-bound tasks.
	// We use 3x here as a balance between parallelism and resource consumption.
	const workersPerCPU = 3
	totalWorkers := workersPerCPU * numCPU()

	// Since the number of concurrent download tasks is limited by readGlobalMaxBlocks,
	// creating more workers beyond this limit offers no performance gain and wastes
	// resources. Hence, we cap total workers to ceil(1.1 * readGlobalMaxBlocks).
	if cappedWorkers := (11*readGlobalMaxBlocks + 9) / 10; int64(totalWorkers) > cappedWorkers {
		totalWorkers = int(cappedWorkers)
	}

	// 10% of total workers for priority, rounded up.
	priorityWorkers := (totalWorkers + 9) / 10
	normalWorkers := totalWorkers - priorityWorkers

	wp, err := NewStaticWorkerPool(uint32(priorityWorkers), uint32(normalWorkers), readGlobalMaxBlocks)
	if err != nil {
		return nil, err
	}

	wp.Start()
	return wp, nil
}

// Start all the workers and wait till they start receiving requests
func (swp *staticWorkerPool) Start() {
	for i := uint32(0); i < swp.priorityWorker; i++ {
		swp.wg.Add(1)
		go swp.do(true)
	}

	for i := uint32(0); i < swp.normalWorker; i++ {
		swp.wg.Add(1)
		go swp.do(false)
	}
}

// Stop all the workers threads and wait for them to finish processing.
func (swp *staticWorkerPool) Stop() {
	// Notify all workers to stop.
	logger.Infof("staticWorkerPool: stopping all the workers.")
	close(swp.stop)

	swp.wg.Wait()

	// Close the channel after all workers are done.
	close(swp.priorityCh)
	close(swp.normalCh)
}

// Schedule schedules tasks to the worker pool.
// Pass urgent as true for priority scheduling.
func (swp *staticWorkerPool) Schedule(urgent bool, task Task) {
	// urgent specifies the priority of this task.
	// true means high priority and false means low priority
	if urgent {
		swp.priorityCh <- task
	} else {
		swp.normalCh <- task
	}
}

// do is the core routine that runs in each worker thread.
// It will keep listening to the channel for tasks and execute them.
func (swp *staticWorkerPool) do(priority bool) {
	defer swp.wg.Done()

	if priority {
		// Worker only listens to the priority channel.
		for {
			select {
			case <-swp.stop:
				return
			default:
				select {
				case <-swp.stop:
					return
				case task := <-swp.priorityCh:
					task.Execute()
				}
			}
		}
	} else {
		// Worker listens to both channels but gives priority to the priority channel.
		for {
			select {
			case <-swp.stop:
				return
			case task := <-swp.priorityCh:
				task.Execute()
			default:
				select {
				case <-swp.stop:
					return
				case task := <-swp.priorityCh:
					task.Execute()
				case task := <-swp.normalCh:
					task.Execute()
				}
			}
		}
	}
}
