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
	"strings"

	"golang.org/x/net/context"
)

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

// Create a map containing all of the key=value mount options to be given to
// the mount helper.
func (c *MountConfig) toMap() (opts map[string]string) {
	isDarwin := runtime.GOOS == "darwin"
	opts = make(map[string]string)

	// Enable permissions checking in the kernel. See the comments on
	// InodeAttributes.Mode.
	opts["default_permissions"] = ""

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
		opts["fsname"] = fsname
	}

	// Read only?
	if c.ReadOnly {
		opts["ro"] = ""
	}

	// OS X: set novncache when appropriate.
	if isDarwin && !c.EnableVnodeCaching {
		opts["novncache"] = ""
	}

	// OS X: disable the use of "Apple Double" (._foo and .DS_Store) files, which
	// just add noise to debug output and can have significant cost on
	// network-based file systems.
	//
	// Cf. https://github.com/osxfuse/osxfuse/wiki/Mount-options
	if isDarwin {
		opts["noappledouble"] = ""
	}

	// Last but not least: other user-supplied options.
	for k, v := range c.Options {
		opts[k] = v
	}

	return
}

func escapeOptionsKey(s string) (res string) {
	res = s
	res = strings.Replace(res, `\`, `\\`, -1)
	res = strings.Replace(res, `,`, `\,`, -1)
	return
}

// Create an options string suitable for passing to the mount helper.
func (c *MountConfig) toOptionsString() string {
	var components []string
	for k, v := range c.toMap() {
		k = escapeOptionsKey(k)

		component := k
		if v != "" {
			component = fmt.Sprintf("%s=%s", k, v)
		}

		components = append(components, component)
	}

	return strings.Join(components, ",")
}
