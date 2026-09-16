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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

func TestIsSensitiveEnvKey(t *testing.T) {
	testCases := []struct {
		key      string
		expected bool
	}{
		{"HOSTNAME", false},
		{"KUBERNETES_SERVICE_HOST", false},
		{"GOOGLE_CLOUD_PROJECT", false},
		{"GOOGLE_APPLICATION_CREDENTIALS", false},
		{"SSH_AUTH_SOCK", false},
		{"AWS_SECRET_ACCESS_KEY", true},
		{"aws_session_token", true},
		{"MY_API_KEY", true},
		{"DB_PASSWORD", true},
		{"SOME_PRIVATE_DATA", true},
	}

	for _, tc := range testCases {
		t.Run(tc.key, func(t *testing.T) {
			assert.Equal(t, tc.expected, isSensitiveEnvKey(tc.key))
		})
	}
}

func TestRedactEnvValue(t *testing.T) {
	t.Run("PlainValueIsReturnedAsIs", func(t *testing.T) {
		assert.Equal(t, "my-project", redactEnvValue("GOOGLE_CLOUD_PROJECT", "my-project"))
	})

	t.Run("SensitiveValueIsRedacted", func(t *testing.T) {
		got := redactEnvValue("AWS_SECRET_ACCESS_KEY", "super-secret")

		assert.NotContains(t, got, "super-secret")
		assert.Equal(t, "<redacted: 12 chars>", got)
	})

	t.Run("LongValueIsTruncated", func(t *testing.T) {
		value := strings.Repeat("a", maxLoggedEnvValueLen+10)

		got := redactEnvValue("SOME_LONG_VAR", value)

		assert.Len(t, got, maxLoggedEnvValueLen+len("...<truncated: 1034 chars total>"))
		assert.True(t, strings.HasSuffix(got, "...<truncated: 1034 chars total>"))
	})
}

func TestEnvironmentVariableLines(t *testing.T) {
	t.Setenv("GCSFUSE_TEST_PLAIN_VAR", "plain-value")
	t.Setenv("GCSFUSE_TEST_SECRET_TOKEN", "do-not-log-me")

	lines := environmentVariableLines()

	joined := strings.Join(lines, "\n")
	assert.Contains(t, joined, "  GCSFUSE_TEST_PLAIN_VAR = plain-value")
	assert.Contains(t, joined, "  GCSFUSE_TEST_SECRET_TOKEN = <redacted: 13 chars>")
	assert.NotContains(t, joined, "do-not-log-me")
	assert.True(t, sortedAscending(lines))
}

func TestResourceAttributeLines(t *testing.T) {
	t.Run("NilResource", func(t *testing.T) {
		assert.Empty(t, resourceAttributeLines(nil))
	})

	t.Run("AttributesAreSorted", func(t *testing.T) {
		res, err := resource.New(t.Context(),
			resource.WithAttributes(
				attribute.String("service.name", "gcsfuse"),
				attribute.String("cloud.region", "us-central1"),
				attribute.Int64("host.id", 42),
			),
		)
		require.NoError(t, err)

		lines := resourceAttributeLines(res)

		assert.Equal(t, []string{
			"  cloud.region = us-central1",
			"  host.id = 42",
			"  service.name = gcsfuse",
		}, lines)
	})
}

func TestLogTelemetryEnvironmentDoesNotPanic(t *testing.T) {
	res, err := resource.New(t.Context(), resource.WithAttributes(attribute.String("service.name", "gcsfuse")))
	require.NoError(t, err)

	assert.NotPanics(t, func() { logTelemetryEnvironment(res) })
	assert.NotPanics(t, func() { logTelemetryEnvironment(nil) })
}

func sortedAscending(lines []string) bool {
	for i := 1; i < len(lines); i++ {
		if lines[i-1] > lines[i] {
			return false
		}
	}
	return true
}
