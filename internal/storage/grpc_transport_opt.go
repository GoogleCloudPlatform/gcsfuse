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
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"math/bits"
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
	grpc_experimental "google.golang.org/grpc/experimental"
	"google.golang.org/grpc/mem"
)

const (
	// minSocketReadBufferSize upgrades ALTS 32.5 KiB socket reads to 512 KiB
	// so each non-blocking TCP read drains up to 512 KiB of queued records
	// in a single syscall.
	minSocketReadBufferSize = 512 * 1024

	// altsReadBufferInitialSize is the exact 33,280-byte size passed by
	// google.golang.org/grpc/credentials/alts/internal/conn.ReadOnReady.
	altsReadBufferInitialSize = 32*1024 + 512

	// altsInitialProtectedCap is the initial 36 KiB capacity allocated by
	// alts.NewConnWithMaxFrameSize when readBufPool is empty (33280 rounded
	// up to a 4 KiB page boundary).
	altsInitialProtectedCap = 36864

	// defaultTCPUserTimeoutMs matches grpc-go default tcpUserTimeoutOp (20s).
	defaultTCPUserTimeoutMs = 20000

	// dirtyBufFooterSize reserves 16 trailing bytes at the end of every pooled
	// buffer's capacity [defaultSize : defaultSize+16] for zero-copy reference
	// counting across MultiRangeDownloader and FUSE writev.
	dirtyBufFooterSize = 16
)

// dirtyBufFooter is placed in the trailing 16 bytes [defaultSize : defaultSize+16]
// of every pooled buffer's backing array. It contains no Go pointers so the
// backing []byte allocation remains valid noscan memory for the Go garbage collector.
type dirtyBufFooter struct {
	magic uint64
	exp   uint32
	refs  atomic.Int32
}

var (
	_ [16 - unsafe.Sizeof(dirtyBufFooter{})]byte
	_ [unsafe.Sizeof(dirtyBufFooter{}) - 16]byte

	// dirtyBufMagic identifies buffers allocated by dirtyTieredBufferPool.
	// Initialized with cryptographic randomness at startup so payload bytes
	// cannot forge a valid footer magic.
	dirtyBufMagic uint64

	sliceHeaderPool = sync.Pool{
		New: func() any {
			var s []byte
			return &s
		},
	}
)

func dirtyFooterTag(ftrPtr unsafe.Pointer, exp uint32) uint32 {
	h := uint64(uintptr(ftrPtr)) ^ dirtyBufMagic
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	return (uint32(h>>40) << 8) | (exp & 0xff)
}

// dirtySizedPool pools fixed-capacity byte slices without zeroing them on Get,
// eliminating runtime.memclrNoHeapPointers overhead on the gRPC read path and
// embedding a 16-byte reference-count footer in the unused capacity tail.
type dirtySizedPool struct {
	pool        sync.Pool
	defaultSize int
	exp         uint32
}

func (p *dirtySizedPool) Get(size int) *[]byte {
	buf, ok := p.pool.Get().(*[]byte)
	if !ok {
		raw := make([]byte, p.defaultSize+dirtyBufFooterSize)
		b := raw[:size]
		buf = &b
	} else {
		*buf = (*buf)[:size]
	}
	ftrPtr := unsafe.Add(unsafe.Pointer(unsafe.SliceData(*buf)), p.defaultSize)
	ftr := (*dirtyBufFooter)(ftrPtr)
	ftr.magic = dirtyBufMagic
	ftr.exp = dirtyFooterTag(ftrPtr, p.exp)
	ftr.refs.Store(1)
	return buf
}

func (p *dirtySizedPool) Put(buf *[]byte) {
	if buf == nil || cap(*buf) != p.defaultSize+dirtyBufFooterSize {
		return
	}
	ftr := (*dirtyBufFooter)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(*buf)), p.defaultSize))
	if ftr.refs.Add(-1) == 0 {
		p.pool.Put(buf)
		return
	}
	*buf = nil
	sliceHeaderPool.Put(buf)
}

// TryPinDirtySlice checks if p is a sub-slice of a buffer allocated from
// defaultDirtyBufferPool and, if so, increments its reference count in O(1)
// so the backing buffer stays alive after gRPC frees its mem.BufferSlice.
// Returns an opaque token to pass to ReleaseDirtyBuffer, or nil if p is not
// from defaultDirtyBufferPool.
func TryPinDirtySlice(p []byte) unsafe.Pointer {
	c := cap(p)
	if len(p) == 0 || c-len(p) < dirtyBufFooterSize {
		return nil
	}
	ftrPtr := unsafe.Add(unsafe.Pointer(unsafe.SliceData(p)), c-dirtyBufFooterSize)
	if uintptr(ftrPtr)&7 != 0 {
		return nil
	}
	ftr := (*dirtyBufFooter)(ftrPtr)
	if ftr.magic != dirtyBufMagic {
		return nil
	}
	rawExp := ftr.exp
	exp := rawExp & 0xff
	if exp < 11 || exp > 22 || c > (1<<exp)+dirtyBufFooterSize || rawExp != dirtyFooterTag(ftrPtr, exp) {
		return nil
	}
	for {
		cur := ftr.refs.Load()
		if cur <= 0 {
			return nil
		}
		if ftr.refs.CompareAndSwap(cur, cur+1) {
			return ftrPtr
		}
	}
}

