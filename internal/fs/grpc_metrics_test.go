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
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/wrappers"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

// StorageServer is a dummy interface for gRPC registration.
type StorageServer any

// reflectFakeServer implements the gRPC Storage service using reflection and dynamicpb
// to avoid importing the conflicting storagepb package. A functional server is
// needed to trigger client-side gRPC metrics and satisfy GCSFuse's stateful
// expectations (e.g., GetObject must succeed before ReadObject is called).
type reflectFakeServer struct{}

func (s *reflectFakeServer) GetObject(ctx context.Context, req any) (any, error) {
	dm := req.(*dynamicpb.Message)
	objectName := dm.Get(dm.Descriptor().Fields().ByName("object")).String()
	bucketName := dm.Get(dm.Descriptor().Fields().ByName("bucket")).String()
	fmt.Printf("Fake GCS: GetObject %s in bucket %s\n", objectName, bucketName)

	if strings.HasSuffix(objectName, "/") {
		return nil, status.Error(codes.NotFound, "not found")
	}

	// Dynamically create an Object message using the global registry.
	msgType, _ := protoregistry.GlobalTypes.FindMessageByName("google.storage.v2.Object")
	obj := dynamicpb.NewMessage(msgType.Descriptor())
	obj.Set(obj.Descriptor().Fields().ByName("name"), protoreflect.ValueOf(objectName))
	obj.Set(obj.Descriptor().Fields().ByName("bucket"), protoreflect.ValueOf(bucketName))
	obj.Set(obj.Descriptor().Fields().ByName("size"), protoreflect.ValueOf(int64(12)))

	return obj, nil
}

func (s *reflectFakeServer) ListObjects(ctx context.Context, req any) (any, error) {
	msgType, _ := protoregistry.GlobalTypes.FindMessageByName("google.storage.v2.ListObjectsResponse")
	return dynamicpb.NewMessage(msgType.Descriptor()), nil
}

func (s *reflectFakeServer) ReadObject(req any, stream grpc.ServerStream) error {
	dm := req.(*dynamicpb.Message)
	objectName := dm.Get(dm.Descriptor().Fields().ByName("object")).String()
	fmt.Printf("Fake GCS: ReadObject %s\n", objectName)

	// Send one response.
	respType, _ := protoregistry.GlobalTypes.FindMessageByName("google.storage.v2.ReadObjectResponse")
	resp := dynamicpb.NewMessage(respType.Descriptor())

	// Set data.
	dataField := resp.Descriptor().Fields().ByName("checksummed_data")
	dataMsg := dynamicpb.NewMessage(dataField.Message())
	dataMsg.Set(dataMsg.Descriptor().Fields().ByName("content"), protoreflect.ValueOf([]byte("test content")))
	resp.Set(dataField, protoreflect.ValueOfMessage(dataMsg))

	// Set metadata.
	metaField := resp.Descriptor().Fields().ByName("metadata")
	metaMsg := dynamicpb.NewMessage(metaField.Message())
	metaMsg.Set(metaMsg.Descriptor().Fields().ByName("name"), protoreflect.ValueOf(objectName))
	metaMsg.Set(metaMsg.Descriptor().Fields().ByName("size"), protoreflect.ValueOf(int64(12)))
	resp.Set(metaField, protoreflect.ValueOfMessage(metaMsg))

	return stream.SendMsg(resp)
}

func (s *reflectFakeServer) StartResumableWrite(ctx context.Context, req any) (any, error) {
	respType, _ := protoregistry.GlobalTypes.FindMessageByName("google.storage.v2.StartResumableWriteResponse")
	resp := dynamicpb.NewMessage(respType.Descriptor())
	resp.Set(resp.Descriptor().Fields().ByName("upload_id"), protoreflect.ValueOf("upload-id"))
	return resp, nil
}

func (s *reflectFakeServer) WriteObject(stream grpc.ServerStream) error {
	msgType, _ := protoregistry.GlobalTypes.FindMessageByName("google.storage.v2.WriteObjectRequest")
	req := dynamicpb.NewMessage(msgType.Descriptor())
	if err := stream.RecvMsg(req); err != nil {
		return err
	}

	specField := req.Descriptor().Fields().ByName("write_object_spec")
	if req.Has(specField) {
		spec := req.Get(specField).Message()
		resField := spec.Descriptor().Fields().ByName("resource")
		if spec.Has(resField) {
			res := spec.Get(resField).Message()
			nameField := res.Descriptor().Fields().ByName("name")
			fmt.Printf("Fake GCS: WriteObject %s\n", res.Get(nameField).String())
		}
	}

	respType, _ := protoregistry.GlobalTypes.FindMessageByName("google.storage.v2.WriteObjectResponse")
	resp := dynamicpb.NewMessage(respType.Descriptor())
	return stream.SendMsg(resp)
}

