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
	"bytes"
	"context"
	"encoding/binary"
	"hash/crc32"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	storagev2 "cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpc_experimental "google.golang.org/grpc/experimental"
	"google.golang.org/grpc/mem"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// twoBytesStorageReader reads at most 2 bytes from the buffer in one go.
type twoBytesStorageReader struct {
	buf *bytes.Buffer
}

func (psr twoBytesStorageReader) ReadHandle() (rh storagev2.ReadHandle) {
	return nil
}

func (psr twoBytesStorageReader) Close() (err error) {
	return nil
}

func (psr twoBytesStorageReader) Read(b []byte) (n int, err error) {
	maxBytes := 2
	bufLen := min(len(b), maxBytes)
	temp := make([]byte, bufLen)
	n, err = psr.buf.Read(temp)
	copy(b, temp)
	return n, err
}

func TestFullReaderCloser(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		bufSize      int
		data         []byte
		expectedData []byte
		expectedErr  error
	}{
		{
			name:         "large_buffer",
			data:         []byte("0123"),
			bufSize:      5,
			expectedData: []byte("0123"),
			expectedErr:  io.EOF,
		},
		{
			name:         "small_buffer",
			data:         []byte("0123"),
			bufSize:      2,
			expectedData: []byte("01"),
			expectedErr:  nil,
		},
		{
			name:         "equal_buffer",
			data:         []byte("0123"),
			bufSize:      4,
			expectedData: []byte("0123"),
			expectedErr:  nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			storageReader := twoBytesStorageReader{
				buf: new(bytes.Buffer),
			}
			storageReader.buf.Write(tc.data)
			fullReadCloser := newGCSFullReadCloser(storageReader)
			buffer := make([]byte, tc.bufSize)

			n, err := fullReadCloser.Read(buffer)

			assert.Equal(t, len(tc.expectedData), n)
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedData[:n], buffer[:n])
		})
	}
}

func TestDirtyTieredBufferPool(t *testing.T) {
	t.Parallel()

	// Exact power-of-two capacities from 2 KiB (1<<11) to 4 MiB (1<<22) without footer waste.
	for exp := 11; exp <= 22; exp++ {
		capSize := 1 << exp
		reqSize := capSize - 123
		if exp == 11 {
			reqSize = 1024
		}
		bufPtr := defaultDirtyBufferPool.Get(reqSize)
		require.NotNil(t, bufPtr)
		assert.Equal(t, reqSize, len(*bufPtr))
		assert.Equal(t, capSize, cap(*bufPtr))

		// Wrap in mem.Buffer to verify gRPC reference counting with defaultDirtyBufferPool.
		mb := mem.NewBuffer(bufPtr, defaultDirtyBufferPool)
		mb.Ref()
		mb.Free()
		mb.Free()
	}

	// Non-positive size returns empty slice.
	emptyPtr := defaultDirtyBufferPool.Get(0)
	require.NotNil(t, emptyPtr)
	assert.Empty(t, *emptyPtr)

	// Oversized (> 4 MiB) allocation is not pooled.
	hugeSize := (4 * 1024 * 1024) + 1024
	hugePtr := defaultDirtyBufferPool.Get(hugeSize)
	require.NotNil(t, hugePtr)
	assert.Equal(t, hugeSize, len(*hugePtr))
	assert.Equal(t, hugeSize, cap(*hugePtr))
	defaultDirtyBufferPool.Put(hugePtr)
}

func flattenVectored(slices [][]byte, fallback []byte, n int) []byte {
	if slices == nil {
		out := make([]byte, n)
		copy(out, fallback[:n])
		return out
	}
	var out []byte
	for _, s := range slices {
		out = append(out, s...)
	}
	return out
}

