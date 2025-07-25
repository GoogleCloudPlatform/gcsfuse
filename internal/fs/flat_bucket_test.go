// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fs_test

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type FlatBucketTests struct {
	RenameFileTests
	fsTest
}

func TestFlatBucketTests(t *testing.T) { suite.Run(t, new(FlatBucketTests)) }

func (t *FlatBucketTests) SetupSuite() {
	t.serverCfg.MetricHandle = common.NewNoopMetrics()
	t.fsTest.SetUpTestSuite()
}

func (t *FlatBucketTests) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func (t *FlatBucketTests) SetupTest() {
	err := t.createFolders([]string{"foo/", "bar/", "foo/test2/", "foo/test/"})
	require.NoError(t.T(), err)

	err = t.createObjects(
		map[string]string{
			"foo/file1.txt":              file1Content,
			"foo/file2.txt":              file2Content,
			"foo/test/file3.txt":         "xyz",
			"foo/implicit_dir/file3.txt": "xxw",
			"bar/file1.txt":              "-1234556789",
		})
	require.NoError(t.T(), err)
}

func (t *FlatBucketTests) TearDownTest() {
	t.fsTest.TearDown()
}
