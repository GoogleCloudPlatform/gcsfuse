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
	"reflect"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	assert.Nil(t.T(), err)
	assert.NotNil(t.T(), storageHandle)
}

func (t *MainTest) TestCreateStorageHandle_WithClientProtocolAsGRPC() {
	newConfig := &cfg.Config{
		GcsConnection: cfg.GcsConnectionConfig{ClientProtocol: cfg.GRPC},
		GcsAuth:       cfg.GcsAuthConfig{KeyFile: "testdata/test_creds.json"},
	}

	storageHandle, err := createStorageHandle(newConfig, "AppName", metrics.NewNoopMetrics())

	assert.Nil(t.T(), err)
	assert.NotNil(t.T(), storageHandle)
}

func (t *MainTest) TestGetUserAgentWhenMetadataImageTypeEnvVarIsSet() {
	t.T().Setenv("GCSFUSE_METADATA_IMAGE_TYPE", "DLVM")
	mountConfig := &cfg.Config{}

	userAgent := getUserAgent("AppName", getConfigForUserAgent(mountConfig), "testFS-123")

	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s AppName (GPN:gcsfuse-DLVM) (Cfg:0:0:0:0:0:0) (mount-id:testFS-123)", common.GetVersion()))
	assert.Equal(t.T(), expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentWhenMetadataImageTypeEnvVarIsNotSet() {
	mountConfig := &cfg.Config{}

	userAgent := getUserAgent("AppName", getConfigForUserAgent(mountConfig), "testFS-123")

	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0:0:0:0) (mount-id:testFS-123)", common.GetVersion()))
	assert.Equal(t.T(), expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentConfigWithNoFileCache() {
	mountConfig := &cfg.Config{}

	userAgent := getUserAgent("AppName", getConfigForUserAgent(mountConfig), "testFS-123")

	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0:0:0:0) (mount-id:testFS-123)", common.GetVersion()))
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
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0:0:0:0) (mount-id:testFS-123)", common.GetVersion())),
		},
		{
			name: "Config with file cache disabled where maxsize is set but cache dir is not set.",
			mountConfig: &cfg.Config{
				FileCache: cfg.FileCacheConfig{
					MaxSizeMb: -1,
				},
			},
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0:0:0:0) (mount-id:testFS-123)", common.GetVersion())),
		},
		{
			name: "Config with file cache enabled but random read disabled.",
			mountConfig: &cfg.Config{
				CacheDir: "//tmp//folder//",
				FileCache: cfg.FileCacheConfig{
					MaxSizeMb: -1,
				},
			},
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:1:0:0:0:0:0) (mount-id:testFS-123)", common.GetVersion())),
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
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:1:1:0:0:0:0) (mount-id:testFS-123)", common.GetVersion())),
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
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0:0:0:0) (mount-id:testFS-123)", common.GetVersion())),
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
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:1:0:1:0:0:0) (mount-id:testFS-123)", common.GetVersion())),
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
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:1:1:1:0:0:0) (mount-id:testFS-123)", common.GetVersion())),
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
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:1:0:1:1:0:0) (mount-id:testFS-123)", common.GetVersion())),
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
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:1:1:0:0:0:0) (mount-id:testFS-123)", common.GetVersion())),
		},
		{
			name: "buffered_read_enabled",
			mountConfig: &cfg.Config{
				Read: cfg.ReadConfig{EnableBufferedRead: true},
			},
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0:0:1:0) (mount-id:testFS-123)", common.GetVersion())),
		},
		{
			name: "buffered_read_disabled",
			mountConfig: &cfg.Config{
				Read: cfg.ReadConfig{EnableBufferedRead: false},
			},
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0:0:0:0) (mount-id:testFS-123)", common.GetVersion())),
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
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:1:0:0:0:1:0) (mount-id:testFS-123)", common.GetVersion())),
		},
		{
			name: "profile_enabled_aiml_training",
			mountConfig: &cfg.Config{
				Profile: cfg.ProfileAIMLTraining,
			},
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0:0:0:1) (mount-id:testFS-123)", common.GetVersion())),
		},
		{
			name: "profile_enabled_aiml_serving",
			mountConfig: &cfg.Config{
				Profile: cfg.ProfileAIMLServing,
			},
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0:0:0:1) (mount-id:testFS-123)", common.GetVersion())),
		},
		{
			name: "profile_enabled_aiml_checkpointing",
			mountConfig: &cfg.Config{
				Profile: cfg.ProfileAIMLCheckpointing,
			},
			expectedUserAgent: strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (Cfg:0:0:0:0:0:1) (mount-id:testFS-123)", common.GetVersion())),
		},
	}

	for _, tc := range testCases {
		t.T().Run(tc.name, func(t *testing.T) {
			userAgent := getUserAgent("AppName", getConfigForUserAgent(tc.mountConfig), "testFS-123")

			assert.Equal(t, tc.expectedUserAgent, userAgent)
		})
	}
}

