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

package mount_retry

import (
	"fmt"
	"os"
	"path"
	"testing"

	emulator_tests "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/emulator_tests/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type mountRetryBase struct {
	bucketName         string
	port               int
	proxyProcessId     int
	proxyServerLogFile string
	flags              []string
	configFileName     string
	testDirPath        string
	isHTTP             bool
	suite.Suite
}

func (m *mountRetryBase) SetupTest() {
	m.proxyServerLogFile = setup.CreateProxyServerLogFile(m.T())
	var err error
	m.port, m.proxyProcessId, err = emulator_tests.StartProxyServer(m.configFileName, m.proxyServerLogFile)
	require.NoError(m.T(), err)

	// Copy flags to avoid mutating the original slice across suites
	mountFlags := append([]string(nil), m.flags...)
	if m.isHTTP {
		mountFlags = append(mountFlags, fmt.Sprintf("--custom-endpoint=http://localhost:%d/storage/v1/", m.port))
	} else {
		mountFlags = append(mountFlags, fmt.Sprintf("--custom-endpoint=localhost:%d", m.port))
	}
	mountFlags = append(mountFlags, "--anonymous-access") // Required for emulator localhost endpoint

	config := &test_suite.TestConfig{
		TestBucket:              m.bucketName,
		GKEMountedDirectory:     setup.MountedDirectory(),
		GCSFuseMountedDirectory: setup.MntDir(),
		LogFile:                 setup.LogFile(),
	}
	setup.MountGCSFuseWithGivenMountWithConfigFunc(config, mountFlags, static_mounting.MountGcsfuseWithStaticMountingWithConfigFile)
	m.testDirPath = setup.SetupTestDirectory(m.T().Name())
}

func (m *mountRetryBase) TearDownTest() {
	setup.UnmountGCSFuse(setup.MntDir())
	if m.proxyProcessId > 0 {
		assert.NoError(m.T(), emulator_tests.KillProxyServerProcess(m.proxyProcessId))
	}
	setup.SaveGCSFuseLogFileInCaseOfFailure(m.T())
	setup.SaveProxyServerLogFileInCaseOfFailure(m.proxyServerLogFile, m.T())
}

func (m *mountRetryBase) verifyMountedDirectory(folderName string) {
	folderPath := path.Join(m.testDirPath, folderName)
	err := os.MkdirAll(folderPath, 0755)
	require.NoError(m.T(), err)

	info, err := os.Stat(folderPath)
	require.NoError(m.T(), err)
	assert.True(m.T(), info.IsDir())
}

func newMountRetryBase(bucketName string, flags []string, configFileName string, isHTTP bool) mountRetryBase {
	return mountRetryBase{
		bucketName:     bucketName,
		flags:          flags,
		configFileName: configFileName,
		isHTTP:         isHTTP,
	}
}

type mountRetrySuite struct {
	mountRetryBase
	folderName string
}

func (s *mountRetrySuite) TestMountSucceeds() {
	s.verifyMountedDirectory(s.folderName)
}

const (
	hnsBucket    = "test-hns-bucket"
	nonHnsBucket = "test-bucket"
)

var (
	hnsFlags = []string{
		"--enable-mount-retries=true",
		"--client-protocol=grpc",
		"--metadata-cache-ttl-secs=0",
	}

	nonHnsFlags = []string{
		"--enable-mount-retries=true",
		"--enable-hns=false",
		"--client-protocol=http1",
		"--metadata-cache-ttl-secs=0",
	}
)

// TestMountSucceedsWithTransient403PermissionDenied_HNS verifies that GCSFuse mount
// completes successfully even when HTTP 403 / gRPC PermissionDenied error is returned
// during initial GetStorageLayout calls, proving that ShouldRetryOnMount correctly
// retries and recovers after 1 attempt.
func TestMountSucceedsWithTransient403PermissionDenied_HNS(t *testing.T) {
	suite.Run(t, &mountRetrySuite{
		mountRetryBase: newMountRetryBase(
			hnsBucket,
			hnsFlags,
			"../configs/mount_retry_403.yaml",
			false,
		),
		folderName: "retry_folder_hns_403",
	})
}

// TestMountSucceedsWithTransient404BucketDoesNotExist_HNS verifies that GCSFuse mount
// completes successfully even when HTTP 404 / gRPC NotFound (bucket does not exist)
// error is returned during initial GetStorageLayout calls, proving that ShouldRetryOnMount
// correctly retries and recovers after 1 attempt.
func TestMountSucceedsWithTransient404BucketDoesNotExist_HNS(t *testing.T) {
	suite.Run(t, &mountRetrySuite{
		mountRetryBase: newMountRetryBase(
			hnsBucket,
			hnsFlags,
			"../configs/mount_retry_404_bucket_not_found.yaml",
			false,
		),
		folderName: "retry_folder_hns_404",
	})
}

// TestMountSucceedsWithTransient403PermissionDenied_NonHNS verifies that GCSFuse mount
// completes successfully even when HTTP 403 error is returned during initial Attrs calls
// in verifyNonHNSBucketAccess, proving that ShouldRetryOnMount correctly retries and recovers after 1 attempt.
func TestMountSucceedsWithTransient403PermissionDenied_NonHNS(t *testing.T) {
	suite.Run(t, &mountRetrySuite{
		mountRetryBase: newMountRetryBase(
			nonHnsBucket,
			nonHnsFlags,
			"../configs/mount_retry_non_hns_403.yaml",
			true,
		),
		folderName: "retry_folder_non_hns_403",
	})
}

// TestMountSucceedsWithTransient404BucketDoesNotExist_NonHNS verifies that GCSFuse mount
// completes successfully even when HTTP 404 (bucket does not exist) error is returned during initial Attrs calls
// in verifyNonHNSBucketAccess, proving that ShouldRetryOnMount correctly retries and recovers after 1 attempt.
func TestMountSucceedsWithTransient404BucketDoesNotExist_NonHNS(t *testing.T) {
	suite.Run(t, &mountRetrySuite{
		mountRetryBase: newMountRetryBase(
			nonHnsBucket,
			nonHnsFlags,
			"../configs/mount_retry_non_hns_404_bucket_not_found.yaml",
			true,
		),
		folderName: "retry_folder_non_hns_404",
	})
}
