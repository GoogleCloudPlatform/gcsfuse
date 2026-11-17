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
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v4/cfg"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/fs/wrappers"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v4/metrics"
	"github.com/googlecloudplatform/gcsfuse/v4/tracing"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
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

type storageProtoDescriptorSet struct {
	object                  protoreflect.MessageDescriptor
	getObjectReq            protoreflect.MessageDescriptor
	listObjectsReq          protoreflect.MessageDescriptor
	listObjectsResp         protoreflect.MessageDescriptor
	startResumableWriteReq  protoreflect.MessageDescriptor
	startResumableWriteResp protoreflect.MessageDescriptor
	readObjectReq           protoreflect.MessageDescriptor
	readObjectResp          protoreflect.MessageDescriptor
	bidiWriteObjectReq      protoreflect.MessageDescriptor
	bidiWriteObjectResp     protoreflect.MessageDescriptor
}

var storageProtoDescs = sync.OnceValue(func() *storageProtoDescriptorSet {
	mustFind := func(name protoreflect.FullName) protoreflect.MessageDescriptor {
		mt, err := protoregistry.GlobalTypes.FindMessageByName(name)
		if err != nil {
			panic(fmt.Sprintf("failed to resolve protobuf descriptor %s: %v", name, err))
		}
		return mt.Descriptor()
	}
	return &storageProtoDescriptorSet{
		object:                  mustFind("google.storage.v2.Object"),
		getObjectReq:            mustFind("google.storage.v2.GetObjectRequest"),
		listObjectsReq:          mustFind("google.storage.v2.ListObjectsRequest"),
		listObjectsResp:         mustFind("google.storage.v2.ListObjectsResponse"),
		startResumableWriteReq:  mustFind("google.storage.v2.StartResumableWriteRequest"),
		startResumableWriteResp: mustFind("google.storage.v2.StartResumableWriteResponse"),
		readObjectReq:           mustFind("google.storage.v2.ReadObjectRequest"),
		readObjectResp:          mustFind("google.storage.v2.ReadObjectResponse"),
		bidiWriteObjectReq:      mustFind("google.storage.v2.BidiWriteObjectRequest"),
		bidiWriteObjectResp:     mustFind("google.storage.v2.BidiWriteObjectResponse"),
	}
})

// reflectFakeServer implements google.storage.v2.Storage using dynamicpb to
// avoid importing the conflicting storagepb package.
type reflectFakeServer struct{}

func (s *reflectFakeServer) GetObject(_ context.Context, req any) (any, error) {
	dm := req.(*dynamicpb.Message)
	objectName := dm.Get(dm.Descriptor().Fields().ByName("object")).String()
	bucketName := dm.Get(dm.Descriptor().Fields().ByName("bucket")).String()

	if strings.HasSuffix(objectName, "/") || strings.Contains(objectName, "nonexistent") {
		return nil, status.Error(codes.NotFound, "not found")
	}

	objDesc := storageProtoDescs().object
	obj := dynamicpb.NewMessage(objDesc)
	obj.Set(objDesc.Fields().ByName("name"), protoreflect.ValueOf(objectName))
	obj.Set(objDesc.Fields().ByName("bucket"), protoreflect.ValueOf(bucketName))
	obj.Set(objDesc.Fields().ByName("size"), protoreflect.ValueOf(int64(12)))
	return obj, nil
}

func (s *reflectFakeServer) ListObjects(_ context.Context, _ any) (any, error) {
	return dynamicpb.NewMessage(storageProtoDescs().listObjectsResp), nil
}

func (s *reflectFakeServer) StartResumableWrite(_ context.Context, _ any) (any, error) {
	respDesc := storageProtoDescs().startResumableWriteResp
	resp := dynamicpb.NewMessage(respDesc)
	resp.Set(respDesc.Fields().ByName("upload_id"), protoreflect.ValueOf("upload-id"))
	return resp, nil
}

