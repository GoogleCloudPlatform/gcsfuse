package fs

/*import (
	"fmt"
	"math"
)
*/
/*type writebuffer struct {
}

type offset struct {
	start, end int64
}

type inmemorybuffer struct {
	buffer []byte
	offset offset
}

func (ib *inmemorybuffer) size() int64 {
	return ib.offset.end - ib.offset.start
}

type inmemorywritebuffer struct {
	previous   inmemorybuffer
	current    inmemorybuffer
	bufferSize int64
}

// TODO: change it to return writebuffer
func Init(buffersize int64) (wb *inmemorywritebuffer) {
	wb = &inmemorywritebuffer{
		bufferSize: buffersize,
	}

	return
}

func (wf *inmemorywritebuffer) write(data []byte, offset int64) {
	wf.ensureCurrent(offset)


  // We only support sequential writes i.e, subsequent write starting at the end offset of previous write
	// Or overwrite the previous written offset in this buffer.
	// We dont support writing at different offsets within the buffer. Since there is no way to ensure entire buffer is filled.
	if offset < wf.current.offset.start || offset > wf.current.offset.end {
		// non-sequential write is happening.
		// upload till this point and continue with edit flow.
	}

	bytesToCopy := int(math.Min(float64(wf.bufferSize-wf.current.size()), float64(len(data))))
	copy(wf.current.buffer[wf.current.offset.end:], data[0:bytesToCopy])
	wf.current.offset.end += int64(bytesToCopy)

	if wf.current.size() == wf.bufferSize {
		wf.ensurePrevious()
		bytesCopied := copy(wf.previous.buffer, wf.current.buffer)
		if bytesCopied != int(wf.bufferSize) {
			// something happened during copy, throw error
		}
		wf.clearCurrent()

		wf.upload()

		if bytesToCopy < len(data) {
			copy(wf.current.buffer, data[bytesToCopy:])
		}
	}

}

// TODO:
func (wf *inmemorywritebuffer) upload() {}

func (wf *inmemorywritebuffer) ensureCurrent(offset int64) {
	if wf.current.buffer == nil {
		if offset != 0 {
			//throw error.
		}
		wf.current.buffer = make([]byte, wf.bufferSize)
		// ensureCurrent should be called only for offset 0 i.e, first time.
		wf.current.offset.end = wf.bufferSize
		}
	}
}

func (wf *inmemorywritebuffer) ensurePrevious() {
	if wf.previous.buffer == nil {
		wf.previous.buffer = make([]byte, wf.bufferSize)
	}
}

func (wf *inmemorywritebuffer) clearCurrent() {
	startOffset := wf.current.offset.end
	clear(wf.current.buffer)
	wf.current.offset.start = startOffset
}*/

/*
type inmemorybuffer struct {
	buffer []byte
	offset offset
}

type filebuffer struct {
	file   *os.File
	offset offset
}

func (ib *inmemorybuffer) size() int64 {
	return ib.offset.end - ib.offset.start
}

func (ib *inmemorybuffer) Reuse() {
	// TODO: Decide what to do here.
}

type blockCache struct {
	blockPool *BlockPool
	current   *inmemorybuffer
}

func (bc *blockCache) write(data []byte, offset int64) (err error) {
	dataWritten := int(0)
	for dataWritten < len(data) {
		if bc.current == nil {
			bc.current, err = bc.getBlock()
			if err != nil {
				return
			}
		}

		bytesToCopy := int(math.Min(float64(bc.blockPool.blockSize-bc.current.size()), float64(len(data))))
		copy(bc.current.buffer[bc.current.offset.end:], data[0:bytesToCopy])
		bc.current.offset.end += int64(bytesToCopy)
		dataWritten += bytesToCopy

		if bc.current.size() == bc.blockPool.blockSize {
			// trigger upload
			bc.current = nil
		}
	}
}

func (bc *blockCache) getBlock() (*inmemorybuffer, error) {
	var b *inmemorybuffer = nil

	select {
	case b = <-bc.blockPool.blocksCh:
		break

	default:
		if bc.blockPool.numBlocks < bc.blockPool.maxBlocks {
			// create new buffer, assign it to b, return
		}
		// TODO: Do we want to return error here or wait forever.
		//return nil, nil
	}

	// Mark the buffer ready for reuse now
	b.Reuse()
	return b, nil
}

//TODO: how to handle ls -lh with list of buffers

func (bc *blockCache) getFile() (*filebuffer, error) {

}
*/
