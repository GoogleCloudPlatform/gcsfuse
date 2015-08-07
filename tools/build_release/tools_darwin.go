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
	"fmt"
	"os/exec"
)

func checkForTools() (err error) {
	// Make a list of tools necessary.
	tools := []struct {
		name         string
		instructions string
	}{
		{"git", "brew install git"},
		{"fpm", "brew install gnu-tar && sudo gem install fpm -V"},
		{"go", "http://tip.golang.org/doc/install-source.html"},
	}

	// Check each.
	for _, t := range tools {
		_, err = exec.LookPath(t.name)
		if err != nil {
			err = fmt.Errorf("%s not found. Install it: %s", t.name, t.instructions)
			return
		}
	}

	return
}
