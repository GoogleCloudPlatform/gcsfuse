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

package storage

import (
	"io"
	"reflect"
	"sync"
	"sync/atomic"
	"unsafe"

	storagev2 "cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"google.golang.org/grpc/mem"
)

const defaultGRPCReadID = int64(1)

// localBufferSliceOffsets mirrors cloud.google.com/go/storage.bufferSliceOffsets
// and bufferSliceOffsetsReadObject. Its exact layout and field offsets are
// verified at runtime via reflection before use.
type localBufferSliceOffsets struct {
	startBuf, endBuf int
	startOff, endOff uint64
	currBuf          int
	currOff          uint64
}

type ifaceHeader struct {
	tab  unsafe.Pointer
	data unsafe.Pointer
}

type vectoredReadState struct {
	slices   [][]byte
	held     []mem.Buffer
	callback func()
}

var vectoredReadStatePool = sync.Pool{
	New: func() any {
		return &vectoredReadState{
			slices: make([][]byte, 0, 72),
			held:   make([]mem.Buffer, 0, 72),
		}
	},
}

func (s *vectoredReadState) release() {
	for i := range s.held {
		s.held[i].Free()
		s.held[i] = nil
	}
	s.held = s.held[:0]
	for i := range s.slices {
		s.slices[i] = nil
	}
	s.slices = s.slices[:0]
	vectoredReadStatePool.Put(s)
}

type grpcZeroCopyLayout struct {
	valid                bool
	isMap                bool
	offGRSeen            uintptr
	offGRLeftovers       uintptr
	offGRCurrMsg         uintptr
	offGRCheckCRC        uintptr
	offGRChunkCRCPresent uintptr
	offDecDatabufs       uintptr
	offDecDataOffsets    uintptr
	offDecDone           uintptr
}

var (
	srOffsetsOnce    sync.Once
	srOffsetsValid   bool
	offSRRemain      uintptr
	offSRBytesRead   uintptr
	offSRReader      uintptr
	offSRUnfinalized uintptr

	layoutMu         sync.Mutex
	tabReadObject    unsafe.Pointer
	layoutReadObject grpcZeroCopyLayout
	tabBidi          unsafe.Pointer
	layoutBidi       grpcZeroCopyLayout
)

func initSROffsets() {
	srOffsetsOnce.Do(func() {
		srType := reflect.TypeOf(storagev2.Reader{})
		fRemain, ok1 := srType.FieldByName("remain")
		fBytesRead, ok2 := srType.FieldByName("bytesRead")
		fReader, ok3 := srType.FieldByName("reader")
		fUnfinalized, ok4 := srType.FieldByName("unfinalized")
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return
		}
		if fRemain.Type.Kind() != reflect.Int64 ||
			fBytesRead.Type.Kind() != reflect.Int64 ||
			fReader.Type.Kind() != reflect.Interface ||
			fUnfinalized.Type.Kind() != reflect.Bool {
			return
		}
		offSRRemain = fRemain.Offset
		offSRBytesRead = fBytesRead.Offset
		offSRReader = fReader.Offset
		offSRUnfinalized = fUnfinalized.Offset
		srOffsetsValid = true
	})
}

func verifyBufferSliceOffsetsType(bsoType reflect.Type) bool {
	localType := reflect.TypeOf(localBufferSliceOffsets{})
	if bsoType.Kind() != reflect.Struct || bsoType.Size() != localType.Size() || bsoType.NumField() != localType.NumField() {
		return false
	}
	for i := 0; i < localType.NumField(); i++ {
		sf := bsoType.Field(i)
		lf := localType.Field(i)
		if sf.Name != lf.Name || sf.Type.Kind() != lf.Type.Kind() || sf.Offset != lf.Offset {
			return false
		}
	}
	return true
}

