package wrappers

import (
	"context"
	"slices"

	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
)

type ignoreInterrupt struct {
	wrapped fuseutil.FileSystem
}

// Add a fuseop to this list if it needs to ignore interrupts
var fsOpsIgnoringInterrupts = []string{
	"LookUpInode",
	"GetInodeAttributes",
	"SetInodeAttributes",
	"MkDir",
	"MkNode",
	"CreateFile",
	"CreateSymlink",
	"RmDir",
	"Rename",
	"Unlink",
	"ReadDir",
	"ReadDirPlus",
	"ReadFile",
	"WriteFile",
	"SyncFile",
	"FlushFile",
}

func (fs *ignoreInterrupt) Destroy() {
	fs.wrapped.Destroy()
}

// WithTracing wraps a FileSystem to handle interrupts and ignore accordingly
func WithIgnoreInterrupt(wrapped fuseutil.FileSystem) fuseutil.FileSystem {
	return &ignoreInterrupt{
		wrapped: wrapped,
	}
}

func (fs *ignoreInterrupt) invokeWrapped(ctx context.Context, opName string, w wrappedCall) error {
	clearParentCtx := slices.Contains(fsOpsIgnoringInterrupts, opName)
	if clearParentCtx {
		ctx = context.Background()
	}

	err := w(ctx)

	return err
}

func (fs *ignoreInterrupt) StatFS(ctx context.Context, op *fuseops.StatFSOp) error {
	return fs.invokeWrapped(ctx, "StatFS", func(ctx context.Context) error { return fs.wrapped.StatFS(ctx, op) })
}

func (fs *ignoreInterrupt) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) error {
	return fs.invokeWrapped(ctx, "LookUpInode", func(ctx context.Context) error { return fs.wrapped.LookUpInode(ctx, op) })
}

func (fs *ignoreInterrupt) GetInodeAttributes(ctx context.Context, op *fuseops.GetInodeAttributesOp) error {
	return fs.invokeWrapped(ctx, "GetInodeAttributes", func(ctx context.Context) error { return fs.wrapped.GetInodeAttributes(ctx, op) })
}

func (fs *ignoreInterrupt) SetInodeAttributes(ctx context.Context, op *fuseops.SetInodeAttributesOp) error {
	return fs.invokeWrapped(ctx, "SetInodeAttributes", func(ctx context.Context) error { return fs.wrapped.SetInodeAttributes(ctx, op) })
}

func (fs *ignoreInterrupt) ForgetInode(ctx context.Context, op *fuseops.ForgetInodeOp) error {
	return fs.invokeWrapped(ctx, "ForgetInode", func(ctx context.Context) error { return fs.wrapped.ForgetInode(ctx, op) })
}

func (fs *ignoreInterrupt) BatchForget(ctx context.Context, op *fuseops.BatchForgetOp) error {
	return fs.invokeWrapped(ctx, "BatchForget", func(ctx context.Context) error { return fs.wrapped.BatchForget(ctx, op) })
}

func (fs *ignoreInterrupt) MkDir(ctx context.Context, op *fuseops.MkDirOp) error {
	return fs.invokeWrapped(ctx, "MkDir", func(ctx context.Context) error { return fs.wrapped.MkDir(ctx, op) })
}

func (fs *ignoreInterrupt) MkNode(ctx context.Context, op *fuseops.MkNodeOp) error {
	return fs.invokeWrapped(ctx, "MkNode", func(ctx context.Context) error { return fs.wrapped.MkNode(ctx, op) })
}

func (fs *ignoreInterrupt) CreateFile(ctx context.Context, op *fuseops.CreateFileOp) error {
	return fs.invokeWrapped(ctx, "CreateFile", func(ctx context.Context) error { return fs.wrapped.CreateFile(ctx, op) })
}

func (fs *ignoreInterrupt) CreateLink(ctx context.Context, op *fuseops.CreateLinkOp) error {
	return fs.invokeWrapped(ctx, "CreateLink", func(ctx context.Context) error { return fs.wrapped.CreateLink(ctx, op) })
}

func (fs *ignoreInterrupt) CreateSymlink(ctx context.Context, op *fuseops.CreateSymlinkOp) error {
	return fs.invokeWrapped(ctx, "CreateSymlink", func(ctx context.Context) error { return fs.wrapped.CreateSymlink(ctx, op) })
}

func (fs *ignoreInterrupt) Rename(ctx context.Context, op *fuseops.RenameOp) error {
	return fs.invokeWrapped(ctx, "Rename", func(ctx context.Context) error { return fs.wrapped.Rename(ctx, op) })
}

func (fs *ignoreInterrupt) RmDir(ctx context.Context, op *fuseops.RmDirOp) error {
	return fs.invokeWrapped(ctx, "RmDir", func(ctx context.Context) error { return fs.wrapped.RmDir(ctx, op) })
}

