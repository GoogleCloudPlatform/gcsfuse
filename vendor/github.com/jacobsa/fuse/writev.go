package fuse

import (
	"syscall"
	"unsafe"
)

func writev(fd int, packet [][]byte) (n int, err error) {
	iovecs := make([]syscall.Iovec, 0, len(packet))
	for _, v := range packet {
		if len(v) == 0 {
			continue
		}
		vec := syscall.Iovec{
			Base: &v[0],
		}
		vec.SetLen(len(v))
		iovecs = append(iovecs, vec)
	}
	n1, _, e1 := syscall.Syscall(
		syscall.SYS_WRITEV,
		uintptr(fd), uintptr(unsafe.Pointer(&iovecs[0])), uintptr(len(iovecs)),
	)
	n = int(n1)
	if e1 != 0 {
		err = syscall.Errno(e1)
	}
	return
}
