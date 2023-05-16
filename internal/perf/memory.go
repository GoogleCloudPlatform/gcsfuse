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

package perf

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/logger"
)

const (
	KiB = 1024
	MiB = 1024 * KiB
)

func HandleMemoryProfileSignals() {
	profileOnce := func(path string) (err error) {
		// Trigger a garbage collection to get up to date information (cf.
		// https://goo.gl/aXVQfL).
		runtime.GC()

		// Open the file.
		var f *os.File
		f, err = os.Create(path)
		if err != nil {
			err = fmt.Errorf("Create: %w", err)
			return
		}

		defer func() {
			closeErr := f.Close()
			if err == nil {
				err = closeErr
			}
		}()

		// Dump to the file.
		err = pprof.Lookup("heap").WriteTo(f, 0)
		if err != nil {
			err = fmt.Errorf("WriteTo: %w", err)
			return
		}

		return
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR2)
	for range c {
		path := fmt.Sprintf("/tmp/mem-%d.pprof", time.Now().UnixNano())

		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		logger.Infof("Heap allocation: %d MiB", m.Alloc/MiB)

		err := profileOnce(path)
		if err == nil {
			logger.Infof("Wrote memory profile to %s.", path)
		} else {
			logger.Infof("Error writing memory profile: %v", err)
		}
	}
}
