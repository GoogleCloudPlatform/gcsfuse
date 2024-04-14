package prefetch

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

const (
	Standard        = "standard"
	FastByte        = "fast-byte"
	TransferManager = "transfer-manager"
)

type Downloader struct {
	bucket             gcs.Bucket
	minObj             *gcs.MinObject
	readCloser         io.ReadCloser
	downloadInProgress bool
	downloadedData     uint64
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

	if d.readCloser != nil {
		return nil, fmt.Errorf("inconsistent state")
	}

	rangeEnd := offset + len
	if rangeEnd > d.minObj.Size {
		rangeEnd = d.minObj.Size
		len = d.minObj.Size - offset
		logger.Tracef("rangeEnd: %d, Size: %d", rangeEnd, d.minObj.Size)
	}

	d.readCloser, err = d.bucket.NewReader(
		context.Background(),
		&gcs.ReadObjectRequest{
			Name:       d.minObj.Name,
			Generation: d.minObj.Generation,
			Range: &gcs.ByteRange{
				Start: offset,
				Limit: rangeEnd,
			},
			ReadCompressed: d.minObj.HasContentEncodingGzip(),
		})

	if err != nil {
		return nil, fmt.Errorf("reader creation failed: %w", err)
	}

	rawBuffer := make([]byte, int(len))

	var copiedSize int
	copiedSize, err = io.ReadFull(d.readCloser, rawBuffer)
	logger.Tracef("copied size: %d, len: %d", copiedSize, len)
	if err != nil || uint64(copiedSize) != len {
		return nil, fmt.Errorf("downloading error: %w", err)
	}

	buff = bytes.NewBuffer(rawBuffer)
	logger.Tracef("length of the buffer allocated: %d", buff.Len())

	//logger.Tracef("downloaded content: %s", strings.TrimSpace(string(buff.Bytes())))

	err = d.readCloser.Close()
	d.readCloser = nil
	if err != nil {
		return nil, fmt.Errorf("error while closing the gcs_reader: %w", err)
	}

	d.downloadedData = offset + len

	return
}