func startFakeVectoredStorageServer(t *testing.T, payload []byte, chunkSize int, withCRC bool) string {
	t.Helper()
	reqROType, err := protoregistry.GlobalTypes.FindMessageByName("google.storage.v2.ReadObjectRequest")
	require.NoError(t, err)
	respROType, err := protoregistry.GlobalTypes.FindMessageByName("google.storage.v2.ReadObjectResponse")
	require.NoError(t, err)
	reqBidiType, err := protoregistry.GlobalTypes.FindMessageByName("google.storage.v2.BidiReadObjectRequest")
	require.NoError(t, err)
	respBidiType, err := protoregistry.GlobalTypes.FindMessageByName("google.storage.v2.BidiReadObjectResponse")
	require.NoError(t, err)

	crcTable := crc32.MakeTable(crc32.Castagnoli)
	fullCRC := crc32.Checksum(payload, crcTable)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	var srvPtr *fakeReadStallStorageServer
	grpcServer := grpc.NewServer()
	grpcServer.RegisterService(&grpc.ServiceDesc{
		ServiceName: "google.storage.v2.Storage",
		HandlerType: srvPtr,
		Streams: []grpc.StreamDesc{
			{
				StreamName:    "ReadObject",
				ServerStreams: true,
				Handler: func(srv any, stream grpc.ServerStream) error {
					req := reqROType.New()
					if err := stream.RecvMsg(req.Interface()); err != nil {
						return err
					}
					offset := int(req.Get(req.Descriptor().Fields().ByName("read_offset")).Int())
					limit := int(req.Get(req.Descriptor().Fields().ByName("read_limit")).Int())
					end := len(payload)
					if limit > 0 && offset+limit < end {
						end = offset + limit
					}
					for pos := offset; pos < end; pos += chunkSize {
						cEnd := min(pos+chunkSize, end)
						chunk := payload[pos:cEnd]
						resp := respROType.New()

						dataField := resp.Descriptor().Fields().ByName("checksummed_data")
						dataMsg := resp.NewField(dataField).Message()
						dataMsg.Set(dataMsg.Descriptor().Fields().ByName("content"), protoreflect.ValueOfBytes(chunk))
						if withCRC {
							cCRC := crc32.Checksum(chunk, crcTable)
							dataMsg.Set(dataMsg.Descriptor().Fields().ByName("crc32c"), protoreflect.ValueOfUint32(cCRC))
						}
						resp.Set(dataField, protoreflect.ValueOfMessage(dataMsg))

						if pos == offset {
							metaField := resp.Descriptor().Fields().ByName("metadata")
							metaMsg := resp.NewField(metaField).Message()
							metaMsg.Set(metaMsg.Descriptor().Fields().ByName("name"), protoreflect.ValueOfString("test-obj"))
							metaMsg.Set(metaMsg.Descriptor().Fields().ByName("size"), protoreflect.ValueOfInt64(int64(len(payload))))
							resp.Set(metaField, protoreflect.ValueOfMessage(metaMsg))

							if withCRC {
								objCSField := resp.Descriptor().Fields().ByName("object_checksums")
								objCSMsg := resp.NewField(objCSField).Message()
								objCSMsg.Set(objCSMsg.Descriptor().Fields().ByName("crc32c"), protoreflect.ValueOfUint32(fullCRC))
								resp.Set(objCSField, protoreflect.ValueOfMessage(objCSMsg))
							}
						}
						if err := stream.SendMsg(resp.Interface()); err != nil {
							return err
						}
					}
					return nil
				},
			},
			{
				StreamName:    "BidiReadObject",
				ClientStreams: true,
				ServerStreams: true,
				Handler: func(srv any, stream grpc.ServerStream) error {
					req := reqBidiType.New()
					if err := stream.RecvMsg(req.Interface()); err != nil {
						return err
					}
					rangesField := req.Descriptor().Fields().ByName("read_ranges")
					rangesList := req.Get(rangesField).List()
					var offset, limit int
					var readID int64 = 1
					if rangesList.Len() > 0 {
						rr := rangesList.Get(0).Message()
						offset = int(rr.Get(rr.Descriptor().Fields().ByName("read_offset")).Int())
						limit = int(rr.Get(rr.Descriptor().Fields().ByName("read_length")).Int())
						readID = rr.Get(rr.Descriptor().Fields().ByName("read_id")).Int()
					}
					end := len(payload)
					if limit > 0 && offset+limit < end {
						end = offset + limit
					}
					for pos := offset; pos < end; pos += chunkSize {
						cEnd := min(pos+chunkSize, end)
						chunk := payload[pos:cEnd]
						resp := respBidiType.New()

						if pos == offset {
							metaField := resp.Descriptor().Fields().ByName("metadata")
							metaMsg := resp.NewField(metaField).Message()
							metaMsg.Set(metaMsg.Descriptor().Fields().ByName("name"), protoreflect.ValueOfString("test-obj"))
							metaMsg.Set(metaMsg.Descriptor().Fields().ByName("size"), protoreflect.ValueOfInt64(int64(len(payload))))
							if withCRC {
								csField := metaMsg.Descriptor().Fields().ByName("checksums")
								csMsg := metaMsg.NewField(csField).Message()
								csMsg.Set(csMsg.Descriptor().Fields().ByName("crc32c"), protoreflect.ValueOfUint32(fullCRC))
								metaMsg.Set(csField, protoreflect.ValueOfMessage(csMsg))
							}
							resp.Set(metaField, protoreflect.ValueOfMessage(metaMsg))
						}

						odrField := resp.Descriptor().Fields().ByName("object_data_ranges")
						odrList := resp.Mutable(odrField).List()
						odrMsg := odrList.NewElement().Message()

						dataField := odrMsg.Descriptor().Fields().ByName("checksummed_data")
						dataMsg := odrMsg.NewField(dataField).Message()
						dataMsg.Set(dataMsg.Descriptor().Fields().ByName("content"), protoreflect.ValueOfBytes(chunk))
						if withCRC {
							cCRC := crc32.Checksum(chunk, crcTable)
							dataMsg.Set(dataMsg.Descriptor().Fields().ByName("crc32c"), protoreflect.ValueOfUint32(cCRC))
						}
						odrMsg.Set(dataField, protoreflect.ValueOfMessage(dataMsg))

						rrField := odrMsg.Descriptor().Fields().ByName("read_range")
						rrMsg := odrMsg.NewField(rrField).Message()
						rrMsg.Set(rrMsg.Descriptor().Fields().ByName("read_offset"), protoreflect.ValueOfInt64(int64(pos)))
						rrMsg.Set(rrMsg.Descriptor().Fields().ByName("read_length"), protoreflect.ValueOfInt64(int64(len(chunk))))
						rrMsg.Set(rrMsg.Descriptor().Fields().ByName("read_id"), protoreflect.ValueOfInt64(readID))
						odrMsg.Set(rrField, protoreflect.ValueOfMessage(rrMsg))

						if cEnd == end {
							odrMsg.Set(odrMsg.Descriptor().Fields().ByName("range_end"), protoreflect.ValueOfBool(true))
						}
						odrList.Append(protoreflect.ValueOfMessage(odrMsg))

						if err := stream.SendMsg(resp.Interface()); err != nil {
							return err
						}
					}
					return nil
				},
			},
		},
		Metadata: "google/storage/v2/storage.proto",
	}, struct{}{})

	go func() {
		_ = grpcServer.Serve(listener)
	}()
	t.Cleanup(func() {
		grpcServer.Stop()
	})
	return listener.Addr().String()
}

