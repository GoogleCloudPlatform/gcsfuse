package fuse

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

func findFusermount() (string, error) {
	path, err := exec.LookPath("fusermount3")
	if err != nil {
		path, err = exec.LookPath("fusermount")
	}
	if err != nil {
		return "", err
	}
	return path, nil
}

func fusermount(dir string, cfg *MountConfig) (*os.File, error) {
	// Create a socket pair.
	fds, err := syscall.Socketpair(syscall.AF_FILE, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, fmt.Errorf("Socketpair: %v", err)
	}

	// Wrap the sockets into os.File objects that we will pass off to fusermount.
	writeFile := os.NewFile(uintptr(fds[0]), "fusermount-child-writes")
	defer writeFile.Close()

	readFile := os.NewFile(uintptr(fds[1]), "fusermount-parent-reads")
	defer readFile.Close()

	// Start fusermount, passing it a buffer in which to write stderr.
	var stderr bytes.Buffer

	fusermount, err := findFusermount()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(
		fusermount,
		"-o", cfg.toOptionsString(),
		"--",
		dir,
	)

	cmd.Env = append(os.Environ(), "_FUSE_COMMFD=3")
	cmd.ExtraFiles = []*os.File{writeFile}
	cmd.Stderr = &stderr

	// Run the command.
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("running fusermount: %v\n\nstderr:\n%s", err, stderr.Bytes())
	}

	// Wrap the socket file in a connection.
	c, err := net.FileConn(readFile)
	if err != nil {
		return nil, fmt.Errorf("FileConn: %v", err)
	}
	defer c.Close()

	// We expect to have a Unix domain socket.
	uc, ok := c.(*net.UnixConn)
	if !ok {
		return nil, fmt.Errorf("Expected UnixConn, got %T", c)
	}

	// Read a message.
	buf := make([]byte, 32) // expect 1 byte
	oob := make([]byte, 32) // expect 24 bytes
	_, oobn, _, _, err := uc.ReadMsgUnix(buf, oob)
	if err != nil {
		return nil, fmt.Errorf("ReadMsgUnix: %v", err)
	}

	// Parse the message.
	scms, err := syscall.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return nil, fmt.Errorf("ParseSocketControlMessage: %v", err)
	}

	// We expect one message.
	if len(scms) != 1 {
		return nil, fmt.Errorf("expected 1 SocketControlMessage; got scms = %#v", scms)
	}

	scm := scms[0]

	// Pull out the FD returned by fusermount
	gotFds, err := syscall.ParseUnixRights(&scm)
	if err != nil {
		return nil, fmt.Errorf("syscall.ParseUnixRights: %v", err)
	}

	if len(gotFds) != 1 {
		return nil, fmt.Errorf("wanted 1 fd; got %#v", gotFds)
	}

	// Turn the FD into an os.File.
	return os.NewFile(uintptr(gotFds[0]), "/dev/fuse"), nil
}

func enableFunc(flag uintptr) func(uintptr) uintptr {
	return func(v uintptr) uintptr {
		return v | flag
	}
}

func disableFunc(flag uintptr) func(uintptr) uintptr {
	return func(v uintptr) uintptr {
		return v &^ flag
	}
}

// As per libfuse/fusermount.c:602: https://bit.ly/2SgtWYM#L602
var mountflagopts = map[string]func(uintptr) uintptr{
	"rw":      disableFunc(unix.MS_RDONLY),
	"ro":      enableFunc(unix.MS_RDONLY),
	"suid":    disableFunc(unix.MS_NOSUID),
	"nosuid":  enableFunc(unix.MS_NOSUID),
	"dev":     disableFunc(unix.MS_NODEV),
	"nodev":   enableFunc(unix.MS_NODEV),
	"exec":    disableFunc(unix.MS_NOEXEC),
	"noexec":  enableFunc(unix.MS_NOEXEC),
	"async":   disableFunc(unix.MS_SYNCHRONOUS),
	"sync":    enableFunc(unix.MS_SYNCHRONOUS),
	"atime":   disableFunc(unix.MS_NOATIME),
	"noatime": enableFunc(unix.MS_NOATIME),
	"dirsync": enableFunc(unix.MS_DIRSYNC),
}

var errFallback = errors.New("sentinel: fallback to fusermount(1)")

func directmount(dir string, cfg *MountConfig) (*os.File, error) {
	// We use syscall.Open + os.NewFile instead of os.OpenFile so that the file
	// is opened in blocking mode. When opened in non-blocking mode, the Go
	// runtime tries to use poll(2), which does not work with /dev/fuse.
	fd, err := syscall.Open("/dev/fuse", syscall.O_RDWR, 0644)
	if err != nil {
		return nil, errFallback
	}
	dev := os.NewFile(uintptr(fd), "/dev/fuse")
	// As per libfuse/fusermount.c:847: https://bit.ly/2SgtWYM#L847
	data := fmt.Sprintf("fd=%d,rootmode=40000,user_id=%d,group_id=%d",
		dev.Fd(), os.Getuid(), os.Getgid())
	// As per libfuse/fusermount.c:749: https://bit.ly/2SgtWYM#L749
	mountflag := uintptr(unix.MS_NODEV | unix.MS_NOSUID)
	opts := cfg.toMap()
	for k := range opts {
		fn, ok := mountflagopts[k]
		if !ok {
			continue
		}
		mountflag = fn(mountflag)
		delete(opts, k)
	}
	delete(opts, "fsname") // handled via fstype mount(2) parameter
	fstype := "fuse"
	if subtype, ok := opts["subtype"]; ok {
		fstype += "." + subtype
	}
	delete(opts, "subtype")
	data += "," + mapToOptionsString(opts)
	if err := unix.Mount(
		cfg.FSName, // source
		dir,        // target
		fstype,     // fstype
		mountflag,  // mountflag
		data,       // data
	); err != nil {
		if err == syscall.EPERM {
			return nil, errFallback

		}
		return nil, err
	}
	return dev, nil
}

// Begin the process of mounting at the given directory, returning a connection
// to the kernel. Mounting continues in the background, and is complete when an
// error is written to the supplied channel. The file system may need to
// service the connection in order for mounting to complete.
func mount(dir string, cfg *MountConfig, ready chan<- error) (*os.File, error) {
	// On linux, mounting is never delayed.
	ready <- nil

	// Try mounting without fusermount(1) first: we might be running as root or
	// have the CAP_SYS_ADMIN capability.
	dev, err := directmount(dir, cfg)
	if err == errFallback {
		return fusermount(dir, cfg)
	}
	return dev, err
}