// ReleaseDirtyBuffer decrements the reference count of a buffer previously
// pinned by TryPinDirtySlice, returning it to defaultDirtyBufferPool when
// the reference count reaches zero.
func ReleaseDirtyBuffer(tok unsafe.Pointer) {
	if tok == nil {
		return
	}
	ftr := (*dirtyBufFooter)(tok)
	if ftr.refs.Add(-1) != 0 {
		return
	}
	exp := ftr.exp & 0xff
	if ftr.magic != dirtyBufMagic || exp < 11 || exp > 22 || ftr.exp != dirtyFooterTag(tok, exp) {
		return
	}
	sp := defaultDirtyBufferPool.pools[exp]
	basePtr := (*byte)(unsafe.Add(tok, -sp.defaultSize))
	hdr, _ := sliceHeaderPool.Get().(*[]byte)
	if hdr == nil {
		s := unsafe.Slice(basePtr, sp.defaultSize+dirtyBufFooterSize)[:sp.defaultSize]
		hdr = &s
	} else {
		*hdr = unsafe.Slice(basePtr, sp.defaultSize+dirtyBufFooterSize)[:sp.defaultSize]
	}
	sp.pool.Put(hdr)
}

// dirtyTieredBufferPool implements mem.BufferPool using power-of-two sub-pools
// without zeroing buffers upon retrieval. Starting at exp=11 (2048 bytes >
// grpc/mem.BufferPoolingThreshold=1024) ensures 100% of allocated buffers are
// wrapped in pooled mem.Buffers and returned via Put.
type dirtyTieredBufferPool struct {
	pools [24]*dirtySizedPool
}

func newDirtyTieredBufferPool() *dirtyTieredBufferPool {
	p := &dirtyTieredBufferPool{}
	for exp := 11; exp <= 22; exp++ {
		p.pools[exp] = &dirtySizedPool{
			defaultSize: 1 << exp,
			exp:         uint32(exp),
		}
	}
	return p
}

func (p *dirtyTieredBufferPool) Get(size int) *[]byte {
	if size <= 0 {
		b := make([]byte, 0)
		return &b
	}
	exp := 11
	if size > 2048 {
		exp = bits.Len(uint(size - 1))
	}
	if exp <= 22 {
		return p.pools[exp].Get(size)
	}
	b := make([]byte, size)
	return &b
}

func (p *dirtyTieredBufferPool) Put(buf *[]byte) {
	if buf == nil {
		return
	}
	c := cap(*buf)
	if c < (1<<11)+dirtyBufFooterSize {
		return
	}
	exp := bits.Len(uint(c)) - 1
	if exp >= 11 && exp <= 22 && c == (1<<exp)+dirtyBufFooterSize {
		p.pools[exp].Put(buf)
	}
}

var (
	defaultDirtyBufferPool = newDirtyTieredBufferPool()
	preseededALTSPool      atomic.Bool
)

func init() {
	var b [8]byte
	_, _ = rand.Read(b[:])
	dirtyBufMagic = binary.LittleEndian.Uint64(b[:])
	if dirtyBufMagic == 0 {
		dirtyBufMagic = 0x474353465553455a // "GCSFUSEZ"
	}
	grpc_experimental.SetDefaultBufferPool(defaultDirtyBufferPool)
}

type largeReadState struct {
	bufSize     int
	origBufSize int
	isALTSPool  bool
	pool        mem.BufferPool
	readError   error
	bytesRead   int
	buf         *[]byte
}

// largeReadTCPConn wraps *net.TCPConn and implements readyreader.Reader
// (ReadOnReady(bufSize int, pool mem.BufferPool) (*[]byte, int, error))
// to coalesce small ALTS socket reads into 512 KiB non-blocking reads.
type largeReadTCPConn struct {
	*net.TCPConn
	raw            syscall.RawConn
	state          largeReadState
	doRead         func(fd uintptr) bool
	firstWriteDone atomic.Bool
	nonALTS        atomic.Bool
	altsInitDone   atomic.Bool
}

