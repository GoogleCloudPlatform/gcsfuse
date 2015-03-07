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
	"time"

	fusefs "bazil.org/fuse/fs"
)

const dirSeparator = '/'

// Implementation detail, do not touch.
//
// How long we remember that we took some action on the contents of a directory
// (linking or unlinking), and pretend the action is reflected in the listing
// even if it is not.
//
// Intended to paper over the fact that GCS doesn't offer list-your-own-writes
// consistency: it may be an arbitrarily long time before you see the creation
// or deletion of an object in a subsequent listing, and even if you see it in
// one listing you may not in the next. The drawback is that modifications to
// recently-modified directories by foreign machines will not be reflected
// locally for awhile.
//
// TODO(jacobsa): Set this according to information about listing staleness
// distributions from the GCS team.
//
// TODO(jacobsa): Can this be moved to a decorator implementation of gcs.Bucket
// instead of living here?
var ChildActionMemoryTTL = 5 * time.Minute

// See the childModifications field of dir.
type childModification struct {
	time time.Time
	name string
	node fusefs.Node
}
