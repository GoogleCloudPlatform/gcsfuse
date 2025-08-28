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

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
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
		GcsAuth:       cfg.GcsAuthConfig{KeyFile: "testdata/test_creds.json"},
	}

	storageHandle, err := createStorageHandle(newConfig, "AppName", metrics.NewNoopMetrics())

	assert.Equal(t.T(), nil, err)
	assert.NotEqual(t.T(), nil, storageHandle)
}

func (t *MainTest) TestCreateStorageHandle_WithClientProtocolAsGRPC() {
	newConfig := &cfg.Config{
		GcsConnection: cfg.GcsConnectionConfig{ClientProtocol: cfg.GRPC},
		GcsAuth:       cfg.GcsAuthConfig{KeyFile: "testdata/test_creds.json"},
	}

	storageHandle, err := createStorageHandle(newConfig, "AppName", metrics.NewNoopMetrics())

	assert.Equal(t.T(), nil, err)
	assert.NotEqual(t.T(), nil, storageHandle)
}

func (t *MainTest) TestGetUserAgentWhenMetadataImageTypeEnvVarIsSet() {
	os.Setenv("GCSFUSE_METADATA_IMAGE_TYPE", "DLVM")
	defer os.Unsetenv("GCSFUSE_METADATA_IMAGE_TYPE")
	mountConfig := &cfg.Config{}

	userAgent := getUserAgent("AppName", getConfigForUserAgent(mountConfig))

	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s AppName (GPN:gcsfuse-DLVM) (Cfg:0:0:0:0:0)", common.GetVersion()))
	assert.Equal(t.T(), expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentWhenMetadataImageTypeEnvVarIsNotSet() {
	mountConfig := &cfg.Config{}

	userAgent := getUserAgent("AppName", getConfigForUserAgent(mountConfig))

	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0:0:0)", common.GetVersion()))
	assert.Equal(t.T(), expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentConfigWithNoFileCache() {
	mountConfig := &cfg.Config{}

	userAgent := getUserAgent("AppName", getConfigForUserAgent(mountConfig))

	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0:0:0)", common.GetVersion()))
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
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0:0:0)", common.GetVersion())),
		},
		{
			name: "Config with file cache disabled where maxsize is set but cache dir is not set.",
			mountConfig: &cfg.Config{
				FileCache: cfg.FileCacheConfig{
					MaxSizeMb: -1,
				},
			},
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0:0:0)", common.GetVersion())),
		},
		{
			name: "Config with file cache enabled but random read disabled.",
			mountConfig: &cfg.Config{
				CacheDir: "//tmp//folder//",
				FileCache: cfg.FileCacheConfig{
					MaxSizeMb: -1,
				},
			},
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:1:0:0:0:0)", common.GetVersion())),
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
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:1:1:0:0:0)", common.GetVersion())),
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
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0:0:0)", common.GetVersion())),
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
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:1:0:1:0:0)", common.GetVersion())),
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
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:1:1:1:0:0)", common.GetVersion())),
		},
		{
			name: "streaming_writes_enabled",
			mountConfig: &cfg.Config{
				CacheDir: "/cache/path",
				FileCache: cfg.FileCacheConfig{
					MaxSizeMb:               -1,
					CacheFileForRangeRead:   false,
					EnableParallelDownloads: true,
				},
				Write: cfg.WriteConfig{EnableStreamingWrites: true},
			},
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:1:0:1:1:0)", common.GetVersion())),
		},
		{
			name: "streaming_writes_disabled",
			mountConfig: &cfg.Config{
				CacheDir: "/cache/path",
				FileCache: cfg.FileCacheConfig{
					MaxSizeMb:               -1,
					CacheFileForRangeRead:   true,
					EnableParallelDownloads: false,
				},
				Write: cfg.WriteConfig{EnableStreamingWrites: false},
			},
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:1:1:0:0:0)", common.GetVersion())),
		},
		{
			name: "buffered_read_enabled",
			mountConfig: &cfg.Config{
				Read: cfg.ReadConfig{EnableBufferedRead: true},
			},
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0:0:1)", common.GetVersion())),
		},
		{
			name: "buffered_read_disabled",
			mountConfig: &cfg.Config{
				Read: cfg.ReadConfig{EnableBufferedRead: false},
			},
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0:0:0)", common.GetVersion())),
		},
		{
			name: "file_cache_enabled_and_buffered_read_enabled",
			mountConfig: &cfg.Config{
				CacheDir:  "/cache/path",
				FileCache: cfg.FileCacheConfig{MaxSizeMb: -1},
				Read:      cfg.ReadConfig{EnableBufferedRead: true},
			},
			// Note: getConfigForUserAgent runs before config rationalization, which
			// would disable buffered-read when file-cache is enabled.
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:1:0:0:0:1)", common.GetVersion())),
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

	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-DLVM) (Cfg:0:0:0:0:0)", common.GetVersion()))
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