func newTestGRPCStorageClient(t *testing.T, addr string, bidi bool) *storagev2.Client {
	t.Helper()
	opts := []option.ClientOption{
		option.WithEndpoint(addr),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithGRPCDialOption(grpc.WithContextDialer(dialLargeReadTCP)),
		option.WithGRPCDialOption(grpc_experimental.WithBufferPool(defaultDirtyBufferPool)),
		option.WithGRPCDialOption(grpc.WithReadBufferSize(0)),
		storagev2.WithDisabledClientMetrics(),
	}
	if bidi {
		opts = append(opts, storagev2.WithGRPCBidiReads())
	}
	client, err := storagev2.NewGRPCClient(context.Background(), opts...)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = client.Close()
	})
	return client
}

func TestReadVectored_NonStorageV2Fallback(t *testing.T) {
	t.Parallel()
	payload := []byte("0123456789abcdef")
	sr := twoBytesStorageReader{buf: bytes.NewBuffer(payload)}
	frc := newGCSFullReadCloser(sr)

	// Zero-length read returns immediately.
	slices, n, cb, err := frc.ReadVectored(nil)
	assert.Nil(t, slices)
	assert.Zero(t, n)
	assert.Nil(t, cb)
	assert.NoError(t, err)

	buf := make([]byte, 10)
	slices, n, cb, err = frc.ReadVectored(buf)
	require.NoError(t, err)
	assert.Nil(t, slices)
	assert.Nil(t, cb)
	assert.Equal(t, 10, n)
	assert.Equal(t, payload[:10], buf)

	// Remaining 6 bytes with 10-byte buffer should return io.EOF (normalized from io.ErrUnexpectedEOF).
	slices, n, cb, err = frc.ReadVectored(buf)
	assert.Equal(t, io.EOF, err)
	assert.Nil(t, slices)
	assert.Nil(t, cb)
	assert.Equal(t, 6, n)
	assert.Equal(t, payload[10:], buf[:6])
}