func resolveGRPCLayout(sr *storagev2.Reader, tab unsafe.Pointer) *grpcZeroCopyLayout {
	layoutMu.Lock()
	defer layoutMu.Unlock()

	if tab == atomic.LoadPointer(&tabReadObject) {
		return &layoutReadObject
	}
	if tab == atomic.LoadPointer(&tabBidi) {
		return &layoutBidi
	}

	srVal := reflect.ValueOf(sr).Elem()
	readerIface := srVal.FieldByName("reader")
	if !readerIface.IsValid() || readerIface.IsNil() || readerIface.Elem().Kind() != reflect.Pointer {
		return nil
	}
	grElem := readerIface.Elem().Elem()
	if grElem.Kind() != reflect.Struct {
		return nil
	}
	grType := grElem.Type()
	grName := grType.Name()
	if grName != "gRPCReadObjectReader" && grName != "gRPCReader" {
		return nil
	}

	fSeen, ok1 := grType.FieldByName("seen")
	fLeftovers, ok2 := grType.FieldByName("leftovers")
	fCurrMsg, ok3 := grType.FieldByName("currMsg")
	fCheckCRC, ok4 := grType.FieldByName("checkCRC")
	fChunkCRC, ok5 := grType.FieldByName("chunkCRCPresent")
	if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 {
		return nil
	}
	if fSeen.Type.Kind() != reflect.Int64 ||
		fLeftovers.Type != reflect.TypeOf([]byte(nil)) ||
		fCurrMsg.Type.Kind() != reflect.Pointer ||
		fCheckCRC.Type.Kind() != reflect.Bool ||
		fChunkCRC.Type.Kind() != reflect.Bool {
		return nil
	}

	decType := fCurrMsg.Type.Elem()
	if decType.Kind() != reflect.Struct {
		return nil
	}
	fDatabufs, ok1 := decType.FieldByName("databufs")
	fDataOffsets, ok2 := decType.FieldByName("dataOffsets")
	fDone, ok3 := decType.FieldByName("done")
	if !ok1 || !ok2 || !ok3 {
		return nil
	}
	if fDatabufs.Type != reflect.TypeOf(mem.BufferSlice(nil)) || fDone.Type.Kind() != reflect.Bool {
		return nil
	}

	var isMap bool
	if grName == "gRPCReader" && decType.Name() == "readResponseDecoder" {
		if fDataOffsets.Type.Kind() != reflect.Map || fDataOffsets.Type.Key().Kind() != reflect.Int64 {
			return nil
		}
		if !verifyBufferSliceOffsetsType(fDataOffsets.Type.Elem()) {
			return nil
		}
		isMap = true
	} else if grName == "gRPCReadObjectReader" && decType.Name() == "readObjectResponseDecoder" {
		if !verifyBufferSliceOffsetsType(fDataOffsets.Type) {
			return nil
		}
		isMap = false
	} else {
		return nil
	}

	layout := grpcZeroCopyLayout{
		valid:                true,
		isMap:                isMap,
		offGRSeen:            fSeen.Offset,
		offGRLeftovers:       fLeftovers.Offset,
		offGRCurrMsg:         fCurrMsg.Offset,
		offGRCheckCRC:        fCheckCRC.Offset,
		offGRChunkCRCPresent: fChunkCRC.Offset,
		offDecDatabufs:       fDatabufs.Offset,
		offDecDataOffsets:    fDataOffsets.Offset,
		offDecDone:           fDone.Offset,
	}

	if isMap {
		layoutBidi = layout
		atomic.StorePointer(&tabBidi, tab)
		return &layoutBidi
	}
	layoutReadObject = layout
	atomic.StorePointer(&tabReadObject, tab)
	return &layoutReadObject
}

// gcsFullReadCloser wraps storage.Reader (initially conforms to io.ReadCloser)
// and forces Read to call io.ReadFull instead.
type gcsFullReadCloser struct {
	wrapped gcs.StorageReader
}

func newGCSFullReadCloser(StorageReader gcs.StorageReader) gcsFullReadCloser {
	return gcsFullReadCloser{
		wrapped: StorageReader,
	}
}

// Re-define the behavior of Read function to return:
// 1. the number of bytes copied and an EOF if response size < buffer size
// 2. n == len(buf) if and only if err == nil.
func (frc gcsFullReadCloser) Read(buf []byte) (n int, err error) {
	n, err = io.ReadFull(frc.wrapped, buf)
	if err == io.ErrUnexpectedEOF {
		err = io.EOF
	}
	return n, err
}

