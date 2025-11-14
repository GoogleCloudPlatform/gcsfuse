package fuse

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func unmount(dir string) error {
	if err := fuserunmount(dir); err != nil {
		// Return custom error for fusermount unmount error for /dev/fd/N mountpoints
		if strings.HasPrefix(dir, "/dev/fd/") {
			return fmt.Errorf("%w: %s", ErrExternallyManagedMountPoint, err)
		}
		return err
	}
	return nil
}

func fuserunmount(dir string) error {
	fusermount, err := findFusermount()
	if err != nil {
		return err
	}
	cmd := exec.Command(fusermount, "-u", dir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) > 0 {
			output = bytes.TrimRight(output, "\n")
			return fmt.Errorf("%v: %s", err, output)
		}

		return err
	}
	return nil
}