func TestReadVectored_GRPCReadObjectAndBidi(t *testing.T) {
	t.Parallel()
	const totalSize = 24 * 1024
	const chunkSize = 8 * 1024
	payload := make([]byte, totalSize)
	for i := range payload {
		payload[i] = byte((i * 31) & 0xff)
	}

	for _, tc := range []struct {
		name string
		bidi bool
	}{
		{name: "ReadObject", bidi: false},
		{name: "BidiReadObject", bidi: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			addr := startFakeVectoredStorageServer(t, payload, chunkSize, false)
			client := newTestGRPCStorageClient(t, addr, tc.bidi)

			// 1. Partial reads within a single 8 KiB message followed by multi-chunk read.
			r1, err := client.Bucket("test-bucket").Object("test-obj").NewRangeReader(context.Background(), 0, -1)
			require.NoError(t, err)
			frc1 := newGCSFullReadCloser(r1)

			var got []byte
			// Read 3 KiB (partial first 8 KiB message).
			buf := make([]byte, 3*1024)
			slices, n, cb, err := frc1.ReadVectored(buf)
			require.NoError(t, err)
			require.NotNil(t, slices)
			require.NotNil(t, cb)
			assert.Equal(t, 3*1024, n)
			assert.Equal(t, len(slices[0]), cap(slices[0]), "returned zero-copy slice capacity must be capped to length")
			got = append(got, flattenVectored(slices, buf, n)...)
			cb()

			// Read 12 KiB spanning the remainder of message 1 (5 KiB), message 2 (7 KiB).
			buf2 := make([]byte, 12*1024)
			slices, n, cb, err = frc1.ReadVectored(buf2)
			require.NoError(t, err)
			require.NotNil(t, slices)
			require.NotNil(t, cb)
			assert.Equal(t, 12*1024, n)
			got = append(got, flattenVectored(slices, buf2, n)...)
			cb()

			// Read remaining 9 KiB with a 16 KiB buffer (hits EOF).
			buf3 := make([]byte, 16*1024)
			slices, n, cb, err = frc1.ReadVectored(buf3)
			assert.Equal(t, io.EOF, err)
			require.NotNil(t, slices)
			require.NotNil(t, cb)
			assert.Equal(t, 9*1024, n)
			got = append(got, flattenVectored(slices, buf3, n)...)
			cb()

			assert.Equal(t, payload, got)
			require.NoError(t, frc1.Close())

			// 2. Mid-message Close() must free unread currMsg.databufs back to the buffer pool.
			trackPool := &trackingTestBufferPool{inner: defaultDirtyBufferPool}
			client2, err := storagev2.NewGRPCClient(
				context.Background(),
				option.WithEndpoint(addr),
				option.WithoutAuthentication(),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
				option.WithGRPCDialOption(grpc.WithContextDialer(dialLargeReadTCP)),
				option.WithGRPCDialOption(grpc_experimental.WithBufferPool(trackPool)),
				option.WithGRPCDialOption(grpc.WithReadBufferSize(0)),
				storagev2.WithDisabledClientMetrics(),
			)
			require.NoError(t, err)
			defer func() { _ = client2.Close() }()

			r2, err := client2.Bucket("test-bucket").Object("test-obj").NewRangeReader(context.Background(), 0, -1)
			require.NoError(t, err)
			frc2 := newGCSFullReadCloser(r2)

			partBuf := make([]byte, 2*1024)
			slices, n, cb, err = frc2.ReadVectored(partBuf)
			require.NoError(t, err)
			require.NotEmpty(t, slices)
			assert.Equal(t, 2*1024, n)

			inUseBeforeRelease := trackPool.inUse.Load()
			assert.GreaterOrEqual(t, inUseBeforeRelease, int64(1))

			// Release the caller's ReadVectored reference; currMsg still holds its reference.
			cb()
			assert.Equal(t, inUseBeforeRelease, trackPool.inUse.Load(), "currMsg.databufs still holds 1 ref before Close")

			// Close mid-message; SDK reader Close() must release unread currMsg.databufs.
			require.NoError(t, frc2.Close())
			assert.Less(t, trackPool.inUse.Load(), inUseBeforeRelease, "Close must free unread currMsg.databufs back to pool")

			// 3. Concurrent ReadVectored calls across multiple readers under -race.
			var wg sync.WaitGroup
			for range 4 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					rc, rErr := client.Bucket("test-bucket").Object("test-obj").NewRangeReader(context.Background(), 0, -1)
					if !assert.NoError(t, rErr) {
						return
					}
					frc := newGCSFullReadCloser(rc)
					b := make([]byte, totalSize)
					sl, nn, release, rErr := frc.ReadVectored(b)
					assert.NoError(t, rErr)
					assert.Equal(t, totalSize, nn)
					assert.Equal(t, payload, flattenVectored(sl, b, nn))
					if release != nil {
						release()
					}
					assert.NoError(t, frc.Close())
				}()
			}
			wg.Wait()
		})
	}
}

func TestReadVectored_CRCFallback(t *testing.T) {
	t.Parallel()
	payload := bytes.Repeat([]byte("crc-verified-payload-"), 512)
	for _, tc := range []struct {
		name string
		bidi bool
	}{
		{name: "ReadObject_CRC", bidi: false},
		{name: "BidiReadObject_CRC", bidi: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			addr := startFakeVectoredStorageServer(t, payload, 4096, true)
			client := newTestGRPCStorageClient(t, addr, tc.bidi)

			r, err := client.Bucket("test-bucket").Object("test-obj").NewRangeReader(context.Background(), 0, -1)
			require.NoError(t, err)
			frc := newGCSFullReadCloser(r)

			buf := make([]byte, len(payload))
			slices, n, cb, err := frc.ReadVectored(buf)
			require.NoError(t, err)
			assert.Nil(t, slices, "CRC-enabled reader should fall back to standard Read into buf")
			assert.Nil(t, cb)
			assert.Equal(t, len(payload), n)
			assert.Equal(t, payload, buf)
			require.NoError(t, frc.Close())
		})
	}
}

type simpleTestBufferPool struct {
	mu      sync.Mutex
	buffers []*[]byte
}

func (p *simpleTestBufferPool) Get(size int) *[]byte {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.buffers) > 0 {
		b := p.buffers[0]
		p.buffers = p.buffers[1:]
		if cap(*b) >= size {
			*b = (*b)[:size]
			return b
		}
		p.buffers = append(p.buffers, b)
	}
	b := make([]byte, size)
	return &b
}

func (p *simpleTestBufferPool) Put(b *[]byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.buffers = append(p.buffers, b)
}

