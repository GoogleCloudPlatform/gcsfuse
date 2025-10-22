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

//go:build libfuse && libfuse3

package fs

/*
#cgo pkg-config: fuse3
#include <fuse_lowlevel.h>
#include <stdlib.h>

// fuse_reply_err replies to a request with an error code.
void fuse_reply_err_cgo(fuse_req_t req, int err) {
	fuse_reply_err(req, err);
}

// fuse_reply_none replies to a request with no data.
void fuse_reply_none_cgo(fuse_req_t req) {
	fuse_reply_err(req, 0);
}

// fuse_req_ctx returns the context of a request.
struct fuse_ctx* fuse_req_ctx_cgo(fuse_req_t req) {
	return fuse_req_ctx(req);
}

// fuse_req_get_userdata returns the user data of a request.
void* fuse_req_get_userdata_cgo(fuse_req_t req) {
	return fuse_req_get_userdata(req);
}

// fuse_session_new creates a new FUSE session.
struct fuse_session* fuse_session_new_cgo(struct fuse_args *args,
                                          const struct fuse_lowlevel_ops *op,
                                          size_t op_size, void *userdata) {
	return fuse_session_new(args, op, op_size, userdata);
}

// fuse_session_mount mounts a FUSE session.
int fuse_session_mount_cgo(struct fuse_session *se, const char *mountpoint) {
	return fuse_session_mount(se, mountpoint);
}

// fuse_session_unmount unmounts a FUSE session.
void fuse_session_unmount_cgo(struct fuse_session *se) {
	fuse_session_unmount(se);
}

// fuse_session_loop_mt runs the FUSE session loop in multi-threaded mode.
int fuse_session_loop_mt_cgo(struct fuse_session *se, int clone_fd) {
	return fuse_session_loop_mt(se, clone_fd);
}

// fuse_session_destroy destroys a FUSE session.
void fuse_session_destroy_cgo(struct fuse_session *se) {
	fuse_session_destroy(se);
}

// fuse_set_signal_handlers sets the signal handlers for a FUSE session.
int fuse_set_signal_handlers_cgo(struct fuse_session *se) {
	return fuse_set_signal_handlers(se);
}

// fuse_remove_signal_handlers removes the signal handlers for a FUSE session.
void fuse_remove_signal_handlers_cgo(struct fuse_session *se) {
	fuse_remove_signal_handlers(se);
}

// fuse_daemonize daemonizes the FUSE process.
int fuse_daemonize_cgo(int foreground) {
	return fuse_daemonize(foreground);
}

// fuse_version returns the version of the libfuse library.
int fuse_version_cgo() {
	return fuse_version();
}

// fuse_reply_attr replies to a getattr request.
int fuse_reply_attr_cgo(fuse_req_t req, const struct stat *attr,
                        double attr_timeout) {
	return fuse_reply_attr(req, attr, attr_timeout);
}

// fuse_reply_entry replies to a lookup request.
int fuse_reply_entry_cgo(fuse_req_t req, const struct fuse_entry_param *e) {
	return fuse_reply_entry(req, e);
}

// fuse_reply_create replies to a create request.
int fuse_reply_create_cgo(fuse_req_t req, const struct fuse_entry_param *e,
                          const struct fuse_file_info *fi) {
	return fuse_reply_create(req, e, fi);
}

// fuse_reply_open replies to an open request.
int fuse_reply_open_cgo(fuse_req_t req, const struct fuse_file_info *fi) {
	return fuse_reply_open(req, fi);
}

// fuse_reply_read replies to a read request.
int fuse_reply_buf_cgo(fuse_req_t req, const char *buf, size_t size) {
	return fuse_reply_buf(req, buf, size);
}

// fuse_reply_write replies to a write request.
int fuse_reply_write_cgo(fuse_req_t req, size_t count) {
	return fuse_reply_write(req, count);
}

// fuse_add_direntry adds a directory entry to a directory buffer.
size_t fuse_add_direntry_cgo(fuse_req_t req, char *buf, size_t bufsize,
                             const char *name, const struct stat *stbuf,
                             off_t off) {
	return fuse_add_direntry(req, buf, bufsize, name, stbuf, off);
}

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
