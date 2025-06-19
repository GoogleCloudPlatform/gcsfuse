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

// Task interface defines the contract for a runnable task.
type Task interface {
	Execute()
}

type WorkerPool interface {
	// Start initializes the worker pool and prepares it to accept tasks.
	Start()

	// Stop gracefully shuts down the worker pool, waiting for all tasks to complete.
	Stop()

	// Schedule adds a task to the worker pool for execution.
	Schedule(urgent bool, task Task)
}
