// Copyright 2026 Google LLC
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

package fs_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v4/metrics"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"google.golang.org/grpc/codes"
)

// storageClientMeterScope is the OpenTelemetry instrumentation scope registered
// by cloud.google.com/go/storage for Cloud Storage client Standard and Debug
// metrics (distinct from GCSFuse's fs/* and gcs/* bucket wrapper metrics in
// gcs_metrics_test.go and grpc-go's grpc.client.* transport metrics in
// grpc_metrics_test.go). Dot-separated OpenTelemetry instrument names (e.g.
// "gcp.storage.client.operations") map 1:1 to slash-separated Cloud Monitoring
// descriptor paths ("gcp/storage/client/operations").
const storageClientMeterScope = "cloud.google.com/go/storage"

var storageClientStandardMetricNames = []string{
	"gcp.storage.client.response.body.size",
	"gcp.storage.client.request.body.size",
	"gcp.storage.client.operation.ttfb",
	"rpc.client.call.duration",
	"gcp.storage.client.attempts",
	"gcp.client.request.duration",
	"gcp.storage.client.operations",
	"gcp.storage.client.errors",
}

var storageClientDebugMetricNames = []string{
	"gcp.storage.client.network.bytes.received",
	"gcp.storage.client.network.bytes.sent",
	"gcp.storage.client.request.active",
	"gcp.storage.client.network.tcp.connect.duration",
	"gcp.storage.client.network.tls.handshake.duration",
	"gcp.storage.client.stall.duration",
	"gcp.storage.client.gfe.header_missing",
	"gcp.storage.client.auth.credential_refresh.duration",
}

var (
	testTLSOnce         sync.Once
	testSAPrivateKeyPEM string
	testTLSServerCert   tls.Certificate
	testTLSInitErr      error
)

// TestStorageClientStandardMetrics_LookUpInodeAndErrors validates Standard
// operation, request duration, attempt, RPC call duration, unary TTFB, and
// error metrics across LookUpInode ("test.txt" -> OK, "test.txt/" -> NOT_FOUND,
// "nonexistent.txt" -> NOT_FOUND) and OpenDir/ReadDir (ListObjects -> OK), and
// verifies that payload body-size and Debug metrics remain unemitted.
func TestStorageClientStandardMetrics_LookUpInodeAndErrors(t *testing.T) {
	ctx := t.Context()
	f := setupStorageClientMetricsFixture(ctx, t, grpcMetricsTestOptions{
		enableOTelMetrics: true,
	})

	require.NoError(t, f.server.LookUpInode(ctx, &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   "test.txt",
	}))
	err := f.server.LookUpInode(ctx, &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   "nonexistent.txt",
	})
	require.ErrorIs(t, err, fuse.ENOENT)

	openDirOp := &fuseops.OpenDirOp{Inode: fuseops.RootInodeID}
	require.NoError(t, f.server.OpenDir(ctx, openDirOp))
	require.NoError(t, f.server.ReadDir(ctx, &fuseops.ReadDirOp{
		Inode:  fuseops.RootInodeID,
		Handle: openDirOp.Handle,
		Offset: 0,
		Dst:    make([]byte, 1024),
	}))
	require.NoError(t, f.server.ReleaseDirHandle(ctx, &fuseops.ReleaseDirHandleOp{
		Handle: openDirOp.Handle,
	}))

	opts := collectScopedOpts(t, ctx, f.reader)

	// ObjectHandle.Attrs sets gcp.client.method="Object.Attrs" on operation-level
	// metrics, whereas BucketHandle.Objects sets gcp.client.method only inside the
	// per-page gRPC fetch ("grpcStorageClient.ObjectsListCall").
	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.operations",
		opAttrs(f.endpointHost, "GetObject", "Object.Attrs", "OK"), 1, append(opts, metrics.WithUnit("1"))...)
	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.operations",
		opAttrs(f.endpointHost, "GetObject", "Object.Attrs", "NOT_FOUND"), 3, opts...)
	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.operations",
		opAttrs(f.endpointHost, "ListObjects", "", "OK"), 1, opts...)

	metrics.VerifyHistogramFull[float64](t, ctx, nil, "gcp.client.request.duration",
		opAttrs(f.endpointHost, "GetObject", "Object.Attrs", "OK"), 1, math.SmallestNonzeroFloat64, nil,
		append(opts, metrics.AtLeastSum(), metrics.WithUnit("s"))...)
	metrics.VerifyHistogramFull[float64](t, ctx, nil, "gcp.client.request.duration",
		opAttrs(f.endpointHost, "GetObject", "Object.Attrs", "NOT_FOUND"), 3, math.SmallestNonzeroFloat64, nil,
		append(opts, metrics.AtLeastSum())...)
	metrics.VerifyHistogramFull[float64](t, ctx, nil, "gcp.client.request.duration",
		opAttrs(f.endpointHost, "ListObjects", "", "OK"), 1, math.SmallestNonzeroFloat64, nil,
		append(opts, metrics.AtLeastSum())...)

	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.attempts",
		attemptAttrs(f.endpointHost, "GetObject", "Object.Attrs", codes.OK, "OK"), 1, append(opts, metrics.WithUnit("1"))...)
	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.attempts",
		attemptAttrs(f.endpointHost, "GetObject", "Object.Attrs", codes.NotFound, "NOT_FOUND"), 3, opts...)
	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.attempts",
		attemptAttrs(f.endpointHost, "ListObjects", "grpcStorageClient.ObjectsListCall", codes.OK, "OK"), 1, opts...)

	metrics.VerifyHistogramFull[float64](t, ctx, nil, "rpc.client.call.duration",
		callDurationAttrs(f.endpointHost, "GetObject", "Object.Attrs", codes.OK, "OK"), 1, math.SmallestNonzeroFloat64, nil,
		append(opts, metrics.AtLeastSum(), metrics.WithUnit("s"))...)
	metrics.VerifyHistogramFull[float64](t, ctx, nil, "rpc.client.call.duration",
		callDurationAttrs(f.endpointHost, "GetObject", "Object.Attrs", codes.NotFound, "NOT_FOUND"), 3, math.SmallestNonzeroFloat64, nil,
		append(opts, metrics.AtLeastSum())...)
	metrics.VerifyHistogramFull[float64](t, ctx, nil, "rpc.client.call.duration",
		callDurationAttrs(f.endpointHost, "ListObjects", "grpcStorageClient.ObjectsListCall", codes.OK, "OK"), 1, math.SmallestNonzeroFloat64, nil,
		append(opts, metrics.AtLeastSum())...)

	metrics.VerifyHistogramFull[float64](t, ctx, nil, "gcp.storage.client.operation.ttfb",
		opAttrs(f.endpointHost, "GetObject", "Object.Attrs", ""), 4, math.SmallestNonzeroFloat64, nil,
		append(opts, metrics.AtLeastSum(), metrics.WithUnit("s"))...)
	metrics.VerifyHistogramFull[float64](t, ctx, nil, "gcp.storage.client.operation.ttfb",
		opAttrs(f.endpointHost, "ListObjects", "grpcStorageClient.ObjectsListCall", ""), 1, math.SmallestNonzeroFloat64, nil,
		append(opts, metrics.AtLeastSum())...)

	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.errors",
		opAttrs(f.endpointHost, "GetObject", "Object.Attrs", "NOT_FOUND"), 3, append(opts, metrics.WithUnit("1"))...)
	metrics.VerifyDataPointNotPresent(t, ctx, nil, "gcp.storage.client.errors",
		opAttrs(f.endpointHost, "GetObject", "Object.Attrs", "OK"), opts...)
	metrics.VerifyDataPointNotPresent(t, ctx, nil, "gcp.storage.client.errors",
		opAttrs(f.endpointHost, "ListObjects", "grpcStorageClient.ObjectsListCall", "OK"), opts...)

	metrics.VerifyMetricNotPresent(t, ctx, nil, "gcp.storage.client.response.body.size", opts...)
	metrics.VerifyMetricNotPresent(t, ctx, nil, "gcp.storage.client.request.body.size", opts...)
	for _, debugMetric := range storageClientDebugMetricNames {
		metrics.VerifyMetricNotPresent(t, ctx, nil, debugMetric, opts...)
	}
}

