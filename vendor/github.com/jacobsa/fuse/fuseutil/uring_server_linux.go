//go:build linux
// +build linux

package fuseutil

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/internal/buffer"
	"github.com/jacobsa/fuse/internal/fusekernel"
	"golang.org/x/sys/unix"
)

func raiseMemlockLimit() error {
	// Try standard Setrlimit first (succeeds if running as root)
	limit := unix.Rlimit{
		Cur: unix.RLIM_INFINITY,
		Max: unix.RLIM_INFINITY,
	}
	err := unix.Setrlimit(unix.RLIMIT_MEMLOCK, &limit)
	if err == nil {
		log.Printf("[FUSE_OVER_IO_URING] Raised RLIMIT_MEMLOCK using direct Setrlimit.")
		return nil
	}

	// Fallback: Use sudo prlimit to set limit for the current process
	pid := os.Getpid()
	log.Printf("[FUSE_OVER_IO_URING Debug] Direct Setrlimit failed (%v). Attempting sudo prlimit fallback for PID %d...", err, pid)
	cmd := exec.Command("sudo", "-n", "prlimit", "--pid", strconv.Itoa(pid), "--memlock=unlimited:unlimited")
	if out, serr := cmd.CombinedOutput(); serr != nil {
		return fmt.Errorf("sudo prlimit failed: %v, output: %q", serr, string(out))
	}

	log.Printf("[FUSE_OVER_IO_URING] Raised RLIMIT_MEMLOCK successfully using sudo prlimit fallback.")
	return nil
}

func inputStructSize(opcode uint32) int {
	switch opcode {
	case fusekernel.OpLookup, fusekernel.OpGetattr, fusekernel.OpSymlink,
		fusekernel.OpUnlink, fusekernel.OpRmdir, fusekernel.OpOpendir,
		fusekernel.OpReadlink, fusekernel.OpStatfs, fusekernel.OpRemovexattr:
		return 0

	case fusekernel.OpSetattr:
		return int(unsafe.Sizeof(fusekernel.SetattrIn{}))
	case fusekernel.OpForget:
		return int(unsafe.Sizeof(fusekernel.ForgetIn{}))
	case fusekernel.OpBatchForget:
		return int(unsafe.Sizeof(fusekernel.BatchForgetCountIn{}))
	case fusekernel.OpMkdir:
		return int(unsafe.Sizeof(fusekernel.MkdirIn{}))
	case fusekernel.OpMknod:
		return int(unsafe.Sizeof(fusekernel.MknodIn{}))
	case fusekernel.OpCreate:
		return int(unsafe.Sizeof(fusekernel.CreateIn{}))
	case fusekernel.OpRename:
		return int(unsafe.Sizeof(fusekernel.RenameIn{}))
	case fusekernel.OpOpen:
		return int(unsafe.Sizeof(fusekernel.OpenIn{}))
	case fusekernel.OpRead, fusekernel.OpReaddir, fusekernel.OpReaddirplus:
		return int(unsafe.Sizeof(fusekernel.ReadIn{}))
	case fusekernel.OpRelease, fusekernel.OpReleasedir:
		return int(unsafe.Sizeof(fusekernel.ReleaseIn{}))
	case fusekernel.OpWrite:
		return int(unsafe.Sizeof(fusekernel.WriteIn{}))
	case fusekernel.OpSyncFS:
		return int(unsafe.Sizeof(fusekernel.SyncFSIn{}))
	case fusekernel.OpFlush:
		return int(unsafe.Sizeof(fusekernel.FlushIn{}))
	case fusekernel.OpInterrupt:
		return int(unsafe.Sizeof(fusekernel.InterruptIn{}))
	case fusekernel.OpInit:
		return int(unsafe.Sizeof(fusekernel.InitIn{}))
	case fusekernel.OpLink:
		return int(unsafe.Sizeof(fusekernel.LinkIn{}))
	case fusekernel.OpGetxattr:
		return int(unsafe.Sizeof(fusekernel.GetxattrIn{}))
	case fusekernel.OpListxattr:
		return int(unsafe.Sizeof(fusekernel.ListxattrIn{}))
	case fusekernel.OpSetxattr:
		return int(unsafe.Sizeof(fusekernel.SetxattrIn{}))
	case fusekernel.OpFallocate:
		return int(unsafe.Sizeof(fusekernel.FallocateIn{}))
	}
	return 0
}

