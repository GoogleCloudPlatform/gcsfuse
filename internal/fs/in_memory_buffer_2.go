package fs

/*
// This is the correct and updated one.
import (
	"math"
	"os"
)

type WriteHandler struct {
	current       Block
	blockpool     BlockPool2
	uploadHandler uploadHandler
}

type Block interface {
	// Implement io.Reader interface
	//io.Reader
	Reuse()
	Size() int64
	Copy(bytes []byte)
	Create() Block
}
type memoryBlock struct {
	buffer []byte
	offset offset2
}

type diskBlock struct {
	buff   *os.File
	offset offset2
}

type offset2 struct {
	start, end int64
}

func (db *memoryBlock) Size() int64 {
	return db.offset.end - db.offset.start
}

func (db *memoryBlock) Reuse() {
}

func (db *memoryBlock) Copy(bytes []byte) {
	bytesCopied := copy(db.buffer[db.offset.end:], bytes)
	db.offset.end += int64(bytesCopied)
	// TODO: return error if bytesCopied != len(bytes)
}

func (db *diskBlock) Reuse() {
}

func (db *diskBlock) Size() int64 {
	return 1
}

func (db *diskBlock) Copy(bytes []byte) {
}

type BlockPool2 struct {
	// Channel holding free blocks
	blocksCh chan *Block

	// Size of each block this pool holds
	blockSize int64

	// Number of block that this pool can handle at max
	maxBlocks uint32

	// Number of blocks yet to be uploaded.
	numBlocks uint32

	// Holds the type of buffers to be created - memory/disk
	blockType string
}

func (ib *WriteHandler) write(data []byte, offset int64) (err error) {
	dataWritten := int(0)
	for dataWritten < len(data) {
		if ib.current == nil {
			ib.current, err = ib.blockpool.get()
			if err != nil {
				return
			}
		}

		bytesToCopy := int(math.Min(float64(ib.blockpool.blockSize-ib.current.Size()), float64(len(data))))
		ib.current.Copy(data[dataWritten:bytesToCopy])
		dataWritten += bytesToCopy

		if ib.current.Size() == ib.blockpool.blockSize {
			ib.uploadHandler.upload(ib.current)
			// trigger upload
			ib.current = nil
		}
	}

	return
}

func (ib *BlockPool2) get() (Block, error) {
	var b Block = nil
	for {
		select {
		case b = <-ib.blocksCh:
			break

		default:
			if ib.numBlocks < ib.maxBlocks {
				b, _ = ib.createBlock()
				break
				// create new buffer, assign it to b, return
			}
			// TODO: Do we want to return error here or wait forever.
			//return nil, nil
		}
	}

	// Mark the buffer ready for reuse now.
	b.Reuse()
	return b, nil
}

func (ib *BlockPool2) createBlock() (Block, error) {
	switch ib.blockType {
	case "memory":
		mb := memoryBlock{}
		return &mb, nil
	case "disk":
		db := diskBlock{}
		return &db, nil
	}
	return nil, nil
}
*/
