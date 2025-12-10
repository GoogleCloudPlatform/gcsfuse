// Copyright 2025 Google Inc. All Rights Reserved.

package folio

import (
	"container/list"
	"fmt"
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
)

// Create a new folio.
func NewFolio(start, end int64, inode *inode.Inode) *Folio {
	return &Folio{
		Start: start,
		End:   end,
		inode: inode,
		done:  make(chan struct{}),
	}
}

// AllocateFolios creates multiple folios for the given range [start, end).
// The range is distributed across multiple folios, where each folio corresponds
// to one or more buffers from the allocated blocks. Each folio's size is determined
// by the buffer length(s) it contains.
func AllocateFolios(start, end int64, inode *inode.Inode, pool *SmartPool) ([]*Folio, error) {
	size := end - start
	if size <= 0 {
		return nil, fmt.Errorf("invalid range size: %d", size)
	}

	// Allocate blocks using the smart pool
	alloc, err := pool.Allocate(int(size))
	if err != nil {
		return nil, fmt.Errorf("failed to allocate blocks: %w", err)
	}

	// Create folios, one per buffer
	folios := make([]*Folio, 0, len(alloc.Blocks))
	currentOffset := start

	for _, poolBlock := range alloc.Blocks {
		bufferSize := int64(len(poolBlock.Data))
		if bufferSize == 0 {
			continue
		}

		// Calculate folio end, ensuring we don't exceed the requested range
		folioEnd := currentOffset + bufferSize
		if folioEnd > end {
			folioEnd = end
			// Trim the buffer to match the actual size needed
			poolBlock.Data = poolBlock.Data[:folioEnd-currentOffset]
		}

		// Create a Block with single buffer for this folio
		block := &Block{
			Data:     poolBlock.Data,
			chunk:    poolBlock.chunk,
			blockIdx: poolBlock.blockIdx,
		}

		folio := &Folio{
			Start: currentOffset,
			End:   folioEnd,
			inode: inode,
			block: block,
			done:  make(chan struct{}),
		}

		folios = append(folios, folio)
		currentOffset = folioEnd

		// Stop if we've covered the entire range
		if currentOffset >= end {
			break
		}
	}

	return folios, nil
}

// A Folio is a reference counted range within an inode that is backed by
// a memory block that can have an active read task on it.
type Folio struct {
	Start    int64
	End      int64
	refcount int

	inode    *inode.Inode
	block    *Block
	listNode *list.Element
	done     chan struct{}
}

func (f *Folio) describe() any {
	return &struct {
		Start, End, Size int64
		Refcount         int
		Cached           bool
	}{
		Start:    f.Start,
		End:      f.End,
		Size:     f.End - f.Start,
		Refcount: f.refcount,
		Cached:   f.IsDone(),
	}
}

func (f *Folio) Size() int64 {
	return f.End - f.Start
}

// Return the folio's reference count
func (f *Folio) Refcount() int {
	return f.refcount
}

// Increase the refcount for this folio. The inode lock must be held.
func (f *Folio) IncRef() {
	f.refcount++
}

// Decrease the refcount for this folio. The inode lock must be held.
// If a folio's refcount reaches zero, it becomes reclaimable.
func (f *Folio) DecRef() {
	if f.refcount == 0 {
		panic("trying to decrease refcount when refcount == 0")
	}
	f.refcount--
}

func (f *Folio) Done() <-chan struct{} {
	return f.done
}

func (f *Folio) IsDone() bool {
	select {
	case <-f.done:
		return true
	default:
		return false
	}
}

// FolioRefs owns references to multiple folios. It is used to protect folios
// from reclamation when reading them in and returning slices to FUSE.
type FolioRefs struct {
	inode       *inode.Inode
	folios      []*Folio
	newReads    int64 // for stats
	cachedReads int64 // for stats
}

func (refs *FolioRefs) describe() any {
	folioDescs := make([]any, 0)
	for _, folio := range refs.folios {
		folioDescs = append(folioDescs, folio.describe())
	}
	return folioDescs
}

func (refs *FolioRefs) Add(folio *Folio) {
	// Normally we'd slice the buffer of the folio here, but in the simulator
	// we slice the inode's mmap.
	folio.IncRef()
	refs.folios = append(refs.folios, folio)
}

// Wait until all folios are done.
func (refs *FolioRefs) Wait() {
	wg := sync.WaitGroup{}
	wg.Add(len(refs.folios))
	for _, folio := range refs.folios {
		go func() { <-folio.Done(); wg.Done() }()
	}
	wg.Wait()
}

// Release references to all folios. Typically, you'd call this method from
// fuseops.ReadFileOp.Callback.
func (refs *FolioRefs) Release() {
	refs.folios = nil
}

// Return a list of slices that cover [start, end).
func (refs *FolioRefs) Slice(start, end int64) (int, [][]byte) {
	sglist := make([][]byte, 0, len(refs.folios))
	nbytes := 0
	for _, folio := range refs.folios {
		if start >= folio.End || end <= folio.Start || start == end {
			continue
		}
		if folio.block == nil || len(folio.block.Data) == 0 {
			continue
		}
		// bufLen can be less than folio.Size() at the end of a file
		bufLen := int64(len(folio.block.Data))
		lo := min(bufLen, max(0, start-folio.Start))
		hi := min(bufLen, max(0, end-folio.Start))
		sglist = append(sglist, folio.block.Data[lo:hi])
		nbytes += int(hi - lo)
	}
	return nbytes, sglist
}
