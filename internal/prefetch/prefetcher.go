package prefetch

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

type Prefetcher struct {
	bufferSizePredictor *BufferSizePredictor
	bufferQueue         *Queue
	downloader          *Downloader
	currentPart         *Part
	minObject           *gcs.MinObject
	errChan             chan error
}

func NewPrefetcher(prefetchConf *Configuration, bucket gcs.Bucket, minObject *gcs.MinObject) *Prefetcher {
	return &Prefetcher{
		bufferSizePredictor: NewBufferSizePredictor(prefetchConf),
		bufferQueue:         NewQueue(),
		downloader:          NewDownloader(bucket, minObject),
		currentPart:         &Part{buff: &bytes.Buffer{}},
		minObject:           minObject,
	}
}

func (p *Prefetcher) downloadAsync(offset uint64, bufferSize uint64, done chan error) {
	buff, err := p.downloader.Download(p.getCorrectInterfaceType(bufferSize), offset, uint64(bufferSize))

	if err != nil {
		logger.Tracef("downloadAsync: %v", err)
		done <- err
		return
	}
	logger.Tracef("download job completed")
	p.bufferQueue.Push(&Part{offset, offset + bufferSize, buff})
	logger.Tracef("Buffer %d to %d has been pushed into queue.", offset, offset+bufferSize)

	done <- nil
	//logger.Tracef("Before pushing to the queue downloaded content: %s", string(buff.Bytes()))
}

// Triggers a job to pre-fetch data based on heuristic.
// Blocks if not enough downloaded content to server the current read request.
func (p *Prefetcher) scheduleNewIfRequired(offset int64, len int) error {
	if p.bufferQueue.IsEmpty() {
		if p.currentPart.buff.Len() < 4*util.MiB {
			// Schedule a job if not any.
			if !p.downloader.downloadInProgress {
				currentBufferSize := p.currentPart.endOffset - p.currentPart.startOffset
				nextBufferSize := p.bufferSizePredictor.GetCorrectBufferSize(currentBufferSize)
				nextPartStartOffset := p.currentPart.endOffset
				if nextPartStartOffset < p.minObject.Size {
					if nextPartStartOffset+nextBufferSize > p.minObject.Size {
						nextBufferSize = p.minObject.Size - nextPartStartOffset
					}
					logger.Tracef("buffer size to download: %d", nextBufferSize)
					p.errChan = make(chan error)
					go p.downloadAsync(nextPartStartOffset, nextBufferSize, p.errChan)
				}
			}
		}

		if p.currentPart.buff.Len() < len {
			err := <-p.errChan // Blocks until we have enough data to server.
			if err != nil {
				return fmt.Errorf("error in download job: %w", err)
			}
		}
	}

	return nil
}

func (p *Prefetcher) refreshCurrentBufferIfNeeded() error {
	if p.currentPart.buff == nil || p.currentPart.buff.Len() == 0 {
		p.currentPart.buff = nil
		if p.bufferQueue.IsEmpty() {
			return fmt.Errorf("buffer-queue shouldn't be empty")
		}

		entry := p.bufferQueue.Pop()
		if buff, ok := entry.(*Part); ok {
			p.currentPart = buff
			//logger.Tracef("After popping from the queue downloaded content: %s", string(buff.Bytes()))
		} else {
			return fmt.Errorf("buffered-queue data-type mismatch")
		}
	}
	return nil
}

// Read
// Waiting call if buffer doesn't contain the requested content.
func (p *Prefetcher) Read(ctx context.Context, dst []byte, offset int64) (n int, err error) {
	logger.Tracef("prefetcher gets the length: %d", len(dst))
	err = p.scheduleNewIfRequired(offset, len(dst))
	if err != nil {
		n = 0
		err = fmt.Errorf("error in scheduling job: %w", err)
		return
	}

	err = p.refreshCurrentBufferIfNeeded()
	if err != nil {
		return 0, fmt.Errorf("error while refreshing the current buffer")
	}

	if p.currentPart.buff.Len() < len(dst) {
		previousBufferRemainingSize := p.currentPart.buff.Len()
		var partialWritten, remainingWritten int
		partialWritten, err = io.ReadFull(p.currentPart.buff, dst[:previousBufferRemainingSize])
		if err != nil || partialWritten != previousBufferRemainingSize {
			return 0, fmt.Errorf("prefetch: partial first-read error %w", err)
		}

		err = p.refreshCurrentBufferIfNeeded()
		if err != nil {
			return 0, fmt.Errorf("error while partialread refreshing the current buffer")
		}

		remainingWritten, err = io.ReadFull(p.currentPart.buff, dst[previousBufferRemainingSize:])
		if err != nil || remainingWritten != (len(dst)-previousBufferRemainingSize) {
			return 0, fmt.Errorf("prefetch: partial remaining-read error %w", err)
		}

		n = len(dst)
		//logger.Tracef("final content: %s", strings.TrimSpace(string(dst)))
		return
	}

	n, err = p.currentPart.buff.Read(dst)
	if err != nil || n != len(dst) {
		return 0, fmt.Errorf("prefetch: read error %w", err)
	}
	//logger.Tracef("final in the end of prefetcher content: %s", string(dst))
	return
}

func (p *Prefetcher) getCorrectInterfaceType(bufferSize uint64) string {
	return Standard
}
