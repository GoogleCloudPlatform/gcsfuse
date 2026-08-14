// Copyright 2023 Google LLC
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

// Provides the RWLocker implementations with optional debug utils.
package locker

import (
	"runtime"
	"sync"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

type RWLocker interface {
	sync.Locker
	RLock()
	RUnlock()
}

// NewRWWithOptions returns an RW locker with potential capability for debugging based on options.
func NewRWWithOptions(name string, check func(), opts Options) RWLocker {
	var l RWLocker = &sync.RWMutex{}

	if opts.EnableInvariantsCheck {
		l = &rwChecker{
			locker: l,
			check:  check,
		}
	}

	if opts.EnableDebugMessages {
		l = &rwDebugger{
			locker: l,
			name:   name,
		}
	}

	return l
}

// NewRW returns a RW locker with potential capability for debugging.
//
// Note: The deadlock detection is done only for writer lock and not for reader
// lock.
// TODO: Deprecate and delete this function once all components migrate to using locker.Options.
func NewRW(name string, check func()) RWLocker {
	return NewRWWithOptions(name, check, Options{
		EnableInvariantsCheck: gEnableInvariantsCheck,
		EnableDebugMessages:   gEnableDebugMessages,
	})
}

type rwChecker struct {
	locker RWLocker
	check  func()
}

func (c *rwChecker) Lock() {
	c.locker.Lock()
	c.check()
}

func (c *rwChecker) Unlock() {
	c.check()
	c.locker.Unlock()
}

func (c *rwChecker) RLock() {
	c.locker.RLock()
	c.check()
}

func (c *rwChecker) RUnlock() {
	c.check()
	c.locker.RUnlock()
}

// Note: rwDebugger doesn't check potential deadlock in case of read only lock as
// doing that is not straight forward because multiple readers can acquire locks
// at the same time and that needs keeping track of every different lock.
type rwDebugger struct {
	locker RWLocker
	name   string
	timer  *time.Timer
}

func (d *rwDebugger) Lock() {
	d.locker.Lock()

	buf := make([]byte, 2048)
	n := runtime.Stack(buf, false /* all */)
	// Use only the bytes written to the buffer to avoid uninitialized values in the string.
	holder := string(buf[:n])

	d.timer = time.AfterFunc(5*time.Second, func() {
		logger.Tracef("debug_mutex: Potential dead lock detected for a lock %q held by: %v\n", d.name, holder)
	})
}

func (d *rwDebugger) Unlock() {
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}

	d.locker.Unlock()
}

func (d *rwDebugger) RLock() {
	d.locker.RLock()
}

func (d *rwDebugger) RUnlock() {
	d.locker.RUnlock()
}
