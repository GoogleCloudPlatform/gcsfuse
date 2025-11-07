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

package read_manager

import (
	"context"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workloadinsight"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewVisualReadManager(t *testing.T) {
	mockReadManager := &MockReadManager{}
	ioRenderer, err := workloadinsight.NewRenderer()
	require.NoError(t, err, "Failed to create IORenderer")

	vrm := NewVisualReadManager(mockReadManager, ioRenderer, cfg.WorkloadInsightConfig{})

	assert.NotNil(t, vrm, "VisualReadManager should not be nil")
	assert.Equal(t, mockReadManager, vrm.wrapped, "Wrapped ReadManager should match the input")
	assert.Equal(t, ioRenderer, vrm.ioRenderer, "IORenderer should match the input")
	assert.Empty(t, vrm.readIOs, "Initial readIOs slice should be empty")
}

func TestVisualReadManager_AcceptRange(t *testing.T) {
	testCase := []struct {
		name           string
		inputRanges    [][2]uint64
		expectedRanges []workloadinsight.Range
	}{
		{
			name: "Non-overlapping ranges",
			inputRanges: [][2]uint64{
				{0, 10},
				{20, 30},
				{40, 50},
			},
			expectedRanges: []workloadinsight.Range{
				{Start: 0, End: 10},
				{Start: 20, End: 30},
				{Start: 40, End: 50},
			},
		},
		{
			name: "Overlapping ranges",
			inputRanges: [][2]uint64{
				{0, 15},
				{10, 25},
				{20, 30},
			},
			expectedRanges: []workloadinsight.Range{
				{Start: 0, End: 15},
				{Start: 10, End: 25},
				{Start: 20, End: 30},
			},
		},
		{
			name: "Adjacent ranges",
			inputRanges: [][2]uint64{
				{0, 10},
				{10, 20},
				{20, 30},
			},
			expectedRanges: []workloadinsight.Range{
				{Start: 0, End: 30},
			},
		},
		{
			name: "Mixed ranges",
			inputRanges: [][2]uint64{
				{0, 10},
				{5, 15},
				{20, 25},
				{25, 30},
				{40, 50},
			},
			expectedRanges: []workloadinsight.Range{
				{Start: 0, End: 10},
				{Start: 5, End: 15},
				{Start: 20, End: 30},
				{Start: 40, End: 50},
			},
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			mockReadManager := &MockReadManager{}
			mockReadManager.On("Object").Return(&gcs.MinObject{Name: "test-object", Size: 100}).Maybe()
			ioRenderer, err := workloadinsight.NewRenderer()
			require.NoError(t, err, "Failed to create IORenderer")
			vrm := NewVisualReadManager(mockReadManager, ioRenderer, cfg.WorkloadInsightConfig{})

			for _, r := range tc.inputRanges {
				vrm.acceptRange(r[0], r[1])
			}

			assert.Equal(t, tc.expectedRanges, vrm.readIOs, "Recorded readIOs should match expected merged ranges")
		})
	}
}

func TestVisualReadManager_MergeRanges(t *testing.T) {
	testCases := []struct {
		name                    string
		forwardMergeThresholdMb uint64
		first                   workloadinsight.Range
		second                  workloadinsight.Range
		expected                workloadinsight.Range
		merge                   bool
	}{
		{
			name:                    "Overlapping ranges",
			forwardMergeThresholdMb: 0,
			first:                   workloadinsight.Range{Start: 0, End: 10},
			second:                  workloadinsight.Range{Start: 5, End: 15},
			expected:                workloadinsight.Range{},
			merge:                   false,
		},
		{
			name:                    "Adjacent ranges",
			forwardMergeThresholdMb: 0,
			first:                   workloadinsight.Range{Start: 10, End: 20},
			second:                  workloadinsight.Range{Start: 20, End: 30},
			expected:                workloadinsight.Range{Start: 10, End: 30},
			merge:                   true,
		},
		{
			name:                    "Non-overlapping ranges",
			forwardMergeThresholdMb: 0,
			first:                   workloadinsight.Range{Start: 0, End: 10},
			second:                  workloadinsight.Range{Start: 15, End: 25},
			expected:                workloadinsight.Range{},
			merge:                   false,
		},
		{
			name:                    "Within forward merge threshold",
			forwardMergeThresholdMb: 1, // 1 MB
			first:                   workloadinsight.Range{Start: 0, End: 10},
			second:                  workloadinsight.Range{Start: 1 * MiB, End: 2 * MiB},
			expected:                workloadinsight.Range{Start: 0, End: 2 * MiB},
			merge:                   true,
		},
		{
			name:                    "Exceeding forward merge threshold",
			forwardMergeThresholdMb: 1, // 1 MB
			first:                   workloadinsight.Range{Start: 0, End: 10},
			second:                  workloadinsight.Range{Start: 2 * MiB, End: 3 * MiB},
			expected:                workloadinsight.Range{},
			merge:                   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockReadManager := &MockReadManager{}
			ioRenderer, err := workloadinsight.NewRenderer()
			require.NoError(t, err, "Failed to create IORenderer")
			vrm := NewVisualReadManager(mockReadManager, ioRenderer, cfg.WorkloadInsightConfig{ForwardMergeThresholdMb: int64(tc.forwardMergeThresholdMb)})

			mergedRange, ok := vrm.mergeRanges(tc.first, tc.second)

			assert.Equal(t, tc.merge, ok, "Merge result should match expected")
			assert.Equal(t, tc.expected, mergedRange, "Merged range should match expected")
		})
	}
}

