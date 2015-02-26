// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package fuseutil

import "bazil.org/fuse"

// An object that terminates one end of the userspace <-> FUSE VFS connection.
type server struct {
}

// Create a server that relays requests to the supplied file system.
func newServer(fs FileSystem) (s *server, err error)

// Serve the fuse connection by repeatedly reading requests from the supplied
// FUSE connection, responding as dictated by the file system. Return when the
// connection is closed or an unexpected error occurs.
func (s *server) Serve(c *fuse.Conn) (err error)