// TestStorageClientStandardMetrics_TransientRetry_AttemptsVsOperations verifies
// that a transient codes.Unavailable error followed by a successful retry
// records two wire attempts (1 UNAVAILABLE + 1 OK) and 1 error, but a single OK
// logical operation.
func TestStorageClientStandardMetrics_TransientRetry_AttemptsVsOperations(t *testing.T) {
	ctx := t.Context()
	f := setupStorageClientMetricsFixture(ctx, t, grpcMetricsTestOptions{
		enableOTelMetrics:             true,
		failFirstGetObjectUnavailable: true,
	})

	require.NoError(t, f.server.LookUpInode(ctx, &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   "test.txt",
	}))

	opts := collectScopedOpts(t, ctx, f.reader)

	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.operations",
		opAttrs(f.endpointHost, "GetObject", "Object.Attrs", "OK"), 1, opts...)
	metrics.VerifyDataPointNotPresent(t, ctx, nil, "gcp.storage.client.operations",
		opAttrs(f.endpointHost, "GetObject", "Object.Attrs", "UNAVAILABLE"), opts...)
	metrics.VerifyHistogramFull[float64](t, ctx, nil, "gcp.client.request.duration",
		opAttrs(f.endpointHost, "GetObject", "Object.Attrs", "OK"), 1, math.SmallestNonzeroFloat64, nil,
		append(opts, metrics.AtLeastSum())...)

	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.attempts",
		attemptAttrs(f.endpointHost, "GetObject", "Object.Attrs", codes.Unavailable, "UNAVAILABLE"), 1, opts...)
	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.attempts",
		attemptAttrs(f.endpointHost, "GetObject", "Object.Attrs", codes.OK, "OK"), 1, opts...)
	metrics.VerifyHistogramFull[float64](t, ctx, nil, "rpc.client.call.duration",
		callDurationAttrs(f.endpointHost, "GetObject", "Object.Attrs", codes.Unavailable, "UNAVAILABLE"), 1, math.SmallestNonzeroFloat64, nil,
		append(opts, metrics.AtLeastSum())...)
	metrics.VerifyHistogramFull[float64](t, ctx, nil, "rpc.client.call.duration",
		callDurationAttrs(f.endpointHost, "GetObject", "Object.Attrs", codes.OK, "OK"), 1, math.SmallestNonzeroFloat64, nil,
		append(opts, metrics.AtLeastSum())...)
	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.errors",
		opAttrs(f.endpointHost, "GetObject", "Object.Attrs", "UNAVAILABLE"), 1, opts...)
}

