// Copyright 2023 Google LLC
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

//go:build libfuse && (libfuse2 || libfuse3)

package fuse

import (
	"context"
	"fmt"
	"os"
	"time"
	"unsafe"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/jacobsa/fuse/fuseops"
)

/*
#include <fuse_lowlevel.h>
#include <stdlib.h>

extern void go_init(void* userdata, struct fuse_conn_info* conn);
extern void go_lookup(fuse_req_t req, fuse_ino_t parent, const char *name);
extern void go_getattr(fuse_req_t req, fuse_ino_t ino, struct fuse_file_info *fi);
extern void go_readdir(fuse_req_t req, fuse_ino_t ino, size_t size, off_t off,
                       struct fuse_file_info *fi);
extern void go_open(fuse_req_t req, fuse_ino_t ino, struct fuse_file_info *fi);
extern void go_read(fuse_req_t req, fuse_ino_t ino, size_t size, off_t off,
                    struct fuse_file_info *fi);
extern void go_write(fuse_req_t req, fuse_ino_t ino, const char *buf,
                     size_t size, off_t off, struct fuse_file_info *fi);
extern void go_create(fuse_req_t req, fuse_ino_t parent, const char *name,
                      mode_t mode, struct fuse_file_info *fi);
extern void go_mkdir(fuse_req_t req, fuse_ino_t parent, const char *name,
                     mode_t mode);
extern void go_unlink(fuse_req_t req, fuse_ino_t parent, const char *name);
extern void go_rmdir(fuse_req_t req, fuse_ino_t parent, const char *name);
extern void go_rename(fuse_req_t req, fuse_ino_t parent_old,
                      const char *name_old, fuse_ino_t parent_new,
                      const char *name_new, unsigned int flags);
extern void go_setattr(fuse_req_t req, fuse_ino_t ino, struct stat *attr,
                       int to_set, struct fuse_file_info *fi);

static inline void set_ops(struct fuse_lowlevel_ops *ops) {
	ops->init = go_init;
	ops->lookup = go_lookup;
	ops->getattr = go_getattr;
	ops->readdir = go_readdir;
	ops->open = go_open;
	ops->read = go_read;
	ops->write = go_write;
	ops->create = go_create;
	ops->mkdir = go_mkdir;
	ops->unlink = go_unlink;
	ops->rmdir = go_rmdir;
	ops->rename = go_rename;
	ops->setattr = go_setattr;
}
*/
import "C"

var (
	theServer *libfuseServer
)

//export go_init
func go_init(userdata unsafe.Pointer, conn *C.struct_fuse_conn_info) {
	// Store the server pointer in the user data.
	// This is not safe, but it's the only way to get the server pointer
	// in the callbacks.
	conn.userdata = unsafe.Pointer(theServer)
}

//export go_lookup
func go_lookup(req C.fuse_req_t, parent C.fuse_ino_t, name *C.char) {
	s := C.fuse_req_get_userdata_cgo(req)
	server := (*libfuseServer)(s)
	op := &fuseops.LookUpInodeOp{
		Parent: fuseops.InodeID(parent),
		Name:   C.GoString(name),
	}
	entry, err := server.fs.LookUpInode(context.Background(), op)
	if err != nil {
		C.fuse_reply_err_cgo(req, C.int(err.(fuse.FuseError).ErrorCode()))
		return
	}
	e := &C.struct_fuse_entry_param{}
	e.ino = C.fuse_ino_t(entry.Child)
	e.attr_timeout = C.double(entry.AttributesExpiration.Sub(time.Now()).Seconds())
	e.entry_timeout = C.double(entry.EntryExpiration.Sub(time.Now()).Seconds())
	toCStat(&entry.Attributes, &e.attr)
	C.fuse_reply_entry_cgo(req, e)
}