func TestLargeReadTCPConn_ReadAndReadOnReady(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()

	// Build a fake ALTS record frame (frameLen = 100, msgType = 6, body = 96 bytes)
	// followed by trailing data that must NOT be consumed by the initial 36 KiB ALTS Read.
	const altsPayloadLen = 96
	const altsFrameLen = 4 + altsPayloadLen // msgType (4) + payload (96) = 100
	const altsTotalFrame = 4 + altsFrameLen // length field (4) + frameLen (100) = 104
	altsRecord := make([]byte, altsTotalFrame)
	binary.LittleEndian.PutUint32(altsRecord[:4], uint32(altsFrameLen))
	binary.LittleEndian.PutUint32(altsRecord[4:8], 6)
	for i := 8; i < altsTotalFrame; i++ {
		altsRecord[i] = byte(i)
	}

	http2FrameHdr := []byte{0, 0, 16, 4, 0, 0, 0, 0, 0}
	readyPayload := bytes.Repeat([]byte("R"), 64*1024)

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		srvConn, aErr := listener.Accept()
		if aErr != nil {
			return
		}
		defer func() { _ = srvConn.Close() }()
		// 1. Send 9-byte HTTP/2 frame header.
		_, _ = srvConn.Write(http2FrameHdr)
		// 2. Send initial ALTS record + readyPayload coalesced together on the socket.
		_, _ = srvConn.Write(altsRecord)
		_, _ = srvConn.Write(readyPayload)
	}()

	clientConn, err := dialLargeReadTCP(context.Background(), listener.Addr().String())
	require.NoError(t, err)
	defer func() { _ = clientConn.Close() }()
	lrc, ok := clientConn.(*largeReadTCPConn)
	require.True(t, ok)

	// 1. 9-byte read (like http2.Framer.ReadFrameHeader with ReadBufferSize(0))
	// must not trigger checkptr or ALTS frame interception.
	hdrBuf := make([]byte, 9)
	n, err := io.ReadFull(lrc, hdrBuf)
	require.NoError(t, err)
	assert.Equal(t, 9, n)
	assert.Equal(t, http2FrameHdr, hdrBuf)

	// 2. Initial 36,864-byte ALTS protected buffer read must read only the exact
	// single ALTS frame (104 bytes) even though readyPayload is already queued.
	initialALTSBuf := make([]byte, altsInitialProtectedCap)
	n, err = lrc.Read(initialALTSBuf)
	require.NoError(t, err)
	assert.Equal(t, altsTotalFrame, n)
	assert.Equal(t, altsRecord, initialALTSBuf[:n])

	// 3. Simulate ALTS dropProtectedIfEmtpy() returning the 36 KiB buffer to pool
	// and then calling ReadOnReady(33280, pool).
	pool := &simpleTestBufferPool{}
	pool.Put(&initialALTSBuf)

	bufPtr, n, err := lrc.ReadOnReady(32*1024+512, pool)
	require.NoError(t, err)
	require.NotNil(t, bufPtr)
	assert.Greater(t, n, 0)
	assert.Equal(t, minSocketReadBufferSize, cap(*bufPtr), "ReadOnReady must request a 512 KiB buffer from the ALTS pool")
	assert.NotSame(t, &initialALTSBuf, bufPtr, "36 KiB buffer handle must not be mutated in-place")
	pool.Put(bufPtr)

	<-serverDone

	t.Run("PartialUnder8ByteReadReturnsImmediately", func(t *testing.T) {
		ln, lErr := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, lErr)
		defer func() { _ = ln.Close() }()

		partial := []byte{0x01, 0x02, 0x03}
		readDone := make(chan struct{})
		srvDone := make(chan struct{})
		go func() {
			defer close(srvDone)
			srvConn, aErr := ln.Accept()
			if aErr != nil {
				return
			}
			defer func() { _ = srvConn.Close() }()
			_, _ = srvConn.Write(partial)
			<-readDone
		}()

		conn, dErr := dialLargeReadTCP(context.Background(), ln.Addr().String())
		require.NoError(t, dErr)
		defer func() { _ = conn.Close() }()
		connLRC := conn.(*largeReadTCPConn)
		require.NoError(t, connLRC.SetReadDeadline(time.Now().Add(5*time.Second)))

		buf := make([]byte, altsInitialProtectedCap)
		rn, rErr := connLRC.Read(buf)
		close(readDone)
		require.NoError(t, rErr)
		assert.Equal(t, len(partial), rn)
		assert.Equal(t, partial, buf[:rn])
		assert.True(t, connLRC.altsInitDone.Load())
		<-srvDone
	})

	t.Run("PropagatesDeadlineErrorOnBodyRead", func(t *testing.T) {
		ln, lErr := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, lErr)
		defer func() { _ = ln.Close() }()

		hdr := make([]byte, 8)
		binary.LittleEndian.PutUint32(hdr[:4], 100) // frameLen >= 20, totalFrame = 104
		binary.LittleEndian.PutUint32(hdr[4:8], 6)   // msgType == 6

		hdrWritten := make(chan struct{})
		done := make(chan struct{})
		srvDone := make(chan struct{})
		go func() {
			defer close(srvDone)
			srvConn, aErr := ln.Accept()
			if aErr != nil {
				return
			}
			defer func() { _ = srvConn.Close() }()
			_, _ = srvConn.Write(hdr)
			close(hdrWritten)
			<-done
		}()

		conn, dErr := dialLargeReadTCP(context.Background(), ln.Addr().String())
		require.NoError(t, dErr)
		defer func() { _ = conn.Close() }()
		connLRC := conn.(*largeReadTCPConn)

		<-hdrWritten
		require.NoError(t, connLRC.SetReadDeadline(time.Now().Add(50*time.Millisecond)))

		buf := make([]byte, altsInitialProtectedCap)
		rn, rErr := connLRC.Read(buf)
		close(done)
		require.Error(t, rErr)
		assert.ErrorIs(t, rErr, os.ErrDeadlineExceeded)
		assert.Equal(t, 8, rn)
		assert.Equal(t, hdr, buf[:rn])
		<-srvDone
	})

	t.Run("HTTP2PrefaceDisablesALTSHeuristic", func(t *testing.T) {
		ln, lErr := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, lErr)
		defer func() { _ = ln.Close() }()

		preface := []byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")
		trailing := bytes.Repeat([]byte("H"), 200)
		srvDone := make(chan struct{})
		go func() {
			defer close(srvDone)
			srvConn, aErr := ln.Accept()
			if aErr != nil {
				return
			}
			defer func() { _ = srvConn.Close() }()
			gotPreface := make([]byte, len(preface))
			_, _ = io.ReadFull(srvConn, gotPreface)
			_, _ = srvConn.Write(append(append([]byte(nil), altsRecord...), trailing...))
		}()

		conn, dErr := dialLargeReadTCP(context.Background(), ln.Addr().String())
		require.NoError(t, dErr)
		defer func() { _ = conn.Close() }()
		connLRC := conn.(*largeReadTCPConn)

		wn, wErr := connLRC.Write(preface)
		require.NoError(t, wErr)
		assert.Equal(t, len(preface), wn)
		assert.True(t, connLRC.firstWriteDone.Load())
		assert.True(t, connLRC.nonALTS.Load())

		buf := make([]byte, altsInitialProtectedCap)
		rn, rErr := connLRC.Read(buf)
		require.NoError(t, rErr)
		assert.False(t, connLRC.altsInitDone.Load())
		assert.Equal(t, altsTotalFrame+len(trailing), rn)
		<-srvDone
	})
}

