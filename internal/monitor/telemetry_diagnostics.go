// Copyright 2025 Google LLC
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

package monitor

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"go.opentelemetry.io/otel/sdk/resource"
)

// maxLoggedEnvValueLen caps the length of an individual environment variable
// value in the diagnostic dump so that a single huge value (e.g. an inlined
// config) doesn't flood the logs.
const maxLoggedEnvValueLen = 1024

// sensitiveEnvKeySubstrings lists the (upper-cased) substrings that mark an
// environment variable as likely holding a credential. Values of such
// variables are redacted before being logged.
var sensitiveEnvKeySubstrings = []string{
	"AUTH",
	"CREDENTIAL",
	"PASSWD",
	"PASSWORD",
	"PRIVATE",
	"SECRET",
	"SESSION",
	"SIGNATURE",
	"TOKEN",
	"KEY",
}

// nonSensitiveEnvKeys lists variables that match sensitiveEnvKeySubstrings but
// only ever hold a file path or a boolean, and are useful while debugging
// telemetry. Their values are logged verbatim.
var nonSensitiveEnvKeys = map[string]bool{
	"GOOGLE_APPLICATION_CREDENTIALS": true,
	"SSH_AUTH_SOCK":                  true,
}

// isSensitiveEnvKey reports whether the value of the given environment
// variable should be redacted before logging.
func isSensitiveEnvKey(key string) bool {
	if nonSensitiveEnvKeys[strings.ToUpper(key)] {
		return false
	}
	upperKey := strings.ToUpper(key)
	for _, s := range sensitiveEnvKeySubstrings {
		if strings.Contains(upperKey, s) {
			return true
		}
	}
	return false
}

// redactEnvValue returns the value to log for the given environment variable:
// the value itself (truncated if very long) for regular variables, or a
// placeholder that only reveals the value's length for sensitive ones.
func redactEnvValue(key, value string) string {
	if isSensitiveEnvKey(key) {
		return fmt.Sprintf("<redacted: %d chars>", len(value))
	}
	if len(value) > maxLoggedEnvValueLen {
		return fmt.Sprintf("%s...<truncated: %d chars total>", value[:maxLoggedEnvValueLen], len(value))
	}
	return value
}

// environmentVariableLines returns one sorted "key = value" line per
// environment variable visible to this process, with sensitive values
// redacted.
func environmentVariableLines() []string {
	environ := os.Environ()
	lines := make([]string, 0, len(environ))
	for _, entry := range environ {
		key, value, found := strings.Cut(entry, "=")
		if !found {
			// Should not happen, but don't drop the entry.
			key, value = entry, ""
		}
		lines = append(lines, fmt.Sprintf("  %s = %s", key, redactEnvValue(key, value)))
	}
	sort.Strings(lines)
	return lines
}

// resourceAttributeLines returns one sorted "key = value" line per attribute
// (a.k.a. resource label) of the detected OTel resource.
func resourceAttributeLines(res *resource.Resource) []string {
	if res == nil {
		return nil
	}
	attrs := res.Attributes()
	lines := make([]string, 0, len(attrs))
	for _, attr := range attrs {
		lines = append(lines, fmt.Sprintf("  %s = %s", string(attr.Key), attr.Value.Emit()))
	}
	sort.Strings(lines)
	return lines
}

// logTelemetryEnvironment logs everything gcsfuse can currently discover about
// the environment it is mounted in:
//   - the OTel resource labels (detected via the GCP resource detector) that
//     get attached to every exported metric/log, at INFO severity, and
//   - the full set of environment variables, at DEBUG severity, since the dump
//     is verbose and may contain user-specific data. Mount with
//     `--log-severity=debug` to see it.
//
// It is invoked while setting up the metric exporters, i.e. on every mount.
func logTelemetryEnvironment(res *resource.Resource) {
	attrLines := resourceAttributeLines(res)
	if len(attrLines) == 0 {
		logger.Infof("Telemetry resource labels: none detected.")
	} else {
		logger.Infof("Telemetry resource labels (%d detected):\n%s", len(attrLines), strings.Join(attrLines, "\n"))
	}
	if res != nil {
		logger.Debugf("Telemetry resource schema URL: %q", res.SchemaURL())
	}

	envLines := environmentVariableLines()
	logger.Debugf("Telemetry environment variables (%d visible; values of credential-like keys are redacted):\n%s",
		len(envLines), strings.Join(envLines, "\n"))
}
