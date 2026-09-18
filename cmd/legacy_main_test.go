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

	storageHandle, err := createStorageHandle(newConfig, "AppName", "AppName-Config", metrics.NewNoopMetrics(), false, false)

	assert.Nil(t.T(), err)
	assert.NotNil(t.T(), storageHandle)
}

func (t *MainTest) TestCreateStorageHandle_WithClientProtocolAsGRPC() {
	newConfig := &cfg.Config{
		GcsConnection: cfg.GcsConnectionConfig{ClientProtocol: cfg.GRPC},
		GcsAuth:       cfg.GcsAuthConfig{KeyFile: "testdata/test_creds.json"},
	}

	storageHandle, err := createStorageHandle(newConfig, "AppName", "AppName-Config", metrics.NewNoopMetrics(), false, false)

	assert.Nil(t.T(), err)
	assert.NotNil(t.T(), storageHandle)
}

func (t *MainTest) TestCreateStorageHandle_WithClientProtocolAsGRPCIsGKE() {
	newConfig := &cfg.Config{
		GcsConnection: cfg.GcsConnectionConfig{ClientProtocol: cfg.GRPC},
		GcsAuth:       cfg.GcsAuthConfig{KeyFile: "testdata/test_creds.json"},
	}

	storageHandle, err := createStorageHandle(newConfig, "AppName", "AppName-Config", metrics.NewNoopMetrics(), true, false)

	assert.Nil(t.T(), err)
	assert.NotNil(t.T(), storageHandle)
}

func (t *MainTest) TestGetUserAgentWhenMetadataImageTypeEnvVarIsSet() {
	t.T().Setenv("GCSFUSE_METADATA_IMAGE_TYPE", "DLVM")

	userAgent := getUserAgent("AppName", "testFS-123")

	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s AppName (GPN:gcsfuse-DLVM) (mount-id:testFS-123)", common.GetVersion()))
	assert.Equal(t.T(), expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentWhenMetadataImageTypeEnvVarIsNotSet() {
	userAgent := getUserAgent("AppName", "testFS-123")

	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (mount-id:testFS-123)", common.GetVersion()))
	assert.Equal(t.T(), expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentWhenMetadataImageTypeEnvVarSetAndAppNameNotSet() {
	t.T().Setenv("GCSFUSE_METADATA_IMAGE_TYPE", "DLVM")
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-DLVM) (mount-id:testFS-123)", common.GetVersion()))

	userAgent := getUserAgent("", "testFS-123")

	assert.Equal(t.T(), expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentWhenNoAppNameAndNoEnvVar() {
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse) (mount-id:testFS-123)", common.GetVersion()))

	userAgent := getUserAgent("", "testFS-123")

	assert.Equal(t.T(), expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentWithConfig_SanitizationAndSerialization() {
	t.T().Setenv("GCSFUSE_METADATA_IMAGE_TYPE", "DLVM")
	mountConfig := &cfg.Config{}

	userAgent := getUserAgentWithConfig("AppName", mountConfig, "testFS-123")

	expectedConfigProto, err := cfg.SerializeConfigToProtoBase64(mountConfig)
	require.NoError(t.T(), err)
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s AppName (GPN:gcsfuse-DLVM) (CfgProto:%s) (mount-id:testFS-123)", common.GetVersion(), expectedConfigProto))
	assert.Equal(t.T(), expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentWithConfig_WithoutEnvVar() {
	mountConfig := &cfg.Config{}

	userAgent := getUserAgentWithConfig("AppName", mountConfig, "testFS-123")

	expectedConfigProto, err := cfg.SerializeConfigToProtoBase64(mountConfig)
	require.NoError(t.T(), err)
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (CfgProto:%s) (mount-id:testFS-123)", common.GetVersion(), expectedConfigProto))
	assert.Equal(t.T(), expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentWithConfig_NilConfig() {
	userAgent := getUserAgentWithConfig("AppName", nil, "testFS-123")

	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName) (CfgProto:) (mount-id:testFS-123)", common.GetVersion()))
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
			name:                     "GOOGLE_CLOUD_PROJECT",
			inputEnvVars:             map[string]string{"GOOGLE_CLOUD_PROJECT": "my-test-project"},
			expectedForwardedEnvVars: []string{"GOOGLE_CLOUD_PROJECT=my-test-project"},
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
		"GOOGLE_CLOUD_PROJECT",
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
