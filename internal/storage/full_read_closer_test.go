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
	"sync"
	"testing"
	"unsafe"

	storagev2 "cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpc_experimental "google.golang.org/grpc/experimental"
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

func TestDirtyBufferPoolPinAndRelease(t *testing.T) {
	t.Parallel()

	bufPtr := defaultDirtyBufferPool.Get(32 * 1024)
	sub1 := (*bufPtr)[100:4096]
	sub2 := (*bufPtr)[4096:16384]

	tok1 := TryPinDirtySlice(sub1)
	tok2 := TryPinDirtySlice(sub2)
	assert.NotNil(t, tok1)
	assert.Equal(t, tok1, tok2)

	// Simulate gRPC mem.BufferSlice.Free() returning bufPtr to pool while pinned.
	defaultDirtyBufferPool.Put(bufPtr)

	// Release both pins.
	ReleaseDirtyBuffer(tok1)
	ReleaseDirtyBuffer(tok2)

	// Non-pooled slice must return nil.
	plain := make([]byte, 4096, 8192)
	assert.Nil(t, TryPinDirtySlice(plain))
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
			tok := TryPinDirtySlice(slices[0])
			assert.NotNil(t, tok)
			ReleaseDirtyBuffer(tok)
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

			// 2. Mid-message Close() must free currMsg.databufs back to defaultDirtyBufferPool.
			r2, err := client.Bucket("test-bucket").Object("test-obj").NewRangeReader(context.Background(), 0, -1)
			require.NoError(t, err)
			frc2 := newGCSFullReadCloser(r2)

			partBuf := make([]byte, 2*1024)
			slices, n, cb, err = frc2.ReadVectored(partBuf)
			require.NoError(t, err)
			require.NotEmpty(t, slices)
			assert.Equal(t, 2*1024, n)

			// Locate the dirtyBufFooter of the underlying pooled buffer before releasing.
			s0 := slices[0]
			ftrPtr := unsafe.Add(unsafe.Pointer(unsafe.SliceData(s0)), cap(s0)-dirtyBufFooterSize)
			ftr := (*dirtyBufFooter)(ftrPtr)
			assert.Equal(t, dirtyBufMagic, ftr.magic)
			assert.Equal(t, int32(1), ftr.refs.Load())

			// Release the caller's vectoredReadState reference; currMsg still holds its reference.
			cb()
			assert.Equal(t, int32(1), ftr.refs.Load(), "currMsg.databufs still holds 1 ref before Close")

			pinTok := TryPinDirtySlice(s0)
			require.NotNil(t, pinTok)

			// Close mid-message; freeUnreadCurrMsg must release currMsg.databufs.
			require.NoError(t, frc2.Close())
			assert.Equal(t, int32(1), ftr.refs.Load(), "Close must decrement unread currMsg.databufs refcount")
			ReleaseDirtyBuffer(pinTok)

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
	t.Parallel()

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
	assert.Equal(t, minSocketReadBufferSize, cap(*bufPtr), "36 KiB buffer must be upgraded in-place to 512 KiB")
	assert.Same(t, &initialALTSBuf, bufPtr, "36 KiB buffer handle should be upgraded in-place")
	pool.Put(bufPtr)

	<-serverDone
}

