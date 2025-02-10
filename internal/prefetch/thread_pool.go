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

package prefetch

import (
	"sync"
)

// ThreadPool is a group of workers that can be used to execute a task
type ThreadPool struct {
	// Number of workers running in this group
	worker uint32

	// Channel to close all the workers
	close chan int

	// Wait group to wait for all workers to finish
	wg sync.WaitGroup

	// Channel to hold pending requests
	priorityCh chan *prefetchTask
	normalCh   chan *prefetchTask

	// Reader method that will actually read the data
	download func(*prefetchTask)
}

// newThreadPool creates a new thread pool
func newThreadPool(count uint32, download func(*prefetchTask)) *ThreadPool {
	if count == 0 || download == nil {
		return nil
	}

	return &ThreadPool{
		worker:     count,
		download:   download,
		close:      make(chan int, count),
		priorityCh: make(chan *prefetchTask, count*2),
		normalCh:   make(chan *prefetchTask, count*5000),
	}
}

// Start all the workers and wait till they start receiving requests
func (t *ThreadPool) Start() {
	// 10% threads will listen only on high priority channel
	highPriority := (t.worker * 10) / 100

	for i := uint32(0); i < t.worker; i++ {
		t.wg.Add(1)
		go t.Do(i < highPriority)
	}
}

// Stop all the workers threads
func (t *ThreadPool) Stop() {
	for i := uint32(0); i < t.worker; i++ {
		t.close <- 1
	}

	t.wg.Wait()

	close(t.close)
	close(t.priorityCh)
	close(t.normalCh)
}

// Schedule the download of a block
func (t *ThreadPool) Schedule(urgent bool, item *prefetchTask) {
	// urgent specifies the priority of this task.
	// true means high priority and false means low priority
	if urgent {
		t.priorityCh <- item
	} else {
		t.normalCh <- item
	}
}

// Do is the core task to be executed by each worker thread
func (t *ThreadPool) Do(priority bool) {
	defer t.wg.Done()

	if priority {
		// This thread will work only on high priority channel
		for {
			select {
			case item := <-t.priorityCh:
				t.download(item)
			case <-t.close:
				return
			}
		}
	} else {
		// This thread will work only on both high and low priority channel
		for {
			select {
			case item := <-t.priorityCh:
				t.download(item)
			case item := <-t.normalCh:
				t.download(item)
			case <-t.close:
				return
			}
		}
	}
}