type testConstBufferPool struct {
	buf      []byte
	getCount int
	putCount int
}

func (p *testConstBufferPool) Get(int) *[]byte {
	p.getCount++
	return &p.buf
}

func (p *testConstBufferPool) Put(*[]byte) {
	p.putCount++
}

func TestLargeReadTCPConn_TLSAndPartialALTSAndConstPool(t *testing.T) {
	t.Run("TLSClientHelloDisablesALTSHeuristic", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer func() { _ = listener.Close() }()

		const altsPayloadLen = 96
		const altsFrameLen = 4 + altsPayloadLen
		const altsTotalFrame = 4 + altsFrameLen
		altsRecord := make([]byte, altsTotalFrame)
		binary.LittleEndian.PutUint32(altsRecord[:4], uint32(altsFrameLen))
		binary.LittleEndian.PutUint32(altsRecord[4:8], 6)
		trailing := bytes.Repeat([]byte("T"), 200)

		serverDone := make(chan struct{})
		go func() {
			defer close(serverDone)
			srvConn, aErr := listener.Accept()
			if aErr != nil {
				return
			}
			defer func() { _ = srvConn.Close() }()
			hello := make([]byte, 10)
			_, _ = io.ReadFull(srvConn, hello)
			_, _ = srvConn.Write(append(altsRecord, trailing...))
		}()

		clientConn, err := dialLargeReadTCP(context.Background(), listener.Addr().String())
		require.NoError(t, err)
		defer func() { _ = clientConn.Close() }()
		lrc := clientConn.(*largeReadTCPConn)

		// Write TLS 1.0/1.2 ClientHello record header (0x16, 0x03, 0x01, ...).
		tlsHello := []byte{0x16, 0x03, 0x01, 0x00, 0x05, 'h', 'e', 'l', 'l', 'o'}
		wn, err := lrc.Write(tlsHello)
		require.NoError(t, err)
		assert.Equal(t, len(tlsHello), wn)
		assert.True(t, lrc.firstWriteDone.Load())
		assert.True(t, lrc.nonALTS.Load())

		// Reading with a 36 KiB slice must NOT trigger the single-frame ALTS split.
		buf := make([]byte, altsInitialProtectedCap)
		rn, err := lrc.Read(buf)
		require.NoError(t, err)
		assert.False(t, lrc.altsInitDone.Load())
		assert.Equal(t, altsTotalFrame+len(trailing), rn)
		<-serverDone
	})

	t.Run("PartialALTSBodyReadPreservesHeaderBytes", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer func() { _ = listener.Close() }()

		const altsFrameLen = 100 // expects 4 + 100 = 104 total bytes (96 body bytes)
		partialRecord := make([]byte, 8+20)
		binary.LittleEndian.PutUint32(partialRecord[:4], uint32(altsFrameLen))
		binary.LittleEndian.PutUint32(partialRecord[4:8], 6)
		for i := 8; i < len(partialRecord); i++ {
			partialRecord[i] = byte(i)
		}

		go func() {
			srvConn, aErr := listener.Accept()
			if aErr != nil {
				return
			}
			_, _ = srvConn.Write(partialRecord)
			_ = srvConn.Close()
		}()

		clientConn, err := dialLargeReadTCP(context.Background(), listener.Addr().String())
		require.NoError(t, err)
		defer func() { _ = clientConn.Close() }()
		lrc := clientConn.(*largeReadTCPConn)

		buf := make([]byte, altsInitialProtectedCap)
		n, err := lrc.Read(buf)
		require.NoError(t, err, "partial frame read after 8-byte header must return nil error so header bytes are not dropped")
		assert.Equal(t, len(partialRecord), n)
		assert.Equal(t, partialRecord, buf[:n])

		// Subsequent read returns EOF.
		n2, err2 := lrc.Read(buf)
		assert.Zero(t, n2)
		assert.Equal(t, io.EOF, err2)
	})

	t.Run("ReadOnReadyWithConstBufferPoolDoesNotReallocateOrPoisonALTSPool", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer func() { _ = listener.Close() }()

		want := []byte("const-buffer-pool-payload-bytes")
		go func() {
			srvConn, aErr := listener.Accept()
			if aErr != nil {
				return
			}
			defer func() { _ = srvConn.Close() }()
			_, _ = srvConn.Write(want)
		}()

		clientConn, err := dialLargeReadTCP(context.Background(), listener.Addr().String())
		require.NoError(t, err)
		defer func() { _ = clientConn.Close() }()
		lrc := clientConn.(*largeReadTCPConn)

		backing := make([]byte, altsReadBufferInitialSize)
		origPtr := unsafe.SliceData(backing)
		constPool := &testConstBufferPool{buf: backing}

		gotBuf, n, err := lrc.ReadOnReady(altsReadBufferInitialSize, constPool)
		require.NoError(t, err)
		require.NotNil(t, gotBuf)
		assert.Same(t, &constPool.buf, gotBuf)
		assert.Equal(t, origPtr, unsafe.SliceData(*gotBuf), "constBufferPool backing slice must not be reallocated")
		assert.Equal(t, altsReadBufferInitialSize, cap(*gotBuf))
		assert.Equal(t, len(want), n)
		assert.Equal(t, want, (*gotBuf)[:n])
		assert.GreaterOrEqual(t, constPool.getCount, 1)
		assert.Equal(t, 1, constPool.getCount-constPool.putCount)
	})
}