// TestStorageClientStandardMetrics_ReadFile_ResponseBodySizeAndStreamingTTFB
// validates gcp.storage.client.response.body.size and streaming ReadObject TTFB
// across single-chunk and multi-chunk server responses.
func TestStorageClientStandardMetrics_ReadFile_ResponseBodySizeAndStreamingTTFB(t *testing.T) {
	for _, tc := range []struct {
		name           string
		multiChunkRead bool
	}{
		{name: "single_chunk_read", multiChunkRead: false},
		{name: "multi_chunk_read", multiChunkRead: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			f := setupStorageClientMetricsFixture(ctx, t, grpcMetricsTestOptions{
				enableOTelMetrics: true,
				multiChunkRead:    tc.multiChunkRead,
			})

			lookupOp := &fuseops.LookUpInodeOp{Parent: fuseops.RootInodeID, Name: "test.txt"}
			require.NoError(t, f.server.LookUpInode(ctx, lookupOp))

			openOp := &fuseops.OpenFileOp{Inode: lookupOp.Entry.Child}
			require.NoError(t, f.server.OpenFile(ctx, openOp))

			readOp := &fuseops.ReadFileOp{
				Inode:  lookupOp.Entry.Child,
				Handle: openOp.Handle,
				Offset: 0,
				Dst:    make([]byte, 12),
			}
			require.NoError(t, f.server.ReadFile(ctx, readOp))
			require.Equal(t, 12, readOp.BytesRead)
			require.Equal(t, "test content", string(readOp.Dst))

			// Reader.Close runs on ReleaseFileHandle, emitting response.body.size,
			// operations, and gcp.client.request.duration.
			require.NoError(t, f.server.ReleaseFileHandle(ctx, &fuseops.ReleaseFileHandleOp{Handle: openOp.Handle}))

			opts := collectScopedOpts(t, ctx, f.reader)

			// For exact-range ReadObject streams where io.ReadFull completes without a
			// trailing EOF RecvMsg, the storage client's metricsState.target remains ""
			// when Reader.Close emits operation-level ReadObject metrics.
			metrics.VerifyHistogramFull[int64](t, ctx, nil, "gcp.storage.client.response.body.size",
				grpcAttrs("", "ReadObject"), 1, 12, nil, append(opts, metrics.WithUnit("By"))...)
			metrics.VerifyHistogramFull[float64](t, ctx, nil, "gcp.storage.client.operation.ttfb",
				grpcAttrs(f.endpointHost, "ReadObject"), 1, math.SmallestNonzeroFloat64, nil,
				append(opts, metrics.AtLeastSum(), metrics.WithUnit("s"))...)
			metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.operations",
				opAttrs("", "ReadObject", "Object.Reader", "OK"), 1, opts...)
			metrics.VerifyHistogramFull[float64](t, ctx, nil, "gcp.client.request.duration",
				opAttrs("", "ReadObject", "Object.Reader", "OK"), 1, math.SmallestNonzeroFloat64, nil,
				append(opts, metrics.AtLeastSum())...)
		})
	}
}