// serveOpsOverIoUring is called from ServeOps when c.UsingIoUring() == true on Linux.
func (s *fileSystemServer) serveOpsOverIoUring(c *fuse.Connection, _ int, numQueues int) {
	if err := raiseMemlockLimit(); err != nil {
		log.Printf("[FUSE_OVER_IO_URING Warning] Failed to raise RLIMIT_MEMLOCK to infinity: %v. Running as non-root may restrict setup memory.", err)
	} else {
		log.Printf("[FUSE_OVER_IO_URING] Successfully raised RLIMIT_MEMLOCK to infinity.")
	}

	log.Printf("[FUSE_OVER_IO_URING] Setting up %d independent io_uring queues for DevFd=%d\n", numQueues, c.DevFd())

	const queueDepth = 2
	queues := make([]*uringQueue, numQueues)

	// 1. SEQUENTIAL creation and registration of queues to prevent kernel lock contention
	for qid := 0; qid < numQueues; qid++ {
		queue, err := newUringQueue(queueDepth)
		if err != nil {
			log.Printf("[FUSE_OVER_IO_URING Error] Failed to setup io_uring for QID=%d: %v\n", qid, err)
			// Cleanup previously setup queues
			for idx := 0; idx < qid; idx++ {
				queues[idx].Close()
			}
			return
		}
		queues[qid] = queue

		// Push initial REGISTER commands for all slots
		for i := 0; i < queueDepth; i++ {
			err := queue.pushCommand(fusekernel.FuseIoUringCmdRegister, uint16(qid), 0, c.DevFd(), i, nil)
			if err != nil {
				log.Printf("[FUSE_OVER_IO_URING Error] QID=%d Failed to push initial register for slot %d: %v\n", qid, i, err)
				for idx := 0; idx <= qid; idx++ {
					queues[idx].Close()
				}
				return
			}
		}

		// Wait for completions of all REGISTER commands for this queue
		for i := 0; i < queueDepth; i++ {
			slotIdx, cqeRes, err := queue.waitEvent()
			if err != nil || cqeRes < 0 {
				log.Printf("[FUSE_OVER_IO_URING Error] QID=%d waitCQE registration returned slotIdx=%d cqeRes=%d err=%v\n", qid, slotIdx, cqeRes, err)
				for idx := 0; idx <= qid; idx++ {
					queues[idx].Close()
				}
				return
			}
			log.Printf("[FUSE_OVER_IO_URING Debug] QID=%d Slot=%d registered successfully with kernel.", qid, slotIdx)
		}
	}

	log.Printf("[FUSE_OVER_IO_URING] All %d queues registered successfully with kernel! Starting workers...\n", numQueues)

	defer func() {
		log.Printf("[FUSE_OVER_IO_URING] All worker queues stopping. Destroying filesystem and unmounting.\n")
		s.opsInFlight.Wait()
		s.fs.Destroy()
		for _, q := range queues {
			q.Close()
		}
	}()

	// 2. Start worker threads (they can now go straight to the event loop)
	var wg sync.WaitGroup
	for qid := 0; qid < numQueues; qid++ {
		wg.Add(1)
		go func(queueID uint16, q *uringQueue) {
			defer wg.Done()
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			s.runUringWorkerLoopWithQueue(c, queueID, q)
		}(uint16(qid), queues[qid])
	}
	wg.Wait()
}

