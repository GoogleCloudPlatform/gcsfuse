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

// Provides integration tests when implicit_dir flag is set.
package implicitdir_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

type Implicitdir struct {
	setup.SetUpAndTearDown
}

func (i *Implicitdir) teardown() {
	fmt.Println("Tulsi")
	os.RemoveAll(setup.MntDir())
}

func TestMain(m *testing.M) {
	var x *Implicitdir

	flags := [][]string{{"--enable-storage-client-library=true", "--implicit-dirs=true"},
		{"--enable-storage-client-library=false"},
		{"--implicit-dirs=true"},
		{"--implicit-dirs=false"}}

	setup.RunTests(flags, m, x)
}
