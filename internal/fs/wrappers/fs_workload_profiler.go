package wrappers

import (
	"context"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workloadprofiler"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"sync"
)

// OperationStats holds statistics for a specific file system operation.
type OperationStats struct {
	TotalCount  int64
	Parallelism int64
}

// ProfilerWrapper wraps an Afero filesystem and collects operation statistics.
type ProfilerWrapper struct {
	workloadprofiler.ProfilerSource
	wrapped fuseutil.FileSystem
	Stats   map[string]OperationStats
	temp    map[string]OperationStats // Temporary stats for parallelism tracking
	mu      sync.RWMutex
}

// NewFsWrapper creates a new ProfilerWrapper with the given Afero filesystem.
func NewFsWrapper(fs fuseutil.FileSystem) *ProfilerWrapper {
	pw := &ProfilerWrapper{
		wrapped: fs,
		Stats:   make(map[string]OperationStats),
		temp:    make(map[string]OperationStats),
	}

	workloadprofiler.AddProfilerSource(pw) // Register this wrapper as a profiler source
	return pw
}

// WithMonitoring takes a FileSystem, returns a FileSystem with monitoring
// on the counts of requests per API.
func WithProfilerWrapper(fs fuseutil.FileSystem) fuseutil.FileSystem {
	return NewFsWrapper(fs)
}

func (fs *ProfilerWrapper) incrementCounter(op string) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	stats := fs.Stats[op]
	stats.TotalCount++
	fs.Stats[op] = stats
}

func (fs *ProfilerWrapper) updateParallelism(op string, increment bool) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	tmp := fs.temp[op]
	if increment {
		tmp.Parallelism++
	} else {
		tmp.Parallelism--
	}
	fs.temp[op] = tmp
	if tmp.Parallelism > fs.Stats[op].Parallelism {
		ss := fs.Stats[op]
		ss.Parallelism = tmp.Parallelism
		fs.Stats[op] = ss
	}
}

