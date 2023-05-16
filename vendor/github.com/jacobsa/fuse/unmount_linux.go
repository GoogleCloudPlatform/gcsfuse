package fuse

import (
	"bytes"
	"fmt"
	"os/exec"
)

func unmount(dir string) error {
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