// TestStorageClientStandardMetrics_WriteFile_BodySize verifies both 0-byte and
// non-zero-byte WriteObject uploads: syncing an empty file records the
// WriteObject operation while omitting request.body.size, whereas syncing a
// 12-byte file records exact request.body.size (Count=1, Sum=12) alongside
// oneshot BidiWriteObject RPC metrics.
func TestStorageClientStandardMetrics_WriteFile_BodySize(t *testing.T) {
	ctx := t.Context()
	f := setupStorageClientMetricsFixture(ctx, t, grpcMetricsTestOptions{
		enableOTelMetrics: true,
	})

	// 1. Sync a newly created 0-byte file: Writer.markClosed omits request.body.size when totalBytes == 0.
	emptyCreate := &fuseops.CreateFileOp{Parent: fuseops.RootInodeID, Name: "empty_file.txt", Mode: 0644}
	require.NoError(t, f.server.CreateFile(ctx, emptyCreate))
	require.NoError(t, f.server.SyncFile(ctx, &fuseops.SyncFileOp{Inode: emptyCreate.Entry.Child, Handle: emptyCreate.Handle}))
	require.NoError(t, f.server.ReleaseFileHandle(ctx, &fuseops.ReleaseFileHandleOp{Handle: emptyCreate.Handle}))

	emptyOpts := collectScopedOpts(t, ctx, f.reader)
	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.operations",
		opAttrs(f.endpointHost, "WriteObject", "Object.Writer", "OK"), 1, emptyOpts...)
	metrics.VerifyMetricNotPresent(t, ctx, nil, "gcp.storage.client.request.body.size", emptyOpts...)

	// 2. Write and sync a 12-byte file: operations advances to 2 while request.body.size records 1 sample of 12 bytes.
	createOp := &fuseops.CreateFileOp{Parent: fuseops.RootInodeID, Name: "new_file.txt", Mode: 0644}
	require.NoError(t, f.server.CreateFile(ctx, createOp))
	payload := []byte("write-sample")
	require.NoError(t, f.server.WriteFile(ctx, &fuseops.WriteFileOp{
		Inode:  createOp.Entry.Child,
		Handle: createOp.Handle,
		Offset: 0,
		Data:   payload,
	}))
	require.NoError(t, f.server.SyncFile(ctx, &fuseops.SyncFileOp{Inode: createOp.Entry.Child, Handle: createOp.Handle}))
	require.NoError(t, f.server.ReleaseFileHandle(ctx, &fuseops.ReleaseFileHandleOp{Handle: createOp.Handle}))

	opts := collectScopedOpts(t, ctx, f.reader)
	metrics.VerifyHistogramFull[int64](t, ctx, nil, "gcp.storage.client.request.body.size",
		grpcAttrs(f.endpointHost, "WriteObject"), 1, int64(len(payload)), nil, append(opts, metrics.WithUnit("By"))...)
	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.operations",
		opAttrs(f.endpointHost, "WriteObject", "Object.Writer", "OK"), 2, opts...)
	metrics.VerifyHistogramFull[float64](t, ctx, nil, "gcp.client.request.duration",
		opAttrs(f.endpointHost, "WriteObject", "Object.Writer", "OK"), 2, math.SmallestNonzeroFloat64, nil,
		append(opts, metrics.AtLeastSum())...)
	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.attempts",
		attemptAttrs(f.endpointHost, "WriteObject", "Object.Writer", codes.OK, "OK"), 2, opts...)
	metrics.VerifyHistogramFull[float64](t, ctx, nil, "gcp.storage.client.operation.ttfb",
		grpcAttrs(f.endpointHost, "WriteObject"), 2, math.SmallestNonzeroFloat64, nil,
		append(opts, metrics.AtLeastSum())...)
	metrics.VerifyHistogramFull[float64](t, ctx, nil, "rpc.client.call.duration",
		callDurationAttrs(f.endpointHost, "BidiWriteObject", "Object.Writer", codes.OK, "OK"), 2, math.SmallestNonzeroFloat64, nil,
		append(opts, metrics.AtLeastSum())...)
}

