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
	"sync"

	"fmt"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workloadinsight"
)

const (
	// mergeThreshold defines the gap tolerance for merging adjacent ranges
	// Currently set to 0 meaning ranges must be exactly adjacent to merge
	mergeThreshold = 0
)

type VisualReadManager struct {
	gcsx.ReadManager
	wrapped gcsx.ReadManager

	// Renderer for visualizing read I/O patterns.
	ioRenderer *workloadinsight.Renderer

	// List of recorded read I/O ranges.
	readIOs []workloadinsight.Range

	// Guards access to readIOs slice.
	mu sync.Mutex

	// Optional output file for writing the visualization.
	outputFilePath string
}

// NewVisualReadManager creates a new VisualReadManager that wraps
// an existing ReadManager and uses the provided IORenderer to visualize
// read I/O patterns.
// The visualization is output to outputFilePath when Destroy() is called.
// In case outputFilePath is empty, output is printed to stdout.
func NewVisualReadManager(wrapped gcsx.ReadManager, ioRenderer *workloadinsight.Renderer, outputFilePath string) *VisualReadManager {
	return &VisualReadManager{
		wrapped:        wrapped,
		ioRenderer:     ioRenderer,
		readIOs:        []workloadinsight.Range{},
		mu:             sync.Mutex{},
		outputFilePath: outputFilePath,
	}
}

// ReadAt records the read I/O range and delegates the read to the wrapped ReadManager.
func (vrm *VisualReadManager) ReadAt(ctx context.Context, p []byte, offset int64) (gcsx.ReaderResponse, error) {
	// Capture the range in the visualizer
	if len(p) > 0 {
		vrm.acceptRange(uint64(offset), uint64(offset)+uint64(len(p)))
	}
	// Delegate to the wrapped ReadManager
	return vrm.wrapped.ReadAt(ctx, p, offset)
}

// Destroy outputs the read I/O visualization and destroys the wrapped ReadManager.
func (vrm *VisualReadManager) Destroy() {
	defer vrm.wrapped.Destroy()

	output, err := vrm.ioRenderer.Render(vrm.wrapped.Object().Name, vrm.wrapped.Object().Size, vrm.readIOs)
	if err != nil {
		logger.Warnf("Failed to render read pattern: %v", err)
		return
	}
	if vrm.outputFilePath != "" {
		f, err := os.OpenFile(vrm.outputFilePath, os.O_APPEND|os.O_WRONLY, 0600)
		if err == nil {
			defer f.Close()
			_, err = f.Write([]byte(output))
			if err == nil {
				return
			}
			logger.Warnf("Failed to create output file: %v", err)
		}
	}

	// Fallback to logging output
	fmt.Println(output)
}

// acceptRange records a read I/O range and merges it with existing ranges if possible.
func (vrm *VisualReadManager) acceptRange(start, end uint64) {
	if end <= start {
		return // Invalid range, ignore
	}

	// Clamp end to object size.
	if end > vrm.wrapped.Object().Size {
		end = vrm.wrapped.Object().Size
	}

	newRange := workloadinsight.Range{
		Start: start,
		End:   end,
	}

	vrm.mu.Lock()
	defer vrm.mu.Unlock()

	// Try to merge with the last range if possible
	if len(vrm.readIOs) > 0 {
		lastRange := &vrm.readIOs[len(vrm.readIOs)-1]

		if mergedRange, ok := vrm.mergeRanges(*lastRange, newRange); ok {
			// Merge the readIOs by extending the last range
			vrm.readIOs[len(vrm.readIOs)-1] = mergedRange
			return
		}
	}

	// No merge possible, add as new range
	vrm.readIOs = append(vrm.readIOs, newRange)
}

// mergeRanges combines two readIOs into a single range.
// Merge happens when second range is ahead of first range.End and within the merge threshold.
func (vrm *VisualReadManager) mergeRanges(first, second workloadinsight.Range) (workloadinsight.Range, bool) {
	if first.End > second.Start || first.End+mergeThreshold < second.Start {
		return workloadinsight.Range{}, false
	}

	return workloadinsight.Range{Start: first.Start, End: second.End}, true
}