func TestVisualReadManager_ReadAt(t *testing.T) {
	mockReadManager := &MockReadManager{}
	mockReadManager.On("Object").Return(&gcs.MinObject{Name: "test-object", Size: 100}).Maybe()
	mockReadManager.On("ReadAt", mock.Anything, mock.Anything, mock.Anything).Return(gcsx.ReaderResponse{}, nil).Once()
	ioRenderer, err := workloadinsight.NewRenderer()
	require.NoError(t, err, "Failed to create IORenderer")
	vrm := NewVisualReadManager(mockReadManager, ioRenderer, cfg.WorkloadInsightConfig{})

	_, err = vrm.ReadAt(context.Background(), make([]byte, 20), 10)
	require.NoError(t, err, "ReadAt should not return an error")

	expectedRange := workloadinsight.Range{Start: 10, End: 30}
	assert.Len(t, vrm.readIOs, 1, "There should be one recorded range")
	assert.Equal(t, expectedRange, vrm.readIOs[0], "Recorded range should match expected")
	mockReadManager.AssertExpectations(t)
}

func TestVisualReadManager_Destroy(t *testing.T) {
	mockReadManager := &MockReadManager{}
	mockReadManager.On("Object").Return(&gcs.MinObject{Name: "test-object", Size: 100}).Maybe()
	mockReadManager.On("Destroy").Return().Once()
	ioRenderer, err := workloadinsight.NewRenderer()
	require.NoError(t, err, "Failed to create IORenderer")
	vrm := NewVisualReadManager(mockReadManager, ioRenderer, cfg.WorkloadInsightConfig{})
	vrm.acceptRange(0, 10)
	vrm.acceptRange(20, 30)

	vrm.Destroy()

	mockReadManager.AssertExpectations(t)
}

func TestVisualReadManager_Destroy_WithOutputFile(t *testing.T) {
	mockReadManager := &MockReadManager{}
	mockReadManager.On("Object").Return(&gcs.MinObject{Name: "test-object", Size: 100}).Maybe()
	mockReadManager.On("Destroy").Return().Once()
	ioRenderer, err := workloadinsight.NewRenderer()
	require.NoError(t, err, "Failed to create IORenderer")
	outputFilePath := "test_output.txt"
	vrm := NewVisualReadManager(mockReadManager, ioRenderer, cfg.WorkloadInsightConfig{OutputFile: outputFilePath})
	vrm.acceptRange(0, 10)
	vrm.acceptRange(20, 40)

	vrm.Destroy()

	// Verify that the output file was created and contains data.
	data, err := os.ReadFile(outputFilePath)
	assert.NoError(t, err, "Should be able to read the output file")
	assert.NotEmpty(t, data, "Output file should not be empty")
	err = os.Remove(outputFilePath)
	assert.NoError(t, err, "Should be able to delete the output file")
	mockReadManager.AssertExpectations(t)
}

func TestAppendToFile_EmptyFile(t *testing.T) {
	outputFilePath := "test_append_output.txt"
	text1 := "First line of text.\n"

	err := appendToFile(outputFilePath, text1)

	assert.NoError(t, err, "First appendToFile should not return an error")
	data, err := os.ReadFile(outputFilePath)
	assert.NoError(t, err, "Should be able to read the output file")
	assert.Equal(t, text1, string(data), "Output file content should match expected")
	err = os.Remove(outputFilePath)
	assert.NoError(t, err, "Should be able to delete the output file")
}

func TestAppendToFile_NonEmptyFile(t *testing.T) {
	outputFilePath := "test_append_output.txt"
	text1 := "First line of text.\n"
	err := appendToFile(outputFilePath, text1)
	require.NoError(t, err, "First appendToFile should not return an error")

	text2 := "Second line of text.\n"
	err = appendToFile(outputFilePath, text2)
	require.NoError(t, err, "Second appendToFile should not return an error")

	data, err := os.ReadFile(outputFilePath)
	assert.NoError(t, err, "Should be able to read the output file")
	assert.Equal(t, text1+text2, string(data), "Output file content should match expected")
	err = os.Remove(outputFilePath)
	assert.NoError(t, err, "Should be able to delete the output file")
}
