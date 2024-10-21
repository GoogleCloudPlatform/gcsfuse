package prefetch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"golang.org/x/sync/errgroup"
)

const (
	Standard        = "standard"
	FastByte        = "fast-byte"
	TransferManager = "transfer-manager"
)

type Downloader struct {
	bucket             gcs.Bucket
	minObj             *gcs.MinObject
	downloadInProgress bool
	downloadedData     uint64
	eG                 errgroup.Group
}

func NewDownloader(bucket gcs.Bucket, object *gcs.MinObject) *Downloader {
	return &Downloader{
		bucket: bucket,
		minObj: object,
	}
}

func (d *Downloader) Download(sdkInterfaceType string, offset uint64, len uint64) (buff *bytes.Buffer, err error) {
	d.downloadInProgress = true
	defer func() {
		d.downloadInProgress = false
	}()

	if len == 0 {
		return
	}

	rangeEnd := offset + len
	if rangeEnd > d.minObj.Size {
		rangeEnd = d.minObj.Size
		len = d.minObj.Size - offset
		logger.Tracef("rangeEnd: %d, Size: %d", rangeEnd, d.minObj.Size)
	}

	rawBuffer := make([]byte, int(len))
	n, err := d.ParallelDownload(offset, rangeEnd, rawBuffer)
	if uint64(n) != len {
		return nil, fmt.Errorf("while parallel download: %w", err)
	}

	buff = bytes.NewBuffer(rawBuffer)
	logger.Tracef("length of the buffer allocated: %d", buff.Len())

	if err != nil {
		return nil, fmt.Errorf("error while closing the gcs_reader: %w", err)
	}

	d.downloadedData = offset + len

	return
}

func (d *Downloader) ParallelDownload(start uint64, end uint64, rawBuffer []byte) (n int, err error) {

	requestSize := end - start

	lenBuf := len(rawBuffer)
	if uint64(lenBuf) != requestSize {
		return 0, errors.New("invalid request")
	}

	if lenBuf == 0 {
		return 0, nil
	}

	if lenBuf < 28*util.MiB {
		return d.SingleDownload(start, end, rawBuffer)
	}

	// One worker will read 16 MB
	for s := start; s < end; s += 16 * util.MiB {
		ss := s
		ee := min(end, s+16*util.MiB)
		d.eG.Go(func() error {
			_, errS := d.SingleDownload(ss, ee, rawBuffer[(ss-start):(ee-start)])
			if errS != nil {
				errS = fmt.Errorf("error in download: %d to %d: %w", ss, ee, errS)
				return errS
			}
			return nil
		})
	}
	err = d.eG.Wait()
	if err != nil {
		return 0, fmt.Errorf("error while parallel download: %w", err)
	}

	return lenBuf, nil
}

func (d *Downloader) SingleDownload(start uint64, end uint64, rawBuffer []byte) (n int, err error) {
	logger.Tracef("Downloading %d to %d", start, end)
	len := end - start

	rc, err := d.bucket.NewReader(
		context.Background(),
		&gcs.ReadObjectRequest{
			Name:       d.minObj.Name,
			Generation: d.minObj.Generation,
			Range: &gcs.ByteRange{
				Start: start,
				Limit: end,
			},
			ReadCompressed: d.minObj.HasContentEncodingGzip(),
		})

	if err != nil {
		return 0, fmt.Errorf("reader creation failed: %w", err)
	}

	var copiedSize int
	copiedSize, err = io.ReadFull(rc, rawBuffer)

	if err != nil || uint64(copiedSize) != len {
		return 0, fmt.Errorf("downloading error: %w", err)
	}

	return int(len), nil
}
