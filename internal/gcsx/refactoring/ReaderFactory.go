package refactoring

import "github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"

type ReaderFactory struct {
	bucket gcs.Bucket
}

type DataReader interface {
	read(offset int, limit int) []byte
}

func (rf ReaderFactory) GetReader(readerType string) DataReader {
	//return SingleRangeReader or MultiRangeReader based bucket type and read type
	return nil
}
