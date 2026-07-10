//go:build !linux
// +build !linux

package fuseutil

import (
	"github.com/jacobsa/fuse"
	"log"
)

func (s *fileSystemServer) serveOpsOverIoUring(c *fuse.Connection, ringFd int, numQueues int) {
	log.Panicf("FUSE_OVER_IO_URING is not supported on non-Linux platforms")
}
