// Copyright 2024 Google Inc. All Rights Reserved.
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

package cmd

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func Test_Main(t *testing.T) {
	suite.Run(t, new(MainTest))
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type MainTest struct {
	suite.Suite
}

func (t *MainTest) TestCreateStorageHandle() {
	newConfig := &cfg.Config{
		GcsConnection: cfg.GcsConnectionConfig{ClientProtocol: cfg.HTTP1},
		GcsAuth:       cfg.GcsAuthConfig{KeyFile: "testdata/test_creds.json"}}

	userAgent := "AppName"
	storageHandle, err := createStorageHandle(newConfig, &config.MountConfig{}, userAgent)

	assert.Equal(t.T(), nil, err)
	assert.NotEqual(t.T(), nil, storageHandle)
}

func (t *MainTest) TestCreateStorageHandle_WithClientProtocolAsGRPC() {
	newConfig := &cfg.Config{
		GcsConnection: cfg.GcsConnectionConfig{ClientProtocol: cfg.GRPC},
		GcsAuth:       cfg.GcsAuthConfig{KeyFile: "testdata/test_creds.json"},
	}

	userAgent := "AppName"
	storageHandle, err := createStorageHandle(newConfig, &config.MountConfig{}, userAgent)

	assert.Equal(t.T(), nil, err)
	assert.NotEqual(t.T(), nil, storageHandle)
}

func (t *MainTest) TestGetUserAgentWhenMetadataImageTypeEnvVarIsSet() {
	os.Setenv("GCSFUSE_METADATA_IMAGE_TYPE", "DLVM")
	defer os.Unsetenv("GCSFUSE_METADATA_IMAGE_TYPE")

	mountConfig := &config.MountConfig{}
	userAgent := getUserAgent("AppName", getConfigForUserAgent(mountConfig))
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s AppName (GPN:gcsfuse-DLVM) (Cfg:0:0)", getVersion()))

	assert.Equal(t.T(), expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentWhenMetadataImageTypeEnvVarIsNotSet() {
	mountConfig := &config.MountConfig{}
	userAgent := getUserAgent("AppName", getConfigForUserAgent(mountConfig))
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0)", getVersion()))

	assert.Equal(t.T(), expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentConfigWithNoFileCache() {
	mountConfig := &config.MountConfig{}
	userAgent := getUserAgent("AppName", getConfigForUserAgent(mountConfig))
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0)", getVersion()))
	assert.Equal(t.T(), expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentConfigWithFileCacheEnabledRandomReadEnabled() {
	mountConfig := &config.MountConfig{
		CacheDir: "//tmp//folder//",
		FileCacheConfig: config.FileCacheConfig{
			MaxSizeMB:             -1,
			CacheFileForRangeRead: true,
		},
	}
	userAgent := getUserAgent("AppName", getConfigForUserAgent(mountConfig))
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:1:1)", getVersion()))
	assert.Equal(t.T(), expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentConfigWithFileCacheEnabledRandomDisabled() {
	// Test File Cache Enabled but Random Read Disabled
	mountConfig := &config.MountConfig{
		CacheDir: "//tmp//folder//",
		FileCacheConfig: config.FileCacheConfig{
			MaxSizeMB: -1,
		},
	}
	userAgent := getUserAgent("AppName", getConfigForUserAgent(mountConfig))
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:1:0)", getVersion()))
	assert.Equal(t.T(), expectedUserAgent, userAgent)
}
func (t *MainTest) TestGetUserAgentConfigWithFileCacheSizeSetCacheDirNotSet() {
	// Test File cache disabled where MaxSize is set but Cache Dir is not set.
	mountConfig := &config.MountConfig{
		FileCacheConfig: config.FileCacheConfig{
			MaxSizeMB: -1,
		},
	}
	userAgent := getUserAgent("AppName", getConfigForUserAgent(mountConfig))
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0)", getVersion()))
	assert.Equal(t.T(), expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentConfigWithCacheDirSetMaxSizeDisabled() {
	// Test File Cache disabled when Cache Dir is given but maxSize is set 0.
	mountConfig := &config.MountConfig{
		CacheDir: "//tmp//folder//",
		FileCacheConfig: config.FileCacheConfig{
			MaxSizeMB: 0,
		},
	}
	userAgent := getUserAgent("AppName", getConfigForUserAgent(mountConfig))
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0)", getVersion()))
	assert.Equal(t.T(), expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentWhenMetadataImageTypeEnvVarSetAndAppNameNotSet() {
	os.Setenv("GCSFUSE_METADATA_IMAGE_TYPE", "DLVM")
	defer os.Unsetenv("GCSFUSE_METADATA_IMAGE_TYPE")

	mountConfig := &config.MountConfig{}
	userAgent := getUserAgent("", getConfigForUserAgent(mountConfig))
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-DLVM) (Cfg:0:0)", getVersion()))

	assert.Equal(t.T(), expectedUserAgent, userAgent)
}

func (t *MainTest) TestCallListRecursiveOnExistingDirectory() {
	// Set up a mini file-system to test on.
	rootdir, err := os.MkdirTemp("/tmp", "TestCallListRecursive-*")
	if err != nil {
		t.T().Fatalf("Failed to set up test. error = %v", err)
	}
	defer os.RemoveAll(rootdir)
	_, err = os.CreateTemp(rootdir, "abc-*.txt")
	if err != nil {
		t.T().Fatalf("Failed to set up test. error = %v", err)
	}

	err = callListRecursive(rootdir)

	assert.Nil(t.T(), err)
}

func (t *MainTest) TestCallListRecursiveOnNonExistingDirectory() {
	// Set up a mini file-system to test on, which must fail.
	rootdir := "/path/to/non/existing/directory"

	err := callListRecursive(rootdir)

	assert.ErrorContains(t.T(), err, "does not exist")
}

func (t *MainTest) TestIsDynamicMount() {
	for _, input := range []struct {
		bucketName string
		isDynamic  bool
	}{
		{
			bucketName: "",
			isDynamic:  true,
		}, {
			bucketName: "_",
			isDynamic:  true,
		}, {
			bucketName: "abc",
			isDynamic:  false,
		},
	} {
		assert.Equal(t.T(), input.isDynamic, isDynamicMount(input.bucketName))
	}
}