// TestStorageClientDebugMetrics_NetworkAuthActiveAndGFE validates the 7
// non-stall Debug metrics in a single TLS + OAuth mount lifecycle:
//   - gcp/storage/client/auth/credential_refresh/duration (OK)
//   - gcp/storage/client/gfe/header_missing (when header absent vs present)
//   - gcp/storage/client/request/active (1 mid-flight, 0 post-completion)
//   - gcp/storage/client/network/tcp/connect/duration
//   - gcp/storage/client/network/tls/handshake/duration
//   - gcp/storage/client/network/bytes/received
//   - gcp/storage/client/network/bytes/sent
func TestStorageClientDebugMetrics_NetworkAuthActiveAndGFE(t *testing.T) {
	ctx := t.Context()
	f := setupStorageClientMetricsFixture(ctx, t, grpcMetricsTestOptions{
		enableOTelMetrics:      true,
		enableOTelDebugMetrics: true,
		blockBidiWriteObject:   true,
	})

	// Phase 1: LookUpInode with sendGFEHeader=false triggers initial OAuth2 token
	// exchange and increments gfe.header_missing on both OK and NOT_FOUND GetObject RPCs.
	lookupOp := &fuseops.LookUpInodeOp{Parent: fuseops.RootInodeID, Name: "test.txt"}
	require.NoError(t, f.server.LookUpInode(ctx, lookupOp))

	// Phase 2: Enable x-goog-gfe-service-time header for subsequent ReadObject and
	// BidiWriteObject streams so gfe.duration is recorded instead of gfe.header_missing.
	f.fakeSrv.sendGFEHeader.Store(true)

	openOp := &fuseops.OpenFileOp{Inode: lookupOp.Entry.Child}
	require.NoError(t, f.server.OpenFile(ctx, openOp))
	require.NoError(t, f.server.ReadFile(ctx, &fuseops.ReadFileOp{
		Inode:  lookupOp.Entry.Child,
		Handle: openOp.Handle,
		Offset: 0,
		Dst:    make([]byte, 12),
	}))
	require.NoError(t, f.server.ReleaseFileHandle(ctx, &fuseops.ReleaseFileHandleOp{Handle: openOp.Handle}))

	// Phase 3: Pause BidiWriteObject mid-flight to verify non-monotonic request.active == 1.
	createOp := &fuseops.CreateFileOp{Parent: fuseops.RootInodeID, Name: "active_file.txt", Mode: 0644}
	require.NoError(t, f.server.CreateFile(ctx, createOp))
	payload := []byte("write-sample")
	require.NoError(t, f.server.WriteFile(ctx, &fuseops.WriteFileOp{
		Inode:  createOp.Entry.Child,
		Handle: createOp.Handle,
		Offset: 0,
		Data:   payload,
	}))

	syncErrCh := make(chan error, 1)
	go func() {
		syncErrCh <- f.server.SyncFile(ctx, &fuseops.SyncFileOp{
			Inode:  createOp.Entry.Child,
			Handle: createOp.Handle,
		})
	}()

	var (
		releaseOnce sync.Once
		waitOnce    sync.Once
		syncErr     error
	)
	releaseAndWaitSync := func() error {
		releaseOnce.Do(func() { close(f.fakeSrv.bidiWriteRelease) })
		waitOnce.Do(func() { syncErr = <-syncErrCh })
		return syncErr
	}
	defer func() { _ = releaseAndWaitSync() }()

	select {
	case <-f.fakeSrv.bidiWriteEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for BidiWriteObject to enter fake server")
	}

	inFlightOpts := collectScopedOpts(t, ctx, f.reader)
	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.request.active",
		grpcAttrs(f.endpointHost, "BidiWriteObject"), 1, append(inFlightOpts, metrics.NonMonotonic(), metrics.WithUnit("1"))...)
	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.request.active",
		grpcAttrs(f.endpointHost, "GetObject"), 0, append(inFlightOpts, metrics.NonMonotonic())...)

	require.NoError(t, releaseAndWaitSync())
	require.NoError(t, f.server.ReleaseFileHandle(ctx, &fuseops.ReleaseFileHandleOp{Handle: createOp.Handle}))

	opts := collectScopedOpts(t, ctx, f.reader)

	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.request.active",
		grpcAttrs(f.endpointHost, "BidiWriteObject"), 0, append(opts, metrics.NonMonotonic())...)

	// metricsTokenProvider wraps authCreds.TokenProvider per RPC (2 GetObjects in
	// LookUpInode + 1 ReadObject + 1 GetObject in CreateFile + 1 BidiWriteObject = 5).
	metrics.VerifyHistogramFull[float64](t, ctx, nil, "gcp.storage.client.auth.credential_refresh.duration",
		attribute.NewSet(attribute.String("error.type", "OK")), 5, math.SmallestNonzeroFloat64, nil,
		append(opts, metrics.AtLeastSum(), metrics.WithUnit("s"))...)
	metrics.VerifyDataPointNotPresent(t, ctx, nil, "gcp.storage.client.auth.credential_refresh.duration",
		attribute.NewSet(attribute.String("error.type", "CLIENT_AUTHENTICATION_ERROR")), opts...)

	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.gfe.header_missing",
		opAttrs(f.endpointHost, "GetObject", "", "OK"), 1, append(opts, metrics.WithUnit("1"))...)
	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.gfe.header_missing",
		opAttrs(f.endpointHost, "GetObject", "", "NOT_FOUND"), 1, opts...)
	metrics.VerifyDataPointNotPresent(t, ctx, nil, "gcp.storage.client.gfe.header_missing",
		attribute.NewSet(attribute.String("rpc.method", "BidiWriteObject")), append(opts, metrics.Subset())...)
	metrics.VerifyHistogramFull[float64](t, ctx, nil, "gcp.storage.client.gfe.duration",
		grpcAttrs(f.endpointHost, "BidiWriteObject"), 1, 0.01, nil, append(opts, metrics.AtLeastSum(), metrics.WithUnit("s"))...)

	metrics.VerifyHistogramFull[float64](t, ctx, nil, "gcp.storage.client.network.tcp.connect.duration",
		grpcAttrs("storage.googleapis.com", ""), 1, math.SmallestNonzeroFloat64, nil,
		append(opts, metrics.AtLeastSum(), metrics.WithUnit("s"))...)
	metrics.VerifyHistogramFull[float64](t, ctx, nil, "gcp.storage.client.network.tls.handshake.duration",
		grpcAttrs("storage.googleapis.com", ""), 1, math.SmallestNonzeroFloat64, nil,
		append(opts, metrics.AtLeastSum(), metrics.WithUnit("s"))...)

	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.network.bytes.received",
		grpcAttrs("storage.googleapis.com", "ReadObject"), 12, append(opts, metrics.AtLeast(), metrics.WithUnit("By"))...)
	for _, method := range []string{"GetObject", "BidiWriteObject"} {
		metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.network.bytes.received",
			grpcAttrs("storage.googleapis.com", method), 1, append(opts, metrics.AtLeast())...)
	}

	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.network.bytes.sent",
		grpcAttrs("storage.googleapis.com", "BidiWriteObject"), int64(len(payload)), append(opts, metrics.AtLeast(), metrics.WithUnit("By"))...)
	for _, method := range []string{"GetObject", "ReadObject"} {
		metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.network.bytes.sent",
			grpcAttrs("storage.googleapis.com", method), 1, append(opts, metrics.AtLeast())...)
	}
}