func (s *fileSystemServer) runUringWorkerLoopWithQueue(c *fuse.Connection, qid uint16, queue *uringQueue) {
	inMsg := buffer.NewInMessage()
	outMsg := new(buffer.OutMessage)

	for {
		outMsg.Reset()

		// Wait for CQE (next request)
		slotIdx, cqeRes, err := queue.waitEvent()
		if err != nil || cqeRes < 0 {
			log.Printf("[FUSE_OVER_IO_URING Worker Exiting] QID=%d waitCQE returned slotIdx=%d cqeRes=%d err=%v\n", qid, slotIdx, cqeRes, err)
			break // Ring closed, connection terminated, or unmounted
		}

		slot := &queue.slots[slotIdx]

		// Extract headers and payload from the completed slot and construct a contiguous InMessage without gaps
		inHdrLen := 40
		opcode := binary.LittleEndian.Uint32(slot.header[4:8])
		structSize := inputStructSize(opcode)

		ent := (*fusekernel.FuseUringEntInOut)(unsafe.Pointer(&slot.header[256]))
		payloadSz := int(ent.PayloadSz)
		commitID := ent.CommitID // Save the kernel-provided transaction CommitID!

		// Copy InHeader (first 40 bytes)
		copy(inMsg.Storage()[0:inHdrLen], slot.header[0:inHdrLen])
		// Copy OpIn struct if present (without gap!)
		if structSize > 0 {
			copy(inMsg.Storage()[inHdrLen:inHdrLen+structSize], slot.header[128:128+structSize])
		}
		// Copy extra payload directly after the struct
		if payloadSz > 0 {
			copy(inMsg.Storage()[inHdrLen+structSize:inHdrLen+structSize+payloadSz], slot.payload[0:payloadSz])
		}

		totalRead := inHdrLen + structSize + payloadSz
		// Re-initialize InMessage state to point to the copied buffer
		inMsg.InitFromUring(totalRead, 40)

		log.Printf("[FUSE_OVER_IO_URING Trace] QID=%d Slot=%d CQERes=%d Opcode=%d StructSize=%d PayloadSz=%d CommitID=%d Unique=%d Nodeid=%d\n",
			qid, slotIdx, cqeRes, opcode, structSize, payloadSz, commitID, inMsg.Header().Unique, inMsg.Header().Nodeid)

		// Parse and dispatch the operation
		s.opsInFlight.Add(1)
		op, parseErr := c.ParseInMessage(inMsg, outMsg)
		if parseErr != nil {
			log.Printf("[FUSE_OVER_IO_URING Error] QID=%d ParseInMessage failed: %v\n", qid, parseErr)
		}
		
		// Use BeginUringOp to create context with stuffed opState!
		ctx := c.BeginUringOp(inMsg, outMsg, op)

		outMsg.Reset()
		handleErr := s.handleOpSync(c, ctx, op, outMsg)
		
		// Call Connection.Reply to serialize response!
		_ = c.Reply(ctx, handleErr)
		
		s.opsInFlight.Done()

		// Print response status and size
		respStatus := outMsg.OutHeader().Error
		respLen := outMsg.Len()
		log.Printf("[FUSE_OVER_IO_URING Trace] QID=%d Slot=%d Unique=%d OutStatus=%d OutLen=%d\n",
			qid, slotIdx, inMsg.Header().Unique, respStatus, respLen)

		// Atomic Commit & Fetch: Submit reply & fetch NEXT request in ONE io_uring cmd!
		err = queue.pushCommand(fusekernel.FuseIoUringCmdCommitAndFetch, qid, commitID, c.DevFd(), slotIdx, outMsg.Bytes())
		if err != nil {
			log.Printf("[FUSE_OVER_IO_URING] QID=%d Failed to submit commit and fetch for slot %d: %v\n", qid, slotIdx, err)
			break
		}
	}
}

// uringSlot represents one request slot buffer set.
type uringSlot struct {
	header  []byte         // 288 bytes (fuse_uring_req_header)
	payload []byte         // 1,048,576 bytes (max_write)
	iov     *[2]unix.Iovec // Pointer to the 2-element iovec array inside mmapBuf
}

