// Copyright 2015 Google Inc. All Rights Reserved.
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

package main

import (
	"errors"
	"os/exec"
	"runtime"
)

// Find the path to the fusermount command for this system, returning "" if
// we're not running on Linux.
func findFusermount() (p string, err error) {
	if runtime.GOOS != "linux" {
		return
	}

	// HACK(jacobsa): Since mount(8) appears to call its helpers with $PATH
	// unset, I can find no better way to do this than searching a hard-coded
	// list of candidates. These are directories where I've seen it live on
	// various distributions.
	candidates := []string{
		"/bin/fusermount",
		"/usr/bin/fusermount",
		"/run/current-system/sw/bin/fusermount",
	}

	for _, c := range candidates {
		_, err = exec.LookPath(c)
		if err == nil {
			p = c
			return
		}
	}

	err = errors.New("Can't find a usable executable.")
	return
}

// Find the path to the gcsfuse program.
func findGcsfuse() (p string, err error) {
	// HACK(jacobsa): Since mount(8) appears to call its helpers with $PATH
	// unset, I can find no better way to do this than searching a hard-coded
	// list of candidates. However, include as a candidate the $PATH-relative
	// version in case we are being called in a context with $PATH set, such as
	// a test.
	candidates := []string{
		"gcsfuse",
		"/usr/bin/gcsfuse",
		"/usr/local/bin/gcsfuse",
		"/run/current-system/sw/bin/gcsfuse",
	}

	for _, c := range candidates {
		_, err = exec.LookPath(c)
		if err == nil {
			p = c
			return
		}
	}

	err = errors.New("Can't find a usable executable.")
	return
}
