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

package fuse

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// MountedFileSystem represents a mounted file system.
type MountedFileSystem struct {
	dir string
	// ready chan error
	// join  chan error
}

// Dir returns the directory on which the file system is mounted.
func (mfs *MountedFileSystem) Dir() string {
	return mfs.dir
}

// Join blocks until the file system has been unmounted.
func (mfs *MountedFileSystem) Join(ctx context.Context) error {
	// <-mfs.join
	return nil
}

// Unmount unmounts the file system.
func (mfs *MountedFileSystem) Unmount() error {
	delay := 10 * time.Millisecond
	for {
		err := unmount(mfs.dir)
		if err == nil {
			break
		}

		if strings.Contains(err.Error(), "resource busy") {
			time.Sleep(delay)
			delay = time.Duration(1.3 * float64(delay))
			continue
		}

		return fmt.Errorf("unmount: %w", err)
	}
	return nil
}