// TestStorageClientDebugMetrics_CredentialRefreshFailure validates
// gcp/storage/client/auth/credential_refresh/duration with
// error.type="CLIENT_AUTHENTICATION_ERROR" when OAuth2 token exchange fails.
func TestStorageClientDebugMetrics_CredentialRefreshFailure(t *testing.T) {
	ctx := t.Context()
	f := setupStorageClientMetricsFixture(ctx, t, grpcMetricsTestOptions{
		enableOTelMetrics:      true,
		enableOTelDebugMetrics: true,
		failAuthTokenRefresh:   true,
	})

	err := f.server.LookUpInode(ctx, &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   "test.txt",
	})
	require.Error(t, err)

	// LookUpInode issues two concurrent GetObject calls ("test.txt" and "test.txt/");
	// with maxRetryAttempts=1, each fails token acquisition once (Count=2).
	opts := collectScopedOpts(t, ctx, f.reader)
	metrics.VerifyHistogramFull[float64](t, ctx, nil, "gcp.storage.client.auth.credential_refresh.duration",
		attribute.NewSet(attribute.String("error.type", "CLIENT_AUTHENTICATION_ERROR")), 2, math.SmallestNonzeroFloat64, nil,
		append(opts, metrics.AtLeastSum(), metrics.WithUnit("s"))...)
	metrics.VerifyDataPointNotPresent(t, ctx, nil, "gcp.storage.client.auth.credential_refresh.duration",
		attribute.NewSet(attribute.String("error.type", "OK")), opts...)
}

// TestStorageClientDebugMetrics_StallDuration_ReadStall validates
// gcp/storage/client/stall/duration when the storage client's dynamic
// read-stall timeout aborts a stalled first ReadObject stream and succeeds on
// retry.
func TestStorageClientDebugMetrics_StallDuration_ReadStall(t *testing.T) {
	ctx := t.Context()
	f := setupStorageClientMetricsFixture(ctx, t, grpcMetricsTestOptions{
		enableOTelMetrics:      true,
		enableOTelDebugMetrics: true,
		stallFirstRead:         true,
		enableReadStallRetry:   true,
	})

	lookupOp := &fuseops.LookUpInodeOp{Parent: fuseops.RootInodeID, Name: "test.txt"}
	require.NoError(t, f.server.LookUpInode(ctx, lookupOp))

	openOp := &fuseops.OpenFileOp{Inode: lookupOp.Entry.Child}
	require.NoError(t, f.server.OpenFile(ctx, openOp))

	readOp := &fuseops.ReadFileOp{
		Inode:  lookupOp.Entry.Child,
		Handle: openOp.Handle,
		Offset: 0,
		Dst:    make([]byte, 12),
	}
	require.NoError(t, f.server.ReadFile(ctx, readOp))
	require.Equal(t, 12, readOp.BytesRead)
	require.NoError(t, f.server.ReleaseFileHandle(ctx, &fuseops.ReleaseFileHandleOp{Handle: openOp.Handle}))

	opts := collectScopedOpts(t, ctx, f.reader)
	metrics.VerifyHistogramFull[float64](t, ctx, nil, "gcp.storage.client.stall.duration",
		grpcAttrs(f.endpointHost, "ReadObject"), 1, 0.01, nil,
		append(opts, metrics.AtLeastSum(), metrics.WithUnit("s"))...)

	// The canceled first ReadObject attempt populates metricsState.target so
	// Reader.Close emits response.body.size with f.endpointHost.
	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.attempts",
		attemptAttrs(f.endpointHost, "ReadObject", "grpcStorageClient.NewRangeReaderReadObject", codes.Canceled, "CANCELLED"), 1, opts...)
	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.errors",
		opAttrs(f.endpointHost, "ReadObject", "grpcStorageClient.NewRangeReaderReadObject", "CANCELLED"), 1, opts...)
	metrics.VerifyHistogramFull[float64](t, ctx, nil, "rpc.client.call.duration",
		callDurationAttrs(f.endpointHost, "ReadObject", "grpcStorageClient.NewRangeReaderReadObject", codes.Canceled, "CANCELLED"), 1, math.SmallestNonzeroFloat64, nil,
		append(opts, metrics.AtLeastSum())...)
	metrics.VerifyHistogramFull[int64](t, ctx, nil, "gcp.storage.client.response.body.size",
		grpcAttrs(f.endpointHost, "ReadObject"), 1, 12, nil, opts...)
}