func (t *MainTest) TestGetUserAgentWhenMetadataImageTypeEnvVarSetAndAppNameNotSet() {
	t.T().Setenv("GCSFUSE_METADATA_IMAGE_TYPE", "DLVM")
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-DLVM) (Cfg:0:0:0:0:0:0) (mount-id:testFS-123)", common.GetVersion()))
	mountConfig := &cfg.Config{}

	userAgent := getUserAgent("", getConfigForUserAgent(mountConfig), "testFS-123")

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
	testCases := []struct {
		name       string
		bucketName string
		isDynamic  bool
	}{
		{
			name:       "Empty bucket name",
			bucketName: "",
			isDynamic:  true,
		},
		{
			name:       "Underscore bucket name",
			bucketName: "_",
			isDynamic:  true,
		},
		{
			name:       "Regular bucket name",
			bucketName: "abc",
			isDynamic:  false,
		},
	}

	for _, tc := range testCases {
		t.T().Run(tc.name, func(t *testing.T) {
			isDynamic := isDynamicMount(tc.bucketName)

			assert.Equal(t, tc.isDynamic, isDynamic)
		})
	}
}

func (t *MainTest) TestFSName() {
	testCases := []struct {
		name       string
		bucketName string
		fsName     string
	}{
		{
			name:       "Empty bucket name",
			bucketName: "",
			fsName:     DynamicMountFSName,
		},
		{
			name:       "Underscore bucket name",
			bucketName: "_",
			fsName:     DynamicMountFSName,
		},
		{
			name:       "Regular bucket name",
			bucketName: "abc",
			fsName:     "abc",
		},
	}

	for _, tc := range testCases {
		t.T().Run(tc.name, func(t *testing.T) {
			actualFSName := fsName(tc.bucketName)

			assert.Equal(t, tc.fsName, actualFSName)
		})
	}
}

func (t *MainTest) TestForwardedEnvVars_AlwaysPresent() {
	// These variables are always added to the forwarded environment.
	homeDir, err := os.UserHomeDir()
	require.NoError(t.T(), err)
	parentDir, err := os.Getwd()
	require.NoError(t.T(), err)
	expectedForwardedEnvVars := []string{
		"GCSFUSE_IN_BACKGROUND_MODE=true",
		"GCSFUSE_MOUNT_UUID=" + logger.MountUUID(),
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + homeDir,
		util.GCSFUSE_PARENT_PROCESS_DIR + "=" + parentDir,
	}

	forwardedEnvVars := forwardedEnvVars()

	assert.Subset(t.T(), forwardedEnvVars, expectedForwardedEnvVars)
}

func (t *MainTest) TestForwardedEnvVars_Precedence() {
	// This test handles cases where the presence of one env var affects another.
	testCases := []struct {
		name                           string
		inputEnvVars                   map[string]string
		expectedForwardedEnvVars       []string
		unexpectedForwardedEnvVarNames []string
	}{
		{
			name:                           "https_proxy is forwarded over http_proxy",
			inputEnvVars:                   map[string]string{"https_proxy": "https-proxy-123", "http_proxy": "http-proxy-123"},
			expectedForwardedEnvVars:       []string{"https_proxy=https-proxy-123"},
			unexpectedForwardedEnvVarNames: []string{"http_proxy"},
		},
		{
			name:                           "http_proxy is forwarded when https_proxy is not set",
			inputEnvVars:                   map[string]string{"http_proxy": "http-proxy-123"},
			expectedForwardedEnvVars:       []string{"http_proxy=http-proxy-123"},
			unexpectedForwardedEnvVarNames: []string{"https_proxy"},
		},
	}

	for _, tc := range testCases {
		t.T().Run(tc.name, func(t *testing.T) {
			for k, v := range tc.inputEnvVars {
				t.Setenv(k, v)
			}

			forwardedEnvVars := forwardedEnvVars()

			assert.Subset(t, forwardedEnvVars, tc.expectedForwardedEnvVars)
			// Verify that none of the unexpected variables were forwarded.
			for _, forwardedVar := range forwardedEnvVars {
				name, _, ok := strings.Cut(forwardedVar, "=")
				require.True(t, ok, "Invalid env var format: %s", forwardedVar)
				assert.NotContains(t, tc.unexpectedForwardedEnvVarNames, name, "unexpected env var %q was forwarded", name)
			}
		})
	}
}