func (s *reflectFakeServer) ReadObject(req any, stream grpc.ServerStream) error {
	dm := req.(*dynamicpb.Message)
	objectName := dm.Get(dm.Descriptor().Fields().ByName("object")).String()
	return sendReadObjectChunk(stream, objectName, []byte("test content"))
}

// sendReadObjectChunk always populates metadata (field 4) after checksummed_data
// (field 1) so checksummed_data.content is never the final byte of the wire
// frame—avoiding cloud.google.com/go/storage's zero-copy readObjectResponseDecoder
// end-of-buffer slice bounds bug.
func sendReadObjectChunk(stream grpc.ServerStream, objectName string, chunk []byte) error {
	respDesc := storageProtoDescs().readObjectResp
	resp := dynamicpb.NewMessage(respDesc)

	dataField := respDesc.Fields().ByName("checksummed_data")
	dataMsg := dynamicpb.NewMessage(dataField.Message())
	dataMsg.Set(dataField.Message().Fields().ByName("content"), protoreflect.ValueOf(chunk))
	resp.Set(dataField, protoreflect.ValueOfMessage(dataMsg))

	metaField := respDesc.Fields().ByName("metadata")
	metaMsg := dynamicpb.NewMessage(metaField.Message())
	metaMsg.Set(metaField.Message().Fields().ByName("name"), protoreflect.ValueOf(objectName))
	metaMsg.Set(metaField.Message().Fields().ByName("size"), protoreflect.ValueOf(int64(12)))
	resp.Set(metaField, protoreflect.ValueOfMessage(metaMsg))

	return stream.SendMsg(resp)
}

func (s *reflectFakeServer) BidiWriteObject(stream grpc.ServerStream) error {
	descs := storageProtoDescs()
	req := dynamicpb.NewMessage(descs.bidiWriteObjectReq)
	finishField := descs.bidiWriteObjectReq.Fields().ByName("finish_write")
	flushField := descs.bidiWriteObjectReq.Fields().ByName("flush")

	var (
		objectName = "new_file.txt"
		bucketName = "projects/_/buckets/test-bucket"
		totalSize  int64
	)

	for {
		req.Reset()
		err := stream.RecvMsg(req)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		totalSize += extractBidiWriteRequestFields(req, &objectName, &bucketName)

		finishWrite := finishField != nil && req.Get(finishField).Bool()
		flush := flushField != nil && req.Get(flushField).Bool()

		if finishWrite {
			resp := dynamicpb.NewMessage(descs.bidiWriteObjectResp)
			obj := dynamicpb.NewMessage(descs.object)
			obj.Set(descs.object.Fields().ByName("name"), protoreflect.ValueOf(objectName))
			obj.Set(descs.object.Fields().ByName("bucket"), protoreflect.ValueOf(bucketName))
			obj.Set(descs.object.Fields().ByName("size"), protoreflect.ValueOf(totalSize))
			resp.Set(descs.bidiWriteObjectResp.Fields().ByName("resource"), protoreflect.ValueOfMessage(obj))
			return stream.SendMsg(resp)
		}
		if flush {
			resp := dynamicpb.NewMessage(descs.bidiWriteObjectResp)
			resp.Set(descs.bidiWriteObjectResp.Fields().ByName("persisted_size"), protoreflect.ValueOf(totalSize))
			if err := stream.SendMsg(resp); err != nil {
				return err
			}
		}
	}
}

func extractBidiWriteRequestFields(req *dynamicpb.Message, objectName, bucketName *string) int64 {
	if specField := req.Descriptor().Fields().ByName("write_object_spec"); specField != nil && req.Has(specField) {
		spec := req.Get(specField).Message()
		if resField := spec.Descriptor().Fields().ByName("resource"); resField != nil && spec.Has(resField) {
			res := spec.Get(resField).Message()
			if nameField := res.Descriptor().Fields().ByName("name"); nameField != nil && res.Has(nameField) {
				*objectName = res.Get(nameField).String()
			}
			if bktField := res.Descriptor().Fields().ByName("bucket"); bktField != nil && res.Has(bktField) {
				*bucketName = res.Get(bktField).String()
			}
		}
	}
	if dataField := req.Descriptor().Fields().ByName("checksummed_data"); dataField != nil && req.Has(dataField) {
		dataMsg := req.Get(dataField).Message()
		if contentField := dataMsg.Descriptor().Fields().ByName("content"); contentField != nil && dataMsg.Has(contentField) {
			return int64(len(dataMsg.Get(contentField).Bytes()))
		}
	}
	return 0
}

