// Copyright 2023 Google Inc. All Rights Reserved.
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

package buffer

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	. "github.com/jacobsa/ogletest"
)

const (
	KiB               = 1024
	smallContent1     = "Taco"
	smallContent1Size = 4
	smallContent2     = "Burrito"
	smallContent2Size = 7
	smallContent3     = "Pizza"
	smallContent3Size = 5
)

func TestMemoryBuffer(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type MemoryBufferTest struct {
	mb *InMemoryWriteBuffer
}

var _ SetUpInterface = &MemoryBufferTest{}
var _ TearDownInterface = &MemoryBufferTest{}

func init() { RegisterTestSuite(&MemoryBufferTest{}) }

func (t *MemoryBufferTest) SetUp(ti *TestInfo) {
	t.mb = &InMemoryWriteBuffer{}
}

func (t *MemoryBufferTest) TearDown() {}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (t *MemoryBufferTest) generateRandomData(size int64) []byte {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	data := make([]byte, size)
	_, err := r.Read(data)
	AssertEq(nil, err)
	return data
}

func (t *MemoryBufferTest) validateInMemoryBuffer(buffer inMemoryBuffer,
	capacity, length, startOffset, endOffset int64) {
	AssertEq(capacity, cap(buffer.buffer))
	AssertEq(length, len(buffer.buffer))
	AssertEq(startOffset, buffer.offset.start)
	AssertEq(endOffset, buffer.offset.end)
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////

func (t *MemoryBufferTest) TestCreateEmptyInMemoryBufferWithValidBufferSize() int64 {
	var err error
	var sizeInMB int64 = 1
	t.mb, err = CreateInMemoryWriteBuffer(uint(sizeInMB))

	AssertEq(nil, err)
	AssertEq(nil, t.mb.current.buffer)
	AssertEq(nil, t.mb.flushed.buffer)
	AssertEq(sizeInMB*MiB, t.mb.bufferSize)
	AssertEq(0, t.mb.fileSize)
	return sizeInMB
}

func (t *MemoryBufferTest) TestCreateEmptyInMemoryBufferWith0BufferSize() {
	var err error
	var sizeInMB int64 = 0
	t.mb, err = CreateInMemoryWriteBuffer(uint(sizeInMB))

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), ZeroSizeBufferError))
	AssertEq(nil, t.mb)
}

func (t *MemoryBufferTest) TestWritingToUninitializedBuffer() {
	// Write to buffer
	err := t.mb.WriteAt([]byte(smallContent1), 0)

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), ZeroSizeBufferError))
}

func (t *MemoryBufferTest) TestEnsureCurrentBuffer() {
	sizeInMB := t.TestCreateEmptyInMemoryBufferWithValidBufferSize()

	err := t.mb.ensureCurrentBuffer()

	AssertEq(nil, err)
	AssertEq(sizeInMB*MiB, t.mb.bufferSize)
	t.validateInMemoryBuffer(t.mb.current, sizeInMB*MiB, 0, 0, sizeInMB*MiB)
}

func (t *MemoryBufferTest) TestEnsureFlushedBuffer() {
	sizeInMB := t.TestCreateEmptyInMemoryBufferWithValidBufferSize()

	err := t.mb.ensureFlushedBuffer()

	AssertEq(nil, err)
	AssertEq(sizeInMB*MiB, t.mb.bufferSize)
	t.validateInMemoryBuffer(t.mb.flushed, sizeInMB*MiB, 0, 0, 0)
}

func (t *MemoryBufferTest) TestEmptyWriteToInMemoryBuffer() {
	t.TestCreateEmptyInMemoryBufferWithValidBufferSize()

	// Write to buffer
	err := t.mb.WriteAt([]byte(""), 3)

	AssertEq(nil, err)
	AssertEq(0, t.mb.fileSize)
	t.validateInMemoryBuffer(t.mb.current, 0, 0, 0, 0)
	t.validateInMemoryBuffer(t.mb.flushed, 0, 0, 0, 0)
}

func (t *MemoryBufferTest) TestSingleWriteToInMemoryBuffer() {
	sizeInMB := t.TestCreateEmptyInMemoryBufferWithValidBufferSize()

	// Write to buffer
	err := t.mb.WriteAt([]byte(smallContent1), 0)

	AssertEq(nil, err)
	AssertEq(smallContent1Size, t.mb.fileSize)
	AssertTrue(bytes.Equal([]byte(smallContent1), t.mb.current.buffer[0:t.mb.fileSize]))
	t.validateInMemoryBuffer(t.mb.current, sizeInMB*MiB, 0, 0, sizeInMB*MiB)
	t.validateInMemoryBuffer(t.mb.flushed, 0, 0, 0, 0)
}

