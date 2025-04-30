package fs_util

import (
	"github.com/jacobsa/fuse/fuseops"
)

// OpenMode represents the file access mode.
type OpenMode int

// Available file open modes.
const (
	Read OpenMode = iota
	Write
	Append
)

// DetermineFileOpenMode analyzes the open flags to determine the file's open mode.
// It returns Read, Write, or Append.
//
// Append mode is returned for both 'a' and 'a+' flags as the kernel handles the distinction.
// Similarly, read/write ('r+') mode is implicitly handled by the kernel, so this function
// focuses on read-only, write-only, and append modes.
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