func makeDynamicUnaryHandler(reqDesc protoreflect.MessageDescriptor, fullMethod string, invoke func(*reflectFakeServer, context.Context, any) (any, error)) grpc.MethodHandler {
	return func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
		in := dynamicpb.NewMessage(reqDesc)
		if err := dec(in); err != nil {
			return nil, err
		}
		if interceptor == nil {
			return invoke(srv.(*reflectFakeServer), ctx, in)
		}
		return interceptor(ctx, in, &grpc.UnaryServerInfo{Server: srv, FullMethod: fullMethod}, func(ctx context.Context, req any) (any, error) {
			return invoke(srv.(*reflectFakeServer), ctx, req)
		})
	}
}

func registerFakeStorageServer(s *grpc.Server, srv *reflectFakeServer) {
	descs := storageProtoDescs()
	desc := &grpc.ServiceDesc{
		ServiceName: "google.storage.v2.Storage",
		HandlerType: (*any)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "GetObject",
				Handler:    makeDynamicUnaryHandler(descs.getObjectReq, "/google.storage.v2.Storage/GetObject", (*reflectFakeServer).GetObject),
			},
			{
				MethodName: "ListObjects",
				Handler:    makeDynamicUnaryHandler(descs.listObjectsReq, "/google.storage.v2.Storage/ListObjects", (*reflectFakeServer).ListObjects),
			},
			{
				MethodName: "StartResumableWrite",
				Handler:    makeDynamicUnaryHandler(descs.startResumableWriteReq, "/google.storage.v2.Storage/StartResumableWrite", (*reflectFakeServer).StartResumableWrite),
			},
		},
		Streams: []grpc.StreamDesc{
			{
				StreamName: "ReadObject",
				Handler: func(srv any, stream grpc.ServerStream) error {
					m := dynamicpb.NewMessage(descs.readObjectReq)
					if err := stream.RecvMsg(m); err != nil {
						return err
					}
					return srv.(*reflectFakeServer).ReadObject(m, stream)
				},
				ServerStreams: true,
			},
			{
				StreamName: "BidiWriteObject",
				Handler: func(srv any, stream grpc.ServerStream) error {
					return srv.(*reflectFakeServer).BidiWriteObject(stream)
				},
				ServerStreams: true,
				ClientStreams: true,
			},
		},
		Metadata: "google/storage/v2/storage.proto",
	}
	s.RegisterService(desc, srv)
}

// startFakeGCEMetadataServer starts an in-process HTTP server emulating GCE
// metadata endpoints for OTel GCP resource detection.
func startFakeGCEMetadataServer(t *testing.T) {
	t.Helper()
	metadataLis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Metadata-Flavor", "Google")
		switch {
		case strings.HasPrefix(r.URL.Path, "/computeMetadata/v1/project/project-id"):
			_, _ = w.Write([]byte("test-project"))
		case strings.HasPrefix(r.URL.Path, "/computeMetadata/v1/instance/zone"):
			_, _ = w.Write([]byte("projects/123456/zones/us-central1-a"))
		case strings.HasPrefix(r.URL.Path, "/computeMetadata/v1/instance/id"):
			_, _ = w.Write([]byte("1234567890"))
		case strings.HasPrefix(r.URL.Path, "/computeMetadata/v1/instance/name"):
			_, _ = w.Write([]byte("test-instance"))
		case strings.HasPrefix(r.URL.Path, "/computeMetadata/v1/instance/machine-type"):
			_, _ = w.Write([]byte("projects/123456/machineTypes/n2-standard-4"))
		case strings.HasPrefix(r.URL.Path, "/computeMetadata/v1/universe/universe-domain"):
			_, _ = w.Write([]byte("googleapis.com"))
		default:
			w.WriteHeader(http.StatusOK)
		}
	})
	metadataServer := &http.Server{Handler: mux}
	go func() {
		_ = metadataServer.Serve(metadataLis)
	}()
	t.Cleanup(func() {
		_ = metadataServer.Close()
	})
	t.Setenv("GCE_METADATA_HOST", metadataLis.Addr().String())
}

