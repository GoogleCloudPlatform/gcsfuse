# Handoff Report - Adversarial Validation and Stress Testing of Read-Locking Optimization

## 1. Observation
- Target Files under review: `internal/fs/fs.go` and `internal/fs/inode/lookup_count.go`.
- Patches under review (provided by `reviewer_m3_1`): `fs.diff` and `lookup_count.diff` (which implement directory read-locking lookup optimizations).
- Lock hierarchy in gcsfuse:
  - Inode lock (`in.Lock()` / `in.RLock()`) must be acquired first.
  - Filesystem lock (`fs.mu.Lock()`) must be acquired second.
- In `internal/fs/fs.go`, inside the lock downgrade flow of `lookUpOrCreateInodeIfNotStale` (introduced in the optimized patch at lines 163-170 and 180-184):
  ```go
  165:						existingInode.Unlock()
  166:						existingInode.(locker.RWLocker).RLock()
  ```
  and
  ```go
  182:				existingInode.Unlock()
  ...
  183:				existingInode.(locker.RWLocker).RLock()
  ```
  These calls release the exclusive write lock and acquire a read lock (`RLock()`) on the inode while holding `fs.mu` (which is acquired at line 152 and not released).
- Direct call to `RmDir` in `fs.go` (lines 2438 and 2473):
  - Line 2438: locks `child` directory inode exclusively (`child, readLocked, err := fs.lookUpOrCreateChildInode(ctx, parent, op.Name, false)`).
  - Line 2473: acquires `fs.mu.Lock()` while holding `child.Lock()` exclusively.
- Executed `go test -v -count=1 -run "TestReadLockingUpgradeDeadlock" ./internal/fs/...` using a mock bucket (which overrides `StatObject` to trigger a remote size change on directory object `target/` and intercepts deletes as noops).
- The test timed out and panicked:
  ```
  --- FAIL: TestReadLockingUpgradeDeadlock (4.02s)
  panic: DEADLOCK DETECTED! Test hung and timed out after 3 seconds.
  ```
- Stack trace from the deadlock shows:
  ```
  goroutine 148 [sync.RWMutex.RLock, 10 minutes]:
  sync.runtime_SemacquireRWMutexR(...)
  sync.(*RWMutex).RLock(...)
  github.com/googlecloudplatform/gcsfuse/v3/internal/locker.(*rwDebugger).RLock(...)
  github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode.(*dirInode).RLock(...)
  github.com/googlecloudplatform/gcsfuse/v3/internal/fs.(*fileSystem).lookUpOrCreateInodeIfNotStale(...)
  	/usr/local/google/home/kislayk/gitproj/gcsfuse/internal/fs/fs.go:1236
  ```
  which is blocked on `RLock()` while downgrading.
  Other goroutines are blocked trying to acquire `in.Lock()` or `fs.mu.Lock()` concurrently:
  ```
  goroutine 183 [sync.Mutex.Lock, 10 minutes]:
  ...
  github.com/googlecloudplatform/gcsfuse/v3/internal/fs.(*fileSystem).lookUpOrCreateChildInode(...)
  	/usr/local/google/home/kislayk/gitproj/gcsfuse/internal/fs/fs.go:1327
  github.com/googlecloudplatform/gcsfuse/v3/internal/fs.(*fileSystem).RmDir(...)
  	/usr/local/google/home/kislayk/gitproj/gcsfuse/internal/fs/fs.go:2438
  ```

## 2. Logic Chain
- Standard lock hierarchy requires: first lock the inode, then lock the filesystem (`fs.mu`).
- When downgrading the lock from write to read in `lookUpOrCreateInodeIfNotStale`, the code unlocks the write lock and then immediately calls `RLock()` (read-lock) on the inode while still holding `fs.mu` locked.
- If Thread A is downgrading, it holds `fs.mu` and releases `existingInode` write lock.
- Before Thread A can acquire `existingInode.RLock()`, Thread B (e.g. running `RmDir` or `Rename` on the same directory) acquires `existingInode.Lock()` exclusively (write lock).
- Thread A is now blocked on `existingInode.RLock()` waiting for Thread B to release the inode lock.
- Thread B then attempts to acquire `fs.mu.Lock()` (e.g. at line 2473 in `RmDir` or during inode disposal).
- Thread B is now blocked on `fs.mu.Lock()` waiting for Thread A to release `fs.mu`.
- This creates a circular dependency: Thread A holds `fs.mu` and waits for `existingInode` lock. Thread B holds `existingInode` lock and waits for `fs.mu`.
- This causes a permanent deadlock that hangs FUSE directory lookup and directory mutation operations.

## 3. Caveats
- Integration and mount tests were not run due to constraint.
- The external domain skill `SKILL.md` was not loaded due to LOAS key permission constraints in the workspace.

## 4. Conclusion
Adversarial stress testing has empirically validated the existence of a critical circular deadlock risk during lock downgrade operations in the proposed read-locking optimization. The verdict is a definitive **REQUEST_CHANGES** (block implementation until lock downgrading is redesigned without lock ordering violations).

## 5. Verification Method
To reproduce the deadlock:
1. Apply the patches:
   ```bash
   git apply .agents/reviewer_m3_1/fs.diff
   git apply .agents/reviewer_m3_1/lookup_count.diff
   ```
2. Save the deadlock test code below to `internal/fs/read_locking_deadlock_test.go`:
   <details>
   <summary>Click to view read_locking_deadlock_test.go code</summary>

   ```go
   package fs_test

   import (
   	"context"
   	"fmt"
   	"sync"
   	"testing"
   	"time"

   	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
   	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs"
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

   	err = storageutil.CreateObjects(ctx, fakeB, map[string][]byte{"target/": []byte("")})
   	require.NoError(t, err)

   	lookupOp := &fuseops.LookUpInodeOp{
   		Parent: fuseops.RootInodeID,
   		Name:   "target",
   	}
   	err = server.LookUpInode(ctx, lookupOp)
   	require.NoError(t, err)

   	stop := make(chan struct{})
   	var wg sync.WaitGroup

   	for range 20 {
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
   				}
   			}
   		}()
   	}

   	for range 20 {
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
   	case <-time.After(3 * time.Second):
   		panic("DEADLOCK DETECTED! Test hung and timed out after 3 seconds.")
   	}
   }
   ```
   </details>
3. Run the test:
   ```bash
   go test -v -count=1 -run "TestReadLockingUpgradeDeadlock" ./internal/fs/...
   ```
   Verify that it exits with status 1 and prints the watchdog panic: `"DEADLOCK DETECTED! Test hung and timed out after 3 seconds."`
