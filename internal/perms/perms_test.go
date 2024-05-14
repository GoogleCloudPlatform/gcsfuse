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

// System permissions-related code unit tests.
package perms_test

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/perms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type PermsTest struct {
	suite.Suite
}

func TestPermsSuite(t *testing.T) {
	suite.Run(t, new(PermsTest))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *PermsTest) MyUserAndGroupNoError() {
	uid, gid, err := perms.MyUserAndGroup()
	assert.NoError(t.T(), err)

	unexpected_id_signed := -1
	unexpected_id := uint32(unexpected_id_signed)
	assert.NotEqual(t.T(), uid, unexpected_id)
	assert.NotEqual(t.T(), gid, unexpected_id)
}
