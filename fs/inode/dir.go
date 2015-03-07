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

package inode

import (
	"fmt"

	"github.com/jacobsa/gcloud/syncutil"
)

type DirInode struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	// The name of the GCS object backing the inode, used as a prefix when
	// listing. Special case: the empty string means this is the root inode.
	//
	// INVARIANT: name == "" || name[len(name)-1] == '/'
	name string

	/////////////////////////
	// Mutable state
	/////////////////////////

	// A mutex that must be held when calling certain methods. See documentation
	// for each method.
	Mu syncutil.InvariantMutex
}

// Create a directory inode for the directory with the given name. The name
// must end with a slash unless this is the root directory, in which case it
// must be empty.
//
// REQUIRES: name == "" || name[len(name)-1] == '/'
func NewDirInode(name string) (d *DirInode) {
	// Set up the basic struct.
	d = &DirInode{
		name: name,
	}

	// Set up invariant checking.
	d.Mu = syncutil.NewInvariantMutex(d.checkInvariants)

	return
}

func (d *DirInode) checkInvariants() {
	// Check the name.
	if !(d.name == "" || d.name[len(d.name)-1] == '/') {
		panic(fmt.Sprintf("Unexpected name: %s", d.name))
	}
}
