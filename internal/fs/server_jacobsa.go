// Copyright 2021 Google LLC
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

//go:build !libfuse
// +build !libfuse

// Copyright 2021 Google LLC
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

//go:build !libfuse
// +build !libfuse

package fs

import (
	"fmt"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/wrappers"
	"github.com/jacobsa/fuse"
)

// NewServer creates a new file system server.
//
// It is not safe to use the server for multiple mounts simultaneously.
func NewServer(newConfig *cfg.Config) (Server, error) {
	// Create a wrappers file system.
	fs, err := wrappers.NewFileSystem(newConfig)
	if err != nil {
		return nil, fmt.Errorf("NewFileSystem: %w", err)
	}

	// Create the server.
	server, err := fuse.NewServer(fs)
	if err != nil {
		return nil, fmt.Errorf("fuse.NewServer: %w", err)
	}
	return &serverFromFileSystem{
		server: server,
	}, nil
}