func registerFakeStorageServer(s *grpc.Server, srv *reflectFakeServer) {
	// Dummy interface type pointer.
	var storageServerPtr *StorageServer

	desc := &grpc.ServiceDesc{
		ServiceName: "google.storage.v2.Storage",
		HandlerType: storageServerPtr,
		Methods: []grpc.MethodDesc{
			{
				MethodName: "GetObject",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					msgType, _ := protoregistry.GlobalTypes.FindMessageByName("google.storage.v2.GetObjectRequest")
					in := dynamicpb.NewMessage(msgType.Descriptor())
					if err := dec(in); err != nil {
						return nil, err
					}
					if interceptor == nil {
						return srv.(*reflectFakeServer).GetObject(ctx, in)
					}
					return interceptor(ctx, in, &grpc.UnaryServerInfo{Server: srv, FullMethod: "/google.storage.v2.Storage/GetObject"}, func(ctx context.Context, req any) (any, error) {
						return srv.(*reflectFakeServer).GetObject(ctx, req)
					})
				},
			},
			{
				MethodName: "ListObjects",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					msgType, _ := protoregistry.GlobalTypes.FindMessageByName("google.storage.v2.ListObjectsRequest")
					in := dynamicpb.NewMessage(msgType.Descriptor())
					if err := dec(in); err != nil {
						return nil, err
					}
					if interceptor == nil {
						return srv.(*reflectFakeServer).ListObjects(ctx, in)
					}
					return interceptor(ctx, in, &grpc.UnaryServerInfo{Server: srv, FullMethod: "/google.storage.v2.Storage/ListObjects"}, func(ctx context.Context, req any) (any, error) {
						return srv.(*reflectFakeServer).ListObjects(ctx, req)
					})
				},
			},
			{
				MethodName: "StartResumableWrite",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					msgType, _ := protoregistry.GlobalTypes.FindMessageByName("google.storage.v2.StartResumableWriteRequest")
					in := dynamicpb.NewMessage(msgType.Descriptor())
					if err := dec(in); err != nil {
						return nil, err
					}
					if interceptor == nil {
						return srv.(*reflectFakeServer).StartResumableWrite(ctx, in)
					}
					return interceptor(ctx, in, &grpc.UnaryServerInfo{Server: srv, FullMethod: "/google.storage.v2.Storage/StartResumableWrite"}, func(ctx context.Context, req any) (any, error) {
						return srv.(*reflectFakeServer).StartResumableWrite(ctx, req)
					})
				},
			},
		},
		Streams: []grpc.StreamDesc{
			{
				StreamName: "ReadObject",
				Handler: func(srv any, stream grpc.ServerStream) error {
					msgType, _ := protoregistry.GlobalTypes.FindMessageByName("google.storage.v2.ReadObjectRequest")
					m := dynamicpb.NewMessage(msgType.Descriptor())
					if err := stream.RecvMsg(m); err != nil {
						return err
					}
					return srv.(*reflectFakeServer).ReadObject(m, stream)
				},
				ServerStreams: true,
			},
			{
				StreamName: "WriteObject",
				Handler: func(srv any, stream grpc.ServerStream) error {
					return srv.(*reflectFakeServer).WriteObject(stream)
				},
				ClientStreams: true,
			},
			{
				StreamName: "BidiWriteObject",
				Handler: func(srv any, stream grpc.ServerStream) error {
					return status.Error(codes.Unimplemented, "unimplemented")
				},
				ServerStreams: true,
				ClientStreams: true,
			},
		},
		Metadata: "google/storage/v2/storage.proto",
	}
	s.RegisterService(desc, srv)
}

type fakeStorageControlServer struct {
	controlpb.UnimplementedStorageControlServer
}

func (s *fakeStorageControlServer) GetStorageLayout(ctx context.Context, req *controlpb.GetStorageLayoutRequest) (*controlpb.StorageLayout, error) {
	return &controlpb.StorageLayout{
		Name: req.Name,
		HierarchicalNamespace: &controlpb.StorageLayout_HierarchicalNamespace{
			Enabled: false,
		},
	}, nil
}

