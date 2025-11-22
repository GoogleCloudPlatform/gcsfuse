package read_manager

import (
	"context"
	"fmt"

	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

type VisualizingReadManager struct {
	gcsx.ReadManager
	wrapped    gcsx.ReadManager
	visualizer *common.ReadPatternVisualizer
}

func NewVisualizingReadManager(wrapped gcsx.ReadManager) *VisualizingReadManager {
	return &VisualizingReadManager{
		wrapped: wrapped,
		visualizer: common.NewReadPatternVisualizerWithFullConfig( // This line was already changed in the previous diff.
			1024, // 1KB default scale
			100,  // 100 characters width
			"",
			wrapped.Object().Name,
		),
	}
}

func (vrm *VisualizingReadManager) ReadAt(ctx context.Context, p []byte, offset int64) (gcsx.ReaderResponse, error) {
	// Capture the range in the visualizer
	if len(p) > 0 {
		vrm.visualizer.AcceptRange(offset, offset+int64(len(p)))
	}
	// Delegate to the wrapped ReadManager
	return vrm.wrapped.ReadAt(ctx, p, offset)
}

func (vrm *VisualizingReadManager) Destroy() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Warnf("Unable to get home directory to dump read pattern: %v", err)
	}
	vrm.visualizer.DumpGraphToFile(homeDir + "/read_pattern.txt")
	vrm.wrapped.Destroy()
}

func (vrm *VisualizingReadManager) Object() *gcs.MinObject {
	return vrm.wrapped.Object()
}