func (fs *ProfilerWrapper) invokeWrapped(ctx context.Context, opName string, w func(ctx context.Context) error) error {
	fs.incrementCounter(opName)
	fs.updateParallelism(opName, true)
	defer fs.updateParallelism(opName, false)

	// Call the wrapped filesystem operation.
	err := w(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (fs *ProfilerWrapper) GetProfileData() map[string]interface{} {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	data := make(map[string]interface{})
	for op, stats := range fs.Stats {
		data[op] = map[string]int64{
			"TotalCount":  stats.TotalCount,
			"Parallelism": stats.Parallelism,
		}
	}
	fs.Stats = make(map[string]OperationStats) // Reset stats after collecting data
	fs.temp = make(map[string]OperationStats)  // Reset temporary stats
	return data
}

func (fs *ProfilerWrapper) Destroy() {
	fs.wrapped.Destroy()
}

func (fs *ProfilerWrapper) StatFS(ctx context.Context, op *fuseops.StatFSOp) error {
	return fs.invokeWrapped(ctx, "StatFS", func(ctx context.Context) error { return fs.wrapped.StatFS(ctx, op) })
}

func (fs *ProfilerWrapper) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) error {
	return fs.invokeWrapped(ctx, "LookUpInode", func(ctx context.Context) error { return fs.wrapped.LookUpInode(ctx, op) })
}

func (fs *ProfilerWrapper) GetInodeAttributes(ctx context.Context, op *fuseops.GetInodeAttributesOp) error {
	return fs.invokeWrapped(ctx, "GetInodeAttributes", func(ctx context.Context) error { return fs.wrapped.GetInodeAttributes(ctx, op) })
}

func (fs *ProfilerWrapper) SetInodeAttributes(ctx context.Context, op *fuseops.SetInodeAttributesOp) error {
	return fs.invokeWrapped(ctx, "SetInodeAttributes", func(ctx context.Context) error { return fs.wrapped.SetInodeAttributes(ctx, op) })
}

func (fs *ProfilerWrapper) ForgetInode(ctx context.Context, op *fuseops.ForgetInodeOp) error {
	return fs.invokeWrapped(ctx, "ForgetInode", func(ctx context.Context) error { return fs.wrapped.ForgetInode(ctx, op) })
}

func (fs *ProfilerWrapper) BatchForget(ctx context.Context, op *fuseops.BatchForgetOp) error {
	return fs.invokeWrapped(ctx, "BatchForget", func(ctx context.Context) error { return fs.wrapped.BatchForget(ctx, op) })
}

func (fs *ProfilerWrapper) MkDir(ctx context.Context, op *fuseops.MkDirOp) error {
	return fs.invokeWrapped(ctx, "MkDir", func(ctx context.Context) error { return fs.wrapped.MkDir(ctx, op) })
}

func (fs *ProfilerWrapper) MkNode(ctx context.Context, op *fuseops.MkNodeOp) error {
	return fs.invokeWrapped(ctx, "MkNode", func(ctx context.Context) error { return fs.wrapped.MkNode(ctx, op) })
}

func (fs *ProfilerWrapper) CreateFile(ctx context.Context, op *fuseops.CreateFileOp) error {
	return fs.invokeWrapped(ctx, "CreateFile", func(ctx context.Context) error { return fs.wrapped.CreateFile(ctx, op) })
}

func (fs *ProfilerWrapper) CreateLink(ctx context.Context, op *fuseops.CreateLinkOp) error {
	return fs.invokeWrapped(ctx, "CreateLink", func(ctx context.Context) error { return fs.wrapped.CreateLink(ctx, op) })
}

func (fs *ProfilerWrapper) CreateSymlink(ctx context.Context, op *fuseops.CreateSymlinkOp) error {
	return fs.invokeWrapped(ctx, "CreateSymlink", func(ctx context.Context) error { return fs.wrapped.CreateSymlink(ctx, op) })
}

func (fs *ProfilerWrapper) Rename(ctx context.Context, op *fuseops.RenameOp) error {
	return fs.invokeWrapped(ctx, "Rename", func(ctx context.Context) error { return fs.wrapped.Rename(ctx, op) })
}

func (fs *ProfilerWrapper) RmDir(ctx context.Context, op *fuseops.RmDirOp) error {
	return fs.invokeWrapped(ctx, "RmDir", func(ctx context.Context) error { return fs.wrapped.RmDir(ctx, op) })
}

func (fs *ProfilerWrapper) Unlink(ctx context.Context, op *fuseops.UnlinkOp) error {
	return fs.invokeWrapped(ctx, "Unlink", func(ctx context.Context) error { return fs.wrapped.Unlink(ctx, op) })
}

func (fs *ProfilerWrapper) OpenDir(ctx context.Context, op *fuseops.OpenDirOp) error {
	return fs.invokeWrapped(ctx, "OpenDir", func(ctx context.Context) error { return fs.wrapped.OpenDir(ctx, op) })
}

func (fs *ProfilerWrapper) ReadDir(ctx context.Context, op *fuseops.ReadDirOp) error {
	return fs.invokeWrapped(ctx, "ReadDir", func(ctx context.Context) error { return fs.wrapped.ReadDir(ctx, op) })
}

func (fs *ProfilerWrapper) ReadDirPlus(ctx context.Context, op *fuseops.ReadDirPlusOp) error {
	return fs.invokeWrapped(ctx, "ReadDirPlus", func(ctx context.Context) error { return fs.wrapped.ReadDirPlus(ctx, op) })
}

func (fs *ProfilerWrapper) ReleaseDirHandle(ctx context.Context, op *fuseops.ReleaseDirHandleOp) error {
	return fs.invokeWrapped(ctx, "ReleaseDirHandle", func(ctx context.Context) error { return fs.wrapped.ReleaseDirHandle(ctx, op) })
}

func (fs *ProfilerWrapper) OpenFile(ctx context.Context, op *fuseops.OpenFileOp) error {
	return fs.invokeWrapped(ctx, "OpenFile", func(ctx context.Context) error { return fs.wrapped.OpenFile(ctx, op) })
}

func (fs *ProfilerWrapper) ReadFile(ctx context.Context, op *fuseops.ReadFileOp) error {
	return fs.invokeWrapped(ctx, "ReadFile", func(ctx context.Context) error { return fs.wrapped.ReadFile(ctx, op) })
}

func (fs *ProfilerWrapper) WriteFile(ctx context.Context, op *fuseops.WriteFileOp) error {
	return fs.invokeWrapped(ctx, "WriteFile", func(ctx context.Context) error { return fs.wrapped.WriteFile(ctx, op) })
}

func (fs *ProfilerWrapper) SyncFile(ctx context.Context, op *fuseops.SyncFileOp) error {
	return fs.invokeWrapped(ctx, "SyncFile", func(ctx context.Context) error { return fs.wrapped.SyncFile(ctx, op) })
}

func (fs *ProfilerWrapper) FlushFile(ctx context.Context, op *fuseops.FlushFileOp) error {
	return fs.invokeWrapped(ctx, "FlushFile", func(ctx context.Context) error { return fs.wrapped.FlushFile(ctx, op) })
}

func (fs *ProfilerWrapper) ReleaseFileHandle(ctx context.Context, op *fuseops.ReleaseFileHandleOp) error {
	return fs.invokeWrapped(ctx, "ReleaseFileHandle", func(ctx context.Context) error { return fs.wrapped.ReleaseFileHandle(ctx, op) })
}

func (fs *ProfilerWrapper) ReadSymlink(ctx context.Context, op *fuseops.ReadSymlinkOp) error {
	return fs.invokeWrapped(ctx, "ReadSymlink", func(ctx context.Context) error { return fs.wrapped.ReadSymlink(ctx, op) })
}

func (fs *ProfilerWrapper) RemoveXattr(ctx context.Context, op *fuseops.RemoveXattrOp) error {
	return fs.invokeWrapped(ctx, "RemoveXattr", func(ctx context.Context) error { return fs.wrapped.RemoveXattr(ctx, op) })
}

func (fs *ProfilerWrapper) GetXattr(ctx context.Context, op *fuseops.GetXattrOp) error {
	return fs.invokeWrapped(ctx, "GetXattr", func(ctx context.Context) error { return fs.wrapped.GetXattr(ctx, op) })
}

func (fs *ProfilerWrapper) ListXattr(ctx context.Context, op *fuseops.ListXattrOp) error {
	return fs.invokeWrapped(ctx, "ListXattr", func(ctx context.Context) error { return fs.wrapped.ListXattr(ctx, op) })
}

func (fs *ProfilerWrapper) SetXattr(ctx context.Context, op *fuseops.SetXattrOp) error {
	return fs.invokeWrapped(ctx, "SetXattr", func(ctx context.Context) error { return fs.wrapped.SetXattr(ctx, op) })
}

func (fs *ProfilerWrapper) Fallocate(ctx context.Context, op *fuseops.FallocateOp) error {
	return fs.invokeWrapped(ctx, "Fallocate", func(ctx context.Context) error { return fs.wrapped.Fallocate(ctx, op) })
}

func (fs *ProfilerWrapper) SyncFS(ctx context.Context, op *fuseops.SyncFSOp) error {
	return fs.invokeWrapped(ctx, "SyncFS", func(ctx context.Context) error { return fs.wrapped.SyncFS(ctx, op) })
}
