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

package folio

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type FolioTests struct {
	suite.Suite
}

func (s *FolioTests) TestBasic() {
	f := NewFolio(10, 50, 0)
	s.Equal(f.Size(), int64(40))
}

func (s *FolioTests) TestRefcount() {
	f := NewFolio(0, 0, 0)
	s.Equal(f.Refcount(), 0)
	for i := range 5 {
		f.IncRef()
		s.Equal(f.Refcount(), i+1)
	}
	for i := range 5 {
		f.DecRef()
		s.Equal(f.Refcount(), 4-i)
	}
	s.Equal(f.Refcount(), 0)
}

func (s *FolioTests) TestDecRefZeroPanics() {
	f := NewFolio(0, 0, 0)
	s.Panics(func() {
		f.DecRef()
	})
}

func (s *FolioTests) TestDone() {
	f := NewFolio(0, 0, 0)
	s.False(f.IsDone())
	close(f.done)
	s.True(f.IsDone())
}

type FolioRefsTests struct {
	suite.Suite
}

func (s *FolioRefsTests) TestAdd() {
	refs := FolioRefs{}
	f := NewFolio(0, 10, 0)
	refs.Add(f)
	s.Equal(f.Refcount(), 1)
}

func (s *FolioRefsTests) TestWait() {
	refs := FolioRefs{}
	for i := range 5 {
		refs.Add(NewFolio(int64(i*10), int64((i+1)*10), 0))
	}
	isDone := false
	go func() {
		refs.Wait()
		isDone = true
	}()
	timeout := 10 * time.Millisecond
	time.Sleep(timeout)
	s.False(isDone)
	for i := range 5 {
		close(refs.folios[i].done)
		time.Sleep(timeout)
		s.Equal(isDone, i == 4)
	}
}

func (s *FolioRefsTests) TestSliceSingle() {
	refs := FolioRefs{}
	folio := NewFolio(0, 4, 0)
	folio.block = &Block{Data: []byte{0, 1, 2, 3}}
	refs.Add(folio)
	n, sg := refs.Slice(1, 3)
	s.Equal(n, 2)
	s.Equal(sg, [][]byte{{1, 2}})
}

func (s *FolioRefsTests) TestSliceEmpty() {
	refs := FolioRefs{}
	folio := NewFolio(0, 4, 0)
	folio.block = &Block{Data: []byte{0, 1, 2, 3}}
	refs.Add(folio)
	n, sg := refs.Slice(1, 1)
	s.Equal(n, 0)
	s.Equal(sg, [][]byte{})
}

func (s *FolioRefsTests) TestSliceMultiple() {
	refs := FolioRefs{}
	for i := range 4 {
		folio := NewFolio(int64(i*4), int64((i+1)*4), 0)
		folio.block = &Block{Data: bytes.Repeat([]byte{byte(i)}, 4)}
		refs.Add(folio)
	}
	n, sg := refs.Slice(0, 4)
	s.Equal(n, 4)
	s.Equal(sg, [][]byte{{0, 0, 0, 0}})
	n, sg = refs.Slice(2, 6)
	s.Equal(n, 4)
	s.Equal(sg, [][]byte{{0, 0}, {1, 1}})
	n, sg = refs.Slice(4, 8)
	s.Equal(n, 4)
	s.Equal(sg, [][]byte{{1, 1, 1, 1}})
	n, sg = refs.Slice(2, 11)
	s.Equal(n, 9)
	s.Equal(sg, [][]byte{{0, 0}, {1, 1, 1, 1}, {2, 2, 2}})
	n, sg = refs.Slice(14, 16)
	s.Equal(n, 2)
	s.Equal(sg, [][]byte{{3, 3}})
}

func (s *FolioRefsTests) TestSliceBeforeAfter() {
	refs := FolioRefs{}
	folio := NewFolio(4, 8, 0)
	folio.block = &Block{Data: make([]byte, 4)}
	refs.Add(folio)
	n, sg := refs.Slice(0, 2)
	s.Equal(n, 0)
	s.Equal(sg, [][]byte{})
	n, sg = refs.Slice(5, 7)
	s.Equal(n, 2)
	s.Equal(sg, [][]byte{{0, 0}})
	n, sg = refs.Slice(6, 10)
	s.Equal(n, 2)
	s.Equal([][]byte{{0, 0}}, sg)
	n, sg = refs.Slice(10, 12)
	s.Equal(n, 0)
	s.Equal(sg, [][]byte{})
	n, sg = refs.Slice(2, 10)
	s.Equal(n, 4)
	s.Equal(sg, [][]byte{{0, 0, 0, 0}})
}

func (s *FolioRefsTests) TestSlicePartial() {
	refs := FolioRefs{}
	folio := NewFolio(0, 10, 0)
	folio.block = &Block{Data: make([]byte, 6)} // Data is smaller than folio
	refs.Add(folio)
	n, sg := refs.Slice(4, 8)
	s.Equal(n, 2)
	s.Equal(sg, [][]byte{{0, 0}})
}

func TestFolio(t *testing.T) {
	suite.Run(t, &FolioTests{})
}

func TestFolioRefs(t *testing.T) {
	suite.Run(t, &FolioRefsTests{})
}

func TestAllocateFolios(t *testing.T) {
	pool, err := NewSmartPool(Block1MB, Block64KB)
	if err != nil {
		t.Fatalf("Failed to create smart pool: %v", err)
	}

	tests := []struct {
		name          string
		start         int64
		end           int64
		expectedCount int
	}{
		{
			name:          "Small range - single folio",
			start:         0,
			end:           32 * 1024, // 32KB
			expectedCount: 1,
		},
		{
			name:          "Medium range - multiple folios",
			start:         0,
			end:           3 * 1024 * 1024, // 3MB
			expectedCount: 3,
		},
		{
			name:          "Large range - many folios",
			start:         1000,
			end:           10*1024*1024 + 1000, // 10MB
			expectedCount: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			folios, err := AllocateFolios(tt.start, tt.end, 0, pool)
			if err != nil {
				t.Fatalf("AllocateFolios failed: %v", err)
			}

			if len(folios) == 0 {
				t.Error("Expected at least one folio")
			}

			// Verify folios cover the entire range
			if len(folios) > 0 {
				if folios[0].Start != tt.start {
					t.Errorf("First folio start = %d, want %d", folios[0].Start, tt.start)
				}
				if folios[len(folios)-1].End != tt.end {
					t.Errorf("Last folio end = %d, want %d", folios[len(folios)-1].End, tt.end)
				}
			}

			// Verify folios are contiguous
			for i := 1; i < len(folios); i++ {
				if folios[i].Start != folios[i-1].End {
					t.Errorf("Folio %d not contiguous: prev.End=%d, curr.Start=%d",
						i, folios[i-1].End, folios[i].Start)
				}
			}

			// Verify each folio has a block with data
			totalSize := int64(0)
			for i, folio := range folios {
				if folio.block == nil {
					t.Errorf("Folio %d has nil block", i)
					continue
				}
				if len(folio.block.Data) == 0 {
					t.Errorf("Folio %d has no data", i)
				}
				totalSize += folio.Size()
			}

			// Verify total size matches requested range
			expectedSize := tt.end - tt.start
			if totalSize != expectedSize {
				t.Errorf("Total size = %d, want %d", totalSize, expectedSize)
			}

			t.Logf("Created %d folios for range [%d, %d)", len(folios), tt.start, tt.end)
		})
	}
}
