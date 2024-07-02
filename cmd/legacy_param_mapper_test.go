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
	"net/url"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	mountpkg "github.com/googlecloudplatform/gcsfuse/v2/internal/mount"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

type mockCLIContext struct {
	cli.Context
	isFlagSet map[string]bool
}

func (m *mockCLIContext) IsSet(name string) bool {
	return m.isFlagSet[name]
}

func TestPopulateConfigFromLegacyFlags(t *testing.T) {
	var populateConfigFromLegacyFlags = []struct {
		testName          string
		legacyFlagStorage *flagStorage
		mockCLICtx        *mockCLIContext
		legacyMountConfig *config.MountConfig
		expectedConfig    *cfg.Config
		expectedErr       error
	}{
		{
			testName:          "nil flags",
			legacyFlagStorage: nil,
			mockCLICtx:        &mockCLIContext{isFlagSet: map[string]bool{}},
			legacyMountConfig: &config.MountConfig{},
			expectedConfig:    nil,
			expectedErr:       fmt.Errorf("PopulateNewConfigFromLegacyFlagsAndConfig: unexpected nil flags or mount config"),
		},
		{
			testName:          "nil config",
			legacyFlagStorage: &flagStorage{},
			mockCLICtx:        &mockCLIContext{isFlagSet: map[string]bool{}},
			legacyMountConfig: nil,
			expectedConfig:    nil,
			expectedErr:       fmt.Errorf("PopulateNewConfigFromLegacyFlagsAndConfig: unexpected nil flags or mount config"),
		},
		{
			testName: "Test decode legacy flags.",
			legacyFlagStorage: &flagStorage{
				AppName:                             "vertex",
				Foreground:                          false,
				ConfigFile:                          "~/Documents/config.yaml",
				DirMode:                             0755,
				FileMode:                            0711,
				Uid:                                 -1,
				Gid:                                 17,
				ImplicitDirs:                        true,
				OnlyDir:                             "abc",
				RenameDirLimit:                      10,
				IgnoreInterrupts:                    false,
				CustomEndpoint:                      nil,
				BillingProject:                      "billing-project",
				KeyFile:                             "~/Documents/key-file",
				TokenUrl:                            "tokenUrl",
				ReuseTokenFromUrl:                   true,
				EgressBandwidthLimitBytesPerSecond:  100,
				OpRateLimitHz:                       50,
				SequentialReadSizeMb:                40,
				AnonymousAccess:                     false,
				MaxRetrySleep:                       10,
				RetryMultiplier:                     2,
				StatCacheCapacity:                   200,
				StatCacheTTL:                        50,
				TypeCacheTTL:                        70,
				KernelListCacheTtlSeconds:           30,
				HttpClientTimeout:                   100,
				MaxRetryDuration:                    10,
				LocalFileCache:                      false,
				TempDir:                             "~/temp",
				MaxConnsPerHost:                     200,
				MaxIdleConnsPerHost:                 150,
				EnableNonexistentTypeCache:          false,
				StackdriverExportInterval:           40,
				OtelCollectorAddress:                "address",
				LogFile:                             "/tmp/log-file.json",
				LogFormat:                           "json",
				ExperimentalEnableJsonRead:          true,
				DebugGCS:                            true,
				DebugFuse:                           true,
				DebugInvariants:                     true,
				DebugMutex:                          true,
				ExperimentalMetadataPrefetchOnMount: "sync",
				ClientProtocol:                      mountpkg.HTTP1,
			},
			mockCLICtx:        &mockCLIContext{isFlagSet: map[string]bool{}},
			legacyMountConfig: &config.MountConfig{},
			expectedConfig: &cfg.Config{
				AppName:    "vertex",
				Foreground: false,
				FileSystem: cfg.FileSystemConfig{
					DirMode:                493, // Octal(755) converted to decimal
					FileMode:               457, // Octal(711) converted to decimal
					Uid:                    -1,
					Gid:                    17,
					RenameDirLimit:         10,
					IgnoreInterrupts:       false,
					DisableParallelDirops:  false,
					FuseOptions:            []string(nil),
					TempDir:                cfg.ResolvedPath(path.Join(os.Getenv("HOME"), "/temp")),
					KernelListCacheTtlSecs: 30,
				},
				ImplicitDirs: true,
				OnlyDir:      "abc",
				CacheDir:     "",
				FileCache: cfg.FileCacheConfig{
					CacheFileForRangeRead:    false,
					ParallelDownloadsPerFile: 0,
					EnableCrc:                false,
					EnableParallelDownloads:  false,
					MaxParallelDownloads:     0,
					MaxSizeMb:                0,
					DownloadChunkSizeMb:      0,
				},
				GcsAuth: cfg.GcsAuthConfig{
					KeyFile:           cfg.ResolvedPath(path.Join(os.Getenv("HOME"), "Documents/key-file")),
					TokenUrl:          "tokenUrl",
					ReuseTokenFromUrl: true,
					AnonymousAccess:   false,
				},
				GcsConnection: cfg.GcsConnectionConfig{
					CustomEndpoint:             nil,
					BillingProject:             "billing-project",
					ClientProtocol:             cfg.Protocol("http1"),
					LimitBytesPerSec:           100,
					LimitOpsPerSec:             50,
					SequentialReadSizeMb:       40,
					MaxConnsPerHost:            200,
					MaxIdleConnsPerHost:        150,
					HttpClientTimeout:          100,
					ExperimentalEnableJsonRead: true,
				},
				GcsRetries: cfg.GcsRetriesConfig{
					MaxRetrySleep: 10,
					Multiplier:    2,
				},
				Logging: cfg.LoggingConfig{
					FilePath: cfg.ResolvedPath("/tmp/log-file.json"),
					Format:   "json",
				},
				MetadataCache: cfg.MetadataCacheConfig{
					DeprecatedStatCacheCapacity:         200,
					DeprecatedStatCacheTtl:              50,
					DeprecatedTypeCacheTtl:              70,
					EnableNonexistentTypeCache:          false,
					ExperimentalMetadataPrefetchOnMount: "sync",
				},
				Metrics: cfg.MetricsConfig{
					StackdriverExportInterval: 40,
				},
				Monitoring: cfg.MonitoringConfig{
					ExperimentalOpentelemetryCollectorAddress: "address",
				},
				Debug: cfg.DebugConfig{
					ExitOnInvariantViolation: true,
					Gcs:                      true,
					LogMutex:                 true,
					Fuse:                     true,
				},
			},
			expectedErr: nil,
		},
		{
			testName: "Test decode legacy config.",
			legacyFlagStorage: &flagStorage{
				ClientProtocol: mountpkg.GRPC,
			},
			mockCLICtx: &mockCLIContext{isFlagSet: map[string]bool{}},
			legacyMountConfig: &config.MountConfig{
				WriteConfig: config.WriteConfig{
					CreateEmptyFile: true,
				},
				LogConfig: config.LogConfig{
					Severity: "info",
					Format:   "text",
					FilePath: "~/Documents/log-file.txt",
					LogRotateConfig: config.LogRotateConfig{
						MaxFileSizeMB:   20,
						BackupFileCount: 2,
						Compress:        true,
					},
				},
				FileCacheConfig: config.FileCacheConfig{
					MaxSizeMB:                20,
					CacheFileForRangeRead:    true,
					EnableParallelDownloads:  true,
					ParallelDownloadsPerFile: 3,
					MaxParallelDownloads:     6,
					DownloadChunkSizeMB:      9,
					EnableCRC:                true,
				},
				CacheDir: "~/cache-dir",
				MetadataCacheConfig: config.MetadataCacheConfig{
					TtlInSeconds:       200,
					TypeCacheMaxSizeMB: 7,
					StatCacheMaxSizeMB: 4,
				},
				ListConfig: config.ListConfig{
					EnableEmptyManagedFolders: true,
				},
				GCSConnection: config.GCSConnection{GRPCConnPoolSize: 29},
				GCSAuth:       config.GCSAuth{AnonymousAccess: true},
				EnableHNS:     true,
				FileSystemConfig: config.FileSystemConfig{
					IgnoreInterrupts:          true,
					DisableParallelDirops:     true,
					KernelListCacheTtlSeconds: 30,
				},
			},
			expectedConfig: &cfg.Config{
				Write: cfg.WriteConfig{CreateEmptyFile: true},
				Logging: cfg.LoggingConfig{
					Severity: "INFO",
					Format:   "text",
					FilePath: cfg.ResolvedPath(path.Join(os.Getenv("HOME"), "Documents/log-file.txt")),
					LogRotate: cfg.LogRotateLoggingConfig{
						MaxFileSizeMb:   20,
						BackupFileCount: 2,
						Compress:        true,
					},
				},
				FileCache: cfg.FileCacheConfig{
					MaxSizeMb:                20,
					CacheFileForRangeRead:    true,
					EnableParallelDownloads:  true,
					ParallelDownloadsPerFile: 3,
					MaxParallelDownloads:     6,
					DownloadChunkSizeMb:      9,
					EnableCrc:                true,
				},
				CacheDir: cfg.ResolvedPath(path.Join(os.Getenv("HOME"), "cache-dir")),
				MetadataCache: cfg.MetadataCacheConfig{
					TtlSecs:            200,
					TypeCacheMaxSizeMb: 7,
					StatCacheMaxSizeMb: 4,
				},
				List: cfg.ListConfig{
					EnableEmptyManagedFolders: true,
				},
				GcsConnection: cfg.GcsConnectionConfig{
					GrpcConnPoolSize: 29,
					ClientProtocol:   cfg.Protocol("grpc")},
				GcsAuth:   cfg.GcsAuthConfig{AnonymousAccess: true},
				EnableHns: true,
				FileSystem: cfg.FileSystemConfig{
					DisableParallelDirops:  true,
					IgnoreInterrupts:       true,
					KernelListCacheTtlSecs: 30,
				},
			},
			expectedErr: nil,
		},
		{
			testName: "Test overlapping flags and configs set.",
			legacyFlagStorage: &flagStorage{
				LogFile:                   "~/Documents/log-flag.txt",
				LogFormat:                 "json",
				IgnoreInterrupts:          false,
				AnonymousAccess:           false,
				KernelListCacheTtlSeconds: -1,
				ClientProtocol:            mountpkg.HTTP2,
			},
			mockCLICtx: &mockCLIContext{
				isFlagSet: map[string]bool{
					"log-file":                   true,
					"log-format":                 true,
					"ignore-interrupts":          true,
					"anonymous-access":           true,
					"kernel-list-cache-ttl-secs": true,
				},
			},
			legacyMountConfig: &config.MountConfig{
				LogConfig: config.LogConfig{
					FilePath: "~/Documents/log-config.txt",
					Format:   "text",
					Severity: "INFO",
				},
				FileSystemConfig: config.FileSystemConfig{
					IgnoreInterrupts:          true,
					KernelListCacheTtlSeconds: 100,
				},
				GCSAuth: config.GCSAuth{
					AnonymousAccess: true,
				},
			},
			expectedConfig: &cfg.Config{
				Logging: cfg.LoggingConfig{
					FilePath: cfg.ResolvedPath(path.Join(os.Getenv("HOME"), "/Documents/log-flag.txt")),
					Format:   "json",
					Severity: "INFO",
				},
				FileSystem: cfg.FileSystemConfig{
					IgnoreInterrupts:       false,
					KernelListCacheTtlSecs: -1,
				},
				GcsAuth: cfg.GcsAuthConfig{
					AnonymousAccess: false,
				},
				GcsConnection: cfg.GcsConnectionConfig{
					ClientProtocol: cfg.Protocol("http2"),
				},
			},
			expectedErr: nil,
		},
	}

	for _, tc := range populateConfigFromLegacyFlags {
		t.Run(tc.testName, func(t *testing.T) {
			resolvedConfig, err := PopulateNewConfigFromLegacyFlagsAndConfig(tc.mockCLICtx, tc.legacyFlagStorage, tc.legacyMountConfig)

			if assert.Equal(t, tc.expectedErr, err) {
				assert.Equal(t, tc.expectedConfig, resolvedConfig)
			}
		})
	}
}