func (t *MemoryBufferTest) TestMultipleSequentialWritesToInMemoryBuffer() {
	sizeInMB := t.TestCreateEmptyInMemoryBufferWithValidBufferSize()

	// Write to buffer
	err := t.mb.WriteAt([]byte(smallContent1), 0)
	AssertEq(nil, err)
	AssertEq(t.mb.fileSize, smallContent1Size)
	err = t.mb.WriteAt([]byte(smallContent2), smallContent1Size)
	AssertEq(nil, err)
	AssertEq(t.mb.fileSize, smallContent1Size+smallContent2Size)
	err = t.mb.WriteAt([]byte(smallContent3), smallContent1Size+smallContent2Size)
	AssertEq(nil, err)
	AssertEq(t.mb.fileSize, smallContent1Size+smallContent2Size+smallContent3Size)

	AssertTrue(bytes.Equal([]byte(smallContent1+smallContent2+smallContent3), t.mb.current.buffer[0:t.mb.fileSize]))
	t.validateInMemoryBuffer(t.mb.current, sizeInMB*MiB, 0, 0, sizeInMB*MiB)
	t.validateInMemoryBuffer(t.mb.flushed, 0, 0, 0, 0)
}

func (t *MemoryBufferTest) Test2MBWriteTo2MBInMemoryBuffer() {
	sizeInMB := t.TestCreateEmptyInMemoryBufferWithValidBufferSize()
	data1 := t.generateRandomData(MiB)
	data2 := t.generateRandomData(MiB)

	// Write to buffer
	err := t.mb.WriteAt(data1, 0)
	AssertEq(nil, err)
	AssertEq(t.mb.fileSize, MiB)
	err = t.mb.WriteAt(data2, MiB)
	AssertEq(nil, err)
	AssertEq(t.mb.fileSize, 2*MiB)

	AssertTrue(bytes.Equal(data1, t.mb.flushed.buffer[0:MiB]))
	AssertTrue(bytes.Equal(data2, t.mb.current.buffer[0:MiB]))
	t.validateInMemoryBuffer(t.mb.current, sizeInMB*MiB, 0, MiB, 2*MiB)
	t.validateInMemoryBuffer(t.mb.flushed, sizeInMB*MiB, 0, 0, MiB)
}

func (t *MemoryBufferTest) Test3MBWriteTo2MBInMemoryBuffer() {
	sizeInMB := t.TestCreateEmptyInMemoryBufferWithValidBufferSize()
	data1 := t.generateRandomData(MiB)
	data2 := t.generateRandomData(MiB)
	data3 := t.generateRandomData(MiB)

	// Write to buffer
	err := t.mb.WriteAt(data1, 0)
	AssertEq(nil, err)
	AssertEq(t.mb.fileSize, MiB)
	err = t.mb.WriteAt(data2, MiB)
	AssertEq(nil, err)
	AssertEq(t.mb.fileSize, 2*MiB)
	err = t.mb.WriteAt(data3, 2*MiB)
	AssertEq(nil, err)
	AssertEq(t.mb.fileSize, 3*MiB)

	AssertEq(t.mb.fileSize, 3*MiB)
	AssertTrue(bytes.Equal(data3, t.mb.current.buffer[0:MiB]))
	AssertTrue(bytes.Equal(data2, t.mb.flushed.buffer[0:MiB]))
	t.validateInMemoryBuffer(t.mb.current, sizeInMB*MiB, 0, 2*MiB, 3*MiB)
	t.validateInMemoryBuffer(t.mb.flushed, sizeInMB*MiB, 0, MiB, 2*MiB)
}

func (t *MemoryBufferTest) Test4MBWriteTo2MBInMemoryBuffer() {
	sizeInMB := t.TestCreateEmptyInMemoryBufferWithValidBufferSize()
	data1 := t.generateRandomData(MiB)
	data2 := t.generateRandomData(MiB)
	data3 := t.generateRandomData(MiB)
	data4 := t.generateRandomData(MiB)

	// Write to buffer
	err := t.mb.WriteAt(data1, 0)
	AssertEq(nil, err)
	err = t.mb.WriteAt(data2, MiB)
	AssertEq(nil, err)
	err = t.mb.WriteAt(data3, 2*MiB)
	AssertEq(nil, err)
	err = t.mb.WriteAt(data4, 3*MiB)
	AssertEq(nil, err)

	AssertEq(t.mb.fileSize, 4*MiB)
	AssertTrue(bytes.Equal(data4, t.mb.current.buffer[0:MiB]))
	AssertTrue(bytes.Equal(data3, t.mb.flushed.buffer[0:MiB]))
	t.validateInMemoryBuffer(t.mb.current, sizeInMB*MiB, 0, 3*MiB, 4*MiB)
	t.validateInMemoryBuffer(t.mb.flushed, sizeInMB*MiB, 0, 2*MiB, 3*MiB)
}

