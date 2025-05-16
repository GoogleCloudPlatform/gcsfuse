package main

import (
	"bufio"
	"context"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"golang.org/x/sync/errgroup"
)

type Downloader struct {
	bucket gcs.Bucket
	minObj *gcs.MinObject
	attrs  *gcs.ExtendedObjectAttributes
	eG     errgroup.Group
}

var globalDownloader *Downloader

func init() {
	globalDownloader = NewDownloader("princer-empty-bucket", "logs1.txt")
}

func NewDownloader(bucketName string, objectName string) *Downloader {
	storageClientConfig := storageutil.StorageClientConfig{
		ClientProtocol:             "http1",
		MaxConnsPerHost:            100,
		MaxIdleConnsPerHost:        100,
		HttpClientTimeout:          0,
		MaxRetrySleep:              20,
		RetryMultiplier:            2,
		UserAgent:                  "prince",
		CustomEndpoint:             nil,
		KeyFile:                    "",
		AnonymousAccess:            false,
		TokenUrl:                   "",
		ReuseTokenFromUrl:          false,
		ExperimentalEnableJsonRead: false,
		GrpcConnPoolSize:           1,
	}
	storageHandle, err := storage.NewStorageHandle(context.Background(), storageClientConfig)
	if err != nil {
		fmt.Println("error while creating storage handle")
		os.Exit(1)
	}

	bucketHandle := storageHandle.BucketHandle(bucketName, "")
	minObject, extendedAttrs, err := bucketHandle.StatObject(context.Background(), &gcs.StatObjectRequest{Name: objectName, ReturnExtendedObjectAttributes: true})

	return &Downloader{
		bucket: bucketHandle,
		minObj: minObject,
		attrs:  extendedAttrs,
	}
}

func (d *Downloader) SingleThreadFullFileDownload(fileName string) (err error) {
	return d.rangeDownload(fileName, 0, d.minObj.Size)
}

func (d *Downloader) MultiThreadFullFileDownload(fileName string) (err error) {
	return d.multiThreadRangeDownload(fileName, uint64(0), d.minObj.Size)
}

func (d *Downloader) IncrementalMultiThreadFullFileDownload(fileName string) (err error) {
	start := uint64(0)
	multiplier := uint64(8)
	downloadSize := uint64(8 * util.MiB)
	end := d.minObj.Size

	for start < end {
		availableEnd := min(start+downloadSize, end)
		downloadSize = availableEnd - start
		fmt.Printf("Downloading %d MiB \n", downloadSize/util.MiB)
		err = d.multiThreadRangeDownload(fileName, start, downloadSize)
		if err != nil {
			err = fmt.Errorf("while incremental download: %d to %d", start, availableEnd)
			return
		}

		start += downloadSize
		downloadSize *= multiplier
	}
	return nil
}

func (d *Downloader) multiThreadRangeDownload(fileName string, offset uint64, len uint64) (err error) {
	end := offset + len
	for s := offset; s < end; s += 32 * util.MiB {
		ss := s
		ee := min(end, s+32*util.MiB)
		d.eG.Go(func() error {
			errS := d.rangeDownload(fileName, ss, ee-ss)
			if errS != nil {
				errS = fmt.Errorf("error in downloading: %d to %d: %w", ss, ee, errS)
				return errS
			}
			return nil
		})
	}
	err = d.eG.Wait()
	if err != nil {
		return fmt.Errorf("error while parallel download: %w", err)
	}
	return nil
}

func (d *Downloader) rangeDownload(fileName string, offset uint64, len uint64) (err error) {
	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			err = fmt.Errorf("error in closing the fileHandle")
		}
	}(f)

	_, err = f.Seek(int64(offset), 0)

	rc, err := d.bucket.NewReader(
		context.Background(),
		&gcs.ReadObjectRequest{
			Name:       d.minObj.Name,
			Generation: d.minObj.Generation,
			Range: &gcs.ByteRange{
				Start: offset,
				Limit: offset + len,
			},
			ReadCompressed: d.minObj.HasContentEncodingGzip(),
		})

	copiedData, err := io.Copy(f, rc)
	if copiedData != int64(len) || (err != nil && err != io.EOF) {
		err = fmt.Errorf("error while downloading")
		return
	}

	return nil
}

const bufferSize = 65536

// CRCReader returns CRC-32-Castagnoli checksum of content in reader
func CRCReader(reader io.Reader) (uint32, error) {
	table := crc32.MakeTable(crc32.Castagnoli)
	checksum := crc32.Checksum([]byte(""), table)
	buf := make([]byte, bufferSize)
	for {
		switch n, err := reader.Read(buf); err {
		case nil:
			checksum = crc32.Update(checksum, table, buf[:n])
		case io.EOF:
			return checksum, nil
		default:
			return 0, err
		}
	}
}

func CRC32(filename string) (uint32, error) {
	if info, err := os.Stat(filename); err != nil || info.IsDir() {
		return 0, err
	}

	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer func() { _ = file.Close() }()

	return CRCReader(bufio.NewReader(file))
}

func main() {
	startTime := time.Now()
	//err := globalDownloader.SingleThreadFullFileDownload("single_download.txt")
	//err := globalDownloader.MultiThreadFullFileDownload("parallel_download.txt")
	err := globalDownloader.IncrementalMultiThreadFullFileDownload("incremental_download.txt")
	if err != nil {
		fmt.Printf("error while downloaing file")
	}

	totalTime := time.Since(startTime)

	fmt.Println("Total time to download file: ", totalTime)
	startTime = time.Now()

	crc32, err := CRC32("incremental_download.txt")
	fmt.Println("Downloaded crc32: ", crc32)
	fmt.Println("Actual crc32: ", *(globalDownloader.attrs.CRC32C))
	totalCrcCalculationTime := time.Since(startTime)
	fmt.Println("CRC-32 calculation time: ", totalCrcCalculationTime)

}