// uringQueue manages a single io_uring submission/completion ring with 128-byte SQEs.
type uringQueue struct {
	fd      int
	sqes    []byte
	sqRing  []byte
	cqRing  []byte
	sqHead  *uint32
	sqTail  *uint32
	sqMask  *uint32
	sqArray []uint32
	cqHead  *uint32
	cqTail  *uint32
	cqMask  *uint32
	cqes    []byte
	mmapBuf []byte
	slots   []uringSlot
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
	var rlimit unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_MEMLOCK, &rlimit); err == nil {
		log.Printf("[FUSE_OVER_IO_URING Debug] newUringQueue: RLIMIT_MEMLOCK: cur=%d, max=%d", rlimit.Cur, rlimit.Max)
	} else {
		log.Printf("[FUSE_OVER_IO_URING Debug] newUringQueue: Getrlimit failed: %v", err)
	}

	const IORING_SETUP_SQE128 uint32 = 1 << 10
	params := &ioUringParams{
		Flags: IORING_SETUP_SQE128,
	}

	fd, err := setupIoUring(entries, params)
	if err != nil {
		log.Printf("[FUSE_OVER_IO_URING Debug] newUringQueue: setupIoUring failed: %v", err)
		return nil, err
	}

	sqes, err := unix.Mmap(fd, int64(ioRING_OFF_SQES), int(params.SqEntries)*128, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED|unix.MAP_POPULATE)
	if err != nil {
		log.Printf("[FUSE_OVER_IO_URING Debug] newUringQueue: Mmap sqes failed: %v", err)
		_ = unix.Close(fd)
		return nil, err
	}

	sqRingSize := params.SqOff.Array + params.SqEntries*4
	sqRing, err := unix.Mmap(fd, int64(ioRING_OFF_SQ_RING), int(sqRingSize), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED|unix.MAP_POPULATE)
	if err != nil {
		log.Printf("[FUSE_OVER_IO_URING Debug] newUringQueue: Mmap sqRing failed: %v", err)
		_ = unix.Munmap(sqes)
		_ = unix.Close(fd)
		return nil, err
	}

	cqRingSize := params.CqOff.Cqes + params.CqEntries*16
	cqRing, err := unix.Mmap(fd, int64(ioRING_OFF_CQ_RING), int(cqRingSize), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED|unix.MAP_POPULATE)
	if err != nil {
		log.Printf("[FUSE_OVER_IO_URING Debug] newUringQueue: Mmap cqRing failed: %v", err)
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

	const queueDepth = 2
	const pageSize = 4096
	const payloadSize = 1048576
	const slotSize = pageSize + payloadSize // 1052672 (page-aligned!)
	
	// Allocate slot memory + 32 bytes per slot for the iovec arrays at the end
	bufSize := queueDepth*slotSize + queueDepth*32
	mmapBuf, err := unix.Mmap(-1, 0, bufSize, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANON|unix.MAP_SHARED|unix.MAP_POPULATE)
	if err != nil {
		log.Printf("[FUSE_OVER_IO_URING Debug] newUringQueue: Mmap mmapBuf failed: %v", err)
		q.Close()
		return nil, err
	}
	q.mmapBuf = mmapBuf

	// Carve mmapBuf into page-aligned slots and setup iovecs at the end of the mmap block
	q.slots = make([]uringSlot, queueDepth)
	iovOffsetStart := queueDepth * slotSize

	for i := 0; i < int(queueDepth); i++ {
		slot := &q.slots[i]
		slotStart := i * slotSize
		
		// The FUSE header structure is 288 bytes, but starts at page boundary (offset 0 of slot)
		slot.header = mmapBuf[slotStart : slotStart+288]
		// The payload starts at next page boundary (offset 4096 of slot)
		slot.payload = mmapBuf[slotStart+pageSize : slotStart+pageSize+payloadSize]

		// The iovec array is stored at the end of mmapBuf
		iovAddr := &mmapBuf[iovOffsetStart + i*32]
		slot.iov = (*[2]unix.Iovec)(unsafe.Pointer(iovAddr))

		slot.iov[0] = unix.Iovec{
			Base: &slot.header[0], // Page-aligned!
			Len:  288,
		}
		slot.iov[1] = unix.Iovec{
			Base: &slot.payload[0], // Page-aligned!
			Len:  payloadSize,
		}
	}

	log.Printf("[FUSE_OVER_IO_URING Debug] newUringQueue: Slots populated successfully. Bypassed fixed buffer registration.\n")
	return q, nil
}

