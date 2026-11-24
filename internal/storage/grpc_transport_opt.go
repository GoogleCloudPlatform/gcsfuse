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
	"errors"
	"io"
	"math/bits"
	"net"
	"sync"
	"sync/atomic"
	"syscall"

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
	if buf == nil || cap(*buf) != p.defaultSize {
		return
	}
	p.pool.Put(buf)
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
	if c < (1 << 11) {
		return
	}
	exp := bits.Len(uint(c)) - 1
	if exp >= 11 && exp <= 22 && c == (1<<exp) {
		p.pools[exp].Put(buf)
	}
}

var defaultDirtyBufferPool = newDirtyTieredBufferPool()

func init() {
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
		setTCPUserTimeout(fd)
	})
	c := &largeReadTCPConn{
		TCPConn: tcpConn,
		raw:     raw,
	}
	c.doRead = func(fd uintptr) bool {
		s := &c.state
		s.buf = s.pool.Get(s.bufSize)
		readSlice := *s.buf
		if !s.isALTSPool {
			readSlice = readSlice[:min(len(readSlice), max(0, s.origBufSize))]
		}
		for {
			s.bytesRead, s.readError = syscall.Read(int(fd), readSlice)
			if s.readError != syscall.EINTR {
				break
			}
		}
		if s.readError != nil {
			s.pool.Put(s.buf)
			s.buf = nil
		}
		return !(s.readError == syscall.EAGAIN || s.readError == syscall.EWOULDBLOCK)
	}
	return c
}

func (c *largeReadTCPConn) Write(b []byte) (int, error) {
	if len(b) > 0 && c.firstWriteDone.CompareAndSwap(false, true) {
		isTLS := len(b) >= 5 && b[0] == 0x16 && b[1] == 0x03 && b[2] >= 0x01 && b[2] <= 0x04
		isH2Preface := len(b) >= 4 && string(b[:4]) == "PRI "
		if isTLS || isH2Preface {
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
	if !c.nonALTS.Load() {
		if len(b) > 64*1024 {
			c.altsInitDone.Store(true)
		} else if len(b) == altsInitialProtectedCap && cap(b) == altsInitialProtectedCap && c.altsInitDone.CompareAndSwap(false, true) {
			n, err := c.TCPConn.Read(b[:8])
			if err != nil {
				if n > 0 && errors.Is(err, io.EOF) {
					return n, nil
				}
				return n, err
			}
			if n < 8 {
				return n, nil
			}
			frameLen := binary.LittleEndian.Uint32(b[:4])
			msgType := binary.LittleEndian.Uint32(b[4:8])
			if (msgType&0xff) == 6 && frameLen >= 20 && frameLen <= 1024*1024-4 && len(b) > 8 {
				totalFrame := 4 + int(frameLen)
				m, rErr := c.TCPConn.Read(b[8:min(totalFrame, len(b))])
				if rErr != nil && !errors.Is(rErr, io.EOF) {
					return n + m, rErr
				}
				return n + m, nil
			}
			return n, nil
		}
	}
	return c.TCPConn.Read(b)
}

func (c *largeReadTCPConn) ReadOnReady(bufSize int, pool mem.BufferPool) (*[]byte, int, error) {
	if bufSize <= 0 {
		return nil, 0, nil
	}
	isALTSPool := !c.nonALTS.Load() && bufSize == altsReadBufferInitialSize && pool != defaultDirtyBufferPool
	origBufSize := bufSize
	if isALTSPool {
		c.altsInitDone.Store(true)
		if bufSize < minSocketReadBufferSize {
			bufSize = minSocketReadBufferSize
		}
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

func enableTCPKeepaliveControl(_, _ string, c syscall.RawConn) error {
	var sockErr error
	if err := c.Control(func(fd uintptr) {
		sockErr = unix.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_KEEPALIVE, 1)
	}); err != nil {
		return err
	}
	return sockErr
}

func dialLargeReadTCP(ctx context.Context, addr string) (net.Conn, error) {
	d := &net.Dialer{
		KeepAlive: -1,
		Control:   enableTCPKeepaliveControl,
	}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	return wrapLargeReadTCPConn(conn), nil
}
