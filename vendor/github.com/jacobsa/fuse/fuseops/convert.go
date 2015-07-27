// Copyright 2015 Google Inc. All Rights Reserved.
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

package fuseops

import (
	"bytes"
	"errors"
	"log"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/jacobsa/fuse/internal/buffer"
	"github.com/jacobsa/fuse/internal/fusekernel"
	"golang.org/x/net/context"
)

// This function is an implementation detail of the fuse package, and must not
// be called by anyone else.
//
// Convert the supplied fuse kernel message to an Op. sendReply will be used to
// send the reply back to the kernel once the user calls o.Respond. If the op
// is unknown, a special unexported type will be used.
//
// The debug logging function and error logger may be nil. The caller is
// responsible for arranging for the message to be destroyed.
func Convert(
	opCtx context.Context,
	m *buffer.InMessage,
	protocol fusekernel.Protocol,
	debugLogForOp func(int, string, ...interface{}),
	errorLogger *log.Logger,
	sendReply replyFunc) (o Op, err error) {
	var co *commonOp

	var io internalOp
	switch m.Header().Opcode {
	case fusekernel.OpLookup:
		buf := m.ConsumeBytes(m.Len())
		n := len(buf)
		if n == 0 || buf[n-1] != '\x00' {
			err = errors.New("Corrupt OpLookup")
			return
		}

		to := &LookUpInodeOp{
			protocol: protocol,
			Parent:   InodeID(m.Header().Nodeid),
			Name:     string(buf[:n-1]),
		}
		io = to
		co = &to.commonOp

	case fusekernel.OpGetattr:
		to := &GetInodeAttributesOp{
			protocol: protocol,
			Inode:    InodeID(m.Header().Nodeid),
		}
		io = to
		co = &to.commonOp

	case fusekernel.OpSetattr:
		type input fusekernel.SetattrIn
		in := (*input)(m.Consume(unsafe.Sizeof(input{})))
		if in == nil {
			err = errors.New("Corrupt OpSetattr")
			return
		}

		to := &SetInodeAttributesOp{
			protocol: protocol,
			Inode:    InodeID(m.Header().Nodeid),
		}

		valid := fusekernel.SetattrValid(in.Valid)
		if valid&fusekernel.SetattrSize != 0 {
			to.Size = &in.Size
		}

		if valid&fusekernel.SetattrMode != 0 {
			mode := convertFileMode(in.Mode)
			to.Mode = &mode
		}

		if valid&fusekernel.SetattrAtime != 0 {
			t := time.Unix(int64(in.Atime), int64(in.AtimeNsec))
			to.Atime = &t
		}

		if valid&fusekernel.SetattrMtime != 0 {
			t := time.Unix(int64(in.Mtime), int64(in.MtimeNsec))
			to.Mtime = &t
		}

		io = to
		co = &to.commonOp

	case fusekernel.OpForget:
		type input fusekernel.ForgetIn
		in := (*input)(m.Consume(unsafe.Sizeof(input{})))
		if in == nil {
			err = errors.New("Corrupt OpForget")
			return
		}

		to := &ForgetInodeOp{
			Inode: InodeID(m.Header().Nodeid),
			N:     in.Nlookup,
		}
		io = to
		co = &to.commonOp

	case fusekernel.OpMkdir:
		in := (*fusekernel.MkdirIn)(m.Consume(fusekernel.MkdirInSize(protocol)))
		if in == nil {
			err = errors.New("Corrupt OpMkdir")
			return
		}

		name := m.ConsumeBytes(m.Len())
		i := bytes.IndexByte(name, '\x00')
		if i < 0 {
			err = errors.New("Corrupt OpMkdir")
			return
		}
		name = name[:i]

		to := &MkDirOp{
			protocol: protocol,
			Parent:   InodeID(m.Header().Nodeid),
			Name:     string(name),

			// On Linux, vfs_mkdir calls through to the inode with at most
			// permissions and sticky bits set (cf. https://goo.gl/WxgQXk), and fuse
			// passes that on directly (cf. https://goo.gl/f31aMo). In other words,
			// the fact that this is a directory is implicit in the fact that the
			// opcode is mkdir. But we want the correct mode to go through, so ensure
			// that os.ModeDir is set.
			Mode: convertFileMode(in.Mode) | os.ModeDir,
		}

		io = to
		co = &to.commonOp

	case fusekernel.OpCreate:
		in := (*fusekernel.CreateIn)(m.Consume(fusekernel.CreateInSize(protocol)))
		if in == nil {
			err = errors.New("Corrupt OpCreate")
			return
		}

		name := m.ConsumeBytes(m.Len())
		i := bytes.IndexByte(name, '\x00')
		if i < 0 {
			err = errors.New("Corrupt OpCreate")
			return
		}
		name = name[:i]

		to := &CreateFileOp{
			protocol: protocol,
			Parent:   InodeID(m.Header().Nodeid),
			Name:     string(name),
			Mode:     convertFileMode(in.Mode),
		}
		io = to
		co = &to.commonOp

	case fusekernel.OpSymlink:
		// The message is "newName\0target\0".
		names := m.ConsumeBytes(m.Len())
		if len(names) == 0 || names[len(names)-1] != 0 {
			err = errors.New("Corrupt OpSymlink")
			return
		}
		i := bytes.IndexByte(names, '\x00')
		if i < 0 {
			err = errors.New("Corrupt OpSymlink")
			return
		}
		newName, target := names[0:i], names[i+1:len(names)-1]

		to := &CreateSymlinkOp{
			protocol: protocol,
			Parent:   InodeID(m.Header().Nodeid),
			Name:     string(newName),
			Target:   string(target),
		}
		io = to
		co = &to.commonOp

	case fusekernel.OpRename:
		type input fusekernel.RenameIn
		in := (*input)(m.Consume(unsafe.Sizeof(input{})))
		if in == nil {
			err = errors.New("Corrupt OpRename")
			return
		}

		names := m.ConsumeBytes(m.Len())
		// names should be "old\x00new\x00"
		if len(names) < 4 {
			err = errors.New("Corrupt OpRename")
			return
		}
		if names[len(names)-1] != '\x00' {
			err = errors.New("Corrupt OpRename")
			return
		}
		i := bytes.IndexByte(names, '\x00')
		if i < 0 {
			err = errors.New("Corrupt OpRename")
			return
		}
		oldName, newName := names[:i], names[i+1:len(names)-1]

		to := &RenameOp{
			OldParent: InodeID(m.Header().Nodeid),
			OldName:   string(oldName),
			NewParent: InodeID(in.Newdir),
			NewName:   string(newName),
		}
		io = to
		co = &to.commonOp

	case fusekernel.OpUnlink:
		buf := m.ConsumeBytes(m.Len())
		n := len(buf)
		if n == 0 || buf[n-1] != '\x00' {
			err = errors.New("Corrupt OpUnlink")
			return
		}

		to := &UnlinkOp{
			Parent: InodeID(m.Header().Nodeid),
			Name:   string(buf[:n-1]),
		}
		io = to
		co = &to.commonOp

	case fusekernel.OpRmdir:
		buf := m.ConsumeBytes(m.Len())
		n := len(buf)
		if n == 0 || buf[n-1] != '\x00' {
			err = errors.New("Corrupt OpRmdir")
			return
		}

		to := &RmDirOp{
			Parent: InodeID(m.Header().Nodeid),
			Name:   string(buf[:n-1]),
		}
		io = to
		co = &to.commonOp

	case fusekernel.OpOpen:
		to := &OpenFileOp{
			Inode: InodeID(m.Header().Nodeid),
		}
		io = to
		co = &to.commonOp

	case fusekernel.OpOpendir:
		to := &OpenDirOp{
			Inode: InodeID(m.Header().Nodeid),
		}
		io = to
		co = &to.commonOp

	case fusekernel.OpRead:
		in := (*fusekernel.ReadIn)(m.Consume(fusekernel.ReadInSize(protocol)))
		if in == nil {
			err = errors.New("Corrupt OpRead")
			return
		}

		to := &ReadFileOp{
			Inode:  InodeID(m.Header().Nodeid),
			Handle: HandleID(in.Fh),
			Offset: int64(in.Offset),
			Size:   int(in.Size),
		}
		io = to
		co = &to.commonOp

	case fusekernel.OpReaddir:
		in := (*fusekernel.ReadIn)(m.Consume(fusekernel.ReadInSize(protocol)))
		if in == nil {
			err = errors.New("Corrupt OpReaddir")
			return
		}

		to := &ReadDirOp{
			Inode:  InodeID(m.Header().Nodeid),
			Handle: HandleID(in.Fh),
			Offset: DirOffset(in.Offset),
			Size:   int(in.Size),
		}
		io = to
		co = &to.commonOp

	case fusekernel.OpRelease:
		type input fusekernel.ReleaseIn
		in := (*input)(m.Consume(unsafe.Sizeof(input{})))
		if in == nil {
			err = errors.New("Corrupt OpRelease")
			return
		}

		to := &ReleaseFileHandleOp{
			Handle: HandleID(in.Fh),
		}
		io = to
		co = &to.commonOp

	case fusekernel.OpReleasedir:
		type input fusekernel.ReleaseIn
		in := (*input)(m.Consume(unsafe.Sizeof(input{})))
		if in == nil {
			err = errors.New("Corrupt OpReleasedir")
			return
		}

		to := &ReleaseDirHandleOp{
			Handle: HandleID(in.Fh),
		}
		io = to
		co = &to.commonOp

	case fusekernel.OpWrite:
		in := (*fusekernel.WriteIn)(m.Consume(fusekernel.WriteInSize(protocol)))
		if in == nil {
			err = errors.New("Corrupt OpWrite")
			return
		}

		buf := m.ConsumeBytes(m.Len())
		if len(buf) < int(in.Size) {
			err = errors.New("Corrupt OpWrite")
			return
		}

		to := &WriteFileOp{
			Inode:  InodeID(m.Header().Nodeid),
			Handle: HandleID(in.Fh),
			Data:   buf,
			Offset: int64(in.Offset),
		}
		io = to
		co = &to.commonOp

	case fusekernel.OpFsync:
		type input fusekernel.FsyncIn
		in := (*input)(m.Consume(unsafe.Sizeof(input{})))
		if in == nil {
			err = errors.New("Corrupt OpFsync")
			return
		}

		to := &SyncFileOp{
			Inode:  InodeID(m.Header().Nodeid),
			Handle: HandleID(in.Fh),
		}
		io = to
		co = &to.commonOp

	case fusekernel.OpFlush:
		type input fusekernel.FlushIn
		in := (*input)(m.Consume(unsafe.Sizeof(input{})))
		if in == nil {
			err = errors.New("Corrupt OpFlush")
			return
		}

		to := &FlushFileOp{
			Inode:  InodeID(m.Header().Nodeid),
			Handle: HandleID(in.Fh),
		}
		io = to
		co = &to.commonOp

	case fusekernel.OpReadlink:
		to := &ReadSymlinkOp{
			Inode: InodeID(m.Header().Nodeid),
		}
		io = to
		co = &to.commonOp

	case fusekernel.OpStatfs:
		to := &InternalStatFSOp{}
		io = to
		co = &to.commonOp

	case fusekernel.OpInterrupt:
		type input fusekernel.InterruptIn
		in := (*input)(m.Consume(unsafe.Sizeof(input{})))
		if in == nil {
			err = errors.New("Corrupt OpInterrupt")
			return
		}

		to := &InternalInterruptOp{
			FuseID: in.Unique,
		}
		io = to
		co = &to.commonOp

	case fusekernel.OpInit:
		type input fusekernel.InitIn
		in := (*input)(m.Consume(unsafe.Sizeof(input{})))
		if in == nil {
			err = errors.New("Corrupt OpInit")
			return
		}

		to := &InternalInitOp{
			Kernel:       fusekernel.Protocol{in.Major, in.Minor},
			MaxReadahead: in.MaxReadahead,
			Flags:        fusekernel.InitFlags(in.Flags),
		}
		io = to
		co = &to.commonOp

	default:
		to := &unknownOp{
			opCode: m.Header().Opcode,
			inode:  InodeID(m.Header().Nodeid),
		}
		io = to
		co = &to.commonOp
	}

	co.init(
		opCtx,
		io,
		m.Header().Unique,
		sendReply,
		debugLogForOp,
		errorLogger)

	o = io
	return
}