func (t *MainTest) TestForwardedEnvVars_PassedWhenSet() {
	// These variables are only forwarded if they are set in the environment.
	testCases := []struct {
		name                     string
		inputEnvVars             map[string]string
		expectedForwardedEnvVars []string
	}{
		{
			name:                     "GCE metadata env vars",
			inputEnvVars:             map[string]string{"GCE_METADATA_HOST": "www.metadata-host.com", "GCE_METADATA_ROOT": "metadata-root", "GCE_METADATA_IP": "99.100.101.102"},
			expectedForwardedEnvVars: []string{"GCE_METADATA_HOST=www.metadata-host.com", "GCE_METADATA_ROOT=metadata-root", "GCE_METADATA_IP=99.100.101.102"},
		},
		{
			name:                     "GOOGLE_APPLICATION_CREDENTIALS",
			inputEnvVars:             map[string]string{"GOOGLE_APPLICATION_CREDENTIALS": "goog-app-cred"},
			expectedForwardedEnvVars: []string{"GOOGLE_APPLICATION_CREDENTIALS=goog-app-cred"},
		},
		{
			name:                     "GRPC debug env vars",
			inputEnvVars:             map[string]string{"GRPC_GO_LOG_VERBOSITY_LEVEL": "99", "GRPC_GO_LOG_SEVERITY_LEVEL": "INFO"},
			expectedForwardedEnvVars: []string{"GRPC_GO_LOG_VERBOSITY_LEVEL=99", "GRPC_GO_LOG_SEVERITY_LEVEL=INFO"},
		},
		{
			name:                     "no_proxy",
			inputEnvVars:             map[string]string{"no_proxy": "no-proxy-123"},
			expectedForwardedEnvVars: []string{"no_proxy=no-proxy-123"},
		},
	}

	for _, tc := range testCases {
		t.T().Run(tc.name, func(t *testing.T) {
			for k, v := range tc.inputEnvVars {
				t.Setenv(k, v)
			}

			forwardedEnvVars := forwardedEnvVars()

			assert.Subset(t, forwardedEnvVars, tc.expectedForwardedEnvVars)
		})
	}
}

func (t *MainTest) TestForwardedEnvVars_NotPassedWhenUnset() {
	// These variables should NOT be forwarded if they are not set.
	unexpectedForwardedEnvVars := []string{
		"GCE_METADATA_HOST",
		"GCE_METADATA_ROOT",
		"GCE_METADATA_IP",
		"GOOGLE_APPLICATION_CREDENTIALS",
		"GRPC_GO_LOG_VERBOSITY_LEVEL",
		"GRPC_GO_LOG_SEVERITY_LEVEL",
		"no_proxy",
	}

	forwardedEnvVars := forwardedEnvVars()

	// Verify that none of the unexpected/unset variables were forwarded.
	for _, forwardedVar := range forwardedEnvVars {
		name, _, ok := strings.Cut(forwardedVar, "=")
		require.True(t.T(), ok, "Invalid env var format: %s", forwardedVar)
		assert.NotContains(t.T(), unexpectedForwardedEnvVars, name, "unexpected env var %q was forwarded", name)
	}
}