//export go_getattr
func go_getattr(req C.fuse_req_t, ino C.fuse_ino_t, fi *C.struct_fuse_file_info) {
	s := C.fuse_req_get_userdata_cgo(req)
	server := (*libfuseServer)(s)
	op := &fuseops.GetInodeAttributesOp{
		Inode: fuseops.InodeID(ino),
	}
	attrs, err := server.fs.GetInodeAttributes(context.Background(), op)
	if err != nil {
		C.fuse_reply_err_cgo(req, C.int(err.(fuse.FuseError).ErrorCode()))
		return
	}
	var st C.struct_stat
	toCStat(&attrs.Attributes, &st)
	C.fuse_reply_attr_cgo(req, &st, C.double(attrs.AttributesExpiration.Sub(time.Now()).Seconds()))
}

//export go_readdir
func go_readdir(req C.fuse_req_t, ino C.fuse_ino_t, size C.size_t, off C.off_t, fi *C.struct_fuse_file_info) {
	s := C.fuse_req_get_userdata_cgo(req)
	server := (*libfuseServer)(s)
	op := &fuseops.ReadDirOp{
		Inode:  fuseops.InodeID(ino),
		Offset: fuseops.DirOffset(off),
		Size:   int(size),
	}
	// TODO: support directory handles.
	// op.Handle = fuseops.HandleID(fi.fh)
	entries, err := server.fs.ReadDir(context.Background(), op)
	if err != nil {
		C.fuse_reply_err_cgo(req, C.int(err.(fuse.FuseError).ErrorCode()))
		return
	}
	buf := make([]byte, 0, op.Size)
	for _, entry := range entries {
		name := C.CString(entry.Name)
		var st C.struct_stat
		st.st_ino = C.fuse_ino_t(entry.Inode)
		st.st_mode = C.uint(entry.Type)
		// TODO: This is not ideal.
		direntSize := C.fuse_add_direntry_cgo(req, nil, 0, name, nil, 0)
		newBuf := make([]byte, len(buf)+int(direntSize))
		copy(newBuf, buf)
		C.fuse_add_direntry_cgo(req, (*C.char)(unsafe.Pointer(&newBuf[len(buf)])), C.size_t(direntSize), name, &st, C.off_t(entry.Offset))
		buf = newBuf
		C.free(unsafe.Pointer(name))
	}
	C.fuse_reply_buf_cgo(req, (*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)))
}

//export go_open
func go_open(req C.fuse_req_t, ino C.fuse_ino_t, fi *C.struct_fuse_file_info) {
	s := C.fuse_req_get_userdata_cgo(req)
	server := (*libfuseServer)(s)
	op := &fuseops.OpenFileOp{
		Inode: fuseops.InodeID(ino),
		Flags: int32(fi.flags),
	}
	err := server.fs.OpenFile(context.Background(), op)
	if err != nil {
		C.fuse_reply_err_cgo(req, C.int(err.(fuse.FuseError).ErrorCode()))
		return
	}
	fi.fh = C.uint64_t(op.Handle)
	fi.keep_cache = 1
	C.fuse_reply_open_cgo(req, fi)
}

//export go_read
func go_read(req C.fuse_req_t, ino C.fuse_ino_t, size C.size_t, off C.off_t, fi *C.struct_fuse_file_info) {
	s := C.fuse_req_get_userdata_cgo(req)
	server := (*libfuseServer)(s)
	op := &fuseops.ReadFileOp{
		Inode:  fuseops.InodeID(ino),
		Handle: fuseops.HandleID(fi.fh),
		Offset: int64(off),
		Size:   int(size),
	}
	buf := make([]byte, op.Size)
	n, err := server.fs.ReadFile(context.Background(), op)
	if err != nil {
		C.fuse_reply_err_cgo(req, C.int(err.(fuse.FuseError).ErrorCode()))
		return
	}
	C.fuse_reply_buf_cgo(req, (*C.char)(unsafe.Pointer(&buf[0])), C.size_t(n))
}

