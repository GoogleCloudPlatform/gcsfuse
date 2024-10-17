package prefetch

import "github.com/googlecloudplatform/gcsfuse/v2/internal/logger"

type BufferSizePredictor struct {
	prefetchConfig     *Configuration
	previousBufferSize uint32

	// Adaptive buffering logic fields.
}

func NewBufferSizePredictor(prefetchConfig *Configuration) *BufferSizePredictor {
	return &BufferSizePredictor{
		prefetchConfig:     prefetchConfig,
		previousBufferSize: 0,
	}
}

func (bsp *BufferSizePredictor) ReadRequeustRange(start uint32, end uint32) {
	// TODO: to implement.
}

func (bsp *BufferSizePredictor) GetCorrectBufferSize(currentBufferSize uint64) (predictedBufferSize uint64) {
	if currentBufferSize == 0 {
		predictedBufferSize = bsp.prefetchConfig.FirstBufferSize
	} else {
		newBufferSize := currentBufferSize * bsp.prefetchConfig.SequentialPrefetchMultiplier

		if newBufferSize > bsp.prefetchConfig.MaxBufferSize {
			newBufferSize = bsp.prefetchConfig.MaxBufferSize
		}
		predictedBufferSize = newBufferSize
	}
	logger.Tracef("CurrentbufferSize: %d, PredictedBufferSize: %d", currentBufferSize, predictedBufferSize)
	return
}
