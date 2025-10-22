package fs

/*
#cgo CFLAGS: -DFUSE_USE_VERSION=29
#cgo pkg-config: fuse
#include <fuse.h>
#include <stdlib.h>
#include <errno.h>

// Forward declarations for Go functions
int go_getattr_callback(char *path, struct stat *stbuf);
int go_readdir_callback(char *path, void *buf, fuse_fill_dir_t filler, off_t offset, struct fuse_file_info *fi);
int go_read_callback(char *path, char *buf, size_t size, off_t offset, struct fuse_file_info *fi);

// C stubs that call the Go functions
static int xmp_getattr(const char *path, struct stat *stbuf) {
    return go_getattr_callback((char*)path, stbuf);
}

static int xmp_readdir(const char *path, void *buf, fuse_fill_dir_t filler, off_t offset, struct fuse_file_info *fi) {
    return go_readdir_callback((char*)path, buf, filler, offset, fi);
}

static int xmp_read(const char *path, char *buf, size_t size, off_t offset, struct fuse_file_info *fi) {
    return go_read_callback((char*)path, buf, size, offset, fi);
}

static struct fuse_operations xmp_oper = {
    .getattr = xmp_getattr,
    .readdir = xmp_readdir,
    .read    = xmp_read,
};

// Helper function to run fuse_main
static int run_fuse(int argc, char *argv[]) {
    return fuse_main(argc, argv, &xmp_oper, NULL);
}

// Helper to call filler
static int helper_c_filler(fuse_fill_dir_t filler, void* buf, const char* name, const struct stat* stbuf, off_t off) {
	return filler(buf, name, stbuf, off);
}
*/

import "C"
import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
)

const (
	helloPath = "/hello"
	helloStr  = "Hello, World!\n"
)

//export go_getattr_callback
func go_getattr_callback(path *C.char, stbuf *C.struct_stat) C.int {
	goPath := C.GoString(path)
	fmt.Println("getattr called for:", goPath)

	stbuf.st_uid = C.uid_t(os.Getuid())
	stbuf.st_gid = C.gid_t(os.Getgid())

	// cgo does not handle #define, so we need to use the actual struct field names.
	now := time.Now()
	stbuf.st_atim.tv_sec = C.long(now.Unix())
	stbuf.st_atim.tv_nsec = C.long(now.Nanosecond())
	stbuf.st_mtim.tv_sec = C.long(now.Unix())
	stbuf.st_mtim.tv_nsec = C.long(now.Nanosecond())
	stbuf.st_ctim.tv_sec = C.long(now.Unix())
	stbuf.st_ctim.tv_nsec = C.long(now.Nanosecond())

	if goPath == "/" {
		stbuf.st_mode = syscall.S_IFDIR | 0755
		stbuf.st_nlink = 2
		return 0
	}

	if goPath == helloPath {
		stbuf.st_mode = syscall.S_IFREG | 0444
		stbuf.st_nlink = 1
		stbuf.st_size = C.off_t(len(helloStr))
		return 0
	}

	return -C.ENOENT
}

//export go_readdir_callback
func go_readdir_callback(path *C.char, buf unsafe.Pointer, filler C.fuse_fill_dir_t, offset C.off_t, fi *C.struct_fuse_file_info) C.int {
	goPath := C.GoString(path)
	fmt.Println("readdir called for:", goPath)

	if goPath != "/" {
		return -C.ENOENT
	}

	cDot := C.CString(".")
	defer C.free(unsafe.Pointer(cDot))
	cDotDot := C.CString("..")
	defer C.free(unsafe.Pointer(cDotDot))
	cHelloPath := C.CString(helloPath[1:])
	defer C.free(unsafe.Pointer(cHelloPath))

	// Standard entries
	C.helper_c_filler(filler, buf, cDot, nil, 0)
	C.helper_c_filler(filler, buf, cDotDot, nil, 0)
	// Our file
	C.helper_c_filler(filler, buf, cHelloPath, nil, 0)

	return 0
}

//export go_read_callback
func go_read_callback(path *C.char, buf *C.char, size C.size_t, offset C.off_t, fi *C.struct_fuse_file_info) C.int {
	goPath := C.GoString(path)
	fmt.Println("read called for:", goPath)

	if goPath != helloPath {
		return -C.ENOENT
	}

	end := int(offset) + int(size)
	if end > len(helloStr) {
		end = len(helloStr)
	}

	copySize := end - int(offset)
	if copySize <= 0 {
		return 0
	}

	cbuf := (*[1 << 30]byte)(unsafe.Pointer(buf))[:size:size]
	copy(cbuf, helloStr[offset:end])

	return C.int(copySize)
}