//export go_write
func go_write(req C.fuse_req_t, ino C.fuse_ino_t, buf *C.char, size C.size_t, off C.off_t, fi *C.struct_fuse_file_info) {
	s := C.fuse_req_get_userdata_cgo(req)
	server := (*libfuseServer)(s)
	op := &fuseops.WriteFileOp{
		Inode:  fuseops.InodeID(ino),
		Handle: fuseops.HandleID(fi.fh),
		Offset: int64(off),
		Data:   C.GoBytes(unsafe.Pointer(buf), C.int(size)),
	}
	err := server.fs.WriteFile(context.Background(), op)
	if err != nil {
		C.fuse_reply_err_cgo(req, C.int(err.(fuse.FuseError).ErrorCode()))
		return
	}
	C.fuse_reply_write_cgo(req, size)
}

//export go_create
func go_create(req C.fuse_req_t, parent C.fuse_ino_t, name *C.char, mode C.mode_t, fi *C.struct_fuse_file_info) {
	s := C.fuse_req_get_userdata_cgo(req)
	server := (*libfuseServer)(s)
	op := &fuseops.CreateFileOp{
		Parent: fuseops.InodeID(parent),
		Name:   C.GoString(name),
		Mode:   os.FileMode(mode),
	}
	entry, err := server.fs.CreateFile(context.Background(), op)
	if err != nil {
		C.fuse_reply_err_cgo(req, C.int(err.(fuse.FuseError).ErrorCode()))
		return
	}
	e := &C.struct_fuse_entry_param{}
	e.ino = C.fuse_ino_t(entry.Child)
	e.attr_timeout = C.double(entry.AttributesExpiration.Sub(time.Now()).Seconds())
	e.entry_timeout = C.double(entry.EntryExpiration.Sub(time.Now()).Seconds())
	toCStat(&entry.Attributes, &e.attr)
	fi.fh = C.uint64_t(op.Handle)
	fi.keep_cache = 1
	C.fuse_reply_create_cgo(req, e, fi)
}

//export go_mkdir
func go_mkdir(req C.fuse_req_t, parent C.fuse_ino_t, name *C.char, mode C.mode_t) {
	s := C.fuse_req_get_userdata_cgo(req)
	server := (*libfuseServer)(s)
	op := &fuseops.MkDirOp{
		Parent: fuseops.InodeID(parent),
		Name:   C.GoString(name),
		Mode:   os.FileMode(mode),
	}
	entry, err := server.fs.MkDir(context.Background(), op)
	if err != nil {
		C.fuse_reply_err_cgo(req, C.int(err.(fuse.FuseError).ErrorCode()))
		return
	}
	e := &C.struct_fuse_entry_param{}
	e.ino = C.fuse_ino_t(entry.Child)
	e.attr_timeout = C.double(entry.AttributesExpiration.Sub(time.Now()).Seconds())
	e.entry_timeout = C.double(entry.EntryExpiration.Sub(time.Now()).Seconds())
	toCStat(&entry.Attributes, &e.attr)
	C.fuse_reply_entry_cgo(req, e)
}

//export go_unlink
func go_unlink(req C.fuse_req_t, parent C.fuse_ino_t, name *C.char) {
	s := C.fuse_req_get_userdata_cgo(req)
	server := (*libfuseServer)(s)
	op := &fuseops.UnlinkOp{
		Parent: fuseops.InodeID(parent),
		Name:   C.GoString(name),
	}
	err := server.fs.Unlink(context.Background(), op)
	if err != nil {
		C.fuse_reply_err_cgo(req, C.int(err.(fuse.FuseError).ErrorCode()))
		return
	}
	C.fuse_reply_err_cgo(req, 0)
}

