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

package gcsproxy

import (
	"errors"
	"io"

	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/googlecloudplatform/gcsfuse/mutable"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

// Create an object syncer that stats the mutable content to see if it's dirty
// before calling through to one of two functions that handle creating the
// underlying object if the content is dirty:
//
// *   syncFull accepts the source object and the full contents with which it
//     should be overwritten.
//
// *   syncAppend accepts the source object and the contents that should be
//     "appended" to it.
//
func createStattingObjectSyncer(
	syncFull func(context.Context, *gcs.Object, io.Reader) (*gcs.Object, error),
	syncAppend func(context.Context, *gcs.Object, io.Reader) (*gcs.Object, error)) (
	os ObjectSyncer) {
	os = &stattingObjectSyncer{
		syncFull:   syncFull,
		syncAppend: syncAppend,
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type stattingObjectSyncer struct {
	syncFull   func(context.Context, *gcs.Object, io.Reader) (*gcs.Object, error)
	syncAppend func(context.Context, *gcs.Object, io.Reader) (*gcs.Object, error)
}

func (os *stattingObjectSyncer) SyncObject(
	ctx context.Context,
	srcObject *gcs.Object,
	content mutable.Content) (rl lease.ReadLease, o *gcs.Object, err error) {
	err = errors.New("TODO")
	return
}