func (t *MainTest) TestGetDeviceMajorMinor() {
	testCases := []struct {
		name        string
		mountPoint  string
		expectedErr bool
	}{
		{
			name:        "Existing mount point",
			mountPoint:  "/tmp",
			expectedErr: false,
		},
		{
			name:        "Non-existing mount point",
			mountPoint:  "/path/to/non/existing/mountpoint",
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.T().Run(tc.name, func(t *testing.T) {
			_, _, err := getDeviceMajorMinor(tc.mountPoint)

			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func (t *MainTest) TestCreateHierarchicalMap_Positive() {
	testCases := []struct {
		name     string
		inputMap map[string]cfg.OptimizationResult
		expected map[string]any
	}{
		{
			name:     "Empty map",
			inputMap: map[string]cfg.OptimizationResult{},
			expected: map[string]any{},
		},
		{
			name: "Flat keys",
			inputMap: map[string]cfg.OptimizationResult{
				"key1": {FinalValue: "value1", OptimizationReason: "reason1"},
				"key2": {FinalValue: 123, OptimizationReason: "reason2"},
			},
			expected: map[string]any{
				"key1": cfg.OptimizationResult{FinalValue: "value1", OptimizationReason: "reason1"},
				"key2": cfg.OptimizationResult{FinalValue: 123, OptimizationReason: "reason2"},
			},
		},
		{
			name: "Single level nesting",
			inputMap: map[string]cfg.OptimizationResult{
				"a.b": {FinalValue: "valueAB", OptimizationReason: "reasonAB"},
				"a.c": {FinalValue: "valueAC", OptimizationReason: "reasonAC"},
			},
			expected: map[string]any{
				"a": map[string]any{
					"b": cfg.OptimizationResult{FinalValue: "valueAB", OptimizationReason: "reasonAB"},
					"c": cfg.OptimizationResult{FinalValue: "valueAC", OptimizationReason: "reasonAC"},
				},
			},
		},
		{
			name: "Multi-level nesting",
			inputMap: map[string]cfg.OptimizationResult{
				"a.b.c": {FinalValue: "valueABC", OptimizationReason: "reasonABC"},
				"a.b.d": {FinalValue: "valueABD", OptimizationReason: "reasonABD"},
				"x.y.z": {FinalValue: true, OptimizationReason: "reasonXYZ"},
			},
			expected: map[string]any{
				"a": map[string]any{
					"b": map[string]any{
						"c": cfg.OptimizationResult{FinalValue: "valueABC", OptimizationReason: "reasonABC"},
						"d": cfg.OptimizationResult{FinalValue: "valueABD", OptimizationReason: "reasonABD"},
					},
				},
				"x": map[string]any{
					"y": map[string]any{
						"z": cfg.OptimizationResult{FinalValue: true, OptimizationReason: "reasonXYZ"},
					},
				},
			},
		},
		{
			name: "No conflict complex keys",
			inputMap: map[string]cfg.OptimizationResult{
				"metadata-cache.ttl-secs":               {FinalValue: int64(-1), OptimizationReason: "reasonTTL"},
				"metadata-cache.stat-cache-max-size-mb": {FinalValue: int64(1024), OptimizationReason: "reasonStat"},
				"file-cache.cache-file-for-range-read":  {FinalValue: true, OptimizationReason: "reasonFileCache"},
			},
			expected: map[string]any{
				"metadata-cache": map[string]any{
					"ttl-secs":               cfg.OptimizationResult{FinalValue: int64(-1), OptimizationReason: "reasonTTL"},
					"stat-cache-max-size-mb": cfg.OptimizationResult{FinalValue: int64(1024), OptimizationReason: "reasonStat"},
				},
				"file-cache": map[string]any{
					"cache-file-for-range-read": cfg.OptimizationResult{FinalValue: true, OptimizationReason: "reasonFileCache"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.T().Run(tc.name, func(t *testing.T) {
			got, err := createHierarchicalMap(tc.inputMap)

			assert.NoError(t, err)
			if !reflect.DeepEqual(tc.expected, got) {
				t.Errorf("createHierarchicalMap() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func (t *MainTest) TestCreateHierarchicalMap_Negative() {
	testCases := []struct {
		name     string
		inputMap map[string]cfg.OptimizationResult
	}{
		{
			name: "Conflict: Prefix as terminal key first",
			inputMap: map[string]cfg.OptimizationResult{
				"a.b":   {FinalValue: "valAB", OptimizationReason: "rAB"},
				"a.b.d": {FinalValue: "valABD", OptimizationReason: "rABD"},
			},
		},
		{
			name: "Conflict: Path key first",
			inputMap: map[string]cfg.OptimizationResult{
				"a.b.d": {FinalValue: "valABD", OptimizationReason: "rABD"},
				"a.b":   {FinalValue: "valAB", OptimizationReason: "rAB"},
			},
		},
		{
			name: "Conflict: Deeper nesting",
			inputMap: map[string]cfg.OptimizationResult{
				"a.b.c":   {FinalValue: "valABC", OptimizationReason: "rABC"},
				"a.b.c.d": {FinalValue: "valABCD", OptimizationReason: "rABCD"},
			},
		},
	}

	for _, tc := range testCases {
		t.T().Run(tc.name, func(t *testing.T) {
			got, err := createHierarchicalMap(tc.inputMap)

			assert.Error(t, err)
			assert.Nil(t, got)
			assert.Contains(t, err.Error(), "key conflict")
		})
	}
}
