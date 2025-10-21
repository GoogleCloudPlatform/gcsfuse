// Copyright 2015 Google LLC
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

package workloadinsight

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIORenderer_NewRenderer(t *testing.T) {
	_, err := NewRenderer()
	require.NoError(t, err, "NewRenderer should not return an error")
}

func TestIORenderer_NewRendererWithSettings_InvalidSettings(t *testing.T) {
	tc := []struct {
		name       string
		plotWidth  int
		labelWidth int
		pad        int
	}{
		{name: "negative plotWidth", plotWidth: -1, labelWidth: 0, pad: 2},
		{name: "negative labelWidth", plotWidth: 80, labelWidth: -5, pad: 2},
		{name: "negative pad", plotWidth: 80, labelWidth: 0, pad: -3},
	}

	for _, test := range tc {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewRendererWithSettings(test.plotWidth, test.labelWidth, test.pad)
			require.Error(t, err, "expected error for invalid settings")
		})
	}
}

func TestIORenderer_NewRendererWithSettings_ValidSettings(t *testing.T) {
	tc := []struct {
		name       string
		plotWidth  int
		labelWidth int
		pad        int
	}{
		{name: "zero labelWidth", plotWidth: 80, labelWidth: 20, pad: 2},
		{name: "positive labelWidth", plotWidth: 100, labelWidth: 15, pad: 4},
		{name: "zero pad", plotWidth: 60, labelWidth: 20, pad: 0},
	}

	for _, test := range tc {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewRendererWithSettings(test.plotWidth, test.labelWidth, test.pad)
			require.NoError(t, err, "NewRendererWithSettings should not return an error for valid settings")
		})
	}
}

func TestHumanReadable(t *testing.T) {
	tc := []struct {
		name     string
		size     uint64
		expected string
	}{
		{name: "zero", size: 0, expected: "0B"},
		{name: "one", size: 1, expected: "1B"},
		{name: "five hundred", size: 500, expected: "500B"},
		{name: "one thousand five hundred", size: 1500, expected: "1.5K"},
		{name: "two megabytes", size: 2 * 1024 * 1024, expected: "2.0M"},
		{name: "three gigabytes", size: 3 * 1024 * 1024 * 1024, expected: "3.0G"},
		{name: "one hundred twenty-three million four hundred fifty-six thousand seven hundred eighty-nine", size: 123456789, expected: "117.7M"},
	}

	for _, test := range tc {
		t.Run(test.name, func(t *testing.T) {
			result := humanReadable(test.size)
			require.Equal(t, test.expected, result, "humanReadable(%d) should be %s", test.size, test.expected)
		})
	}
}

func TestIORenderer_Render(t *testing.T) {
	tc := []struct {
		name               string
		plotWidth          int
		labelWidth         int
		pad                int
		expectedOutputFile string
	}{
		{
			name:               "default settings",
			plotWidth:          80,
			labelWidth:         12, // len(labelHeader)
			pad:                2,
			expectedOutputFile: "testdata/io_renderer/default_settings.txt",
		},
		{
			name:               "custom settings",
			plotWidth:          50,
			labelWidth:         20,
			pad:                4,
			expectedOutputFile: "testdata/io_renderer/custom_settings.txt",
		},
	}

	for _, test := range tc {
		t.Run(test.name, func(t *testing.T) {
			r, err := NewRendererWithSettings(test.plotWidth, test.labelWidth, test.pad)
			require.NoError(t, err, "NewRendererWithSettings should not return an error for valid settings")

			name := "demo.txt"
			size := uint64(1000)
			ranges := []Range{
				{Start: 0, End: 100},
				{Start: 200, End: 300},
				{Start: 400, End: 600},
				{Start: 800, End: 1000},
			}

			out := r.Render(name, size, ranges)
			os.WriteFile(test.expectedOutputFile, []byte(out), 0644) // Uncomment to create a new test output file.

			goldenBytes, err := os.ReadFile(test.expectedOutputFile)
			require.NoError(t, err, "should be able to read golden file: %s", test.expectedOutputFile)
			golden := string(goldenBytes)
			require.Equal(t, golden, out, "visual output should exactly match the golden ASCII representation for %s", test.name)
		})
	}
}
