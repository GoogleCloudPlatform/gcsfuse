package fuse

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/jacobsa/fuse/internal/buffer"
	"github.com/jacobsa/fuse/internal/fusekernel"
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

	// Open device manually (false) or receive the FD through a UNIX socket,
	// like with fusermount (true)
	UseCommFD bool
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
			UseCommFD:    true,
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

const FUSET_SRV_PATH = "/usr/local/bin/go-nfsv4"

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

func convertMountArgs(daemonVar string, libVar string,
	cfg *MountConfig) ([]string, []string, error) {

	// The mount helper doesn't understand any escaping.
	for k, v := range cfg.toMap() {
		if strings.Contains(k, ",") || strings.Contains(v, ",") {
			return nil, nil, fmt.Errorf(
				"mount options cannot contain commas on darwin: %q=%q",
				k,
				v)
		}
	}

	env := []string{libVar + "="}
	if daemonVar != "" {
		env = append(env, daemonVar+"="+os.Args[0])
	}
	argv := []string{
		"-o", cfg.toOptionsString(),
		// Tell osxfuse-kext how large our buffer is. It must split
		// writes larger than this into multiple writes.
		//
		// OSXFUSE seems to ignore InitResponse.MaxWrite, and uses
		// this instead.
		"-o", "iosize=" + strconv.FormatUint(buffer.MaxWriteSize, 10),
	}

	return argv, env, nil
}