func createTestFileSystemWithGrpcMetrics(ctx context.Context, t *testing.T, params *serverConfigParams) (storage.StorageHandle, fuseutil.FileSystem, metrics.MetricHandle, *metric.ManualReader) {
	t.Helper()

	// Bypass the protobuf registration conflict panic.
	_ = os.Setenv("GOLANG_PROTOBUF_REGISTRATION_CONFLICT", "ignore")
	_ = os.Setenv("GOOGLE_CLOUD_PROJECT", "test-project")

	// Start fake gRPC server
	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	grpcServer := grpc.NewServer()
	registerFakeStorageServer(grpcServer, &reflectFakeServer{})
	controlpb.RegisterStorageControlServer(grpcServer, &fakeStorageControlServer{})
	go func() {
		_ = grpcServer.Serve(lis)
	}()
	t.Cleanup(func() {
		grpcServer.Stop()
	})

	// Start fake metadata server to provide project ID for gRPC metrics
	metadataLis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	mux := http.NewServeMux()
	mux.HandleFunc("/computeMetadata/v1/project/project-id", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Metadata-Flavor", "Google")
		_, _ = w.Write([]byte("test-project"))
	})
	metadataServer := &http.Server{Handler: mux}
	go func() {
		_ = metadataServer.Serve(metadataLis)
	}()
	t.Cleanup(func() {
		_ = metadataServer.Close()
	})
	_ = os.Setenv("GCE_METADATA_HOST", metadataLis.Addr().String())
	t.Cleanup(func() { _ = os.Unsetenv("GCE_METADATA_HOST") })

	origProvider := otel.GetMeterProvider()
	t.Cleanup(func() { otel.SetMeterProvider(origProvider) })
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)

	mh, err := metrics.NewOTelMetrics(ctx, 1, 100)
	require.NoError(t, err, "metrics.NewOTelMetrics")

	clientConfig := storageutil.StorageClientConfig{
		ClientProtocol:    cfg.GRPC,
		CustomEndpoint:    lis.Addr().String(),
		EnableGrpcMetrics: true,
		IsGKE:             true,
		AnonymousAccess:   true,
		MetricHandle:      mh,
	}

	sh, err := storage.NewStorageHandle(ctx, clientConfig, "")
	require.NoError(t, err)

	// Poke the storage client to trigger internal DirectPath checks with a short timeout.
	// This prevents the subsequent real operations from hanging for 60s.
	shortCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	_, _ = sh.BucketHandle(shortCtx, "test-bucket", "", false)
	cancel()

	bucketName := "test-bucket"
	bucketConfig := gcsx.BucketConfig{
		BillingProject:       "",
		StatCacheMaxSizeMB:   32,
		StatCacheTTL:         time.Minute,
		NegativeStatCacheTTL: time.Minute,
		TmpObjectPrefix:      ".gcsfuse_tmp/",
	}

	bm := gcsx.NewBucketManager(bucketConfig, sh)

	serverCfg := &fs.ServerConfig{
		NewConfig: &cfg.Config{
			Write: cfg.WriteConfig{
				GlobalMaxBlocks: 1,
			},
			Read: cfg.ReadConfig{
				EnableBufferedRead: params.enableBufferedRead,
				GlobalMaxBlocks:    1,
				BlockSizeMb:        1,
				MaxBlocksPerHandle: 10,
			},
			EnableNewReader: true,
			Metrics: cfg.MetricsConfig{
				ExperimentalEnableGrpcMetrics: true,
			},
		},
		MetricHandle:  mh,
		TraceHandle:   tracing.NewNoopTracer(),
		CacheClock:    &timeutil.SimulatedClock{},
		BucketName:    bucketName,
		BucketManager: bm,
	}

	if params.enableFileCache || params.enableSparseFileCache {
		cacheDir := t.TempDir()
		serverCfg.NewConfig.CacheDir = cfg.ResolvedPath(cacheDir)
		serverCfg.NewConfig.FileCache = cfg.FileCacheConfig{
			MaxSizeMb:                              100,
			CacheFileForRangeRead:                  true,
			ExperimentalEnableChunkCache:           params.enableSparseFileCache,
			DownloadChunkSizeMb:                    1,
			EnableParallelDownloads:                params.enableParallelDownloads,
			ExperimentalParallelDownloadsDefaultOn: params.enableParallelDownloadsBlocking,
			ParallelDownloadsPerFile:               16,
		}
	}
	if serverCfg.NewConfig.MetadataCache.TtlSecs == 0 {
		serverCfg.NewConfig.MetadataCache.TtlSecs = 60
	}

	server, err := fs.NewFileSystem(ctx, serverCfg)
	require.NoError(t, err, "NewFileSystem")
	return sh, server, mh, reader
}