func TestPopulateConfigFromLegacyFlags_KeyFileResolution(t *testing.T) {
	currentWorkingDir, err := os.Getwd()
	require.Nil(t, err)
	var keyFileTests = []struct {
		testName        string
		givenKeyFile    string
		expectedKeyFile cfg.ResolvedPath
	}{
		{
			testName:        "absolute path",
			givenKeyFile:    "/tmp/key-file.json",
			expectedKeyFile: "/tmp/key-file.json",
		},
		{
			testName:        "relative path",
			givenKeyFile:    "~/Documents/key-file.json",
			expectedKeyFile: cfg.ResolvedPath(path.Join(os.Getenv("HOME"), "/Documents/key-file.json")),
		},
		{
			testName:        "current working directory",
			givenKeyFile:    "key-file.json",
			expectedKeyFile: cfg.ResolvedPath(path.Join(currentWorkingDir, "key-file.json")),
		},
		{
			testName:        "empty path",
			givenKeyFile:    "",
			expectedKeyFile: "",
		},
	}

	for _, tc := range keyFileTests {
		t.Run(tc.testName, func(t *testing.T) {
			mockCLICtx := &mockCLIContext{}
			legacyFlagStorage := &flagStorage{
				ClientProtocol: mountpkg.HTTP2,
				KeyFile:        tc.givenKeyFile,
			}
			legacyMountCfg := &config.MountConfig{}

			resolvedConfig, err := PopulateNewConfigFromLegacyFlagsAndConfig(mockCLICtx, legacyFlagStorage, legacyMountCfg)

			if assert.Nil(t, err) {
				assert.Equal(t, tc.expectedKeyFile, resolvedConfig.GcsAuth.KeyFile)
			}
		})
	}
}

