package block

type BlockPool struct {
	// Channel holding free blocks
	blocksCh chan Block

	// Size of each block this pool holds
	blockSize int64

	// Number of block that this pool can handle at max
	maxBlocks uint32

	// Number of blocks created so far.
	numBlocks uint32

	// Holds the type of buffers to be created - memory/disk
	blockType string

	// disk path for files incase of diskBuffers.
	diskPath string
}

// InitBlockPool - Get all required params and do init
func InitBlockPool(bs int64, mb uint32) *BlockPool {
	return &BlockPool{
		blocksCh:  make(chan Block, mb),
		blockSize: bs,
		maxBlocks: mb,
		numBlocks: 0,
		blockType: "memory",
		diskPath:  "",
	}
}

func (ib *BlockPool) Get() (Block, error) {
	var b Block

	for {
		select {
		case b = <-ib.blocksCh:
			return b, nil

		default:
			if ib.numBlocks < ib.maxBlocks {
				var err error
				b, err = ib.createBlock()
				if err != nil {
					return nil, err
				}
				// Mark the buffer ready for reuse now.
				b.Reuse(ib.blocksCh)
			}
		}
	}
}

func (ib *BlockPool) createBlock() (Block, error) {
	ib.numBlocks++
	switch ib.blockType {
	case "memory":
		mb := memoryBlock{
			buffer:         make([]byte, ib.blockSize, ib.blockSize),
			offset:         offset{0, 0},
			readerPosition: 0,
		}
		return &mb, nil
		//case "disk":
		//	db := diskBlock{}
		//	return &db, nil
	}

	return nil, nil
}
