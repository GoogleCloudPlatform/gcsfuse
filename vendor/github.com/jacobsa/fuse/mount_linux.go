package fuse

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

func lineLogger(wg *sync.WaitGroup, prefix string, r io.ReadCloser) {
	defer wg.Done()

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		switch line := scanner.Text(); line {
		case `fusermount: failed to open /etc/fuse.conf: Permission denied`:
			// Silence this particular message, it occurs way too
			// commonly and isn't very relevant to whether the mount
			// succeeds or not.
			continue
		default:
			log.Printf("%s: %s", prefix, line)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("%s, error reading: %v", prefix, err)
	}
}

// Begin the process of mounting at the given directory, returning a connection
// to the kernel. Mounting continues in the background, and is complete when an
// error is written to the supplied channel. The file system may need to
// service the connection in order for mounting to complete.
func mount(
	dir string,
	cfg *MountConfig,
	ready chan<- error) (dev *os.File, err error) {
	// On linux, mounting is never delayed.
	ready <- nil

	// Create a socket pair.
	fds, err := syscall.Socketpair(syscall.AF_FILE, syscall.SOCK_STREAM, 0)
	if err != nil {
		err = fmt.Errorf("Socketpair: %v", err)
		return
	}

	// Wrap the sockets into os.File objects that we will pass off to fusermount.
	writeFile := os.NewFile(uintptr(fds[0]), "fusermount-child-writes")
	defer writeFile.Close()

	readFile := os.NewFile(uintptr(fds[1]), "fusermount-parent-reads")
	defer readFile.Close()

	// Start fusermount, passing it pipes for stdout and stderr.
	cmd := exec.Command(
		"fusermount",
		"-o", cfg.toOptionsString(),
		"--",
		dir,
	)

	cmd.Env = append(os.Environ(), "_FUSE_COMMFD=3")
	cmd.ExtraFiles = []*os.File{writeFile}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		err = fmt.Errorf("StdoutPipe: %v", err)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		err = fmt.Errorf("StderrPipe: %v", err)
		return
	}

	err = cmd.Start()
	if err != nil {
		err = fmt.Errorf("Starting fusermount: %v", err)
		return
	}

	// Log fusermount output until it closes stdout and stderr.
	var wg sync.WaitGroup
	wg.Add(2)
	go lineLogger(&wg, "mount helper output", stdout)
	go lineLogger(&wg, "mount helper error", stderr)
	wg.Wait()

	// Wait for the command.
	err = cmd.Wait()
	if err != nil {
		err = fmt.Errorf("fusermount: %v", err)
		return
	}

	// Wrap the socket file in a connection.
	c, err := net.FileConn(readFile)
	if err != nil {
		err = fmt.Errorf("FileConn: %v", err)
		return
	}
	defer c.Close()

	// We expect to have a Unix domain socket.
	uc, ok := c.(*net.UnixConn)
	if !ok {
		err = fmt.Errorf("Expected UnixConn, got %T", c)
		return
	}

	// Read a message.
	buf := make([]byte, 32) // expect 1 byte
	oob := make([]byte, 32) // expect 24 bytes
	_, oobn, _, _, err := uc.ReadMsgUnix(buf, oob)
	if err != nil {
		err = fmt.Errorf("ReadMsgUnix: %v", err)
		return
	}

	// Parse the message.
	scms, err := syscall.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		err = fmt.Errorf("ParseSocketControlMessage: %v", err)
		return
	}

	// We expect one message.
	if len(scms) != 1 {
		err = fmt.Errorf("expected 1 SocketControlMessage; got scms = %#v", scms)
		return
	}

	scm := scms[0]

	// Pull out the FD returned by fusermount
	gotFds, err := syscall.ParseUnixRights(&scm)
	if err != nil {
		err = fmt.Errorf("syscall.ParseUnixRights: %v", err)
		return
	}

	if len(gotFds) != 1 {
		err = fmt.Errorf("wanted 1 fd; got %#v", gotFds)
		return
	}

	// Turn the FD into an os.File.
	dev = os.NewFile(uintptr(gotFds[0]), "/dev/fuse")

	return
}
