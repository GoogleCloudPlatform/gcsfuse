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
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

// staticWorkerPool starts all the workers (goroutines) on startup and keeps them running.
// It keep two types of workers - priority and normal. Priority workers will only
// execute tasks that are marked as urgent while scheduling, while normal workers will
// execute both urgent and normal tasks.
type staticWorkerPool struct {
	priorityWorker uint32 // Number of priority workers in this pool.
	normalWorker   uint32 // Number of normal workers in this pool.

	// Channel to close all the workers.
	close chan int

	// Wait group to wait for all workers to finish.
	wg sync.WaitGroup

	// Channels for normal and priority tasks.
	priorityCh chan Task
	normalCh   chan Task
}

// newWorkerPool creates a new thread pool
func NewStaticWorkerPool(priorityWorker uint32, normalWorker uint32) (*staticWorkerPool, error) {
	totalWorkers := priorityWorker + normalWorker
	if totalWorkers == 0 {
		return nil, fmt.Errorf("staticWorkerPool: can't create with 0 workers, priority: %d, normal: %d", priorityWorker, normalWorker)
	}

	logger.Infof("staticWorkerPool: creating with %d normal, and %d priority workers.", normalWorker, priorityWorker)

	return &staticWorkerPool{
		priorityWorker: priorityWorker,
		normalWorker:   normalWorker,
		close:          make(chan int, totalWorkers),
		// Keep the size of the channels large enough to handle burst of tasks.
		priorityCh: make(chan Task, totalWorkers*200),
		normalCh:   make(chan Task, totalWorkers*5000),
	}, nil
}

// Start all the workers and wait till they start receiving requests
func (t *staticWorkerPool) Start() {
	for i := uint32(0); i < t.priorityWorker; i++ {
		t.wg.Add(1)
		go t.do(true)
	}

	for i := uint32(0); i < t.normalWorker; i++ {
		t.wg.Add(1)
		go t.do(false)
	}
}

// Stop all the workers threads and wait for them to finish processing.
func (t *staticWorkerPool) Stop() {
	for i := uint32(0); i < t.priorityWorker; i++ {
		t.close <- 1
	}

	for i := uint32(0); i < t.normalWorker; i++ {
		t.close <- 1
	}

	t.wg.Wait()

	close(t.close)
	close(t.priorityCh)
	close(t.normalCh)
}

// Schedule schedules tasks to the worker pool.
// Pass urgent as true for priority scheduling.
func (t *staticWorkerPool) Schedule(urgent bool, item Task) {
	// urgent specifies the priority of this task.
	// true means high priority and false means low priority
	if urgent {
		t.priorityCh <- item
	} else {
		t.normalCh <- item
	}
}

// do is the core routine that runs in each worker thread.
// It will keep listening to the channel for tasks and execute them.
func (t *staticWorkerPool) do(priority bool) {
	defer t.wg.Done()

	if priority {
		// This thread will work only on high priority channel
		for {
			select {
			case item := <-t.priorityCh:
				item.Execute()
			case <-t.close:
				return
			}
		}
	} else {
		// This thread will work only on both high and low priority channel
		for {
			select {
			case item := <-t.priorityCh:
				item.Execute()
			case item := <-t.normalCh:
				item.Execute()
			case <-t.close:
				return
			}
		}
	}
}
