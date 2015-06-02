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

	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

// Given an object record and content that was originally derived from that
// object's contents (and potentially modified):
//
// *   If the content has not been modified, return a nil read proxy and a nil
//     new object.
//
// *   Otherwise, write out a new generation in the bucket (failing with
//     *gcs.PreconditionError if the source generation is no longer current)
//     and return a read proxy for that object's contents.
//
//     In this case the MutableContent is destroyed and must not be used again.
//
func Sync(
	ctx context.Context,
	sourceObject *gcs.Object,
	newContent MutableContent,
	bucket gcs.Bucket) (
	newProxy lease.ReadProxy, newObject *gcs.Object, err error) {
	err = errors.New("TODO")
	return
}
