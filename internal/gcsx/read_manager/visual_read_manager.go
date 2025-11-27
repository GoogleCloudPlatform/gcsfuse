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
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workloadinsight"
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

	// Configuration for workload insight visualization.
	cfg cfg.WorkloadInsightConfig

	// forwardMergeThreshold is the threshold in bytes for merging adjacent readIOs.
	forwardMergeThreshold uint64
}

// NewVisualReadManager creates a new VisualReadManager that wraps
// an existing ReadManager and uses the provided IORenderer to visualize
// read I/O patterns.
// The visualization is output to outputFilePath when Destroy() is called.
// In case outputFilePath is empty, output is printed to stdout.
func NewVisualReadManager(wrapped gcsx.ReadManager, ioRenderer *workloadinsight.Renderer, cfg cfg.WorkloadInsightConfig) *VisualReadManager {
	return &VisualReadManager{
		wrapped:               wrapped,
		ioRenderer:            ioRenderer,
		readIOs:               []workloadinsight.Range{},
		mu:                    sync.Mutex{},
		cfg:                   cfg,
		forwardMergeThreshold: uint64(cfg.ForwardMergeThresholdMb * gcsx.MiB),
	}
}

// ReadAt records the read I/O range and delegates the read to the wrapped ReadManager.
func (vrm *VisualReadManager) ReadAt(ctx context.Context, req *gcsx.ReadRequest) (gcsx.ReadResponse, error) {
	// Capture the range in the visualizer
	if len(req.Buffer) > 0 {
		vrm.acceptRange(uint64(req.Offset), uint64(req.Offset)+uint64(len(req.Buffer)))
	}
	// Delegate to the wrapped ReadManager
	return vrm.wrapped.ReadAt(ctx, req)
}

// Destroy outputs the read I/O visualization and destroys the wrapped ReadManager.
func (vrm *VisualReadManager) Destroy() {
	defer vrm.wrapped.Destroy()

	output, err := vrm.ioRenderer.Render(vrm.Object().Name, vrm.Object().Size, vrm.readIOs)
	if err != nil {
		logger.Warnf("Failed to render read pattern: %v", err)
		return
	}
	if vrm.cfg.OutputFile == "" {
		fmt.Println(output)
		return
	}

	if err := appendToFile(vrm.cfg.OutputFile, output); err != nil {
		fmt.Println(output)
		logger.Warnf("Failed to append to output file: %v", err)
		return
	}
}

func (vrm *VisualReadManager) Object() *gcs.MinObject {
	return vrm.wrapped.Object()
}

func (vrm *VisualReadManager) CheckInvariants() {
	vrm.wrapped.CheckInvariants()
}

// acceptRange records a read I/O range and merges it with existing ranges if possible.
func (vrm *VisualReadManager) acceptRange(start, end uint64) {
	if end <= start {
		return // Invalid range, ignore
	}

	// Clamp end to object size.
	end = min(end, vrm.Object().Size)

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
func (vrm *VisualReadManager) mergeRanges(first, second workloadinsight.Range) (workloadinsight.Range, bool) {
	if first.End+vrm.forwardMergeThreshold < second.Start {
		return workloadinsight.Range{}, false
	}

	if first.End > second.Start {
		return workloadinsight.Range{}, false
	}

	return workloadinsight.Range{Start: first.Start, End: second.End}, true
}

// appendToFile appends the given text to the specified output file.
// If the file does not exist, it is created.
func appendToFile(outputFilePath, text string) error {
	if outputFilePath == "" {
		return errors.New("output file path is empty")
	}

	f, err := os.OpenFile(outputFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write([]byte(text)); err != nil {
		return fmt.Errorf("failed to write to output file: %w", err)
	}
	return nil
}