func convertTime(t time.Time) (secs uint64, nsec uint32) {
	totalNano := t.UnixNano()
	secs = uint64(totalNano / 1e9)
	nsec = uint32(totalNano % 1e9)
	return
}

func convertAttributes(
	inodeID InodeID,
	in *InodeAttributes,
	out *fusekernel.Attr) {
	out.Ino = uint64(inodeID)
	out.Size = in.Size
	out.Atime, out.AtimeNsec = convertTime(in.Atime)
	out.Mtime, out.MtimeNsec = convertTime(in.Mtime)
	out.Ctime, out.CtimeNsec = convertTime(in.Ctime)
	out.SetCrtime(convertTime(in.Crtime))
	out.Nlink = uint32(in.Nlink) // TODO(jacobsa): Make the public field uint32?
	out.Uid = in.Uid
	out.Gid = in.Gid

	// Set the mode.
	out.Mode = uint32(in.Mode) & 0777
	switch {
	default:
		out.Mode |= syscall.S_IFREG
	case in.Mode&os.ModeDir != 0:
		out.Mode |= syscall.S_IFDIR
	case in.Mode&os.ModeDevice != 0:
		if in.Mode&os.ModeCharDevice != 0 {
			out.Mode |= syscall.S_IFCHR
		} else {
			out.Mode |= syscall.S_IFBLK
		}
	case in.Mode&os.ModeNamedPipe != 0:
		out.Mode |= syscall.S_IFIFO
	case in.Mode&os.ModeSymlink != 0:
		out.Mode |= syscall.S_IFLNK
	case in.Mode&os.ModeSocket != 0:
		out.Mode |= syscall.S_IFSOCK
	}
}

