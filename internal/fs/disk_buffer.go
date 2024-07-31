package fs

/*
import (
	"math"
	"os"
)

type DiskBuffer struct {
	WriteBuffer
	current   *fileBuffer
	blockSize int64
}

type fileBuffer struct {
	buff   *os.File
	offset offset
}

func (db *DiskBuffer) write(data []byte, offset int64) (err error) {
	dataWritten := int(0)
	for dataWritten < len(data) {
		if db.current == nil {
			db.current, err = db.getFileBuffer()
			if err != nil {
				return
			}
		}

		bytesToCopy := int(math.Min(float64(db.blockSize-db.current.size()), float64(len(data))))
		// io.Copy(db.current.buff, data[dataWritten:bytesToCopy])
		db.current.offset.end += int64(bytesToCopy)
		dataWritten += bytesToCopy

		fileInfo, _ := db.current.buff.Stat()
		if fileInfo.Size() == db.blockSize {
			// trigger upload
			db.current.buff.Close()
		}
	}

	return
}

func (db *DiskBuffer) getFileBuffer() (*fileBuffer, error) {
	// Create a new file.
	return nil, nil
}
*/