func TestGrpcMetrics_LookUpInode(t *testing.T) {
	ctx := context.Background()
	params := defaultServerConfigParams()

	_, server, mh, reader := createTestFileSystemWithGrpcMetrics(ctx, t, params)
	server = wrappers.WithMonitoring(server, mh)

	fileName := "test.txt"

	lookupOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}

	// HACK: Every gRPC-enabled mount in GCSFuse performs a synchronous
	// "DirectPath connectivity check" during the first few operations on a bucket.
	// In non-GCP environments (like local development), this library check
	// hangs for exactly 60 seconds before failing and falling back to standard gRPC.
	//
	// We perform a dummy LookUpInode with a 100ms timeout here to "poke" this check
	// and allow it to fail early. This prevents the initial test operation from
	// appearing to hang indefinitely in an IDE, although subsequent real calls
	// will still wait for the library's internal 60s timeout to expire naturally.
	//
	// This approach is used to keep the production codebase (storage_handle.go)
	// completely pristine while still allowing the tests to eventually pass.
	shortCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	_ = server.LookUpInode(shortCtx, lookupOp)
	cancel()

	// Now run the real call. It should be fast because DirectPath check (even if hanging)
	// won't block the actual gRPC calls once the client is initialized.
	err := server.LookUpInode(ctx, lookupOp)
	require.NoError(t, err)

	waitForMetricsProcessing()
	time.Sleep(501 * time.Millisecond)

	// Verify that grpc.client.attempt.started was emitted.
	metrics.VerifyCounterMetric(t, ctx, reader, "grpc.client.attempt.started",
		attribute.NewSet(attribute.String("grpc.method", "google.storage.v2.Storage/GetObject")),
		1, metrics.AtLeast(), metrics.Subset())
	metrics.VerifyHistogramMetric(t, ctx, reader, "grpc.client.call.duration",
		attribute.NewSet(attribute.String("grpc.method", "google.storage.v2.Storage/GetObject")),
		1, metrics.AtLeast(), metrics.Subset())
}

func TestGrpcMetrics_ReadFile(t *testing.T) {
	ctx := context.Background()
	params := defaultServerConfigParams()

	_, server, mh, reader := createTestFileSystemWithGrpcMetrics(ctx, t, params)
	server = wrappers.WithMonitoring(server, mh)

	fileName := "test.txt"

	lookupOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
	}
	// Bypass the DirectPath hang (see explanation in TestGrpcMetrics_LookUpInode).
	shortCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	_ = server.LookUpInode(shortCtx, lookupOp)
	cancel()
	err := server.LookUpInode(ctx, lookupOp)
	require.NoError(t, err)

	openOp := &fuseops.OpenFileOp{
		Inode: lookupOp.Entry.Child,
	}
	err = server.OpenFile(ctx, openOp)
	require.NoError(t, err)

	readOp := &fuseops.ReadFileOp{
		Inode:  lookupOp.Entry.Child,
		Handle: openOp.Handle,
		Offset: 0,
		Dst:    make([]byte, 12),
	}
	err = server.ReadFile(ctx, readOp)
	require.NoError(t, err)
	waitForMetricsProcessing()
	time.Sleep(501 * time.Millisecond)

	// Verify ReadObject metric.
	metrics.VerifyCounterMetric(t, ctx, reader, "grpc.client.attempt.started",
		attribute.NewSet(attribute.String("grpc.method", "google.storage.v2.Storage/ReadObject")),
		1, metrics.AtLeast(), metrics.Subset())
	metrics.VerifyHistogramMetric(t, ctx, reader, "grpc.client.call.duration",
		attribute.NewSet(attribute.String("grpc.method", "google.storage.v2.Storage/ReadObject")),
		1, metrics.AtLeast(), metrics.Subset())
}

func TestGrpcMetrics_CreateFile(t *testing.T) {
	ctx := context.Background()
	params := defaultServerConfigParams()

	_, server, mh, reader := createTestFileSystemWithGrpcMetrics(ctx, t, params)
	server = wrappers.WithMonitoring(server, mh)

	fileName := "new_file.txt"

	createOp := &fuseops.CreateFileOp{
		Parent: fuseops.RootInodeID,
		Name:   fileName,
		Mode:   0644,
	}
	// Bypass the DirectPath hang (see explanation in TestGrpcMetrics_LookUpInode).
	shortCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	_ = server.CreateFile(shortCtx, createOp)
	cancel()
	err := server.CreateFile(ctx, createOp)
	require.NoError(t, err)

	syncOp := &fuseops.SyncFileOp{
		Inode:  createOp.Entry.Child,
		Handle: createOp.Handle,
	}
	// SyncFile might fail if BidiWriteObject is unimplemented, but we check metrics.
	_ = server.SyncFile(ctx, syncOp)

	waitForMetricsProcessing()
	time.Sleep(501 * time.Millisecond)

	// Verify BidiWriteObject metric.
	metrics.VerifyCounterMetric(t, ctx, reader, "grpc.client.attempt.started",
		attribute.NewSet(attribute.String("grpc.method", "google.storage.v2.Storage/BidiWriteObject")),
		1, metrics.AtLeast(), metrics.Subset())
	metrics.VerifyHistogramMetric(t, ctx, reader, "grpc.client.call.duration",
		attribute.NewSet(attribute.String("grpc.method", "google.storage.v2.Storage/BidiWriteObject")),
		1, metrics.AtLeast(), metrics.Subset())
}