func (t *MainTest) TestForwardedEnvVars() {
	for _, input := range []struct {
		inputEnvVars                   map[string]string
		expectedForwardedEnvVars       []string
		unexpectedForwardedEnvVarNames []string
	}{{
		inputEnvVars:             map[string]string{"GCE_METADATA_HOST": "www.metadata-host.com", "GCE_METADATA_ROOT": "metadata-root", "GCE_METADATA_IP": "99.100.101.102"},
		expectedForwardedEnvVars: []string{"GCE_METADATA_HOST=www.metadata-host.com", "GCE_METADATA_ROOT=metadata-root", "GCE_METADATA_IP=99.100.101.102"},
	}, {
		inputEnvVars:                   map[string]string{"https_proxy": "https-proxy-123", "http_proxy": "http-proxy-123", "no_proxy": "no-proxy-123"},
		expectedForwardedEnvVars:       []string{"https_proxy=https-proxy-123", "no_proxy=no-proxy-123"},
		unexpectedForwardedEnvVarNames: []string{"http_proxy"},
	}, {
		inputEnvVars:                   map[string]string{"http_proxy": "http-proxy-123", "no_proxy": "no-proxy-123"},
		expectedForwardedEnvVars:       []string{"http_proxy=http-proxy-123", "no_proxy=no-proxy-123"},
		unexpectedForwardedEnvVarNames: []string{"https_proxy"},
	}, {
		inputEnvVars:             map[string]string{"GOOGLE_APPLICATION_CREDENTIALS": "goog-app-cred"},
		expectedForwardedEnvVars: []string{"GOOGLE_APPLICATION_CREDENTIALS=goog-app-cred"},
	}, {
		expectedForwardedEnvVars:       []string{"GCSFUSE_IN_BACKGROUND_MODE=true"},
		unexpectedForwardedEnvVarNames: []string{"GRPC_GO_LOG_VERBOSITY_LEVEL", "GRPC_GO_LOG_SEVERITY_LEVEL", "GCE_METADATA_HOST", "GCE_METADATA_IP", "GCE_METADATA_ROOT", "http_proxy", "https_proxy", "no_proxy", "GOOGLE_APPLICATION_CREDENTIALS"},
	}, {
		inputEnvVars:             map[string]string{"GRPC_GO_LOG_VERBOSITY_LEVEL": "99", "GRPC_GO_LOG_SEVERITY_LEVEL": "INFO"},
		expectedForwardedEnvVars: []string{"GRPC_GO_LOG_VERBOSITY_LEVEL=99", "GRPC_GO_LOG_SEVERITY_LEVEL=INFO"},
	},
	} {
		for envvar, envval := range input.inputEnvVars {
			os.Setenv(envvar, envval)
		}

		forwardedEnvVars := forwardedEnvVars()

		assert.Subset(t.T(), forwardedEnvVars, input.expectedForwardedEnvVars)
		for _, forwardedEnvVar := range forwardedEnvVars {
			forwardedEnvVarName, _, ok := strings.Cut(forwardedEnvVar, "=")
			assert.True(t.T(), ok)
			assert.NotContains(t.T(), input.unexpectedForwardedEnvVarNames, forwardedEnvVarName)
		}
		assert.Contains(t.T(), forwardedEnvVars, fmt.Sprintf("PATH=%s", os.Getenv("PATH")))
		for envvar := range input.inputEnvVars {
			os.Unsetenv(envvar)
		}
	}
}
