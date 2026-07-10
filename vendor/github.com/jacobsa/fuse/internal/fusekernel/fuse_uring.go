package fusekernel

const (
	// InitFlag indicating that both kernel and client support FUSE over io_uring.
	//	InitOverIoUring uint64 = 1 << 41
	// FUSE uring SQE command opcodes (in SQE.cmd_op when opcode == IORING_OP_URING_CMD)
	FuseIoUringCmdInvalid        = 0
	FuseIoUringCmdRegister       = 1 // Register request buffer & fetch first request
	FuseIoUringCmdCommitAndFetch = 2 // Commit response & fetch next request atomic loop
)

// FuseUringCmdReq resides inside the 80-byte SQE command payload area (`sqe.cmd`).
type FuseUringCmdReq struct {
	Flags    uint64
	CommitID uint64 // Request ID of the completed op being committed
	QID      uint16 // Queue Index (per-CPU or worker queue ID)
	Padding  [6]uint8
}

// FuseUringEntInOut is the layout of the ring memory buffer registered with the kernel.
type FuseUringEntInOut struct {
	Flags     uint64
	CommitID  uint64
	PayloadSz uint32
	Padding   uint32
}
