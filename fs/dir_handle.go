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

package fs

import (
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/gcloud/syncutil"
	"github.com/jacobsa/gcsfuse/fs/inode"
	"golang.org/x/net/context"
)

// State required for reading from directories.
type dirHandle struct {
	Mu syncutil.InvariantMutex
}

// Create a directory handle that obtains listings from the supplied inode.
func newDirHandle(in *inode.DirInode) *dirHandle

// Handle a request to read from the directory.
//
// Because the GCS API for listing objects has no notion of a stable offset
// like the posix telldir/seekdir API does, there is no way for us to
// efficiently support seeking backwards. Therefore we return EINVAL when an
// offset for an entry that we no longer have buffered is encountered.
//
// Special case: we assume that a zero offset indicates that rewinddir has been
// called (since fuse gives us no way to intercept and know for sure), and
// start the listing process over again.
//
// EXCLUSIVE_LOCKS_REQUIRED(dh.Mu)
// LOCKS_EXCLUDED(dh.in.Mu)
func (dh *dirHandle) ReadDir(
	ctx context.Context,
	req *fuse.ReadDirRequest) (resp *fuse.ReadDirResponse, err error)