// ReadVectored reads up to len(buf) bytes from a gRPC-backed *storagev2.Reader
// by referencing the underlying gRPC mem.BufferSlice directly without copying
// payload bytes into buf.
func (frc gcsFullReadCloser) ReadVectored(buf []byte) (data [][]byte, n int, callback func(), err error) {
	want := len(buf)
	if want == 0 {
		return nil, 0, nil, nil
	}
	sr, ok := frc.wrapped.(*storagev2.Reader)
	if !ok {
		n, err = frc.Read(buf)
		return nil, n, nil, err
	}
	initSROffsets()
	if !srOffsetsValid {
		n, err = frc.Read(buf)
		return nil, n, nil, err
	}

	srPtr := unsafe.Pointer(sr)
	rIface := (*ifaceHeader)(unsafe.Add(srPtr, offSRReader))
	grPtr := rIface.data
	tab := rIface.tab
	if grPtr == nil || tab == nil {
		n, err = frc.Read(buf)
		return nil, n, nil, err
	}

	var layout *grpcZeroCopyLayout
	if tab == atomic.LoadPointer(&tabReadObject) {
		layout = &layoutReadObject
	} else if tab == atomic.LoadPointer(&tabBidi) {
		layout = &layoutBidi
	} else {
		layout = resolveGRPCLayout(sr, tab)
		if layout == nil || !layout.valid {
			n, err = frc.Read(buf)
			return nil, n, nil, err
		}
	}

	// Fall back to standard Read if CRC checking is active or leftovers exist.
	if *(*bool)(unsafe.Add(grPtr, layout.offGRCheckCRC)) ||
		*(*bool)(unsafe.Add(grPtr, layout.offGRChunkCRCPresent)) ||
		len(*(*[]byte)(unsafe.Add(grPtr, layout.offGRLeftovers))) > 0 {
		n, err = frc.Read(buf)
		return nil, n, nil, err
	}

	state := vectoredReadStatePool.Get().(*vectoredReadState)
	if state.callback == nil {
		state.callback = state.release
	}
	totalRead := 0

	for totalRead < want {
		currMsg := *(*unsafe.Pointer)(unsafe.Add(grPtr, layout.offGRCurrMsg))
		if currMsg == nil || *(*bool)(unsafe.Add(currMsg, layout.offDecDone)) {
			// A 0-byte Read on sr triggers gRPCReadObjectReader.recv() or
			// gRPCReader.recv() (including retries and protobuf header decoding)
			// and returns 0, nil with currMsg ready.
			_, err = sr.Read(nil)
			if err != nil {
				if err == io.ErrUnexpectedEOF {
					err = io.EOF
				}
				break
			}
			if *(*bool)(unsafe.Add(grPtr, layout.offGRCheckCRC)) ||
				*(*bool)(unsafe.Add(grPtr, layout.offGRChunkCRCPresent)) {
				remN, remErr := frc.Read(buf[totalRead:])
				if remN > 0 {
					state.slices = append(state.slices, buf[totalRead:totalRead+remN])
					totalRead += remN
				}
				err = remErr
				break
			}
			currMsg = *(*unsafe.Pointer)(unsafe.Add(grPtr, layout.offGRCurrMsg))
			if currMsg == nil || *(*bool)(unsafe.Add(currMsg, layout.offDecDone)) {
				continue
			}
		}

		databufsPtr := (*mem.BufferSlice)(unsafe.Add(currMsg, layout.offDecDatabufs))
		databufs := *databufsPtr
		var offsets localBufferSliceOffsets
		var dataOffsetsMap map[int64]localBufferSliceOffsets
		if layout.isMap {
			dataOffsetsMap = *(*map[int64]localBufferSliceOffsets)(unsafe.Add(currMsg, layout.offDecDataOffsets))
			var ok bool
			offsets, ok = dataOffsetsMap[defaultGRPCReadID]
			if !ok || len(databufs) == 0 {
				*(*bool)(unsafe.Add(currMsg, layout.offDecDone)) = true
				databufs.Free()
				continue
			}
		} else {
			if len(databufs) == 0 {
				*(*bool)(unsafe.Add(currMsg, layout.offDecDone)) = true
				continue
			}
			offsets = *(*localBufferSliceOffsets)(unsafe.Add(currMsg, layout.offDecDataOffsets))
		}

		var msgRead int64
		for totalRead < want {
			if offsets.currBuf >= len(databufs) || (offsets.currBuf == offsets.endBuf && offsets.currOff >= offsets.endOff) {
				*(*bool)(unsafe.Add(currMsg, layout.offDecDone)) = true
				databufs.Free()
				break
			}
			databuf := databufs[offsets.currBuf]
			raw := databuf.ReadOnlyData()
			start := offsets.currOff
			end := uint64(len(raw))
			if offsets.currBuf == offsets.endBuf {
				end = offsets.endOff
			}

			if end > start {
				avail := int(end - start)
				need := want - totalRead
				take := avail
				if take > need {
					take = need
				}
				databuf.Ref()
				state.held = append(state.held, databuf)
				state.slices = append(state.slices, raw[start:start+uint64(take)])
				totalRead += take
				msgRead += int64(take)
				offsets.currOff += uint64(take)
			}

			if offsets.currBuf != offsets.endBuf && offsets.currOff >= uint64(len(raw)) {
				offsets.currOff = 0
				offsets.currBuf++
			}
			if offsets.currBuf >= len(databufs) || (offsets.currBuf == offsets.endBuf && offsets.currOff >= offsets.endOff) {
				*(*bool)(unsafe.Add(currMsg, layout.offDecDone)) = true
				databufs.Free()
				break
			}
		}

		if layout.isMap {
			dataOffsetsMap[defaultGRPCReadID] = offsets
		} else {
			*(*localBufferSliceOffsets)(unsafe.Add(currMsg, layout.offDecDataOffsets)) = offsets
		}
		if msgRead > 0 {
			*(*int64)(unsafe.Add(grPtr, layout.offGRSeen)) += msgRead
			if !*(*bool)(unsafe.Add(srPtr, offSRUnfinalized)) && !sr.Attrs.Decompressed {
				atomic.AddInt64((*int64)(unsafe.Add(srPtr, offSRRemain)), -msgRead)
			}
			atomic.AddInt64((*int64)(unsafe.Add(srPtr, offSRBytesRead)), msgRead)
		}
	}

	if totalRead == 0 {
		state.release()
		return nil, 0, nil, err
	}
	return state.slices, totalRead, state.callback, err
}

