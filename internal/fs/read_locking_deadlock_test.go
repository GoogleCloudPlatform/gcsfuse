package fs_test

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/require"
)

type mockBucket struct {
	gcs.Bucket
	statObjectFunc   func(ctx context.Context, req *gcs.StatObjectRequest) (*gcs.MinObject, *gcs.ExtendedObjectAttributes, error)
	deleteObjectFunc func(ctx context.Context, req *gcs.DeleteObjectRequest) error
	deleteFolderFunc func(ctx context.Context, folderName string) error
}

func (mb *mockBucket) StatObject(ctx context.Context, req *gcs.StatObjectRequest) (*gcs.MinObject, *gcs.ExtendedObjectAttributes, error) {
	if mb.statObjectFunc != nil {
		return mb.statObjectFunc(ctx, req)
	}
	return mb.Bucket.StatObject(ctx, req)
}

func (mb *mockBucket) DeleteObject(ctx context.Context, req *gcs.DeleteObjectRequest) error {
	if mb.deleteObjectFunc != nil {
		return mb.deleteObjectFunc(ctx, req)
	}
	return mb.Bucket.DeleteObject(ctx, req)
}

func (mb *mockBucket) DeleteFolder(ctx context.Context, folderName string) error {
	if mb.deleteFolderFunc != nil {
		return mb.deleteFolderFunc(ctx, folderName)
	}
	return mb.Bucket.DeleteFolder(ctx, folderName)
}

func TestReadLockingUpgradeDeadlock(t *testing.T) {
	err := logger.InitLogFile(cfg.LoggingConfig{Severity: "ERROR"}, "test-mount")
	require.NoError(t, err)

	ctx := context.Background()
	bucketName := "test-bucket"
	bucketType := gcs.BucketType{Zonal: true}
	fakeB := fake.NewFakeBucket(timeutil.RealClock(), bucketName, bucketType)

	var mu sync.Mutex
	objectSize := uint64(0)
	generation := int64(1)

	mb := &mockBucket{
		Bucket: fakeB,
		statObjectFunc: func(ctx context.Context, req *gcs.StatObjectRequest) (*gcs.MinObject, *gcs.ExtendedObjectAttributes, error) {
			minObj, attrs, err := fakeB.StatObject(ctx, req)
			if err != nil {
				return nil, nil, err
			}
			mu.Lock()
			sz := objectSize
			gen := generation
			mu.Unlock()

			minObj.Size = sz
			minObj.Generation = gen
			return minObj, attrs, nil
		},
		deleteObjectFunc: func(ctx context.Context, req *gcs.DeleteObjectRequest) error {
			return nil
		},
		deleteFolderFunc: func(ctx context.Context, folderName string) error {
			return nil
		},
	}

	serverCfg := &fs.ServerConfig{
		NewConfig: &cfg.Config{
			Write: cfg.WriteConfig{
				GlobalMaxBlocks:       1,
				EnableStreamingWrites: false,
				BlockSizeMb:           1,
				MaxBlocksPerFile:      10,
			},
			Read: cfg.ReadConfig{
				EnableBufferedRead: false,
				GlobalMaxBlocks:    1,
				BlockSizeMb:        1,
				MaxBlocksPerHandle: 10,
			},
			EnableNewReader: true,
			FileSystem: cfg.FileSystemConfig{
				EnableKernelReader: false,
			},
		},
		MetricHandle: metrics.NewNoopMetrics(),
		TraceHandle:  tracing.NewNoopTracer(),
		CacheClock:   &timeutil.SimulatedClock{},
		BucketName:   bucketName,
		BucketManager: &fakeBucketManager{
			buckets: map[string]gcs.Bucket{
				bucketName: mb,
			},
		},
		SequentialReadSizeMb: 200,
	}

	server, err := fs.NewFileSystem(ctx, serverCfg)
	require.NoError(t, err)

	err = storageutil.CreateObjects(ctx, fakeB, map[string][]byte{"target": []byte("hello")})
	require.NoError(t, err)

	lookupOp := &fuseops.LookUpInodeOp{
		Parent: fuseops.RootInodeID,
		Name:   "target",
	}
	err = server.LookUpInode(ctx, lookupOp)
	require.NoError(t, err)

	stop := make(chan struct{})
	var wg sync.WaitGroup

	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					op := &fuseops.LookUpInodeOp{
						Parent: fuseops.RootInodeID,
						Name:   "target",
					}
					_ = server.LookUpInode(ctx, op)
					time.Sleep(10 * time.Microsecond)
				}
			}
		}()
	}

	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					rmdirOp := &fuseops.RmDirOp{
						Parent: fuseops.RootInodeID,
						Name:   "target",
					}
					_ = server.RmDir(ctx, rmdirOp)
					time.Sleep(10 * time.Microsecond)
				}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				mu.Lock()
				objectSize += 5
				mu.Unlock()
				time.Sleep(100 * time.Microsecond)
			}
		}
	}()

	time.Sleep(1 * time.Second)
	close(stop)

	wgChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(wgChan)
	}()

	select {
	case <-wgChan:
	case <-time.After(10 * time.Second):
		buf := make([]byte, 1<<20)
		stacklen := runtime.Stack(buf, true)
		fmt.Printf("=== DEADLOCK STACKS ===\n%s\n", buf[:stacklen])
		panic("DEADLOCK DETECTED! Test hung and timed out after 10 seconds.")
	}
}
