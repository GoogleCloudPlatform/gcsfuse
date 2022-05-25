//go:build !linux
// +build !linux

package fuse

import (
	"os"
	"syscall"
)

func unmount(dir string) error {
	if err := syscall.Unmount(dir, 0); err != nil {
		return &os.PathError{Op: "unmount", Path: dir, Err: err}
	}

	return nil
}
