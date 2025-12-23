// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gcsx

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

type mrdEntry struct {
	mrd gcs.MultiRangeDownloader
	mu  sync.RWMutex
	wg  sync.WaitGroup
}

type mrdPool struct {
	entries        []*mrdEntry
	size           int
	current        uint64
	ctx            context.Context
	cancelCreation context.CancelFunc
	creationWg     sync.WaitGroup
	bucket         gcs.Bucket
	object         *gcs.MinObject
}

func newMRDPool(bucket gcs.Bucket, object *gcs.MinObject, handle []byte, size int) (*mrdPool, error) {
	if size <= 0 {
		size = 1
	}
	p := &mrdPool{
		entries: make([]*mrdEntry, size),
		size:    size,
		bucket:  bucket,
		object:  object,
	}
	for i := range p.entries {
		p.entries[i] = &mrdEntry{}
	}

	// Create the first MRD synchronously.
	mrd, err := bucket.NewMultiRangeDownloader(context.Background(), &gcs.MultiRangeDownloaderRequest{
		Name:           object.Name,
		Generation:     object.Generation,
		ReadCompressed: object.HasContentEncodingGzip(),
		ReadHandle:     handle,
	})
	if err != nil {
		return nil, err
	}
	p.entries[0].mrd = mrd

	// Create the rest of the MRDs asynchronously.
	if size > 1 {
		handle := mrd.GetHandle()
		p.ctx, p.cancelCreation = context.WithCancel(context.Background())
		p.creationWg.Add(1)
		go func() {
			defer p.creationWg.Done()
			p.createRemainingMRDs(handle)
		}()
	}

	return p, nil
}

func (p *mrdPool) createRemainingMRDs(handle []byte) {
	for i := 1; i < p.size; i++ {
		if p.ctx.Err() != nil {
			return
		}
		mrd, err := p.bucket.NewMultiRangeDownloader(context.Background(), &gcs.MultiRangeDownloaderRequest{
			Name:           p.object.Name,
			Generation:     p.object.Generation,
			ReadCompressed: p.object.HasContentEncodingGzip(),
			ReadHandle:     handle,
		})
		if err == nil {
			p.entries[i].mu.Lock()
			p.entries[i].mrd = mrd
			p.entries[i].mu.Unlock()
		}
	}
}

func (p *mrdPool) next() *mrdEntry {
	idx := atomic.AddUint64(&p.current, 1) % uint64(p.size)
	return p.entries[idx]
}

func (p *mrdPool) recreateMRD(entry *mrdEntry, fallbackHandle []byte) {
	entry.mu.Lock()
	defer entry.mu.Unlock()

	var handle []byte
	if entry.mrd != nil {
		handle = entry.mrd.GetHandle()
		entry.mrd.Close()
	} else {
		handle = fallbackHandle
	}

	mrd, err := p.bucket.NewMultiRangeDownloader(context.Background(), &gcs.MultiRangeDownloaderRequest{
		Name:           p.object.Name,
		Generation:     p.object.Generation,
		ReadCompressed: p.object.HasContentEncodingGzip(),
		ReadHandle:     handle,
	})

	if err == nil {
		entry.mrd = mrd
	}
}

func (p *mrdPool) close() (handle []byte) {
	if p.cancelCreation != nil {
		p.cancelCreation()
	}
	p.creationWg.Wait()

	for _, entry := range p.entries {
		entry.mu.Lock()
		if entry.mrd != nil {
			entry.wg.Wait()
			if handle == nil {
				handle = entry.mrd.GetHandle()
			}
			entry.mrd.Close()
			entry.mrd = nil
		}
		entry.mu.Unlock()
	}
	return
}
