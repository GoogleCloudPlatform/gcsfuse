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

package lru

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// longestCommonPrefixBCE is the bounds-check-eliminated (BCE) version of longestCommonPrefix.
// By calculating minLen upfront and bounding loop iterations by i < minLen, the Go compiler's
// SSA analysis can statically prove that i < len(a) and i < len(b) for all iterations.
// This eliminates runtime bounds checks (panicIndex) within the loop body.
func longestCommonPrefixBCE(a, b string) int {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	for i := 0; i < minLen; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return minLen
}

func TestLongestCommonPrefixBCE_Parity(t *testing.T) {
	testCases := []struct {
		name string
		a    string
		b    string
	}{
		{"Empty strings", "", ""},
		{"One empty string", "foo/bar", ""},
		{"No common prefix", "apple", "banana"},
		{"Identical strings", "usr/local/bin/gcsfuse", "usr/local/bin/gcsfuse"},
		{"Prefix of another", "usr/local/", "usr/local/bin/gcsfuse"},
		{"Long common prefix with divergence at end", strings.Repeat("a", 200) + "b", strings.Repeat("a", 200) + "c"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expected := longestCommonPrefix(tc.a, tc.b)
			actual := longestCommonPrefixBCE(tc.a, tc.b)
			assert.Equal(t, expected, actual, "BCE implementation diverged from standard implementation")
		})
	}
}

// BenchmarkLongestCommonPrefix compares the default loop with the BCE-optimized loop
// across different string lengths and prefix characteristics typical in filesystem path routing.
func BenchmarkLongestCommonPrefix(b *testing.B) {
	benchmarks := []struct {
		name string
		a    string
		b    string
	}{
		{
			name: "NoCommonPrefix_Short",
			a:    "apple",
			b:    "banana",
		},
		{
			name: "ShortPrefix_5Bytes",
			a:    "usr/local/bin",
			b:    "usr/share/doc",
		},
		{
			name: "MediumPrefix_50Bytes",
			a:    "foo/bar/baz/qux/directory_structure/file_number_one.txt",
			b:    "foo/bar/baz/qux/directory_structure/file_number_two.txt",
		},
		{
			name: "LongPrefix_200Bytes",
			a:    strings.Repeat("usr/local/google/home/kislayk/gitproj/gcsfuse3/internal/cache/lru/path/", 3) + "file_a.go",
			b:    strings.Repeat("usr/local/google/home/kislayk/gitproj/gcsfuse3/internal/cache/lru/path/", 3) + "file_b.go",
		},
		{
			name: "Identical_100Bytes",
			a:    strings.Repeat("a", 100),
			b:    strings.Repeat("a", 100),
		},
	}

	for _, bm := range benchmarks {
		b.Run("Default_"+bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = longestCommonPrefix(bm.a, bm.b)
			}
		})

		b.Run("BCE_"+bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = longestCommonPrefixBCE(bm.a, bm.b)
			}
		})
	}
}
