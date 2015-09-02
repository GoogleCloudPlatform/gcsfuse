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
	"runtime"
)

// Set with `-ldflags -X main.gcsfuseVersion=1.2.3` by tools/build_gcsfuse. If
// not defined, we use "unknown" in getVersion.
var gcsfuseVersion string

func getVersion() string {
	v := gcsfuseVersion
	if v == "" {
		v = "unknown"
	}

	return fmt.Sprintf("%s (Go version %s)", v, runtime.Version())
}