func (frc gcsFullReadCloser) ReadHandle() (rh storagev2.ReadHandle) {
	return frc.wrapped.ReadHandle()
}

func (frc gcsFullReadCloser) freeUnreadCurrMsg(sr *storagev2.Reader) {
	initSROffsets()
	if !srOffsetsValid {
		return
	}
	srPtr := unsafe.Pointer(sr)
	rIface := (*ifaceHeader)(unsafe.Add(srPtr, offSRReader))
	grPtr := rIface.data
	tab := rIface.tab
	if grPtr == nil || tab == nil {
		return
	}
	var layout *grpcZeroCopyLayout
	if tab == atomic.LoadPointer(&tabReadObject) {
		layout = &layoutReadObject
	} else if tab == atomic.LoadPointer(&tabBidi) {
		layout = &layoutBidi
	} else {
		layout = resolveGRPCLayout(sr, tab)
	}
	if layout == nil || !layout.valid {
		return
	}
	currMsg := *(*unsafe.Pointer)(unsafe.Add(grPtr, layout.offGRCurrMsg))
	if currMsg == nil || *(*bool)(unsafe.Add(currMsg, layout.offDecDone)) {
		return
	}
	*(*bool)(unsafe.Add(currMsg, layout.offDecDone)) = true
	databufsPtr := (*mem.BufferSlice)(unsafe.Add(currMsg, layout.offDecDatabufs))
	databufsPtr.Free()
	*databufsPtr = nil
}

func (frc gcsFullReadCloser) Close() (err error) {
	if sr, ok := frc.wrapped.(*storagev2.Reader); ok {
		frc.freeUnreadCurrMsg(sr)
	}
	return frc.wrapped.Close()
}
