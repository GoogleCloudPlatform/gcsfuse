package fs

/*
import "math"

type InMemoryBuffer struct {
	WriteBuffer
	Block
	current       *bytebuffer
	blockpool     BlockPool
	uploadHandler uploadHandler
}

type offset struct {
	start, end int64
}

type bytebuffer struct {
	buffer []byte
	offset offset
}

func (db *bytebuffer) size() int64 {
	return db.offset.end - db.offset.start
}

func (db *bytebuffer) reuse() {
}

type BlockPool struct {
	// Channel holding free blocks
	blocksCh chan *Block

	// Size of each block this pool holds
	blockSize int64

	// Number of block that this pool can handle at max
	maxBlocks uint32

	// Number of blocks yet to be uploaded.
	numBlocks uint32
}

func (p BlockPool) get() (*Block, error) {

}

func (ib *InMemoryBuffer) write(data []byte, offset int64) (err error) {
	dataWritten := int(0)
	for dataWritten < len(data) {
		if ib.current == nil {
			ib.current, err = ib.getDataBuffer()
			if err != nil {
				returning
			}
		}

		bytesToCopy := int(math.Min(float64(ib.blockpool.blockSize-ib.current.size()), float64(len(data))))
		copy(ib.current.buffer[ib.current.offset.end:], data[dataWritten:bytesToCopy])
		ib.current.offset.end += int64(bytesToCopy)
		dataWritten += bytesToCopy

		if ib.current.size() == ib.blockpool.blockSize {

			block := inmemorybuffer{}
			ib.uploadHandler.upload(block)
			// trigger upload
			ib.current = nil
		}
	}

	return
}

func (ib *InMemoryBuffer) getDataBuffer() (*databuffer, error) {
	var b *databuffer = nil

	select {
	case b = <-ib.blockpool.blocksCh:
		break

	default:
		if ib.blockpool.numBlocks < ib.blockpool.maxBlocks {
			// create new buffer, assign it to b, return
		}
		// TODO: Do we want to return error here or wait forever.
		//return nil, nil
	}

	// Mark the buffer ready for reuse now.
	b.reuse()
	return b, nil
}
*/