func TestPopulateConfigFromLegacyFlags_LogFileResolution(t *testing.T) {
	currentWorkingDir, err := os.Getwd()
	require.Nil(t, err)
	var logFileTests = []struct {
		testName        string
		givenLogFile    string
		expectedLogFile cfg.ResolvedPath
	}{
		{
			testName:        "absolute path",
			givenLogFile:    "/tmp/log-file.json",
			expectedLogFile: "/tmp/log-file.json",
		},
		{
			testName:        "relative path",
			givenLogFile:    "~/Documents/log-file.json",
			expectedLogFile: cfg.ResolvedPath(path.Join(os.Getenv("HOME"), "Documents/log-file.json")),
		},
		{
			testName:        "current working directory",
			givenLogFile:    "log-file.json",
			expectedLogFile: cfg.ResolvedPath(path.Join(currentWorkingDir, "log-file.json")),
		},
		{
			testName:        "empty path",
			givenLogFile:    "",
			expectedLogFile: "",
		},
	}

	for _, tc := range logFileTests {
		t.Run(tc.testName, func(t *testing.T) {
			mockCLICtx := &mockCLIContext{}
			legacyFlagStorage := &flagStorage{
				ClientProtocol: mountpkg.HTTP2,
				LogFile:        tc.givenLogFile,
			}
			legacyMountCfg := &config.MountConfig{}

			resolvedConfig, err := PopulateNewConfigFromLegacyFlagsAndConfig(mockCLICtx, legacyFlagStorage, legacyMountCfg)

			if assert.Nil(t, err) {
				assert.Equal(t, tc.expectedLogFile, resolvedConfig.Logging.FilePath)
			}
		})
	}
}

func TestCustomEndpointResolutionFromFlags(t *testing.T) {
	u, err := url.Parse("http://abc.xyz")
	require.Nil(t, err)
	legacyFlagStorage := &flagStorage{
		ClientProtocol: mountpkg.HTTP2,
		CustomEndpoint: u,
	}

	resolvedConfig, err := PopulateNewConfigFromLegacyFlagsAndConfig(&mockCLIContext{}, legacyFlagStorage, &config.MountConfig{})

	if assert.Nil(t, err) && assert.NotNil(t, resolvedConfig.GcsConnection.CustomEndpoint) {
		assert.Equal(t, *resolvedConfig.GcsConnection.CustomEndpoint, *u)
	}
}
