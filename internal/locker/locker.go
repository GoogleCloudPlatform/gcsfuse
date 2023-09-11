// Copyright 2021 Google Inc. All Rights Reserved.
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

// Provides the Locker implementations with optional debug utils.
package locker

import (
	"runtime"
	"sync"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/logger"
)

var gEnableInvariantsCheck bool
var gEnableDebugMessages bool

// Enable the check for invariants in the locks. Must be set before creating
// any lockers.
func EnableInvariantsCheck() {
	gEnableInvariantsCheck = true
}

// Enable the debug messages to diagnose dead locks. Must be set before creating
// any lockers.
func EnableDebugMessages() {
	gEnableDebugMessages = true
}

type Locker sync.Locker

// Returns a locker with potential capability for debugging.
func New(name string, check func()) Locker {
	var l Locker = &sync.Mutex{}

	if gEnableInvariantsCheck {
		l = &checker{
			locker: l,
			check:  check,
		}
	}

	if gEnableDebugMessages {
		l = &debugger{
			locker: l,
			name:   name,
		}
	}

	return l
}

type checker struct {
	locker Locker
	check  func()
}

func (c *checker) Lock() {
	c.locker.Lock()
	c.check()
}

func (c *checker) Unlock() {
	c.check()
	c.locker.Unlock()
}

type debugger struct {
	locker Locker
	name   string
	holder string
	timer  *time.Timer
}

func (d *debugger) Lock() {
	d.locker.Lock()

	buf := make([]byte, 2048)
	runtime.Stack(buf, false /* all */)
	d.holder = string(buf)

	d.timer = time.AfterFunc(5*time.Second, func() {
		logger.Tracef("debug_mutex: Potential dead lock detected for a lock %q held by: %v\n", d.name, d.holder)
	})
}

func (d *debugger) Unlock() {
	d.holder = ""
	d.timer.Stop()
	d.timer = nil

	d.locker.Unlock()
}