func wrapLargeReadTCPConn(conn net.Conn) net.Conn {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return conn
	}
	raw, err := tcpConn.SyscallConn()
	if err != nil {
		return conn
	}
	_ = raw.Control(func(fd uintptr) {
		_ = unix.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, unix.TCP_USER_TIMEOUT, defaultTCPUserTimeoutMs)
	})
	c := &largeReadTCPConn{
		TCPConn: tcpConn,
		raw:     raw,
	}
	c.doRead = func(fd uintptr) bool {
		s := &c.state
		var readSlice []byte
		if s.isALTSPool {
			cand := s.pool.Get(altsInitialProtectedCap)
			if len(*cand) == altsInitialProtectedCap && cap(*cand) >= altsInitialProtectedCap {
				if preseededALTSPool.CompareAndSwap(false, true) {
					for range 256 {
						b := make([]byte, s.bufSize)
						s.pool.Put(&b)
					}
				}
				if cap(*cand) >= s.bufSize {
					*cand = (*cand)[:s.bufSize]
				} else {
					*cand = make([]byte, s.bufSize)
				}
				s.buf = cand
				readSlice = *s.buf
			} else {
				readSlice = (*cand)[:min(len(*cand), s.origBufSize)]
				s.buf = cand
			}
		} else {
			s.buf = s.pool.Get(s.bufSize)
			readSlice = (*s.buf)[:min(len(*s.buf), s.origBufSize)]
		}
		s.bytesRead, s.readError = syscall.Read(int(fd), readSlice)
		if s.readError != nil {
			s.pool.Put(s.buf)
			s.buf = nil
		}
		return !(s.readError == syscall.EAGAIN || s.readError == syscall.EWOULDBLOCK || s.readError == syscall.EINTR)
	}
	return c
}

func readFullEOF(r io.Reader, buf []byte) (int, error) {
	n, err := io.ReadFull(r, buf)
	if errors.Is(err, io.ErrUnexpectedEOF) {
		err = io.EOF
	}
	return n, err
}

func (c *largeReadTCPConn) Write(b []byte) (int, error) {
	if len(b) > 0 && c.firstWriteDone.CompareAndSwap(false, true) {
		if len(b) >= 5 && b[0] == 0x16 && b[1] == 0x03 && b[2] >= 0x01 && b[2] <= 0x04 {
			c.nonALTS.Store(true)
		}
	}
	return c.TCPConn.Write(b)
}

func (c *largeReadTCPConn) Read(b []byte) (int, error) {
	// An initial 36 KiB ALTS protected buffer passed by alts.conn.ReadOnReady
	// has len(b) == altsInitialProtectedCap && cap(b) == altsInitialProtectedCap
	// (36,864 bytes), whereas ALTS handshake reads use 64 KiB and upgraded
	// buffers use 512 KiB.
	// Read only the exact bytes of the single initial ALTS record so nextFrame
	// is empty after decryption, causing dropProtectedIfEmtpy() to release the
	// 36 KiB buffer and invoke ReadOnReady to upgrade the connection to a
	// 512 KiB protected buffer.
	if !c.nonALTS.Load() && len(b) == altsInitialProtectedCap && cap(b) == altsInitialProtectedCap && c.altsInitDone.CompareAndSwap(false, true) {
		n, err := readFullEOF(c.TCPConn, b[:8])
		if err != nil {
			if n > 0 {
				return n, nil
			}
			return 0, err
		}
		frameLen := int(binary.LittleEndian.Uint32(b[:4]))
		msgType := binary.LittleEndian.Uint32(b[4:8])
		totalFrame := 4 + frameLen
		if msgType == 6 && frameLen >= 20 && totalFrame <= 1024*1024 {
			m, _ := readFullEOF(c.TCPConn, b[8:min(totalFrame, len(b))])
			return n + m, nil
		}
		return n, nil
	}
	return c.TCPConn.Read(b)
}

func (c *largeReadTCPConn) ReadOnReady(bufSize int, pool mem.BufferPool) (*[]byte, int, error) {
	isALTSPool := bufSize == altsReadBufferInitialSize && pool != defaultDirtyBufferPool
	origBufSize := bufSize
	if isALTSPool && bufSize < minSocketReadBufferSize {
		bufSize = minSocketReadBufferSize
	}
	c.state = largeReadState{
		bufSize:     bufSize,
		origBufSize: origBufSize,
		isALTSPool:  isALTSPool,
		pool:        pool,
	}
	err := c.raw.Read(c.doRead)
	buf := c.state.buf
	n := c.state.bytesRead
	readErr := c.state.readError
	c.state = largeReadState{}

	if err != nil {
		if buf != nil {
			pool.Put(buf)
		}
		return nil, 0, err
	}
	if readErr != nil {
		return nil, 0, readErr
	}
	if n == 0 {
		if buf != nil {
			pool.Put(buf)
		}
		return nil, 0, io.EOF
	}
	return buf, n, nil
}

func dialLargeReadTCP(ctx context.Context, addr string) (net.Conn, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	return wrapLargeReadTCPConn(conn), nil
}
