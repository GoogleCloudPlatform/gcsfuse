package openMode

import (
	"github.com/jacobsa/fuse/fuseops"
)

type OpenMode int

const (
	Read OpenMode = iota
	Write
	Append
)

func DetermineFileOpenMode(op *fuseops.OpenFileOp) OpenMode {
	switch {
	case op.OpenFlags.IsAppend():
		return Append
	case op.OpenFlags.IsWriteOnly():
		return Write
	default:
		return Read
	}
}
