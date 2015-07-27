// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fuse

import (
	"fmt"
	"log"
	"runtime"

	"github.com/jacobsa/bazilfuse"
	"golang.org/x/net/context"
)

// A type that knows how to serve ops read from a connection.
type Server interface {
	// Read and serve ops from the supplied connection until EOF. Do not return
	// until all operations have been responded to. Must not be called more than
	// once.
	ServeOps(*Connection)
}

// A struct representing the status of a mount operation, with a method that
// waits for unmounting.
type MountedFileSystem struct {
	dir string

	// The result to return from Join. Not valid until the channel is closed.
	joinStatus          error
	joinStatusAvailable chan struct{}
}

// Return the directory on which the file system is mounted (or where we
// attempted to mount it.)
func (mfs *MountedFileSystem) Dir() string {
	return mfs.dir
}

// Block until a mounted file system has been unmounted. Do not return
// successfully until all ops read from the connection have been responded to
// (i.e. the file system server has finished processing all in-flight ops).
//
// The return value will be non-nil if anything unexpected happened while
// serving. May be called multiple times.
func (mfs *MountedFileSystem) Join(ctx context.Context) error {
	select {
	case <-mfs.joinStatusAvailable:
		return mfs.joinStatus
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Optional configuration accepted by Mount.
type MountConfig struct {
	// The context from which every op read from the connetion by the sever
	// should inherit. If nil, context.Background() will be used.
	OpContext context.Context

	// If non-empty, the name of the file system as displayed by e.g. `mount`.
	// This is important because the `umount` command requires root privileges if
	// it doesn't agree with /etc/fstab.
	FSName string

	// Mount the file system in read-only mode. File modes will appear as normal,
	// but opening a file for writing and metadata operations like chmod,
	// chtimes, etc. will fail.
	ReadOnly bool

	// A logger to use for logging errors. All errors are logged, with the
	// exception of a few blacklisted errors that are expected. If nil, no error
	// logging is performed.
	ErrorLogger *log.Logger

	// A logger to use for logging debug information. If nil, no debug logging is
	// performed.
	DebugLogger *log.Logger

	// OS X only.
	//
	// Normally on OS X we mount with the novncache option
	// (cf. http://goo.gl/1pTjuk), which disables entry caching in the kernel.
	// This is because osxfuse does not honor the entry expiration values we
	// return to it, instead caching potentially forever (cf.
	// http://goo.gl/8yR0Ie), and it is probably better to fail to cache than to
	// cache for too long, since the latter is more likely to hide consistency
	// bugs that are difficult to detect and diagnose.
	//
	// This field disables the use of novncache, restoring entry caching. Beware:
	// the value of ChildInodeEntry.EntryExpiration is ignored by the kernel, and
	// entries will be cached for an arbitrarily long time.
	EnableVnodeCaching bool

	// Additional key=value options to pass unadulterated to the underlying mount
	// command. See `man 8 mount`, the fuse documentation, etc. for
	// system-specific information.
	//
	// For expert use only! May invalidate other guarantees made in the
	// documentation for this package.
	Options map[string]string
}

// Convert to mount options to be passed to package bazilfuse.
func (c *MountConfig) bazilfuseOptions() (opts []bazilfuse.MountOption) {
	isDarwin := runtime.GOOS == "darwin"

	// Enable permissions checking in the kernel. See the comments on
	// InodeAttributes.Mode.
	opts = append(opts, bazilfuse.SetOption("default_permissions", ""))

	// HACK(jacobsa): Work around what appears to be a bug in systemd v219, as
	// shipped in Ubuntu 15.04, where it automatically unmounts any file system
	// that doesn't set an explicit name.
	//
	// When Ubuntu contains systemd v220, this workaround should be removed and
	// the systemd bug reopened if the problem persists.
	//
	// Cf. https://github.com/bazil/fuse/issues/89
	// Cf. https://bugs.freedesktop.org/show_bug.cgi?id=90907
	fsname := c.FSName
	if runtime.GOOS == "linux" && fsname == "" {
		fsname = "some_fuse_file_system"
	}

	// Special file system name?
	if fsname != "" {
		opts = append(opts, bazilfuse.FSName(fsname))
	}

	// Read only?
	if c.ReadOnly {
		opts = append(opts, bazilfuse.ReadOnly())
	}

	// OS X: set novncache when appropriate.
	if isDarwin && !c.EnableVnodeCaching {
		opts = append(opts, bazilfuse.SetOption("novncache", ""))
	}

	// OS X: disable the use of "Apple Double" (._foo and .DS_Store) files, which
	// just add noise to debug output and can have significant cost on
	// network-based file systems.
	//
	// Cf. https://github.com/osxfuse/osxfuse/wiki/Mount-options
	if isDarwin {
		opts = append(opts, bazilfuse.SetOption("noappledouble", ""))
	}

	// Ask the Linux kernel for larger read requests.
	//
	// As of 2015-03-26, the behavior in the kernel is:
	//
	//  *  (http://goo.gl/bQ1f1i, http://goo.gl/HwBrR6) Set the local variable
	//     ra_pages to be init_response->max_readahead divided by the page size.
	//
	//  *  (http://goo.gl/gcIsSh, http://goo.gl/LKV2vA) Set
	//     backing_dev_info::ra_pages to the min of that value and what was sent
	//     in the request's max_readahead field.
	//
	//  *  (http://goo.gl/u2SqzH) Use backing_dev_info::ra_pages when deciding
	//     how much to read ahead.
	//
	//  *  (http://goo.gl/JnhbdL) Don't read ahead at all if that field is zero.
	//
	// Reading a page at a time is a drag. Ask for a larger size.
	const maxReadahead = 1 << 20
	opts = append(opts, bazilfuse.MaxReadahead(maxReadahead))

	// Last but not least: other user-supplied options.
	for k, v := range c.Options {
		opts = append(opts, bazilfuse.SetOption(k, v))
	}

	return
}

// Attempt to mount a file system on the given directory, using the supplied
// Server to serve connection requests. This function blocks until the file
// system is successfully mounted.
func Mount(
	dir string,
	server Server,
	config *MountConfig) (mfs *MountedFileSystem, err error) {
	// Initialize the struct.
	mfs = &MountedFileSystem{
		dir:                 dir,
		joinStatusAvailable: make(chan struct{}),
	}

	// Open a bazilfuse connection.
	bfConn, err := bazilfuse.Mount(mfs.dir, config.bazilfuseOptions()...)
	if err != nil {
		err = fmt.Errorf("bazilfuse.Mount: %v", err)
		return
	}

	// Choose a parent context for ops.
	opContext := config.OpContext
	if opContext == nil {
		opContext = context.Background()
	}

	// Create our own Connection object wrapping it.
	connection, err := newConnection(
		opContext,
		config.DebugLogger,
		config.ErrorLogger,
		bfConn)

	if err != nil {
		bfConn.Close()
		err = fmt.Errorf("newConnection: %v", err)
		return
	}

	// Serve the connection in the background. When done, set the join status.
	go func() {
		server.ServeOps(connection)
		mfs.joinStatus = connection.close()
		close(mfs.joinStatusAvailable)
	}()

	// Wait for the connection to say it is ready.
	if err = connection.waitForReady(); err != nil {
		err = fmt.Errorf("WaitForReady: %v", err)
		return
	}

	return
}
