package poc

import (
	"context"
	"io"
	"sync"
)

type MultiRangeDownloader struct {
	mu         sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
	readHandle []byte
}

func NewMultiRangeDownloader(ctx context.Context) (*MultiRangeDownloader, error) {

	ctx, cancel := context.WithCancel(ctx)
	mrd := &MultiRangeDownloader{
		ctx:    ctx,
		cancel: cancel,
	}

	return mrd, nil
}

func (mrd *MultiRangeDownloader) Add(output io.Writer, offset, length int64, callback func(int64, int64)) int {
	// Downloads the data and adds in output
	return 0
}

func (mrd *MultiRangeDownloader) Close() error {
	// Close closes the MultiRangeDownloader and cancels all in-progress downloads.
	return nil
}
func (mrd *MultiRangeDownloader) Wait() {
	// Wait waits for all ongoing requests to finish processing.
}

// GetHandle returns the read handle for the object, if available.
func (mrd *MultiRangeDownloader) GetHandle() []byte {
	return mrd.readHandle
}
