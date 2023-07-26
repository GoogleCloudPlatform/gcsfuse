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

// System permissions-related code.
package perms

import (
	"fmt"
	"os"
)

// MyUserAndGroup returns the UID and GID of this process.
func MyUserAndGroup() (uid, gid uint32, err error) {
	signed_uid := os.Getuid()
	signed_gid := os.Getgid()

	// Not sure in what scenarios uid/gid could be returned as negative. The only
	// documented scenario at pkg.go.dev/os#Getuid is windows OS.
	if signed_gid < 0 || signed_uid < 0 {
		err = fmt.Errorf("failed to get uid/gid. UID = %d, GID = %d", signed_uid, signed_gid)

		// An untested improvement idea to fallback here is to invoke os.current.User()
		// and use its partial output even when os.current.User() returned error, as
		// the partial output would still be useful.

		return
	}

	uid = uint32(signed_uid)
	gid = uint32(signed_gid)

	return
}
