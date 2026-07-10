//go:build linux
// +build linux

package fuseutil

import (
	"encoding/binary"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/internal/buffer"
	"github.com/jacobsa/fuse/internal/fusekernel"
	"golang.org/x/sys/unix"
)

// serveOpsOverIoUring is called from ServeOps when c.UsingIoUring() == true on Linux.
func (s *fileSystemServer) serveOpsOverIoUring(c *fuse.Connection, _ int, numQueues int) {
	log.Printf("[FUSE_OVER_IO_URING] Starting %d independent io_uring worker queues for DevFd=%d\n", numQueues, c.DevFd())
	defer func() {
		log.Printf("[FUSE_OVER_IO_URING] All worker queues stopped. Destroying filesystem and unmounting.\n")
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

			s.runUringWorkerLoop(c, queueID)
		}(uint16(qid))
	}
	wg.Wait()
}

func (s *fileSystemServer) runUringWorkerLoop(c *fuse.Connection, qid uint16) {
	// 1. Create a dedicated io_uring instance for this queue with 128-byte SQEs
	queue, err := newUringQueue(64)
	if err != nil {
		log.Printf("[FUSE_OVER_IO_URING] QID=%d Failed to setup io_uring: %v\n", qid, err)
		return
	}
	defer queue.Close()

	inMsg := buffer.NewInMessage()
	if mmapBuf, err := unix.Mmap(-1, 0, len(inMsg.Storage()), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANON|unix.MAP_PRIVATE); err == nil {
		inMsg.SetStorage(mmapBuf)
		defer unix.Munmap(mmapBuf)
	}
	outMsg := new(buffer.OutMessage)
	outMsg.Reset()

	// 2. Initial Registration: Tell kernel to register this buffer and fetch first request
	queue.pushCommand(fusekernel.FuseIoUringCmdRegister, qid, 0, c.DevFd(), inMsg.Storage())

	for {
		// 3. Wait for CQE (FUSE request deposited directly into inMsg.Storage() by kernel)
		cqeRes, err := queue.waitEvent()
		if err != nil || cqeRes < 0 {
			log.Printf("[FUSE_OVER_IO_URING Worker Exiting] QID=%d waitCQE returned cqeRes=%d err=%v\n", qid, cqeRes, err)
			break // Ring closed, connection terminated, or unmounted
		}

		s.opsInFlight.Add(1)
		op, _ := c.ParseInMessage(inMsg, outMsg)
		ctx := c.BeginOp(inMsg.Header().Opcode, inMsg.Header().Unique)

		outMsg.Reset()
		_ = s.handleOpSync(c, ctx, op, outMsg)
		s.opsInFlight.Done()

		// 4. Atomic Commit & Fetch: Submit reply & fetch NEXT request in ONE io_uring cmd!
		queue.pushCommand(fusekernel.FuseIoUringCmdCommitAndFetch, qid, inMsg.Header().Unique, c.DevFd(), outMsg.Bytes())
	}
}

type uringIovec struct {
	base uintptr
	len  uint64
}

// uringQueue manages a single io_uring submission/completion ring with 128-byte SQEs.
type uringQueue struct {
	fd          int
	sqes        []byte
	sqRing      []byte
	cqRing      []byte
	sqHead      *uint32
	sqTail      *uint32
	sqMask      *uint32
	sqArray     []uint32
	cqHead      *uint32
	cqTail      *uint32
	cqMask      *uint32
	cqes        []byte
	iov         uringIovec
}

const (
	sys_IO_URING_SETUP        = 425
	ioRING_OFF_SQ_RING uint64 = 0
	ioRING_OFF_CQ_RING uint64 = 0x8000000
	ioRING_OFF_SQES    uint64 = 0x10000000
)

type ioSqringOffsets struct {
	Head        uint32
	Tail        uint32
	RingMask    uint32
	RingEntries uint32
	Flags       uint32
	Dropped     uint32
	Array       uint32
	Resv1       uint32
	Resv2       uint64
}

type ioCqringOffsets struct {
	Head        uint32
	Tail        uint32
	RingMask    uint32
	RingEntries uint32
	Overflow    uint32
	Cqes        uint32
	Flags       uint32
	Resv1       uint32
	Resv2       uint64
}

type ioUringParams struct {
	SqEntries    uint32
	CqEntries    uint32
	Flags        uint32
	SqThreadCpu  uint32
	SqThreadIdle uint32
	Features     uint32
	WqFd         uint32
	Resv         [3]uint32
	SqOff        ioSqringOffsets
	CqOff        ioCqringOffsets
}

func setupIoUring(entries uint32, params *ioUringParams) (int, error) {
	r1, _, errno := syscall.Syscall(sys_IO_URING_SETUP, uintptr(entries), uintptr(unsafe.Pointer(params)), 0)
	if errno != 0 {
		return -1, errno
	}
	return int(r1), nil
}