func (fs *ignoreInterrupt) Unlink(ctx context.Context, op *fuseops.UnlinkOp) error {
	return fs.invokeWrapped(ctx, "Unlink", func(ctx context.Context) error { return fs.wrapped.Unlink(ctx, op) })
}

func (fs *ignoreInterrupt) OpenDir(ctx context.Context, op *fuseops.OpenDirOp) error {
	return fs.invokeWrapped(ctx, "OpenDir", func(ctx context.Context) error { return fs.wrapped.OpenDir(ctx, op) })
}

func (fs *ignoreInterrupt) ReadDir(ctx context.Context, op *fuseops.ReadDirOp) error {
	return fs.invokeWrapped(ctx, "ReadDir", func(ctx context.Context) error { return fs.wrapped.ReadDir(ctx, op) })
}

func (fs *ignoreInterrupt) ReadDirPlus(ctx context.Context, op *fuseops.ReadDirPlusOp) error {
	return fs.invokeWrapped(ctx, "ReadDirPlus", func(ctx context.Context) error { return fs.wrapped.ReadDirPlus(ctx, op) })
}

func (fs *ignoreInterrupt) ReleaseDirHandle(ctx context.Context, op *fuseops.ReleaseDirHandleOp) error {
	return fs.invokeWrapped(ctx, "ReleaseDirHandle", func(ctx context.Context) error { return fs.wrapped.ReleaseDirHandle(ctx, op) })
}

func (fs *ignoreInterrupt) OpenFile(ctx context.Context, op *fuseops.OpenFileOp) error {
	return fs.invokeWrapped(ctx, "OpenFile", func(ctx context.Context) error { return fs.wrapped.OpenFile(ctx, op) })
}

func (fs *ignoreInterrupt) ReadFile(ctx context.Context, op *fuseops.ReadFileOp) error {
	return fs.invokeWrapped(ctx, "ReadFile", func(ctx context.Context) error { return fs.wrapped.ReadFile(ctx, op) })
}

func (fs *ignoreInterrupt) WriteFile(ctx context.Context, op *fuseops.WriteFileOp) error {
	return fs.invokeWrapped(ctx, "WriteFile", func(ctx context.Context) error { return fs.wrapped.WriteFile(ctx, op) })
}

func (fs *ignoreInterrupt) SyncFile(ctx context.Context, op *fuseops.SyncFileOp) error {
	return fs.invokeWrapped(ctx, "SyncFile", func(ctx context.Context) error { return fs.wrapped.SyncFile(ctx, op) })
}

func (fs *ignoreInterrupt) FlushFile(ctx context.Context, op *fuseops.FlushFileOp) error {
	return fs.invokeWrapped(ctx, "FlushFile", func(ctx context.Context) error { return fs.wrapped.FlushFile(ctx, op) })
}

func (fs *ignoreInterrupt) ReleaseFileHandle(ctx context.Context, op *fuseops.ReleaseFileHandleOp) error {
	return fs.invokeWrapped(ctx, "ReleaseFileHandle", func(ctx context.Context) error { return fs.wrapped.ReleaseFileHandle(ctx, op) })
}

func (fs *ignoreInterrupt) ReadSymlink(ctx context.Context, op *fuseops.ReadSymlinkOp) error {
	return fs.invokeWrapped(ctx, "ReadSymlink", func(ctx context.Context) error { return fs.wrapped.ReadSymlink(ctx, op) })
}

func (fs *ignoreInterrupt) RemoveXattr(ctx context.Context, op *fuseops.RemoveXattrOp) error {
	return fs.invokeWrapped(ctx, "RemoveXattr", func(ctx context.Context) error { return fs.wrapped.RemoveXattr(ctx, op) })
}

func (fs *ignoreInterrupt) GetXattr(ctx context.Context, op *fuseops.GetXattrOp) error {
	return fs.invokeWrapped(ctx, "GetXattr", func(ctx context.Context) error { return fs.wrapped.GetXattr(ctx, op) })
}

func (fs *ignoreInterrupt) ListXattr(ctx context.Context, op *fuseops.ListXattrOp) error {
	return fs.invokeWrapped(ctx, "ListXattr", func(ctx context.Context) error { return fs.wrapped.ListXattr(ctx, op) })
}

func (fs *ignoreInterrupt) SetXattr(ctx context.Context, op *fuseops.SetXattrOp) error {
	return fs.invokeWrapped(ctx, "SetXattr", func(ctx context.Context) error { return fs.wrapped.SetXattr(ctx, op) })
}

func (fs *ignoreInterrupt) Fallocate(ctx context.Context, op *fuseops.FallocateOp) error {
	return fs.invokeWrapped(ctx, "Fallocate", func(ctx context.Context) error { return fs.wrapped.Fallocate(ctx, op) })
}

func (fs *ignoreInterrupt) SyncFS(ctx context.Context, op *fuseops.SyncFSOp) error {
	return fs.invokeWrapped(ctx, "SyncFS", func(ctx context.Context) error { return fs.wrapped.SyncFS(ctx, op) })
}
