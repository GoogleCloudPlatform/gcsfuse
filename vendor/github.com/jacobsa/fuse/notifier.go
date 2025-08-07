package fuse

import (
	"unsafe"

	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/internal/fusekernel"
)

// Notifier coordinates low-level notifications from the fuse daemon to the
// kernel. A Notifier may be used by the ServeOps implementation of a Server. In
// order to deliver notifications, wrap the server with NewServerWithNotifier.
type Notifier struct {
	inodeInvalidations  chan invalidateInodeCommand
	dentryInvalidations chan invalidateEntryCommand
}

func NewNotifier() *Notifier {
	return &Notifier{
		inodeInvalidations:  make(chan invalidateInodeCommand),
		dentryInvalidations: make(chan invalidateEntryCommand),
	}
}

type invalidateInodeCommand struct {
	inode  fuseops.InodeID
	offset int64
	length int64
	done   chan<- error
}

type invalidateEntryCommand struct {
	parent fuseops.InodeID
	name   string
	// If fusekernel.NotifyInvalEntryOut is updated to use its padding as flags,
	// we can support the expire flag in this command as well.
	done chan<- error
}

// InvalidateInode notifies the kernel to invalidate an inode cache entry. See
// the libfuse documentation at
// https://libfuse.github.io/doxygen/fuse__lowlevel_8h.html#a9cb974af9745294ff446d11cba2422f1
// for more details.
//
// InvalidateInode blocks until the kernel write completes, and returns the
// error from the kernel, if any. ENOSYS indicates that the kernel does not
// support inode invalidations.
func (n *Notifier) InvalidateInode(inode fuseops.InodeID, offset, length int64) error {
	done := make(chan error)
	n.inodeInvalidations <- invalidateInodeCommand{inode, offset, length, done}
	return <-done
}

// InvalidateEntry notifies to the kernel to invalidate a dentry cache entry.
// See the libfuse documentation at
// https://libfuse.github.io/doxygen/fuse__lowlevel_8h.html#ab14032b74b0a57a2b3155dd6ba8d6095
// for more details.
//
// InvalidateEntry blocks until the kernel write completes, and returns the
// error from the kernel, if any. ENOSYS indicates that the kernel does not
// support dentry invalidations.
func (n *Notifier) InvalidateEntry(parent fuseops.InodeID, name string) error {
	done := make(chan error)
	n.dentryInvalidations <- invalidateEntryCommand{parent, name, done}
	return <-done
}

func serviceInodeInvalidation(c *Connection, inode fuseops.InodeID, offset, length int64) error {
	outMsg := c.getOutMessage()
	defer c.putOutMessage(outMsg)

	cmd := fusekernel.NotifyInvalInodeOut{
		Ino: uint64(inode),
		Off: offset,
		Len: length,
	}
	outMsg.Append(unsafe.Slice((*byte)(unsafe.Pointer(&cmd)), int(unsafe.Sizeof(cmd))))

	outMsg.OutHeader().Error = fusekernel.NotifyCodeInvalInode
	outMsg.OutHeader().Len = uint32(outMsg.Len())

	return c.writeOutMessage(outMsg)
}

func serviceEntryInval(c *Connection, parent fuseops.InodeID, name string) error {
	outMsg := c.getOutMessage()
	defer c.putOutMessage(outMsg)

	cmd := fusekernel.NotifyInvalEntryOut{
		Parent:  uint64(parent),
		Namelen: uint32(len(name)),
	}
	outMsg.Append(unsafe.Slice((*byte)(unsafe.Pointer(&cmd)), int(unsafe.Sizeof(cmd))))

	// The name must be represented as a C string with a null-terminator.
	outMsg.AppendString(name)
	outMsg.Append([]byte{0})

	outMsg.OutHeader().Error = fusekernel.NotifyCodeInvalEntry
	outMsg.OutHeader().Len = uint32(outMsg.Len())
	return c.writeOutMessage(outMsg)
}

func (n *Notifier) notify(c *Connection, terminate <-chan struct{}) {
	for {
		select {
		case i := <-n.inodeInvalidations:
			i.done <- serviceInodeInvalidation(c, i.inode, i.offset, i.length)
		case e := <-n.dentryInvalidations:
			e.done <- serviceEntryInval(c, e.parent, e.name)
		case <-terminate:
			return
		}
	}
}

type notifierServer struct {
	n *Notifier
	s Server
}

func (s *notifierServer) ServeOps(c *Connection) {
	terminate := make(chan struct{})

	go s.n.notify(c, terminate)
	s.s.ServeOps(c)
	close(terminate)
}

func NewServerWithNotifier(n *Notifier, s Server) Server {
	return &notifierServer{n, s}
}