//export go_rename_2
func go_rename_2(req C.fuse_req_t, parent_old C.fuse_ino_t, name_old *C.char, parent_new C.fuse_ino_t, name_new *C.char) {
	s := C.fuse_req_get_userdata_cgo(req)
	server := (*libfuseServer)(s)
	op := &fuseops.RenameOp{
		OldParent: fuseops.InodeID(parent_old),
		OldName:   C.GoString(name_old),
		NewParent: fuseops.InodeID(parent_new),
		NewName:   C.GoString(name_new),
	}
	err := server.fs.Rename(context.Background(), op)
	if err != nil {
		C.fuse_reply_err_cgo(req, C.int(err.(fuse.FuseError).ErrorCode()))
		return
	}
	C.fuse_reply_err_cgo(req, 0)
}

//export go_rmdir
func go_rmdir(req C.fuse_req_t, parent C.fuse_ino_t, name *C.char) {
	s := C.fuse_req_get_userdata_cgo(req)
	server := (*libfuseServer)(s)
	op := &fuseops.RmDirOp{
		Parent: fuseops.InodeID(parent),
		Name:   C.GoString(name),
	}
	err := server.fs.RmDir(context.Background(), op)
	if err != nil {
		C.fuse_reply_err_cgo(req, C.int(err.(fuse.FuseError).ErrorCode()))
		return
	}
	C.fuse_reply_err_cgo(req, 0)
}

//export go_rename
func go_rename(req C.fuse_req_t, parent_old C.fuse_ino_t, name_old *C.char, parent_new C.fuse_ino_t, name_new *C.char, flags C.uint) {
	s := C.fuse_req_get_userdata_cgo(req)
	server := (*libfuseServer)(s)
	op := &fuseops.RenameOp{
		OldParent: fuseops.InodeID(parent_old),
		OldName:   C.GoString(name_old),
		NewParent: fuseops.InodeID(parent_new),
		NewName:   C.GoString(name_new),
	}
	err := server.fs.Rename(context.Background(), op)
	if err != nil {
		C.fuse_reply_err_cgo(req, C.int(err.(fuse.FuseError).ErrorCode()))
		return
	}
	C.fuse_reply_err_cgo(req, 0)
}

//export go_setattr
func go_setattr(req C.fuse_req_t, ino C.fuse_ino_t, attr *C.struct_stat, to_set C.int, fi *C.struct_fuse_file_info) {
	s := C.fuse_req_get_userdata_cgo(req)
	server := (*libfuseServer)(s)
	op := &fuseops.SetInodeAttributesOp{
		Inode: fuseops.InodeID(ino),
	}
	if to_set&C.FUSE_SET_ATTR_MODE != 0 {
		op.Mode = new(os.FileMode)
		*op.Mode = os.FileMode(attr.st_mode)
	}
	if to_set&C.FUSE_SET_ATTR_UID != 0 {
		op.Uid = new(uint32)
		*op.Uid = uint32(attr.st_uid)
	}
	if to_set&C.FUSE_SET_ATTR_GID != 0 {
		op.Gid = new(uint32)
		*op.Gid = uint32(attr.st_gid)
	}
	if to_set&C.FUSE_SET_ATTR_SIZE != 0 {
		op.Size = new(uint64)
		*op.Size = uint64(attr.st_size)
	}
	if to_set&C.FUSE_SET_ATTR_ATIME != 0 {
		op.Atime = new(time.Time)
		*op.Atime = time.Unix(int64(attr.st_atime), int64(attr.st_atime_nsec))
	}
	if to_set&C.FUSE_SET_ATTR_MTIME != 0 {
		op.Mtime = new(time.Time)
		*op.Mtime = time.Unix(int64(attr.st_mtime), int64(attr.st_mtime_nsec))
	}
	// TODO: support file handles.
	// if to_set&C.FUSE_SET_ATTR_FUSE_SET_ATTR_FH != 0 {
	// 	op.Handle = new(fuseops.HandleID)
	// 	*op.Handle = fuseops.HandleID(fi.fh)
	// }
	attrs, err := server.fs.SetInodeAttributes(context.Background(), op)
	if err != nil {
		C.fuse_reply_err_cgo(req, C.int(err.(fuse.FuseError).ErrorCode()))
		return
	}
	var st C.struct_stat
	toCStat(&attrs.Attributes, &st)
	C.fuse_reply_attr_cgo(req, &st, C.double(attrs.AttributesExpiration.Sub(time.Now()).Seconds()))
}

