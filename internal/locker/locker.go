// Copyright 2021 Google LLC
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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

// TODO: Migrate all usages of these global variables to explicitly instantiate and pass locker.Options.
var gEnableInvariantsCheck bool
var gEnableDebugMessages bool

// EnableInvariantsCheck enables the check for invariants in the locks. Must be set before creating
// any lockers.
// TODO: Deprecate and delete this function once all components migrate to using locker.Options.
func EnableInvariantsCheck() {
	gEnableInvariantsCheck = true
}

// EnableDebugMessages enables the debug messages to diagnose dead locks. Must be set before creating
// any lockers.
// TODO: Deprecate and delete this function once all components migrate to using locker.Options.
func EnableDebugMessages() {
	gEnableDebugMessages = true
}

type Locker sync.Locker

// Options contains configuration for creating lockers.
// By using options, callers can avoid relying on the package-level global variables.
type Options struct {
	EnableInvariantsCheck bool
	EnableDebugMessages   bool
}

// NewWithOptions returns a locker with potential capability for debugging based on options.
func NewWithOptions(name string, check func(), opts Options) Locker {
	var l Locker = &sync.Mutex{}

	if opts.EnableInvariantsCheck {
		l = &checker{
			locker: l,
			check:  check,
		}
	}

	if opts.EnableDebugMessages {
		l = &debugger{
			locker: l,
			name:   name,
		}
	}

	return l
}

// New returns a locker with potential capability for debugging.
// TODO: Deprecate and delete this function once all components migrate to using locker.Options.
func New(name string, check func()) Locker {
	return NewWithOptions(name, check, Options{
		EnableInvariantsCheck: gEnableInvariantsCheck,
		EnableDebugMessages:   gEnableDebugMessages,
	})
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
	timer  *time.Timer
}

func (d *debugger) Lock() {
	d.locker.Lock()

	buf := make([]byte, 2048)
	n := runtime.Stack(buf, false /* all */)
	// Use only the bytes written to the buffer to avoid uninitialized values in the string.
	holder := string(buf[:n])

	d.timer = time.AfterFunc(5*time.Second, func() {
		logger.Tracef("debug_mutex: Potential dead lock detected for a lock %q held by: %v\n", d.name, holder)
	})
}

func (d *debugger) Unlock() {
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}

	d.locker.Unlock()
}
