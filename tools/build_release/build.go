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
	"log"
)

// Build release binaries according to the supplied settings, returning the
// path to a directory containing exactly the output binaries.
func buildBinaries(
	version string,
	commit string,
	osys string,
	arch string) (dir string, err error) {
	log.Printf("Building %s at %s for %s (%s).", version, commit, osys, arch)
	err = errors.New("TODO")
	return
}
