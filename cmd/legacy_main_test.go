// Copyright 2024 Google LLC
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
	"github.com/googlecloudplatform/gcsfuse/v2/common"
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
	storageHandle, err := createStorageHandle(newConfig, userAgent)

	assert.Equal(t.T(), nil, err)
	assert.NotEqual(t.T(), nil, storageHandle)
}

func (t *MainTest) TestCreateStorageHandle_WithClientProtocolAsGRPC() {
	newConfig := &cfg.Config{
		GcsConnection: cfg.GcsConnectionConfig{ClientProtocol: cfg.GRPC},
		GcsAuth:       cfg.GcsAuthConfig{KeyFile: "testdata/test_creds.json"},
	}

	userAgent := "AppName"
	storageHandle, err := createStorageHandle(newConfig, userAgent)

	assert.Equal(t.T(), nil, err)
	assert.NotEqual(t.T(), nil, storageHandle)
}

func (t *MainTest) TestGetUserAgentWhenMetadataImageTypeEnvVarIsSet() {
	os.Setenv("GCSFUSE_METADATA_IMAGE_TYPE", "DLVM")
	defer os.Unsetenv("GCSFUSE_METADATA_IMAGE_TYPE")

	mountConfig := &cfg.Config{}
	userAgent := getUserAgent("AppName", getConfigForUserAgent(mountConfig))
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s AppName (GPN:gcsfuse-DLVM) (Cfg:0:0:0)", common.GetVersion()))

	assert.Equal(t.T(), expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentWhenMetadataImageTypeEnvVarIsNotSet() {
	mountConfig := &cfg.Config{}
	userAgent := getUserAgent("AppName", getConfigForUserAgent(mountConfig))
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0)", common.GetVersion()))

	assert.Equal(t.T(), expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentConfigWithNoFileCache() {
	mountConfig := &cfg.Config{}
	userAgent := getUserAgent("AppName", getConfigForUserAgent(mountConfig))
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0)", common.GetVersion()))
	assert.Equal(t.T(), expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentConfig() {
	testCases := []struct {
		name              string
		mountConfig       *cfg.Config
		expectedUserAgent string
	}{
		{
			name: "Config with file cache disabled when cache dir is given but maxsize is set 0.",
			mountConfig: &cfg.Config{
				CacheDir: "//tmp//folder//",
				FileCache: cfg.FileCacheConfig{
					MaxSizeMb: 0,
				},
			},
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0)", common.GetVersion())),
		},
		{
			name: "Config with file cache disabled where maxsize is set but cache dir is not set.",
			mountConfig: &cfg.Config{
				FileCache: cfg.FileCacheConfig{
					MaxSizeMb: -1,
				},
			},
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0)", common.GetVersion())),
		},
		{
			name: "Config with file cache enabled but random read disabled.",
			mountConfig: &cfg.Config{
				CacheDir: "//tmp//folder//",
				FileCache: cfg.FileCacheConfig{
					MaxSizeMb: -1,
				},
			},
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:1:0:0)", common.GetVersion())),
		},
		{
			name: "Config with file cache and random read enabled.",
			mountConfig: &cfg.Config{
				CacheDir: "//tmp//folder//",
				FileCache: cfg.FileCacheConfig{
					MaxSizeMb:             -1,
					CacheFileForRangeRead: true,
				},
			},
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:1:1:0)", common.GetVersion())),
		},
		{
			name: "Config with file cache disabled and enable parallel downloads set.",
			mountConfig: &cfg.Config{
				CacheDir: "",
				FileCache: cfg.FileCacheConfig{
					MaxSizeMb:               -1,
					EnableParallelDownloads: true,
				},
			},
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0)", common.GetVersion())),
		},
		{
			name: "Config with file cache and parallel downloads enabled.",
			mountConfig: &cfg.Config{
				CacheDir: "/cache/path",
				FileCache: cfg.FileCacheConfig{
					MaxSizeMb:               -1,
					EnableParallelDownloads: true,
				},
			},
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:1:0:1)", common.GetVersion())),
		},
		{
			name: "Config with file cache, random reads and parallel downloads enabled.",
			mountConfig: &cfg.Config{
				CacheDir: "/cache/path",
				FileCache: cfg.FileCacheConfig{
					MaxSizeMb:               -1,
					CacheFileForRangeRead:   true,
					EnableParallelDownloads: true,
				},
			},
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:1:1:1)", common.GetVersion())),
		},
	}

	for _, tc := range testCases {
		t.T().Run(tc.name, func(t *testing.T) {
			userAgent := getUserAgent("AppName", getConfigForUserAgent(tc.mountConfig))
			assert.Equal(t, tc.expectedUserAgent, userAgent)
		})
	}
}

func (t *MainTest) TestGetUserAgentWhenMetadataImageTypeEnvVarSetAndAppNameNotSet() {
	os.Setenv("GCSFUSE_METADATA_IMAGE_TYPE", "DLVM")
	defer os.Unsetenv("GCSFUSE_METADATA_IMAGE_TYPE")

	mountConfig := &cfg.Config{}
	userAgent := getUserAgent("", getConfigForUserAgent(mountConfig))
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-DLVM) (Cfg:0:0:0)", common.GetVersion()))

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
