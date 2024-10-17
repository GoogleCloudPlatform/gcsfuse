package block

import (
	"fmt"
	"syscall"
)

// BlockPool handles the creation of blocks as per the user configuration.
type BlockPool struct {
	// Channel holding free blocks.
	blocksCh chan Block

	// Size of each block this pool holds.
	blockSize int64

	// Max number of blocks this blockPool can create.
	maxBlocks int32

	// Total number of blocks created so far.
	totalBlocks int32
}

// InitBlockPool initializes the blockPool based on the user configuration.
func InitBlockPool(blockSize int64, maxBlocks int32) (bp *BlockPool, err error) {
	if blockSize <= 0 || maxBlocks <= 0 {
		err = fmt.Errorf("invalid configuration provided for blockPool, blocksize: %d, maxBlocks: %d", blockSize, maxBlocks)
		return
	}

	bp = &BlockPool{
		blocksCh:    make(chan Block, maxBlocks),
		blockSize:   blockSize,
		maxBlocks:   maxBlocks,
		totalBlocks: 0,
	}
	return
}

// Get returns a block. It returns an existing block if it's ready for reuse or
// creates a new one if required.
func (ib *BlockPool) Get() (Block, error) {
	for {
		select {
		case b := <-ib.blocksCh:
			// Reset the block for reuse.
			b.Reuse()
			return b, nil

		default:
			// No lock is required here since blockPool is per file and all write
			// calls to a single file are serialized because of inode.lock().
			if ib.totalBlocks < ib.maxBlocks {
				b, err := ib.createBlock()
				if err != nil {
					return nil, err
				}

				ib.totalBlocks++
				return b, nil
			}
		}
	}
}

// createBlock creates a new block.
func (ib *BlockPool) createBlock() (Block, error) {
	prot, flags := syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE
	addr, err := syscall.Mmap(-1, 0, int(ib.blockSize), prot, flags)
	if err != nil {
		return nil, fmt.Errorf("mmap error: %v", err)
	}

	mb := memoryBlock{
		buffer: addr,
		offset: offset{0, 0},
	}
	return &mb, nil
}

// BlockSize returns the block size used by the blockPool.
func (ib *BlockPool) BlockSize() int64 {
	return ib.blockSize
}