type trackingTestBufferPool struct {
	inner mem.BufferPool
	inUse atomic.Int64
}

func (p *trackingTestBufferPool) Get(size int) *[]byte {
	p.inUse.Add(1)
	return p.inner.Get(size)
}

func (p *trackingTestBufferPool) Put(b *[]byte) {
	p.inUse.Add(-1)
	p.inner.Put(b)
}

func TestReadVectored_HTTPReaderFallback(t *testing.T) {
	t.Parallel()
	const payload = "http-reader-non-grpc-cache-test"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "31")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	ctx := context.Background()
	httpClient, err := storagev2.NewClient(ctx, option.WithEndpoint(srv.URL), option.WithoutAuthentication())
	require.NoError(t, err)
	defer func() { _ = httpClient.Close() }()

	r, err := httpClient.Bucket("test-bucket").Object("test-obj").NewReader(ctx)
	require.NoError(t, err)
	frc := newGCSFullReadCloser(r)

	buf := make([]byte, len(payload))
	slices, n, cb, err := frc.ReadVectored(buf)
	require.NoError(t, err)
	assert.Nil(t, slices)
	assert.Nil(t, cb)
	assert.Equal(t, len(payload), n)
	assert.Equal(t, payload, string(buf))
	require.NoError(t, frc.Close())
}

func TestDirtyTieredBufferPool_RejectsMismatchedCapOnPut(t *testing.T) {
	t.Parallel()
	const bufSize = 32 * 1024
	buf := defaultDirtyBufferPool.Get(bufSize)
	require.NotNil(t, buf)
	require.Equal(t, bufSize, cap(*buf))

	// A slice whose capacity is not an exact power of two in [2 KiB, 4 MiB] must be ignored by Put.
	nonPowerOfTwo := make([]byte, bufSize, bufSize+16)
	defaultDirtyBufferPool.Put(&nonPowerOfTwo)

	defaultDirtyBufferPool.Put(buf)
}

