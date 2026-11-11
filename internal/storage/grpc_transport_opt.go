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
	"encoding/binary"
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

	// altsInitialProtectedCap is the initial 36 KiB capacity allocated by
	// alts.NewConnWithMaxFrameSize when readBufPool is empty (33280 rounded
	// up to a 4 KiB page boundary).
	altsInitialProtectedCap = 36864

	// defaultTCPUserTimeoutMs matches grpc-go default tcpUserTimeoutOp (20s).
	defaultTCPUserTimeoutMs = 20000
)

// dirtySizedPool pools fixed-capacity byte slices without zeroing them on Get,
// eliminating runtime.memclrNoHeapPointers overhead on the gRPC read path.
type dirtySizedPool struct {
	pool        sync.Pool
	defaultSize int
}

func (p *dirtySizedPool) Get(size int) *[]byte {
	buf, ok := p.pool.Get().(*[]byte)
	if !ok {
		b := make([]byte, size, p.defaultSize)
		return &b
	}
	*buf = (*buf)[:size]
	return buf
}

func (p *dirtySizedPool) Put(buf *[]byte) {
	if buf == nil || cap(*buf) < p.defaultSize {
		return
	}
	p.pool.Put(buf)
}

// dirtyTieredBufferPool implements mem.BufferPool using power-of-two sub-pools
// without zeroing buffers upon retrieval.
type dirtyTieredBufferPool struct {
	pools [24]*dirtySizedPool
}

func newDirtyTieredBufferPool() *dirtyTieredBufferPool {
	p := &dirtyTieredBufferPool{}
	for exp := 8; exp <= 22; exp++ {
		p.pools[exp] = &dirtySizedPool{defaultSize: 1 << exp}
	}
	return p
}

func (p *dirtyTieredBufferPool) Get(size int) *[]byte {
	if size <= 0 {
		b := make([]byte, 0)
		return &b
	}
	exp := 8
	if size > 256 {
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
	if c < 256 {
		return
	}
	exp := bits.Len(uint(c)) - 1
	if exp >= 8 && exp <= 22 {
		p.pools[exp].Put(buf)
	}
}

var (
	defaultDirtyBufferPool = newDirtyTieredBufferPool()
	preseededALTSPool      atomic.Bool
)

func init() {
	grpc_experimental.SetDefaultBufferPool(defaultDirtyBufferPool)
}

type largeReadState struct {
	bufSize   int
	pool      mem.BufferPool
	readError error
	bytesRead int
	buf       *[]byte
}

// largeReadTCPConn wraps *net.TCPConn and implements readyreader.Reader
// (ReadOnReady(bufSize int, pool mem.BufferPool) (*[]byte, int, error))
// to coalesce small ALTS socket reads into 512 KiB non-blocking reads.
type largeReadTCPConn struct {
	*net.TCPConn
	raw    syscall.RawConn
	state  largeReadState
	doRead func(fd uintptr) bool
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
		if preseededALTSPool.CompareAndSwap(false, true) {
			for range 64 {
				b := make([]byte, s.bufSize)
				s.pool.Put(&b)
			}
		}
		cand := s.pool.Get(altsInitialProtectedCap)
		if cap(*cand) >= s.bufSize {
			*cand = (*cand)[:s.bufSize]
			s.buf = cand
		} else {
			// Discard any initial 36 KiB ALTS buffer so only >= 512 KiB buffers
			// remain in readBufPool.
			s.buf = s.pool.Get(s.bufSize)
		}
		s.bytesRead, s.readError = syscall.Read(int(fd), *s.buf)
		if s.readError != nil {
			s.pool.Put(s.buf)
			s.buf = nil
		}
		return !(s.readError == syscall.EAGAIN || s.readError == syscall.EWOULDBLOCK || s.readError == syscall.EINTR)
	}
	return c
}

func (c *largeReadTCPConn) Read(b []byte) (int, error) {
	// Any read into the initial 36 KiB ALTS protected buffer has
	// cap(b) <= altsInitialProtectedCap (36,864 bytes), whereas ALTS handshake
	// reads use 64 KiB and upgraded buffers use 512 KiB.
	// Read only the exact remaining bytes of the single in-progress ALTS record
	// so nextFrame is empty after decryption, causing dropProtectedIfEmtpy() to
	// release the 36 KiB buffer and invoke ReadOnReady to upgrade the connection
	// to a 512 KiB protected buffer.
	if len(b) > 0 && len(b) == cap(b) && cap(b) <= altsInitialProtectedCap {
		already := altsInitialProtectedCap - cap(b)
		basePtr := unsafe.Add(unsafe.Pointer(&b[0]), -already)
		fullBuf := unsafe.Slice((*byte)(basePtr), altsInitialProtectedCap)
		readSoFar := 0
		if already < 4 {
			needHdr := 4 - already
			if needHdr <= len(b) {
				n, err := io.ReadFull(c.TCPConn, b[:needHdr])
				readSoFar += n
				if err != nil {
					return readSoFar, err
				}
			}
		}
		frameLen := int(binary.LittleEndian.Uint32(fullBuf[:4]))
		totalFrame := 4 + frameLen
		needTotal := totalFrame - already
		if frameLen > 0 && needTotal > readSoFar && needTotal <= len(b) {
			n, err := io.ReadFull(c.TCPConn, b[readSoFar:needTotal])
			return readSoFar + n, err
		}
		if readSoFar > 0 {
			return readSoFar, nil
		}
	}
	return c.TCPConn.Read(b)
}

func (c *largeReadTCPConn) ReadOnReady(bufSize int, pool mem.BufferPool) (*[]byte, int, error) {
	if bufSize < minSocketReadBufferSize {
		bufSize = minSocketReadBufferSize
	}
	c.state = largeReadState{
		bufSize: bufSize,
		pool:    pool,
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