// TestStorageClientDebugMetrics_StallDuration_WriteStall validates the storage
// client's gRPC write-stall watchdog recording gcp/storage/client/stall/duration
// when a resumable streaming BidiWriteObject upload stalls and recovers via
// QueryWriteStatus.
func TestStorageClientDebugMetrics_StallDuration_WriteStall(t *testing.T) {
	ctx := t.Context()
	// ChunkTransferTimeoutSecs (1s min) and WriteConfig.BlockSizeMb (1 MiB min)
	// have integer config granularity; writing > 1 MiB is required so gRPCWriter
	// initiates a resumable upload rather than a oneshot upload (which skips the
	// write-stall watchdog).
	f := setupStorageClientMetricsFixture(ctx, t, grpcMetricsTestOptions{
		enableOTelMetrics:        true,
		enableOTelDebugMetrics:   true,
		enableStreamingWrites:    true,
		stallFirstBidiWrite:      true,
		chunkTransferTimeoutSecs: 1,
		chunkRetryDeadlineSecs:   10,
	})

	createOp := &fuseops.CreateFileOp{Parent: fuseops.RootInodeID, Name: "stalled_write.txt", Mode: 0644}
	require.NoError(t, f.server.CreateFile(ctx, createOp))

	firstBlock := make([]byte, 1024*1024)
	tail := []byte("write-sample")
	require.NoError(t, f.server.WriteFile(ctx, &fuseops.WriteFileOp{
		Inode:  createOp.Entry.Child,
		Handle: createOp.Handle,
		Offset: 0,
		Data:   firstBlock,
	}))
	require.NoError(t, f.server.WriteFile(ctx, &fuseops.WriteFileOp{
		Inode:  createOp.Entry.Child,
		Handle: createOp.Handle,
		Offset: int64(len(firstBlock)),
		Data:   tail,
	}))
	// FlushFile finalizes non-zonal streaming writes synchronously (Writer.Close).
	require.NoError(t, f.server.FlushFile(ctx, &fuseops.FlushFileOp{
		Inode:  createOp.Entry.Child,
		Handle: createOp.Handle,
	}))
	require.NoError(t, f.server.ReleaseFileHandle(ctx, &fuseops.ReleaseFileHandleOp{Handle: createOp.Handle}))

	opts := collectScopedOpts(t, ctx, f.reader)
	totalBytes := int64(len(firstBlock) + len(tail))
	metrics.VerifyHistogramFull[float64](t, ctx, nil, "gcp.storage.client.stall.duration",
		grpcAttrs(f.endpointHost, "BidiWriteObject"), 1, 1.0, nil,
		append(opts, metrics.AtLeastSum(), metrics.WithUnit("s"))...)
	metrics.VerifyHistogramFull[int64](t, ctx, nil, "gcp.storage.client.request.body.size",
		grpcAttrs(f.endpointHost, "WriteObject"), 1, totalBytes, nil, opts...)
	metrics.VerifyCounterMetric(t, ctx, nil, "gcp.storage.client.operations",
		opAttrs(f.endpointHost, "WriteObject", "Object.Writer", "OK"), 1, opts...)
}

// TestStorageClientMetrics_Disabled verifies that all 16 Cloud Storage client
// metrics are suppressed both when GCP_STORAGE_GO_ENABLE_OTEL_METRICS=false
// (keeping grpc.client.* transport metrics active) and when EnableGrpcMetrics
// is false (passing storage.WithDisabledClientMetrics() to override OTel env vars).
func TestStorageClientMetrics_Disabled(t *testing.T) {
	allMetrics := append(append([]string(nil), storageClientStandardMetricNames...), storageClientDebugMetricNames...)
	for _, tc := range []struct {
		name              string
		opts              grpcMetricsTestOptions
		expectGRPCMetrics bool
	}{
		{
			name:              "otel_env_var_false",
			opts:              grpcMetricsTestOptions{enableOTelMetrics: false},
			expectGRPCMetrics: true,
		},
		{
			name: "grpc_metrics_flag_false",
			opts: grpcMetricsTestOptions{
				enableOTelMetrics:      true,
				enableOTelDebugMetrics: true,
				disableGRPCMetricsFlag: true,
			},
			expectGRPCMetrics: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			f := setupStorageClientMetricsFixture(ctx, t, tc.opts)

			require.NoError(t, f.server.LookUpInode(ctx, &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   "test.txt",
			}))

			rm := metrics.CollectResourceMetrics(t, ctx, f.reader)
			rmOpt := metrics.WithResourceMetrics(rm)
			metrics.VerifyCounterMetric(t, ctx, nil, "fs/ops_count",
				attribute.NewSet(attribute.String("fs_op", "LookUpInode")), 1, rmOpt, metrics.Subset())

			if tc.expectGRPCMetrics {
				metrics.VerifyCounterMetric(t, ctx, nil, "grpc.client.attempt.started",
					attribute.NewSet(attribute.String("grpc.method", "google.storage.v2.Storage/GetObject")),
					1, rmOpt, metrics.AtLeast(), metrics.Subset())
			} else {
				metrics.VerifyMetricNotPresent(t, ctx, nil, "grpc.client.attempt.started", rmOpt)
				metrics.VerifyMetricNotPresent(t, ctx, nil, "grpc.client.call.duration", rmOpt)
			}

			scoped := []metrics.VerifyOption{rmOpt, metrics.WithScope(storageClientMeterScope)}
			for _, name := range allMetrics {
				metrics.VerifyMetricNotPresent(t, ctx, nil, name, scoped...)
			}
		})
	}
}

// setupStorageClientMetricsFixture enables TLS + service-account OAuth whenever
// Cloud Storage client OTel metrics are active because cloud.google.com/go/storage's
// initGRPCMetricsAndWrapCredentials requires non-nil *auth.Credentials (panicking
// on option.WithoutAuthentication) and gRPC per-RPC OAuth requires TLS.
func setupStorageClientMetricsFixture(ctx context.Context, t *testing.T, opts grpcMetricsTestOptions) *grpcMetricsTestFixture {
	t.Helper()
	if opts.enableOTelMetrics {
		opts.useTLSAndAuth = true
	}
	return setupGRPCMetricsFixture(ctx, t, opts)
}