// setupOTelManualReader installs a view-free OpenTelemetry MeterProvider backed
// by a ManualReader for the test duration.
func setupOTelManualReader(ctx context.Context, t *testing.T) (metrics.MetricHandle, *metric.ManualReader) {
	t.Helper()
	origProvider := otel.GetMeterProvider()
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
		otel.SetMeterProvider(origProvider)
	})

	mh, err := metrics.NewOTelMetrics(ctx, 1, 100)
	require.NoError(t, err, "metrics.NewOTelMetrics")
	t.Cleanup(func() {
		mh.Close()
	})
	return mh, reader
}

type grpcMetricsTestFixture struct {
	server fuseutil.FileSystem
	reader *metric.ManualReader
}

func setupGRPCMetricsFixture(ctx context.Context, t *testing.T) *grpcMetricsTestFixture {
	t.Helper()

	t.Setenv("GOLANG_PROTOBUF_REGISTRATION_CONFLICT", "ignore")
	t.Setenv("GOOGLE_CLOUD_PROJECT", "test-project")
	t.Setenv("GOOGLE_CLOUD_DISABLE_DIRECT_PATH", "true")
	t.Setenv("GCP_STORAGE_GO_ENABLE_OTEL_METRICS", "false")

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	registerFakeStorageServer(grpcServer, &reflectFakeServer{})
	go func() {
		_ = grpcServer.Serve(lis)
	}()
	t.Cleanup(grpcServer.Stop)

	startFakeGCEMetadataServer(t)
	mh, reader := setupOTelManualReader(ctx, t)

	clientConfig := storageutil.StorageClientConfig{
		ClientProtocol:    cfg.GRPC,
		CustomEndpoint:    lis.Addr().String(),
		EnableGrpcMetrics: true,
		IsGKE:             true,
		AnonymousAccess:   true,
		GrpcConnPoolSize:  1,
		MaxRetryAttempts:  3,
		RetryMultiplier:   2,
		MaxRetrySleep:     5 * time.Millisecond,
		MetricHandle:      mh,
		WriteConfig:       &cfg.WriteConfig{},
	}

	sh, err := storage.NewStorageHandle(ctx, clientConfig, "")
	require.NoError(t, err)

	bm := gcsx.NewBucketManager(gcsx.BucketConfig{
		StatCacheMaxSizeMB:       32,
		StatCacheTTL:             time.Minute,
		NegativeStatCacheTTL:     time.Minute,
		ChunkRetryDeadlineSecs:   120,
		ChunkTransferTimeoutSecs: 10,
		TmpObjectPrefix:          ".gcsfuse_tmp/",
	}, sh, mh)

	server, err := fs.NewFileSystem(ctx, &fs.ServerConfig{
		SequentialReadSizeMb: 200,
		NewConfig: &cfg.Config{
			Write: cfg.WriteConfig{
				BlockSizeMb:      1,
				MaxBlocksPerFile: 10,
				GlobalMaxBlocks:  4,
			},
			Read: cfg.ReadConfig{
				BlockSizeMb:        1,
				MaxBlocksPerHandle: 10,
				GlobalMaxBlocks:    1,
			},
			GcsRetries: cfg.GcsRetriesConfig{
				ChunkRetryDeadlineSecs:   120,
				ChunkTransferTimeoutSecs: 10,
			},
			MetadataCache: cfg.MetadataCacheConfig{
				TtlSecs: 60,
			},
			EnableNewReader: true,
			Metrics: cfg.MetricsConfig{
				ExperimentalEnableGrpcMetrics: true,
			},
		},
		MetricHandle:  mh,
		TraceHandle:   tracing.NewNoopTracer(),
		CacheClock:    &timeutil.SimulatedClock{},
		BucketName:    "test-bucket",
		BucketManager: bm,
	})
	require.NoError(t, err, "NewFileSystem")
	t.Cleanup(server.Destroy)

	return &grpcMetricsTestFixture{
		server: wrappers.WithMonitoring(server, mh),
		reader: reader,
	}
}

