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
	"fmt"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/fs/wrappers"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseutil"
	"golang.org/x/net/context"
)

// Create a fuse file system server according to the supplied configuration.
func NewServer(ctx context.Context, cfg *ServerConfig) (fuse.Server, error) {
	fs, err := NewFileSystem(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create file system: %w", err)
	}

	fs = wrappers.WithErrorMapping(fs)
	fs = wrappers.WithMonitoring(fs)
	return fuseutil.NewFileSystemServer(fs), nil
}
