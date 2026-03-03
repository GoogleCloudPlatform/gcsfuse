// Copyright 2026 Google LLC
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

package symlink_handling_test

import (
	"os"
	"path"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

type symlinkTestCase struct {
	name   string
	target string
}

var commonTestCases = []symlinkTestCase{
	{
		name:   "file_target",
		target: "target_file",
	},
	{
		name:   "dir_target",
		target: "target_dir",
	},
	{
		name:   "relative_path",
		target: "../target_file",
	},
	{
		name:   "absolute_path",
		target: "/etc/hosts",
	},
}

func (s *BaseSymlinkSuite) createGCSSymlinkObject(linkName string, content string, metadata map[string]string) {
	fullLinkPath := path.Join(TestDirName, linkName)
	bucketName, objectName := setup.GetBucketAndObjectBasedOnTypeOfMount(fullLinkPath)
	objHandle := testEnv.storageClient.Bucket(bucketName).Object(objectName)

	w, err := client.NewWriter(testEnv.ctx, objHandle, testEnv.storageClient)
	s.Require().NoError(err)
	w.ObjectAttrs.Metadata = metadata
	_, err = w.Write([]byte(content))
	s.Require().NoError(err)
	s.Require().NoError(w.Close())
	operations.WaitForSizeUpdate(setup.IsZonalBucketRun(), operations.WaitDurationAfterCloseZB)
}

func (s *BaseSymlinkSuite) runReadSymlinkTests(testCases []symlinkTestCase, prefix string, createFunc func(string, string)) {
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			linkName := prefix + tc.name
			createFunc(linkName, tc.target)

			linkPath := path.Join(s.testDirPath, linkName)
			result, err := os.Readlink(linkPath)
			s.Require().NoError(err)
			s.Assert().Equal(tc.target, result)
		})
	}
}

func (s *StandardSymlinksTestSuite) TestReadSymlink() {
	testCases := append(commonTestCases, symlinkTestCase{
		name:   "long_target",
		target: strings.Repeat("a", 100),
	})

	s.runReadSymlinkTests(testCases, "read_standard_symlink_", func(linkName, target string) {
		s.createGCSSymlinkObject(linkName, target, map[string]string{
			StandardSymlinkMetadataKey: "true",
		})
	})
}

func (s *LegacySymlinksTestSuite) TestReadSymlink() {
	s.runReadSymlinkTests(commonTestCases, "read_legacy_symlink_", func(linkName, target string) {
		s.createGCSSymlinkObject(linkName, "", map[string]string{
			SymlinkMetadataKey: target,
		})
	})
}