func (t *MemoryBufferTest) TestMultipleRandomWritesToInMemoryBuffer() {
	sizeInMB := t.TestCreateEmptyInMemoryBufferWithValidBufferSize()

	// Write to buffer
	err := t.mb.WriteAt([]byte(smallContent1), 0)
	AssertEq(nil, err)
	err = t.mb.WriteAt([]byte(smallContent2), 2)
	AssertEq(nil, err)
	err = t.mb.WriteAt([]byte(smallContent3), 7)
	AssertEq(nil, err)

	AssertEq(t.mb.fileSize, 12)
	AssertTrue(bytes.Equal([]byte("TaBurriPizza"), t.mb.current.buffer[0:t.mb.fileSize]))
	t.validateInMemoryBuffer(t.mb.current, sizeInMB*MiB, 0, 0, MiB)
	t.validateInMemoryBuffer(t.mb.flushed, 0, 0, 0, 0)
}

func (t *MemoryBufferTest) TestWriteJustBeforeChunkSizeOffset() {
	sizeInMB := t.TestCreateEmptyInMemoryBufferWithValidBufferSize()

	// Write to buffer
	err := t.mb.WriteAt([]byte(smallContent1), MiB-1)
	AssertEq(nil, err)

	AssertEq(t.mb.fileSize, MiB+3)
	AssertTrue(bytes.Equal([]byte("T"), t.mb.flushed.buffer[MiB-1:MiB]))
	AssertTrue(bytes.Equal([]byte("aco"), t.mb.current.buffer[0:3]))
	t.validateInMemoryBuffer(t.mb.current, sizeInMB*MiB, 0, MiB, 2*MiB)
	t.validateInMemoryBuffer(t.mb.flushed, sizeInMB*MiB, 0, 0, MiB)
}

func (t *MemoryBufferTest) TestWriteAtChunkSizeOffset() {
	sizeInMB := t.TestCreateEmptyInMemoryBufferWithValidBufferSize()

	// Write to buffer
	err := t.mb.WriteAt([]byte(smallContent1), MiB)
	AssertEq(nil, err)

	AssertEq(t.mb.fileSize, MiB+smallContent1Size)
	AssertTrue(bytes.Equal([]byte(smallContent1), t.mb.current.buffer[0:smallContent1Size]))
	t.validateInMemoryBuffer(t.mb.current, sizeInMB*MiB, 0, MiB, 2*MiB)
	t.validateInMemoryBuffer(t.mb.flushed, sizeInMB*MiB, 0, 0, MiB)
}

func (t *MemoryBufferTest) TestWriteJustAfterChunkSizeOffset() {
	t.TestCreateEmptyInMemoryBufferWithValidBufferSize()

	// Write to buffer
	err := t.mb.WriteAt([]byte(smallContent1), MiB+1)

	AssertNe(nil, err)
	AssertEq(NonSequentialWriteError, err.Error())
	AssertEq(t.mb.fileSize, 0)
	t.validateInMemoryBuffer(t.mb.current, MiB, 0, 0, MiB)
	t.validateInMemoryBuffer(t.mb.flushed, 0, 0, 0, 0)
}

func (t *MemoryBufferTest) TestWriteRollOverFromCurrentToFlushedBuffer() {
	sizeInMB := t.TestCreateEmptyInMemoryBufferWithValidBufferSize()
	data1 := t.generateRandomData(MiB)
	data2 := t.generateRandomData(2 * KiB)

	// Write to buffer
	err := t.mb.WriteAt(data1, 0)
	AssertEq(nil, err)
	err = t.mb.WriteAt(data2, MiB)
	AssertEq(nil, err)
	err = t.mb.WriteAt(data2, 2*MiB-KiB)
	AssertEq(nil, err)

	AssertEq(t.mb.fileSize, 2*MiB+KiB)
	AssertTrue(bytes.Equal(data2[0:KiB], t.mb.flushed.buffer[MiB-KiB:MiB]))
	AssertTrue(bytes.Equal(data2[KiB:], t.mb.current.buffer[0:KiB]))
	t.validateInMemoryBuffer(t.mb.current, sizeInMB*MiB, 0, 2*MiB, 3*MiB)
	t.validateInMemoryBuffer(t.mb.flushed, sizeInMB*MiB, 0, MiB, 2*MiB)
}

