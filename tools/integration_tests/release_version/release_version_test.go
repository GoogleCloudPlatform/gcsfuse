// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package release_version

import (
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

func TestReleaseVersion(t *testing.T) {
	cmd := exec.Command("gcsfuse", "--version")

	outputBytes, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Failed to execute 'gcsfuse --version': %v\nOutput: %s", err, string(outputBytes))
	}
	output := strings.TrimSpace(string(outputBytes))
	t.Logf("gcsfuse --version output:\n%s", output) // Log the output for debugging
	expectedPattern := `^gcsfuse version (\d+\.\d+\.\d+) \(Go version (go.+)\)$`
	r := regexp.MustCompile(expectedPattern)
	// Match the output against the pattern
	matches := r.FindStringSubmatch(output)
	if len(matches) != 3 { // Expect 3 elements: full match, version, go version
		t.Errorf("Output did not match expected pattern.\nExpected pattern: %q\nActual output: %q\nMatches: %v", expectedPattern, output, matches)
	} else {
		version := matches[1]
		goVersion := matches[2]
		if version == "" {
			t.Errorf("Extracted gcsfuse version is empty")
		}
		if goVersion == "" {
			t.Errorf("Extracted Go version is empty")
		}
	}
}
