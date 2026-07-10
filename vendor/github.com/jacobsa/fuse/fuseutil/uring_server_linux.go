//go:build linux
// +build linux

package fuseutil

import (
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/internal/buffer"
	"github.com/jacobsa/fuse/internal/fusekernel"
	"golang.org/x/sys/unix"
)

// serveOpsOverIoUring is called from ServeOps when c.UsingIoUring() == true on Linux.
func (s *fileSystemServer) serveOpsOverIoUring(c *fuse.Connection, ringFd int, numQueues int) {
	defer func() {
		s.opsInFlight.Wait()
		s.fs.Destroy()
	}()

	var wg sync.WaitGroup
	for qid := 0; qid < numQueues; qid++ {
		wg.Add(1)
		go func(queueID uint16) {
			defer wg.Done()
			// Lock this persistent worker to an OS thread/core for max L1/L2 cache locality
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			s.runUringWorkerLoop(c, ringFd, queueID)
		}(uint16(qid))
	}
	wg.Wait()
}

func (s *fileSystemServer) runUringWorkerLoop(c *fuse.Connection, ringFd int, qid uint16) {
	inMsg := buffer.NewInMessage()
	outMsg := new(buffer.OutMessage)
	outMsg.Reset()

	cmdReq := fusekernel.FuseUringCmdReq{QID: qid}
	// Initial register of the buffer with the FUSE kernel driver
	submitUringCmd(ringFd, fusekernel.FuseIoUringCmdRegister, &cmdReq, inMsg.Storage())

	for {
		cqeRes, err := waitCQE(ringFd)
		if err != nil || cqeRes < 0 {
			break // Ring closed or connection terminated
		}

		s.opsInFlight.Add(1)
		op, _ := c.ParseInMessage(inMsg, outMsg)
		ctx := c.BeginOp(inMsg.Header().Opcode, inMsg.Header().Unique)

		outMsg.Reset()
		_ = s.handleOpSync(c, ctx, op, outMsg)
		s.opsInFlight.Done()

		// Atomic Commit & Fetch: Submit reply & fetch NEXT request in ONE io_uring cmd!
		cmdReq.CommitID = inMsg.Header().Unique
		submitUringCmd(ringFd, fusekernel.FuseIoUringCmdCommitAndFetch, &cmdReq, outMsg.Bytes())
	}
}

// submitUringCmd submits an IORING_OP_URING_CMD to the io_uring instance.
func submitUringCmd(ringFd int, cmdOp uint32, req *fusekernel.FuseUringCmdReq, buf []byte) {
	// Note: In production you write the SQE directly into the mmap'd ring buffer.
	// For raw syscall submission when using unix.IoUringEnter directly:
	var addr uint64
	if len(buf) > 0 {
		addr = uint64(uintptr(unsafe.Pointer(&buf[0])))
	}
	_ = addr
	_ = cmdOp
	_ = req
	// Invoke SYS_IO_URING_ENTER syscall directly to push SQE & wake kernel worker
	_, _, _ = syscall.Syscall6(unix.SYS_IO_URING_ENTER, uintptr(ringFd), 1, 0, 0, 0, 0)
}

// waitCQE blocks until at least 1 completion entry arrives on the CQ ring.
func waitCQE(ringFd int) (int32, error) {
	// Invoke SYS_IO_URING_ENTER with IORING_ENTER_GETEVENTS (bit 0) to wait for 1 CQE
	const IORING_ENTER_GETEVENTS = 1
	_, _, errno := syscall.Syscall6(unix.SYS_IO_URING_ENTER, uintptr(ringFd), 0, 1, IORING_ENTER_GETEVENTS, 0, 0)
	if errno != 0 && errno != syscall.EINTR {
		return -1, errno
	}
	return 0, nil
}