func (q *uringQueue) Close() {
	if q.mmapBuf != nil {
		_ = unix.Munmap(q.mmapBuf)
	}
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
func (q *uringQueue) pushCommand(cmdOp uint32, qid uint16, commitID uint64, devFd int, slotIdx int, payload []byte) error {
	tail := atomic.LoadUint32(q.sqTail)
	head := atomic.LoadUint32(q.sqHead)
	if tail-head >= uint32(len(q.sqArray)) {
		return fmt.Errorf("SQ Ring is full")
	}

	idx := tail & *q.sqMask
	sqeOff := int(idx) * 128
	sqe := q.sqes[sqeOff : sqeOff+128]
	for i := range sqe {
		sqe[i] = 0
	}

	slot := &q.slots[slotIdx]

	// Handle response write-back for COMMIT_AND_FETCH
	if cmdOp == fusekernel.FuseIoUringCmdCommitAndFetch && len(payload) > 0 {
		// Copy OutHeader (first 16 bytes of reply)
		copy(slot.header[0:16], payload[0:16])
		// Copy response payload (remaining bytes)
		payloadSz := len(payload) - 16
		if payloadSz > 0 {
			copy(slot.payload[0:payloadSz], payload[16:])
		}
		// Write payload size back to slot header metadata area
		ent := (*fusekernel.FuseUringEntInOut)(unsafe.Pointer(&slot.header[256]))
		ent.PayloadSz = uint32(payloadSz)
	}

	const IORING_OP_URING_CMD uint8 = 46
	sqe[0] = IORING_OP_URING_CMD                          // Opcode (offset 0)
	binary.LittleEndian.PutUint32(sqe[4:8], uint32(devFd)) // Fd (offset 4..7)
	binary.LittleEndian.PutUint32(sqe[8:12], cmdOp)        // cmd_op (offset 8..11)

	// Set addr to the address of the 2-element iovec array inside the mmap buffer
	binary.LittleEndian.PutUint64(sqe[16:24], uint64(uintptr(unsafe.Pointer(&slot.iov[0]))))
	binary.LittleEndian.PutUint32(sqe[24:28], 2) // 2 segments!
	binary.LittleEndian.PutUint16(sqe[40:42], 0) // buf_index = 0
	binary.LittleEndian.PutUint32(sqe[28:32], 0) // uring_cmd_flags = 0 (non-fixed)

	// Set user_data in SQE to the slot index
	binary.LittleEndian.PutUint64(sqe[32:40], uint64(slotIdx))

	// Write struct fuse_uring_cmd_req { uint64 flags; uint64 commit_id; uint16 qid; uint8 padding[6]; }
	cmdArea := sqe[48 : 48+24]
	binary.LittleEndian.PutUint64(cmdArea[0:8], 0)         // flags
	binary.LittleEndian.PutUint64(cmdArea[8:16], commitID) // commit_id
	binary.LittleEndian.PutUint16(cmdArea[16:18], qid)     // qid

	q.sqArray[idx] = idx
	atomic.StoreUint32(q.sqTail, tail+1)

	log.Printf("[FUSE_OVER_IO_URING Debug] pushCommand: cmdOp=%d qid=%d slotIdx=%d commitID=%d devFd=%d\n",
		cmdOp, qid, slotIdx, commitID, devFd)

	// Enter syscall to push SQE and wake kernel worker
	_, _, _ = syscall.Syscall6(unix.SYS_IO_URING_ENTER, uintptr(q.fd), 1, 0, 0, 0, 0)
	return nil
}

// waitEvent checks the Completion Queue and blocks if necessary for 1 CQE.
func (q *uringQueue) waitEvent() (slotIdx int, cqeRes int32, err error) {
	for {
		head := atomic.LoadUint32(q.cqHead)
		tail := atomic.LoadUint32(q.cqTail)
		if head != tail {
			idx := head & *q.cqMask
			cqeOff := int(idx) * 16
			cqe := q.cqes[cqeOff : cqeOff+16]
			slotIdx = int(binary.LittleEndian.Uint64(cqe[0:8]))   // user_data is offset 0..7
			cqeRes = int32(binary.LittleEndian.Uint32(cqe[8:12])) // res is offset 8..11
			atomic.StoreUint32(q.cqHead, head+1)
			return slotIdx, cqeRes, nil
		}
		// Enter syscall with IORING_ENTER_GETEVENTS (bit 0) to wait for at least 1 CQE
		_, _, errno := syscall.Syscall6(unix.SYS_IO_URING_ENTER, uintptr(q.fd), 0, 1, 1, 0, 0)
		if errno != 0 && errno != syscall.EINTR {
			return -1, -1, errno
		}
	}
}
