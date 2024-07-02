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
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"

	mountpkg "github.com/googlecloudplatform/gcsfuse/v2/internal/mount"
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
	flags := &flagStorage{
		MaxConnsPerHost:     5,
		MaxIdleConnsPerHost: 100,
		HttpClientTimeout:   5,
		MaxRetrySleep:       7,
		RetryMultiplier:     2,
	}
	mountConfig := &config.MountConfig{}
	newConfig := &cfg.Config{
		GcsConnection: cfg.GcsConnectionConfig{ClientProtocol: cfg.HTTP1},
		GcsAuth:       cfg.GcsAuthConfig{KeyFile: "testdata/test_creds.json"}}

	userAgent := "AppName"
	storageHandle, err := createStorageHandle(newConfig, flags, mountConfig, userAgent)

	assert.Equal(t.T(), nil, err)
	assert.NotEqual(t.T(), nil, storageHandle)
}

func (t *MainTest) TestCreateStorageHandle_WithClientProtocolAsGRPC() {
	flags := &flagStorage{
		MaxConnsPerHost:     5,
		MaxIdleConnsPerHost: 100,
		HttpClientTimeout:   5,
		MaxRetrySleep:       7,
		RetryMultiplier:     2,
	}
	mountConfig := &config.MountConfig{
		GCSConnection: config.GCSConnection{GRPCConnPoolSize: 1},
	}
	newConfig := &cfg.Config{
		GcsConnection: cfg.GcsConnectionConfig{ClientProtocol: cfg.GRPC},
		GcsAuth:       cfg.GcsAuthConfig{KeyFile: "testdata/test_creds.json"},
	}

	userAgent := "AppName"
	storageHandle, err := createStorageHandle(newConfig, flags, mountConfig, userAgent)

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

func (t *MainTest) TestStringifyShouldReturnAllFlagsPassedInMountConfigAsMarshalledString() {
	mountConfig := &config.MountConfig{
		WriteConfig: config.WriteConfig{
			CreateEmptyFile: false,
		},
		LogConfig: config.LogConfig{
			Severity: config.TRACE,
			FilePath: "\"path\"to\"file\"",
			LogRotateConfig: config.LogRotateConfig{
				MaxFileSizeMB:   2,
				BackupFileCount: 2,
				Compress:        true,
			},
		},
		ListConfig: config.ListConfig{
			EnableEmptyManagedFolders: false,
		},
		EnableHNS: true,
	}

	actual, err := util.Stringify(mountConfig)
	assert.Equal(t.T(), nil, err)

	expected := strings.Join([]string{
		`{"CreateEmptyFile":false`,
		`"Severity":"TRACE"`,
		`"Format":""`,
		`"FilePath":"\"path\"to\"file\""`,
		`"LogRotateConfig":{"MaxFileSizeMB":2`,
		`"BackupFileCount":2`,
		`"Compress":true}`,
		`"MaxSizeMB":0`,
		`"CacheFileForRangeRead":false`,
		`"EnableParallelDownloads":false`,
		`"ParallelDownloadsPerFile":0`,
		`"MaxParallelDownloads":0`,
		`"DownloadChunkSizeMB":0`,
		`"EnableCRC":false`,
		`"CacheDir":""`,
		`"TtlInSeconds":0`,
		`"TypeCacheMaxSizeMB":0`,
		`"StatCacheMaxSizeMB":0`,
		`"EnableEmptyManagedFolders":false`,
		`"GRPCConnPoolSize":0`,
		`"AnonymousAccess":false`,
		`"EnableHNS":true`,
		`"IgnoreInterrupts":false`,
		`"DisableParallelDirops":false`,
		`"KernelListCacheTtlSeconds":0}`,
	}, ",")
	assert.Equal(t.T(), expected, actual)
}

func (t *MainTest) TestEnableHNSFlagFalse() {
	mountConfig := &config.MountConfig{
		EnableHNS: false,
	}

	actual, err := util.Stringify(mountConfig)
	assert.Equal(t.T(), nil, err)

	expected := strings.Join([]string{
		`{"CreateEmptyFile":false`,
		`"Severity":""`,
		`"Format":""`,
		`"FilePath":""`,
		`"LogRotateConfig":{"MaxFileSizeMB":0`,
		`"BackupFileCount":0`,
		`"Compress":false}`,
		`"MaxSizeMB":0`,
		`"CacheFileForRangeRead":false`,
		`"EnableParallelDownloads":false`,
		`"ParallelDownloadsPerFile":0`,
		`"MaxParallelDownloads":0`,
		`"DownloadChunkSizeMB":0`,
		`"EnableCRC":false`,
		`"CacheDir":""`,
		`"TtlInSeconds":0`,
		`"TypeCacheMaxSizeMB":0`,
		`"StatCacheMaxSizeMB":0`,
		`"EnableEmptyManagedFolders":false`,
		`"GRPCConnPoolSize":0`,
		`"AnonymousAccess":false`,
		`"EnableHNS":false`,
		`"IgnoreInterrupts":false`,
		`"DisableParallelDirops":false`,
		`"KernelListCacheTtlSeconds":0}`,
	}, ",")
	assert.Equal(t.T(), expected, actual)
}

func (t *MainTest) TestStringifyShouldReturnAllFlagsPassedInFlagStorageAsMarshalledString() {
	mountOptions := map[string]string{
		"1": "one",
		"2": "two",
		"3": "three",
	}
	flags := &flagStorage{
		SequentialReadSizeMb:      10,
		ClientProtocol:            mountpkg.ClientProtocol("http4"),
		MountOptions:              mountOptions,
		KernelListCacheTtlSeconds: -1,
	}

	actual, err := util.Stringify(flags)
	assert.Equal(t.T(), nil, err)

	expected := strings.Join([]string{
		`{"AppName":""`,
		`"Foreground":false`,
		`"ConfigFile":""`,
		`"MountOptions":{"1":"one"`,
		`"2":"two"`,
		`"3":"three"}`,
		`"DirMode":0`,
		`"FileMode":0`,
		`"Uid":0`,
		`"Gid":0`,
		`"ImplicitDirs":false`,
		`"OnlyDir":""`,
		`"RenameDirLimit":0`,
		`"IgnoreInterrupts":false`,
		`"CustomEndpoint":null`,
		`"BillingProject":""`,
		`"KeyFile":""`,
		`"TokenUrl":""`,
		`"ReuseTokenFromUrl":false`,
		`"EgressBandwidthLimitBytesPerSecond":0`,
		`"OpRateLimitHz":0`,
		`"SequentialReadSizeMb":10`,
		`"AnonymousAccess":false`,
		`"MaxRetrySleep":0`,
		`"StatCacheCapacity":0`,
		`"StatCacheTTL":0`,
		`"TypeCacheTTL":0`,
		`"KernelListCacheTtlSeconds":-1`,
		`"HttpClientTimeout":0`,
		`"MaxRetryDuration":0`,
		`"RetryMultiplier":0`,
		`"LocalFileCache":false`,
		`"TempDir":""`,
		`"ClientProtocol":"http4"`,
		`"MaxConnsPerHost":0`,
		`"MaxIdleConnsPerHost":0`,
		`"EnableNonexistentTypeCache":false`,
		`"StackdriverExportInterval":0`,
		`"OtelCollectorAddress":""`,
		`"LogFile":""`,
		`"LogFormat":""`,
		`"ExperimentalEnableJsonRead":false`,
		`"DebugFuseErrors":false`,
		`"DebugFuse":false`,
		`"DebugFS":false`,
		`"DebugGCS":false`,
		`"DebugHTTP":false`,
		`"DebugInvariants":false`,
		`"DebugMutex":false`,
		`"ExperimentalMetadataPrefetchOnMount":""}`,
	}, ",")
	assert.Equal(t.T(), expected, actual)
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
