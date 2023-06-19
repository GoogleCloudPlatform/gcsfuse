// Copyright 2023 Google Inc. All Rights Reserved.
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

// Provide tests when implicit directory present and mounted bucket with --implicit-dir flag.
package implicit_dir_test

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup/implicit_and_explicit_dir_setup"
)

const ImplicitDirectory = "implicitDirectory"

func TestMain(m *testing.M) {
	flags := [][]string{{"--implicit-dirs"}, {"--enable-storage-client-library=false", "--implicit-dirs"}}

	implicit_and_explicit_dir_setup.RunTestsForImplicitDirAndExplicitDir(flags, m)
}