func TestGrpcMetrics_LookUpInode(t *testing.T) {
	ctx := t.Context()
	f := setupGRPCMetricsFixture(ctx, t)

	require.NoError(t, f.server.LookUpInode(ctx, &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   "test.txt",
	}))

	rmOpt := metrics.WithResourceMetrics(metrics.CollectResourceMetrics(t, ctx, f.reader))
	metrics.VerifyCounterMetric(t, ctx, nil, "grpc.client.attempt.started",
		attribute.NewSet(attribute.String("grpc.method", "google.storage.v2.Storage/GetObject")),
		1, rmOpt, metrics.AtLeast(), metrics.Subset())
	metrics.VerifyHistogramMetric(t, ctx, nil, "grpc.client.call.duration",
		attribute.NewSet(attribute.String("grpc.method", "google.storage.v2.Storage/GetObject")),
		1, rmOpt, metrics.AtLeast(), metrics.Subset())
}

func TestGrpcMetrics_ReadFile(t *testing.T) {
	ctx := t.Context()
	f := setupGRPCMetricsFixture(ctx, t)

	lookupOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   "test.txt",
	}
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
	require.NoError(t, f.server.ReleaseFileHandle(ctx, &fuseops.ReleaseFileHandleOp{Handle: openOp.Handle}))

	// Reader.Close cancels the ReadObject stream context; grpc-go's stream
	// context-watcher goroutine records grpc.client.call.duration asynchronously
	// via OnFinish once <-ctx.Done() is scheduled.
	readAttrs := attribute.NewSet(attribute.String("grpc.method", "google.storage.v2.Storage/ReadObject"))
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		rmOpt := metrics.WithResourceMetrics(metrics.CollectResourceMetrics(c, ctx, f.reader))
		metrics.VerifyCounterMetric(c, ctx, nil, "grpc.client.attempt.started", readAttrs, 1, rmOpt, metrics.AtLeast(), metrics.Subset())
		metrics.VerifyHistogramMetric(c, ctx, nil, "grpc.client.call.duration", readAttrs, 1, rmOpt, metrics.AtLeast(), metrics.Subset())
	}, time.Second, time.Millisecond)
}

func TestGrpcMetrics_CreateFile(t *testing.T) {
	ctx := t.Context()
	f := setupGRPCMetricsFixture(ctx, t)

	createOp := &fuseops.CreateFileOp{
		Parent: fuseops.RootInodeID,
		Name:   "new_file.txt",
		Mode:   0644,
	}
	require.NoError(t, f.server.CreateFile(ctx, createOp))
	require.NoError(t, f.server.SyncFile(ctx, &fuseops.SyncFileOp{
		Inode:  createOp.Entry.Child,
		Handle: createOp.Handle,
	}))
	require.NoError(t, f.server.ReleaseFileHandle(ctx, &fuseops.ReleaseFileHandleOp{Handle: createOp.Handle}))

	rmOpt := metrics.WithResourceMetrics(metrics.CollectResourceMetrics(t, ctx, f.reader))
	metrics.VerifyCounterMetric(t, ctx, nil, "grpc.client.attempt.started",
		attribute.NewSet(attribute.String("grpc.method", "google.storage.v2.Storage/BidiWriteObject")),
		1, rmOpt, metrics.AtLeast(), metrics.Subset())
	metrics.VerifyHistogramMetric(t, ctx, nil, "grpc.client.call.duration",
		attribute.NewSet(attribute.String("grpc.method", "google.storage.v2.Storage/BidiWriteObject")),
		1, rmOpt, metrics.AtLeast(), metrics.Subset())
}