func NewServer(newConfig *cfg.Config) (Server, error) {
	theServer = &libfuseServer{
		newConfig: newConfig,
	}
	return theServer, nil
}

type libfuseServer struct {
	newConfig *cfg.Config
	fs        fuse.Server
	mounted   bool
}

func (s *libfuseServer) Mount(
	ctx context.Context,
	mountPoint string,
	fsCreator func(context.Context, *fs.ServerConfig) (fs.FileSystem, error),
	serverCfg *fs.ServerConfig) (mfs *MountedFileSystem, err error) {
	// Create a file system server.
	logger.Infof("Creating a new server...\n")
	s.fs, err = fsCreator(ctx, serverCfg)
	if err != nil {
		err = fmt.Errorf("fs.NewServer: %w", err)
		return
	}

	// Mount the file system.
	fsName := serverCfg.BucketName
	if fsName == "" || fsName == "_" {
		// mounting all the buckets at once
		fsName = "gcsfuse"
	}
	logger.Infof("Mounting file system %q...", fsName)
	mountCfg := getFuseMountConfig(fsName, s.newConfig)

	args := []string{"gcsfuse"}
	for k, v := range mountCfg.Options {
		if v == "" {
			args = append(args, "-o", k)
		} else {
			args = append(args, "-o", fmt.Sprintf("%s=%s", k, v))
		}
	}
	args = append(args, mountPoint)
	cArgs := make([]*C.char, len(args))
	for i, arg := range args {
		cArgs[i] = C.CString(arg)
	}
	defer func() {
		for _, arg := range cArgs {
			C.free(unsafe.Pointer(arg))
		}
	}()
	fuseArgs := C.struct_fuse_args{
		argc: C.int(len(args)),
		argv: &cArgs[0],
	}

	// Create a new FUSE session.
	var ops C.struct_fuse_lowlevel_ops
	C.set_ops(&ops)
	se := C.fuse_session_new_cgo(&fuseArgs, &ops, C.sizeof_struct_fuse_lowlevel_ops, nil)
	if se == nil {
		return nil, fmt.Errorf("fuse_session_new: failed to create session")
	}
	defer C.fuse_session_destroy_cgo(se)

	// Set signal handlers.
	if C.fuse_set_signal_handlers_cgo(se) != 0 {
		return nil, fmt.Errorf("fuse_set_signal_handlers: failed to set signal handlers")
	}
	defer C.fuse_remove_signal_handlers_cgo(se)

	// Mount the session.
	if C.fuse_session_mount_cgo(se, C.CString(mountPoint)) != 0 {
		return nil, fmt.Errorf("fuse_session_mount: failed to mount session")
	}
	defer C.fuse_session_unmount_cgo(se)

	// Daemonize the process.
	if C.fuse_daemonize_cgo(0) != 0 {
		return nil, fmt.Errorf("fuse_daemonize: failed to daemonize")
	}

	// Run the session loop.
	if C.fuse_session_loop_mt_cgo(se, 0) != 0 {
		return nil, fmt.Errorf("fuse_session_loop_mt: failed to run session loop")
	}

	return
}

func toCStat(attrs *fuseops.InodeAttributes, st *C.struct_stat) {
	st.st_ino = C.ulong(attrs.Inode)
	st.st_mode = C.uint(attrs.Mode)
	st.st_nlink = C.uint(attrs.Nlink)
	st.st_uid = C.uint(attrs.Uid)
	st.st_gid = C.uint(attrs.Gid)
	st.st_size = C.long(attrs.Size)
	st.st_atime = C.long(attrs.Atime.Unix())
	st.st_mtime = C.long(attrs.Mtime.Unix())
	st.st_ctime = C.long(attrs.Ctime.Unix())
}
