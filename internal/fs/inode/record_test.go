// Copyright 2021 Google Inc. All Rights Reserved.
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

package inode_test

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/fs/inode"
	"github.com/jacobsa/gcloud/gcs"
)

func TestRecord(t *testing.T) {
	var r *inode.Record
	if r.Exists() {
		t.Errorf("%v does not exist", r)
	}

	for tp, obj := range map[inode.Type]*gcs.Object{
		inode.SymlinkType: {
			Name: "bar",
			Metadata: map[string]string{
				inode.SymlinkMetadataKey: "",
			},
		},
		inode.RegularFileType: {Name: "bar"},
		inode.ExplicitDirType: {Name: "bar/"},
		inode.ImplicitDirType: nil,
	} {
		r = &inode.Record{obj}
		if got, want := r.Type(), tp; got != want {
			t.Errorf("Inode type of %v, wanted %v, got %v", r, want, got)
		}
	}

}
