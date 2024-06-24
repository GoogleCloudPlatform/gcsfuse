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
	"net/url"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	mountpkg "github.com/googlecloudplatform/gcsfuse/v2/internal/mount"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/urfave/cli"
)

type LegacyToNewConfigTestSuite struct {
	suite.Suite
}

func TestLegacyToNewConfigTestSuite(t *testing.T) {
	suite.Run(t, new(LegacyToNewConfigTestSuite))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

type MockContext struct {
	cli.Context
	isFlagSet map[string]bool
}

func (m *MockContext) IsSet(name string) bool {
	return m.isFlagSet[name]
}

func (t *LegacyToNewConfigTestSuite) TestPopulateConfigFromLegacyFlags() {
	var populateConfigFromLegacyFlags = []struct {
		testName          string
		legacyFlagStorage *flagStorage
		isFlagSet         map[string]bool
		legacyMountConfig *config.MountConfig
		expectedConfig    *cfg.Config
	}{
		{testName: "Test decode legacy flags.",
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
				DebugInvariants:                     true,
				DebugMutex:                          true,
				ExperimentalMetadataPrefetchOnMount: "sync",
				ClientProtocol:                      mountpkg.HTTP1,
			},
			isFlagSet:         map[string]bool{},
			legacyMountConfig: &config.MountConfig{},
			expectedConfig: &cfg.Config{
				AppName:    "vertex",
				Foreground: false,
				FileSystem: cfg.FileSystemConfig{
					DirMode:               493, // Octal(755) converted to decimal
					FileMode:              457, // Octal(711) converted to decimal
					Uid:                   -1,
					Gid:                   17,
					RenameDirLimit:        10,
					IgnoreInterrupts:      false,
					DisableParallelDirops: false,
					FuseOptions:           []string(nil),
					TempDir:               cfg.ResolvedPath(path.Join(os.Getenv("HOME"), "/temp")),
				},
				ImplicitDirs: true,
				OnlyDir:      "abc",
				CacheDir:     "",
				FileCache: cfg.FileCacheConfig{
					CacheFileForRangeRead:      false,
					DownloadParallelismPerFile: 0,
					EnableCrcCheck:             false,
					EnableParallelDownloads:    false,
					MaxDownloadParallelism:     0,
					MaxSizeMb:                  0,
					ReadRequestSizeMb:          0,
				},
				GcsAuth: cfg.GcsAuthConfig{
					KeyFile:           cfg.ResolvedPath(path.Join(os.Getenv("HOME"), "Documents/key-file")),
					TokenUrl:          "tokenUrl",
					ReuseTokenFromUrl: true,
					AnonymousAccess:   false,
				},
				GcsConnection: cfg.GcsConnectionConfig{
					CustomEndpoint:             url.URL{},
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
				List: cfg.ListConfig{
					KernelListCacheTtlSecs: 30,
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
				},
			},
		},
		{testName: "Test decode legacy config.",
			legacyFlagStorage: &flagStorage{
				ClientProtocol: mountpkg.GRPC,
			},
			isFlagSet: map[string]bool{},
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
					MaxSizeMB:                  20,
					CacheFileForRangeRead:      true,
					EnableParallelDownloads:    true,
					DownloadParallelismPerFile: 3,
					MaxDownloadParallelism:     6,
					ReadRequestSizeMB:          9,
					EnableCrcCheck:             true,
				},
				CacheDir: "~/cache-dir",
				MetadataCacheConfig: config.MetadataCacheConfig{
					TtlInSeconds:       200,
					TypeCacheMaxSizeMB: 7,
					StatCacheMaxSizeMB: 4,
				},
				ListConfig: config.ListConfig{
					EnableEmptyManagedFolders: true,
					KernelListCacheTtlSeconds: 30,
				},
				GCSConnection: config.GCSConnection{GRPCConnPoolSize: 29},
				GCSAuth:       config.GCSAuth{AnonymousAccess: true},
				EnableHNS:     true,
				FileSystemConfig: config.FileSystemConfig{
					IgnoreInterrupts:      true,
					DisableParallelDirops: true,
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
					MaxSizeMb:                  20,
					CacheFileForRangeRead:      true,
					EnableParallelDownloads:    true,
					DownloadParallelismPerFile: 3,
					MaxDownloadParallelism:     6,
					ReadRequestSizeMb:          9,
					EnableCrcCheck:             true,
				},
				CacheDir: cfg.ResolvedPath(path.Join(os.Getenv("HOME"), "cache-dir")),
				MetadataCache: cfg.MetadataCacheConfig{
					TtlSecs:            200,
					TypeCacheMaxSizeMb: 7,
					StatCacheMaxSizeMb: 4,
				},
				List: cfg.ListConfig{
					EnableEmptyManagedFolders: true,
					KernelListCacheTtlSecs:    30,
				},
				GcsConnection: cfg.GcsConnectionConfig{
					GrpcConnPoolSize: 29,
					ClientProtocol:   cfg.Protocol("grpc")},
				GcsAuth:   cfg.GcsAuthConfig{AnonymousAccess: true},
				EnableHns: true,
				FileSystem: cfg.FileSystemConfig{
					DisableParallelDirops: true,
					IgnoreInterrupts:      true,
				},
			},
		},
		{testName: "Test overlapping flags and configs set.",
			legacyFlagStorage: &flagStorage{
				LogFile:                   "~/Documents/log-flag.txt",
				LogFormat:                 "json",
				IgnoreInterrupts:          false,
				AnonymousAccess:           false,
				KernelListCacheTtlSeconds: -1,
				ClientProtocol:            mountpkg.HTTP2,
			},
			isFlagSet: map[string]bool{
				"log-file":                   true,
				"log-format":                 true,
				"ignore-interrupts":          true,
				"anonymous-access":           true,
				"kernel-list-cache-ttl-secs": true,
			},
			legacyMountConfig: &config.MountConfig{
				LogConfig: config.LogConfig{
					FilePath: "~/Documents/log-config.txt",
					Format:   "text",
					Severity: "INFO",
				},
				FileSystemConfig: config.FileSystemConfig{
					IgnoreInterrupts: true,
				},
				GCSAuth: config.GCSAuth{
					AnonymousAccess: true,
				},
				ListConfig: config.ListConfig{
					KernelListCacheTtlSeconds: 100,
				},
			},
			expectedConfig: &cfg.Config{
				Logging: cfg.LoggingConfig{
					FilePath: cfg.ResolvedPath(path.Join(os.Getenv("HOME"), "/Documents/log-flag.txt")),
					Format:   "json",
					Severity: "INFO",
				},
				FileSystem: cfg.FileSystemConfig{
					IgnoreInterrupts: false,
				},
				GcsAuth: cfg.GcsAuthConfig{
					AnonymousAccess: false,
				},
				List: cfg.ListConfig{
					KernelListCacheTtlSecs: -1,
				},
				GcsConnection: cfg.GcsConnectionConfig{
					ClientProtocol: cfg.Protocol("http2"),
				},
			},
		},
	}

	for _, tt := range populateConfigFromLegacyFlags {
		t.T().Run(tt.testName, func(m *testing.T) {
			testContext := &MockContext{isFlagSet: tt.isFlagSet}

			resolvedConfig, err := PopulateNewConfigFromLegacyFlagsAndConfig(testContext, tt.legacyFlagStorage, tt.legacyMountConfig)

			assert.Nil(t.T(), err)
			assert.Equal(t.T(), tt.expectedConfig, resolvedConfig)
		})
	}
}