func collectScopedOpts(t *testing.T, ctx context.Context, reader *metric.ManualReader) []metrics.VerifyOption {
	t.Helper()
	return []metrics.VerifyOption{
		metrics.WithResourceMetrics(metrics.CollectResourceMetrics(t, ctx, reader)),
		metrics.WithScope(storageClientMeterScope),
	}
}

// ensureTestTLSFixture lazily generates an ECDSA P-256 localhost/127.0.0.1 TLS
// certificate and PKCS#8 service-account PEM once per test binary, installing
// the certificate into Go's x509 fallback root pool so TLS verification works
// portably regardless of test execution order.
func ensureTestTLSFixture(t *testing.T) {
	t.Helper()
	testTLSOnce.Do(func() {
		testTLSInitErr = func() error {
			priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				return err
			}
			pkcs8Bytes, err := x509.MarshalPKCS8PrivateKey(priv)
			if err != nil {
				return err
			}
			keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8Bytes})
			tmpl := &x509.Certificate{
				SerialNumber:          big.NewInt(1),
				Subject:               pkix.Name{CommonName: "127.0.0.1"},
				NotBefore:             time.Now().Add(-time.Hour),
				NotAfter:              time.Now().Add(24 * time.Hour),
				KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
				ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
				BasicConstraintsValid: true,
				IsCA:                  true,
				IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
				DNSNames:              []string{"localhost"},
			}
			derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
			if err != nil {
				return err
			}
			certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
			cert, err := tls.X509KeyPair(certPEM, keyPEM)
			if err != nil {
				return err
			}

			// Use a non-system CertPool (systemPool=false) rather than SystemCertPool():
			// on darwin/windows, SystemCertPool() preserves systemPool=true, causing
			// Certificate.Verify (opts.Roots == nil) to call the OS verifier and ignore
			// appended PEM roots; on Linux, NewCertPool() also skips /etc/ssl/certs I/O.
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM(certPEM)

			// Go 1.21+ internal/godebug updates Setting.Value() synchronously on
			// os.Setenv("GODEBUG", ...), and x509.SetFallbackRoots eagerly reads
			// x509usefallbackroots.Value() (latching useFallbackRoots if initSystemRoots
			// has not run yet, or replacing systemRoots in-place under systemRootsMu if
			// it already has), so GODEBUG can be restored right after SetFallbackRoots.
			prevGODEBUG, hadGODEBUG := os.LookupEnv("GODEBUG")
			if prevGODEBUG == "" {
				_ = os.Setenv("GODEBUG", "x509usefallbackroots=1")
			} else {
				_ = os.Setenv("GODEBUG", prevGODEBUG+",x509usefallbackroots=1")
			}
			x509.SetFallbackRoots(pool)
			if hadGODEBUG {
				_ = os.Setenv("GODEBUG", prevGODEBUG)
			} else {
				_ = os.Unsetenv("GODEBUG")
			}

			testSAPrivateKeyPEM = string(keyPEM)
			testTLSServerCert = cert
			return nil
		}()
	})
	require.NoError(t, testTLSInitErr)
}

func writeFakeServiceAccountKeyFile(t *testing.T, tokenURL string) string {
	t.Helper()
	saJSON, err := json.Marshal(map[string]string{
		"type":            "service_account",
		"project_id":      "test-project",
		"private_key_id":  "test-key-id",
		"private_key":     testSAPrivateKeyPEM,
		"client_email":    "test@test-project.iam.gserviceaccount.com",
		"token_uri":       tokenURL,
		"universe_domain": "googleapis.com",
	})
	require.NoError(t, err)
	keyPath := filepath.Join(t.TempDir(), "sa-key.json")
	require.NoError(t, os.WriteFile(keyPath, saJSON, 0600))
	return keyPath
}

func grpcAttrs(host, method string, extra ...attribute.KeyValue) attribute.Set {
	kvs := make([]attribute.KeyValue, 0, 3+len(extra))
	kvs = append(kvs,
		attribute.String("rpc.system.name", "grpc"),
		attribute.String("server.address", host),
	)
	if method != "" {
		kvs = append(kvs, attribute.String("rpc.method", method))
	}
	return attribute.NewSet(append(kvs, extra...)...)
}

func opAttrs(host, method, clientMethod, errType string, extra ...attribute.KeyValue) attribute.Set {
	if clientMethod != "" {
		extra = append(extra, attribute.String("gcp.client.method", clientMethod))
	}
	if errType != "" {
		extra = append(extra, attribute.String("error.type", errType))
	}
	return grpcAttrs(host, method, extra...)
}

func attemptAttrs(host, method, clientMethod string, code codes.Code, errType string) attribute.Set {
	return opAttrs(host, method, clientMethod, errType, attribute.Int64("rpc.grpc.status_code", int64(code)))
}

func callDurationAttrs(host, method, clientMethod string, code codes.Code, errType string) attribute.Set {
	return opAttrs(host, method, clientMethod, errType,
		attribute.String("rpc.service", "google.storage.v2.Storage"),
		attribute.Int64("rpc.grpc.status_code", int64(code)),
		attribute.Int64("rpc.response.status_code", int64(code)),
	)
}
