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
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
)

func HandleCPUProfileSignals() {
	profileOnce := func(duration time.Duration, path string) (err error) {
		// Set up the file.
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

		// Profile.
		err = pprof.StartCPUProfile(f)
		if err != nil {
			logger.Errorf("StartCPUProfile failed: %v", err)
			return
		}
		time.Sleep(duration)
		pprof.StopCPUProfile()
		return
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR1)
	for range c {
		path := fmt.Sprintf("/tmp/cpu-%d.pprof", time.Now().UnixNano())
		const duration = 10 * time.Second

		logger.Infof("Writing %v CPU profile to %s...", duration, path)

		err := profileOnce(duration, path)
		if err == nil {
			logger.Infof("Done writing CPU profile to %s.", path)
		} else {
			logger.Infof("Error writing CPU profile: %v", err)
		}
	}
}