func TestLargeReadTCPConn_Preseeded512KBufferMarksALTSInitDone(t *testing.T) {
	t.Run("LargeReadOver64KMarksALTSInitDone", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer func() { _ = listener.Close() }()

		firstPayload := []byte("preseeded-512k-payload")
		secondPayload := []byte("ok") // < 8 bytes; would block if ALTS header read ran
		firstReadDone := make(chan struct{})
		secondReadDone := make(chan struct{})
		serverDone := make(chan struct{})

		go func() {
			defer close(serverDone)
			srvConn, aErr := listener.Accept()
			if aErr != nil {
				return
			}
			defer func() { _ = srvConn.Close() }()
			_, _ = srvConn.Write(firstPayload)
			<-firstReadDone
			_, _ = srvConn.Write(secondPayload)
			<-secondReadDone
		}()

		clientConn, err := dialLargeReadTCP(context.Background(), listener.Addr().String())
		require.NoError(t, err)
		defer func() { _ = clientConn.Close() }()
		lrc := clientConn.(*largeReadTCPConn)
		require.False(t, lrc.altsInitDone.Load())

		// A > 64 KiB read (e.g. 512 KiB) must mark altsInitDone == true.
		bigBuf := make([]byte, minSocketReadBufferSize)
		n, err := lrc.Read(bigBuf)
		require.NoError(t, err)
		assert.Equal(t, firstPayload, bigBuf[:n])
		assert.True(t, lrc.altsInitDone.Load())
		close(firstReadDone)

		// Subsequent 36,864-byte read must pass straight through TCPConn.Read
		// without blocking for an 8-byte ALTS frame header.
		buf36K := make([]byte, altsInitialProtectedCap)
		n2, err := lrc.Read(buf36K)
		require.NoError(t, err)
		assert.Equal(t, secondPayload, buf36K[:n2])
		close(secondReadDone)
		<-serverDone
	})

	t.Run("ReadOnReadyWithALTSPoolMarksALTSInitDone", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer func() { _ = listener.Close() }()

		firstPayload := []byte("read-on-ready-alts-payload")
		secondPayload := []byte("hi") // < 8 bytes; would block if ALTS header read ran
		firstReadDone := make(chan struct{})
		secondReadDone := make(chan struct{})
		serverDone := make(chan struct{})

		go func() {
			defer close(serverDone)
			srvConn, aErr := listener.Accept()
			if aErr != nil {
				return
			}
			defer func() { _ = srvConn.Close() }()
			_, _ = srvConn.Write(firstPayload)
			<-firstReadDone
			_, _ = srvConn.Write(secondPayload)
			<-secondReadDone
		}()

		clientConn, err := dialLargeReadTCP(context.Background(), listener.Addr().String())
		require.NoError(t, err)
		defer func() { _ = clientConn.Close() }()
		lrc := clientConn.(*largeReadTCPConn)
		require.False(t, lrc.altsInitDone.Load())

		pool := &simpleTestBufferPool{}
		gotBuf, n, err := lrc.ReadOnReady(altsReadBufferInitialSize, pool)
		require.NoError(t, err)
		require.NotNil(t, gotBuf)
		assert.Equal(t, firstPayload, (*gotBuf)[:n])
		pool.Put(gotBuf)
		assert.True(t, lrc.altsInitDone.Load())
		close(firstReadDone)

		// Subsequent 36,864-byte read must pass straight through TCPConn.Read
		// without blocking for an 8-byte ALTS frame header.
		buf36K := make([]byte, altsInitialProtectedCap)
		n2, err := lrc.Read(buf36K)
		require.NoError(t, err)
		assert.Equal(t, secondPayload, buf36K[:n2])
		close(secondReadDone)
		<-serverDone
	})

	t.Run("OversizedFrameLenAndNegativeReadOnReadySize_DoNotPanic", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer func() { _ = listener.Close() }()

		// Craft an 8-byte ALTS header whose frameLen = 0x7ffffffd (which overflows
		// 4+int(frameLen) on 32-bit int), followed by extra trailing bytes.
		var frame [16]byte
		binary.LittleEndian.PutUint32(frame[:4], 0x7ffffffd)
		binary.LittleEndian.PutUint32(frame[4:8], 6)
		copy(frame[8:], "trailing")

		done := make(chan struct{})
		go func() {
			defer close(done)
			srvConn, aErr := listener.Accept()
			if aErr != nil {
				return
			}
			defer func() { _ = srvConn.Close() }()
			_, _ = srvConn.Write(frame[:])
		}()

		clientConn, err := dialLargeReadTCP(context.Background(), listener.Addr().String())
		require.NoError(t, err)
		defer func() { _ = clientConn.Close() }()
		lrc := clientConn.(*largeReadTCPConn)

		// Non-positive bufSize in ReadOnReady must not panic or return io.EOF on an open socket.
		for _, sz := range []int{-1, 0} {
			buf, nReady, rErr := lrc.ReadOnReady(sz, &simpleTestBufferPool{})
			assert.Nil(t, buf)
			assert.Zero(t, nReady)
			assert.NoError(t, rErr)
		}

		// Queued socket bytes must remain unread and intact after non-positive ReadOnReady calls.
		var got [16]byte
		_, rErr := io.ReadFull(lrc, got[:])
		require.NoError(t, rErr)
		assert.Equal(t, frame[:], got[:])

		// Reset altsInitDone to exercise the 8-byte ALTS frame header path with the
		// oversized length on a fresh connection.
		<-done

		listener2, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer func() { _ = listener2.Close() }()
		done2 := make(chan struct{})
		go func() {
			defer close(done2)
			srvConn, aErr := listener2.Accept()
			if aErr != nil {
				return
			}
			defer func() { _ = srvConn.Close() }()
			_, _ = srvConn.Write(frame[:])
		}()

		clientConn2, err := dialLargeReadTCP(context.Background(), listener2.Addr().String())
		require.NoError(t, err)
		defer func() { _ = clientConn2.Close() }()
		buf36K := make([]byte, altsInitialProtectedCap)
		n, err := clientConn2.Read(buf36K)
		require.NoError(t, err)
		assert.Equal(t, 8, n)
		assert.Equal(t, frame[:8], buf36K[:n])
		<-done2
	})
}

