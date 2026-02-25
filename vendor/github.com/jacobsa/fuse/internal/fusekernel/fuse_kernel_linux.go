package fusekernel

import (
	"syscall"
	"time"
)

type Attr struct {
	Ino       uint64
	Size      uint64
	Blocks    uint64
	Atime     uint64
	Mtime     uint64
	Ctime     uint64
	AtimeNsec uint32
	MtimeNsec uint32
	CtimeNsec uint32
	Mode      uint32
	Nlink     uint32
	Uid       uint32
	Gid       uint32
	Rdev      uint32
	Blksize   uint32
	padding   uint32
}

func (a *Attr) Crtime() time.Time {
	return time.Time{}
}

func (a *Attr) SetCrtime(s uint64, ns uint32) {
	// Ignored on Linux.
}

func (a *Attr) SetFlags(f uint32) {
	// Ignored on Linux.
}

type SetattrIn struct {
	setattrInCommon
}

func (in *SetattrIn) BkupTime() time.Time {
	return time.Time{}
}

func (in *SetattrIn) Chgtime() time.Time {
	return time.Time{}
}

func (in *SetattrIn) Flags() uint32 {
	return 0
}

const OpenDirect OpenFlags = syscall.O_DIRECT

// Return true if OpenDirect is set.
func (fl OpenFlags) IsDirect() bool {
	return fl&OpenDirect != 0
}

func init() {
	openFlagNames = append(openFlagNames, flagName{
		bit:  uint32(OpenDirect),
		name: "OpenDirect",
	})
}

type GetxattrIn struct {
	getxattrInCommon
}

type SetxattrIn struct {
	setxattrInCommon
}