func newUringQueue(entries uint32) (*uringQueue, error) {
	const IORING_SETUP_SQE128 uint32 = 1 << 10
	params := &ioUringParams{
		Flags: IORING_SETUP_SQE128,
	}

	fd, err := setupIoUring(entries, params)
	if err != nil {
		return nil, err
	}

	sqes, err := unix.Mmap(fd, int64(ioRING_OFF_SQES), int(params.SqEntries)*128, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED|unix.MAP_POPULATE)
	if err != nil {
		_ = unix.Close(fd)
		return nil, err
	}

	sqRingSize := params.SqOff.Array + params.SqEntries*4
	sqRing, err := unix.Mmap(fd, int64(ioRING_OFF_SQ_RING), int(sqRingSize), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED|unix.MAP_POPULATE)
	if err != nil {
		_ = unix.Munmap(sqes)
		_ = unix.Close(fd)
		return nil, err
	}

	cqRingSize := params.CqOff.Cqes + params.CqEntries*16
	cqRing, err := unix.Mmap(fd, int64(ioRING_OFF_CQ_RING), int(cqRingSize), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED|unix.MAP_POPULATE)
	if err != nil {
		_ = unix.Munmap(sqRing)
		_ = unix.Munmap(sqes)
		_ = unix.Close(fd)
		return nil, err
	}

	q := &uringQueue{
		fd:     fd,
		sqes:   sqes,
		sqRing: sqRing,
		cqRing: cqRing,
		sqHead: (*uint32)(unsafe.Pointer(&sqRing[params.SqOff.Head])),
		sqTail: (*uint32)(unsafe.Pointer(&sqRing[params.SqOff.Tail])),
		sqMask: (*uint32)(unsafe.Pointer(&sqRing[params.SqOff.RingMask])),
		cqHead: (*uint32)(unsafe.Pointer(&cqRing[params.CqOff.Head])),
		cqTail: (*uint32)(unsafe.Pointer(&cqRing[params.CqOff.Tail])),
		cqMask: (*uint32)(unsafe.Pointer(&cqRing[params.CqOff.RingMask])),
	}

	arrayPtr := (*[1 << 20]uint32)(unsafe.Pointer(&sqRing[params.SqOff.Array]))
	q.sqArray = arrayPtr[:params.SqEntries]

	cqesPtr := (*[1 << 20]byte)(unsafe.Pointer(&cqRing[params.CqOff.Cqes]))
	q.cqes = cqesPtr[:int(params.CqEntries)*16]

	return q, nil
}

func (q *uringQueue) Close() {
	if q.cqRing != nil {
		_ = unix.Munmap(q.cqRing)
	}
	if q.sqRing != nil {
		_ = unix.Munmap(q.sqRing)
	}
	if q.sqes != nil {
		_ = unix.Munmap(q.sqes)
	}
	if q.fd > 0 {
		_ = unix.Close(q.fd)
	}
}

// pushCommand formats a 128-byte SQE with IORING_OP_URING_CMD and pushes it to the kernel.
func (q *uringQueue) pushCommand(cmdOp uint32, qid uint16, commitID uint64, devFd int, payload []byte) {
	tail := atomic.LoadUint32(q.sqTail)
	idx := tail & *q.sqMask
	sqeOff := int(idx) * 128
	sqe := q.sqes[sqeOff : sqeOff+128]
	for i := range sqe {
		sqe[i] = 0
	}

	const IORING_OP_URING_CMD uint8 = 46
	sqe[0] = IORING_OP_URING_CMD                          // Opcode (offset 0)
	binary.LittleEndian.PutUint32(sqe[4:8], uint32(devFd)) // Fd (offset 4..7)
	binary.LittleEndian.PutUint32(sqe[8:12], cmdOp)        // cmd_op (offset 8..11)

	if len(payload) > 0 {
		q.iov.base = uintptr(unsafe.Pointer(&payload[0]))
		q.iov.len = uint64(len(payload))
		binary.LittleEndian.PutUint64(sqe[16:24], uint64(uintptr(unsafe.Pointer(&q.iov))))
		binary.LittleEndian.PutUint32(sqe[24:28], 1) // 1 segment/iovec
	} else {
		binary.LittleEndian.PutUint32(sqe[24:28], 0)
	}

	binary.LittleEndian.PutUint32(sqe[28:32], 0)           // UringCmdFlags (offset 28..31)
	binary.LittleEndian.PutUint64(sqe[32:40], uint64(qid)) // user_data (offset 32..39)

	// Write struct fuse_uring_cmd_req { uint64 flags; uint64 commit_id; uint16 qid; uint8 padding[6]; }
	// into the 80-byte SQE command payload area right at offset 48:
	cmdArea := sqe[48 : 48+24]
	binary.LittleEndian.PutUint64(cmdArea[0:8], 0)         // flags
	binary.LittleEndian.PutUint64(cmdArea[8:16], commitID) // commit_id
	binary.LittleEndian.PutUint16(cmdArea[16:18], qid)     // qid

	q.sqArray[idx] = idx
	atomic.StoreUint32(q.sqTail, tail+1)

	// Enter syscall to push SQE and wake kernel worker
	_, _, _ = syscall.Syscall6(unix.SYS_IO_URING_ENTER, uintptr(q.fd), 1, 0, 0, 0, 0)
}

// waitEvent checks the Completion Queue and blocks if necessary for 1 CQE.
func (q *uringQueue) waitEvent() (int32, error) {
	for {
		head := atomic.LoadUint32(q.cqHead)
		tail := atomic.LoadUint32(q.cqTail)
		if head != tail {
			idx := head & *q.cqMask
			cqeOff := int(idx) * 16
			cqe := q.cqes[cqeOff : cqeOff+16]
			res := int32(binary.LittleEndian.Uint32(cqe[8:12]))
			atomic.StoreUint32(q.cqHead, head+1)
			return res, nil
		}
		// Enter syscall with IORING_ENTER_GETEVENTS (bit 0) to wait for at least 1 CQE
		_, _, errno := syscall.Syscall6(unix.SYS_IO_URING_ENTER, uintptr(q.fd), 0, 1, 1, 0, 0)
		if errno != 0 && errno != syscall.EINTR {
			return -1, errno
		}
	}
}