func callMount(
	bin string,
	daemonVar string,
	libVar string,
	dir string,
	cfg *MountConfig,
	dev *os.File,
	ready chan<- error) error {

	argv, env, err := convertMountArgs(daemonVar, libVar, cfg)
	if err != nil {
		return err
	}

	// Call the mount helper, passing in the device file and saving output into a
	// buffer.
	argv = append(argv,
		// refers to fd passed in cmd.ExtraFiles
		"3",
		dir,
	)
	cmd := exec.Command(bin, argv...)
	cmd.ExtraFiles = []*os.File{dev}
	cmd.Env = env

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

func callMountCommFD(
	bin string,
	daemonVar string,
	libVar string,
	dir string,
	cfg *MountConfig) (*os.File, error) {

	argv, env, err := convertMountArgs(daemonVar, libVar, cfg)
	if err != nil {
		return nil, err
	}
	env = append(env, "_FUSE_COMMVERS=2")
	argv = append(argv, dir)

	return fusermount(bin, argv, env, false, cfg.DebugLogger)
}

// Begin the process of mounting at the given directory, returning a connection
// to the kernel. Mounting continues in the background, and is complete when an
// error is written to the supplied channel. The file system may need to
// service the connection in order for mounting to complete.
func mountOsxFuse(
	dir string,
	cfg *MountConfig,
	ready chan<- error) (dev *os.File, err error) {
	// Find the version of osxfuse installed on this machine.
	for _, loc := range osxfuseInstallations {
		if _, err := os.Stat(loc.Mount); os.IsNotExist(err) {
			// try the other locations
			continue
		}

		if loc.UseCommFD {
			// Call the mount binary with the device.
			ready <- nil
			dev, err = callMountCommFD(loc.Mount, loc.DaemonVar, loc.LibVar, dir, cfg)
			if err != nil {
				return nil, fmt.Errorf("callMount: %v", err)
			}
			return
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

func fusetBinary() (string, error) {
	srv_path := os.Getenv("FUSE_NFSSRV_PATH")
	if srv_path == "" {
		srv_path = FUSET_SRV_PATH
	}

	if _, err := os.Stat(srv_path); err == nil {
		return srv_path, nil
	}

	return "", fmt.Errorf("FUSE-T not found")
}

func unixgramSocketpair() (l, r *os.File, err error) {
	fd, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, nil, os.NewSyscallError("socketpair",
			err.(syscall.Errno))
	}
	l = os.NewFile(uintptr(fd[0]), fmt.Sprintf("socketpair-half%d", fd[0]))
	r = os.NewFile(uintptr(fd[1]), fmt.Sprintf("socketpair-half%d", fd[1]))
	return
}

var local, local_mon, remote, remote_mon *os.File

func startFuseTServer(binary string, argv []string,
	additionalEnv []string,
	wait bool,
	debugLogger *log.Logger,
	ready chan<- error) (*os.File, error) {
	if debugLogger != nil {
		debugLogger.Println("Creating a socket pair")
	}

	var err error
	local, remote, err = unixgramSocketpair()
	if err != nil {
		return nil, err
	}
	defer remote.Close()

	local_mon, remote_mon, err = unixgramSocketpair()
	if err != nil {
		return nil, err
	}
	defer remote_mon.Close()

	syscall.CloseOnExec(int(local.Fd()))
	syscall.CloseOnExec(int(local_mon.Fd()))

	if debugLogger != nil {
		debugLogger.Println("Creating files to wrap the sockets")
	}

	if debugLogger != nil {
		debugLogger.Println("Starting fusermount/os mount")
	}
	// Start fusermount/mount_macfuse/mount_osxfuse.
	cmd := exec.Command(binary, argv...)
	cmd.Env = append(os.Environ(), "_FUSE_COMMFD=3")
	cmd.Env = append(cmd.Env, "_FUSE_MONFD=4")
	cmd.Env = append(cmd.Env, additionalEnv...)
	cmd.ExtraFiles = []*os.File{remote, remote_mon}
	cmd.Stderr = nil
	cmd.Stdout = nil
	// daemonize
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	// Run the command.
	err = cmd.Start()
	cmd.Process.Release()
	if err != nil {
		return nil, fmt.Errorf("running %v: %v", binary, err)
	}

	if debugLogger != nil {
		debugLogger.Println("Wrapping socket pair in a connection")
	}

	if debugLogger != nil {
		debugLogger.Println("Checking that we have a unix domain socket")
	}

	if debugLogger != nil {
		debugLogger.Println("Read a message from socket")
	}

	go func() {
		if _, err = local_mon.Write([]byte("mount")); err != nil {
			err = fmt.Errorf("fuse-t failed: %v", err)
		} else {
			reply := make([]byte, 4)
			if _, err = local_mon.Read(reply); err != nil {
				fmt.Printf("mount read  %v\n", err)
				err = fmt.Errorf("fuse-t failed: %v", err)
			}
		}

		ready <- err
		close(ready)
	}()

	if debugLogger != nil {
		debugLogger.Println("Successfully read the socket message.")
	}

	return local, nil
}

func mountFuset(
	dir string,
	cfg *MountConfig,
	ready chan<- error) (dev *os.File, err error) {
	fuseTBin, err := fusetBinary()
	if err != nil {
		return nil, err
	}

	fusekernel.IsPlatformFuseT = true
	env := []string{}
	argv := []string{
		fmt.Sprintf("--rwsize=%d", buffer.MaxWriteSize),
	}

	if cfg.VolumeName != "" {
		argv = append(argv, "--volname")
		argv = append(argv, cfg.VolumeName)
	}
	if cfg.ReadOnly {
		argv = append(argv, "-r")
	}

	env = append(env, "_FUSE_COMMVERS=2")
	argv = append(argv, dir)

	return startFuseTServer(fuseTBin, argv, env, false, cfg.DebugLogger, ready)
}

func mount(
	dir string,
	cfg *MountConfig,
	ready chan<- error) (dev *os.File, err error) {

	fusekernel.IsPlatformFuseT = false
	switch cfg.FuseImpl {
	case FUSEImplMacFUSE:
		dev, err = mountOsxFuse(dir, cfg, ready)
	case FUSEImplFuseT:
		fallthrough
	default:
		dev, err = mountFuset(dir, cfg, ready)
	}
	return
}