// Convert an absolute cache expiration time to a relative time from now for
// consumption by the fuse kernel module.
func convertExpirationTime(t time.Time) (secs uint64, nsecs uint32) {
	// Fuse represents durations as unsigned 64-bit counts of seconds and 32-bit
	// counts of nanoseconds (cf. http://goo.gl/EJupJV). So negative durations
	// are right out. There is no need to cap the positive magnitude, because
	// 2^64 seconds is well longer than the 2^63 ns range of time.Duration.
	d := t.Sub(time.Now())
	if d > 0 {
		secs = uint64(d / time.Second)
		nsecs = uint32((d % time.Second) / time.Nanosecond)
	}

	return
}

func convertChildInodeEntry(
	in *ChildInodeEntry,
	out *fusekernel.EntryOut) {
	out.Nodeid = uint64(in.Child)
	out.Generation = uint64(in.Generation)
	out.EntryValid, out.EntryValidNsec = convertExpirationTime(in.EntryExpiration)
	out.AttrValid, out.AttrValidNsec = convertExpirationTime(in.AttributesExpiration)

	convertAttributes(in.Child, &in.Attributes, &out.Attr)
}

func convertFileMode(unixMode uint32) os.FileMode {
	mode := os.FileMode(unixMode & 0777)
	switch unixMode & syscall.S_IFMT {
	case syscall.S_IFREG:
		// nothing
	case syscall.S_IFDIR:
		mode |= os.ModeDir
	case syscall.S_IFCHR:
		mode |= os.ModeCharDevice | os.ModeDevice
	case syscall.S_IFBLK:
		mode |= os.ModeDevice
	case syscall.S_IFIFO:
		mode |= os.ModeNamedPipe
	case syscall.S_IFLNK:
		mode |= os.ModeSymlink
	case syscall.S_IFSOCK:
		mode |= os.ModeSocket
	default:
		// no idea
		mode |= os.ModeDevice
	}
	if unixMode&syscall.S_ISUID != 0 {
		mode |= os.ModeSetuid
	}
	if unixMode&syscall.S_ISGID != 0 {
		mode |= os.ModeSetgid
	}
	return mode
}
