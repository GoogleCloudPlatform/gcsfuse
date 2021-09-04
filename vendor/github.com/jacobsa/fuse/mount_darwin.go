package fuse

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/jacobsa/fuse/internal/buffer"
)

var errNoAvail = errors.New("no available fuse devices")
var errNotLoaded = errors.New("osxfuse is not loaded")

// errOSXFUSENotFound is returned from Mount when the OSXFUSE installation is
// not detected. Make sure OSXFUSE is installed.
var errOSXFUSENotFound = errors.New("cannot locate OSXFUSE")

// osxfuseInstallation describes the paths used by an installed OSXFUSE
// version.
type osxfuseInstallation struct {
	// Prefix for the device file. At mount time, an incrementing number is
	// suffixed until a free FUSE device is found.
	DevicePrefix string

	// Path of the load helper, used to load the kernel extension if no device
	// files are found.
	Load string

	// Path of the mount helper, used for the actual mount operation.
	Mount string

	// Environment variable used to pass the path to the executable calling the
	// mount helper.
	DaemonVar string

	// Environment variable used to pass the "called by library" flag.
	LibVar string
}

var (
	osxfuseInstallations = []osxfuseInstallation{
		// v4
		{
			DevicePrefix: "/dev/macfuse",
			Load:         "/Library/Filesystems/macfuse.fs/Contents/Resources/load_macfuse",
			Mount:        "/Library/Filesystems/macfuse.fs/Contents/Resources/mount_macfuse",
			DaemonVar:    "_FUSE_DAEMON_PATH",
			LibVar:       "_FUSE_CALL_BY_LIB",
		},

		// v3
		{
			DevicePrefix: "/dev/osxfuse",
			Load:         "/Library/Filesystems/osxfuse.fs/Contents/Resources/load_osxfuse",
			Mount:        "/Library/Filesystems/osxfuse.fs/Contents/Resources/mount_osxfuse",
			DaemonVar:    "MOUNT_OSXFUSE_DAEMON_PATH",
			LibVar:       "MOUNT_OSXFUSE_CALL_BY_LIB",
		},

		// v2
		{
			DevicePrefix: "/dev/osxfuse",
			Load:         "/Library/Filesystems/osxfusefs.fs/Support/load_osxfusefs",
			Mount:        "/Library/Filesystems/osxfusefs.fs/Support/mount_osxfusefs",
			DaemonVar:    "MOUNT_FUSEFS_DAEMON_PATH",
			LibVar:       "MOUNT_FUSEFS_CALL_BY_LIB",
		},
	}
)

func loadOSXFUSE(bin string) error {
	cmd := exec.Command(bin)
	cmd.Dir = "/"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return err
}

func openOSXFUSEDev(devPrefix string) (dev *os.File, err error) {
	// Try each device name.
	for i := uint64(0); ; i++ {
		path := devPrefix + strconv.FormatUint(i, 10)
		dev, err = os.OpenFile(path, os.O_RDWR, 0000)
		if os.IsNotExist(err) {
			if i == 0 {
				// Not even the first device was found. Fuse must not be loaded.
				return nil, errNotLoaded
			}

			// Otherwise we've run out of kernel-provided devices
			return nil, errNoAvail
		}

		if err2, ok := err.(*os.PathError); ok && err2.Err == syscall.EBUSY {
			// This device is in use; try the next one.
			continue
		}

		return dev, nil
	}
}

func callMount(
	bin string,
	daemonVar string,
	libVar string,
	dir string,
	cfg *MountConfig,
	dev *os.File,
	ready chan<- error) error {

	// The mount helper doesn't understand any escaping.
	for k, v := range cfg.toMap() {
		if strings.Contains(k, ",") || strings.Contains(v, ",") {
			return fmt.Errorf(
				"mount options cannot contain commas on darwin: %q=%q",
				k,
				v)
		}
	}

	// Call the mount helper, passing in the device file and saving output into a
	// buffer.
	cmd := exec.Command(
		bin,
		"-o", cfg.toOptionsString(),
		// Tell osxfuse-kext how large our buffer is. It must split
		// writes larger than this into multiple writes.
		//
		// OSXFUSE seems to ignore InitResponse.MaxWrite, and uses
		// this instead.
		"-o", "iosize="+strconv.FormatUint(buffer.MaxWriteSize, 10),
		// refers to fd passed in cmd.ExtraFiles
		"3",
		dir,
	)
	cmd.ExtraFiles = []*os.File{dev}
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, libVar+"=")

	daemon := os.Args[0]
	if daemonVar != "" {
		cmd.Env = append(cmd.Env, daemonVar+"="+daemon)
	}

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Start(); err != nil {
		return err
	}

	// In the background, wait for the command to complete.
	go func() {
		err := cmd.Wait()
		if err != nil {
			if buf.Len() > 0 {
				output := buf.Bytes()
				output = bytes.TrimRight(output, "\n")
				err = fmt.Errorf("%v: %s", err, output)
			}
		}

		ready <- err
	}()

	return nil
}

// Begin the process of mounting at the given directory, returning a connection
// to the kernel. Mounting continues in the background, and is complete when an
// error is written to the supplied channel. The file system may need to
// service the connection in order for mounting to complete.
func mount(
	dir string,
	cfg *MountConfig,
	ready chan<- error) (dev *os.File, err error) {
	// Find the version of osxfuse installed on this machine.
	for _, loc := range osxfuseInstallations {
		if _, err := os.Stat(loc.Mount); os.IsNotExist(err) {
			// try the other locations
			continue
		}

		// Open the device.
		dev, err = openOSXFUSEDev(loc.DevicePrefix)

		// Special case: we may need to explicitly load osxfuse. Load it, then
		// try again.
		if err == errNotLoaded {
			err = loadOSXFUSE(loc.Load)
			if err != nil {
				return nil, fmt.Errorf("loadOSXFUSE: %v", err)
			}

			dev, err = openOSXFUSEDev(loc.DevicePrefix)
		}

		// Propagate errors.
		if err != nil {
			return nil, fmt.Errorf("openOSXFUSEDev: %v", err)
		}

		// Call the mount binary with the device.
		if err := callMount(loc.Mount, loc.DaemonVar, loc.LibVar, dir, cfg, dev, ready); err != nil {
			dev.Close()
			return nil, fmt.Errorf("callMount: %v", err)
		}

		return dev, nil
	}

	return nil, errOSXFUSENotFound
}
