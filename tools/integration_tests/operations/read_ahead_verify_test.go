// Copyright 2026 Google LLC
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

package operations_test

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

func TestVerifyReadAheadKB(t *testing.T) {
	expectedKB := setup.ReadAheadKB()
	if expectedKB <= 0 {
		t.Skip("Skipping read-ahead verification as read-ahead-kb is not configured.")
	}

	err := mounting.VerifyReadAhead(setup.MntDir(), expectedKB)
	if err != nil {
		t.Errorf("read-ahead verification failed: %v", err)
	}
}