func (t *MemoryBufferTest) TestRandomWriteOnAnAlreadyUploadedBufferBlockShouldFail() {
	sizeInMB := t.TestCreateEmptyInMemoryBufferWithValidBufferSize()
	data1 := t.generateRandomData(MiB)
	data2 := t.generateRandomData(2 * KiB)

	// Write to buffer
	err := t.mb.WriteAt(data1, 0)
	AssertEq(nil, err)
	err = t.mb.WriteAt(data2, MiB)
	AssertEq(nil, err)
	// Write to an already uploaded offset
	err = t.mb.WriteAt([]byte(smallContent3), 0)

	AssertNe(nil, err)
	AssertEq(NonSequentialWriteError, err.Error())
	AssertEq(t.mb.fileSize, MiB+2*KiB)
	t.validateInMemoryBuffer(t.mb.current, sizeInMB*MiB, 0, MiB, 2*MiB)
	t.validateInMemoryBuffer(t.mb.flushed, sizeInMB*MiB, 0, 0, MiB)
}

func (t *MemoryBufferTest) TestRandomWriteBeyondCurrentBufferAfterSequentialWritesShouldFail() {
	sizeInMB := t.TestCreateEmptyInMemoryBufferWithValidBufferSize()
	data1 := t.generateRandomData(MiB)
	data2 := t.generateRandomData(2 * KiB)

	// Sequential writes
	err := t.mb.WriteAt(data1, 0)
	AssertEq(nil, err)
	err = t.mb.WriteAt(data2, MiB)
	AssertEq(nil, err)
	// Random write beyond current buffer block
	err = t.mb.WriteAt(data2, 2*MiB+1)

	AssertNe(nil, err)
	AssertEq(NonSequentialWriteError, err.Error())
	AssertEq(t.mb.fileSize, MiB+2*KiB)
	t.validateInMemoryBuffer(t.mb.current, sizeInMB*MiB, 0, MiB, 2*MiB)
	t.validateInMemoryBuffer(t.mb.flushed, sizeInMB*MiB, 0, 0, MiB)
}

func (t *MemoryBufferTest) TestSmallContentWriteWith2ChunksSkippedShouldFail() {
	sizeInMB := t.TestCreateEmptyInMemoryBufferWithValidBufferSize()
	data1 := t.generateRandomData(MiB)
	data2 := t.generateRandomData(2 * KiB)

	// Write to buffer sequentially
	err := t.mb.WriteAt(data1, 0)
	AssertEq(nil, err)
	err = t.mb.WriteAt(data2, MiB)
	AssertEq(nil, err)
	// Write to an offset with 2 chunks skipped
	err = t.mb.WriteAt([]byte(smallContent1), 5*MiB)

	AssertNe(nil, err)
	AssertEq(NonSequentialWriteError, err.Error())
	AssertEq(t.mb.fileSize, MiB+2*KiB)
	t.validateInMemoryBuffer(t.mb.current, sizeInMB*MiB, 0, MiB, 2*MiB)
	t.validateInMemoryBuffer(t.mb.flushed, sizeInMB*MiB, 0, 0, MiB)
}

func (t *MemoryBufferTest) TestBigContentWriteWith2ChunksSkippedShouldFail() {
	sizeInMB := t.TestCreateEmptyInMemoryBufferWithValidBufferSize()
	data1 := t.generateRandomData(MiB)
	data2 := t.generateRandomData(2 * KiB)
	data3 := t.generateRandomData(2 * MiB)

	// Write to buffer
	err := t.mb.WriteAt(data1, 0)
	AssertEq(nil, err)
	err = t.mb.WriteAt(data2, MiB)
	AssertEq(nil, err)
	// Write to an offset with
	err = t.mb.WriteAt([]byte(data3), 5*MiB)

	AssertNe(nil, err)
	AssertEq(NonSequentialWriteError, err.Error())
	AssertEq(t.mb.fileSize, MiB+2*KiB)
	t.validateInMemoryBuffer(t.mb.current, sizeInMB*MiB, 0, MiB, 2*MiB)
	t.validateInMemoryBuffer(t.mb.flushed, sizeInMB*MiB, 0, 0, MiB)
}

func (t *MemoryBufferTest) TestCopyDataToBufferWithinRange() {
	t.TestCreateEmptyInMemoryBufferWithValidBufferSize()
	t.TestEnsureCurrentBuffer()
	data := t.generateRandomData(MiB)

	err := t.mb.copyDataToBuffer(0, MiB, data)

	AssertEq(nil, err)
	AssertEq(MiB, t.mb.fileSize)
	AssertTrue(bytes.Equal(data, t.mb.current.buffer[:MiB]))
}

func (t *MemoryBufferTest) TestCopyDataToBufferBeyondRange() {
	t.TestCreateEmptyInMemoryBufferWithValidBufferSize()
	t.TestEnsureCurrentBuffer()
	data := t.generateRandomData(MiB)

	err := t.mb.copyDataToBuffer(50, MiB, data)

	AssertNe(nil, err)
	fmt.Println(err.Error())
	AssertTrue(strings.Contains(err.Error(), NotEnoughSpaceInCurrentBuffer))
}
