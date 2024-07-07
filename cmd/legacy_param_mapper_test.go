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
	"math"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

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
				MaxRetryAttempts:                    100,
				RetryMultiplier:                     2,
				StatCacheCapacity:                   200,
				StatCacheTTL:                        50,
				TypeCacheTTL:                        70,
				KernelListCacheTtlSeconds:           30,
				HttpClientTimeout:                   100,
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
					MaxRetrySleep:    10,
					MaxRetryAttempts: 100,
					Multiplier:       2,
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
				MaxRetryAttempts:          100,
				ClientProtocol:            mountpkg.HTTP2,
			},
			mockCLICtx: &mockCLIContext{
				isFlagSet: map[string]bool{
					"log-file":                   true,
					"log-format":                 true,
					"ignore-interrupts":          true,
					"anonymous-access":           true,
					"kernel-list-cache-ttl-secs": true,
					"max-retry-attempts":         true,
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
				GCSRetries: config.GCSRetries{
					MaxRetryAttempts: 15,
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
				GcsRetries: cfg.GcsRetriesConfig{
					MaxRetryAttempts: 100,
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

func TestValidClientProtocol(t *testing.T) {
	flags := &flagStorage{
		ClientProtocol: mountpkg.ClientProtocol("http1"),
	}

	v, err := PopulateNewConfigFromLegacyFlagsAndConfig(&mockCLIContext{}, flags, &config.MountConfig{})

	if assert.Nil(t, err) {
		assert.Equal(t, v.GcsConnection.ClientProtocol, cfg.Protocol("http1"))
	}
}

func TestInvalidClientProtocol(t *testing.T) {
	flags := &flagStorage{
		ClientProtocol: mountpkg.ClientProtocol("http3"),
	}

	_, err := PopulateNewConfigFromLegacyFlagsAndConfig(&mockCLIContext{}, flags, &config.MountConfig{})

	assert.NotNil(t, err)
}

func TestMetadataStatCacheResolution(t *testing.T) {
	testcases := []struct {
		flagValue     int
		configValue   int64
		expectedValue int64
	}{
		{
			flagValue:     10000,
			configValue:   100,
			expectedValue: 100,
		},
		{
			flagValue:     10000,
			configValue:   100,
			expectedValue: 100,
		},
		{
			flagValue:     20460,
			configValue:   math.MinInt64,
			expectedValue: 32,
		},
	}
	for idx, tt := range testcases {
		t.Run(fmt.Sprintf("metadata-stat-cache-size-mb resolution: %d", idx), func(t *testing.T) {
			newCfg, err := PopulateNewConfigFromLegacyFlagsAndConfig(&mockCLIContext{},
				&flagStorage{
					StatCacheCapacity: tt.flagValue,
					ClientProtocol:    "http1",
				},
				&config.MountConfig{
					MetadataCacheConfig: config.MetadataCacheConfig{
						StatCacheMaxSizeMB: tt.configValue,
					},
					LogConfig: config.LogConfig{Severity: "INFO"},
				})

			if assert.Nil(t, err) {
				assert.Equal(t, tt.expectedValue, newCfg.MetadataCache.StatCacheMaxSizeMb)
			}
		})
	}
}

func TestMetadataCacheTtlResolution(t *testing.T) {
	testcases := []struct {
		statCacheTTL  time.Duration
		typeCacheTTL  time.Duration
		configTTLSecs int64
		expectedValue int64
	}{
		{
			statCacheTTL:  60 * time.Second,
			typeCacheTTL:  60 * time.Second,
			configTTLSecs: config.TtlInSecsUnsetSentinel,
			expectedValue: 60,
		},
		{
			statCacheTTL:  60 * time.Second,
			typeCacheTTL:  50 * time.Second,
			configTTLSecs: config.TtlInSecsUnsetSentinel,
			expectedValue: 50,
		},
		{
			statCacheTTL:  60 * time.Second,
			typeCacheTTL:  60 * time.Second,
			configTTLSecs: -1,
			expectedValue: int64(time.Duration(math.MaxInt64).Seconds()),
		},
		{
			statCacheTTL:  5 * time.Minute,
			typeCacheTTL:  time.Hour,
			configTTLSecs: 10800,
			expectedValue: 10800,
		},
	}
	for idx, tt := range testcases {
		t.Run(fmt.Sprintf("metadata-stat-cache-ttl resolution: %d", idx), func(t *testing.T) {
			newCfg, err := PopulateNewConfigFromLegacyFlagsAndConfig(&mockCLIContext{},
				&flagStorage{
					StatCacheTTL:   tt.statCacheTTL,
					TypeCacheTTL:   tt.typeCacheTTL,
					ClientProtocol: "http1",
				},
				&config.MountConfig{
					MetadataCacheConfig: config.MetadataCacheConfig{
						TtlInSeconds: tt.configTTLSecs,
					},
					LogConfig: config.LogConfig{Severity: "INFO"},
				})

			if assert.Nil(t, err) {
				assert.Equal(t, tt.expectedValue, newCfg.MetadataCache.TtlSecs)
			}
		})
	}
}

func TestResolveMetadataCacheTTL(t *testing.T) {
	inputs := []struct {
		// Equivalent of user-setting of --stat-cache-ttl.
		statCacheTTL time.Duration

		// Equivalent of user-setting of --type-cache-ttl.
		typeCacheTTL time.Duration

		// Equivalent of user-setting of metadata-cache:ttl-secs in --config-file.
		ttlInSeconds             int64
		expectedMetadataCacheTTL time.Duration
	}{
		{
			// Most common scenario, when user doesn't set any of the TTL config parameters.
			statCacheTTL:             DefaultStatOrTypeCacheTTL,
			typeCacheTTL:             DefaultStatOrTypeCacheTTL,
			ttlInSeconds:             config.TtlInSecsUnsetSentinel,
			expectedMetadataCacheTTL: DefaultStatOrTypeCacheTTL,
		},
		{
			// Scenario where user sets only metadata-cache:ttl-secs and sets it to -1.
			statCacheTTL:             DefaultStatOrTypeCacheTTL,
			typeCacheTTL:             DefaultStatOrTypeCacheTTL,
			ttlInSeconds:             -1,
			expectedMetadataCacheTTL: time.Duration(math.MaxInt64),
		},
		{
			// Scenario where user sets only metadata-cache:ttl-secs and sets it to 0.
			statCacheTTL:             DefaultStatOrTypeCacheTTL,
			typeCacheTTL:             DefaultStatOrTypeCacheTTL,
			ttlInSeconds:             0,
			expectedMetadataCacheTTL: 0,
		},
		{
			// Scenario where user sets only metadata-cache:ttl-secs and sets it to a positive value.
			statCacheTTL:             DefaultStatOrTypeCacheTTL,
			typeCacheTTL:             DefaultStatOrTypeCacheTTL,
			ttlInSeconds:             30,
			expectedMetadataCacheTTL: 30 * time.Second,
		},
		{
			// Scenario where user sets only metadata-cache:ttl-secs and sets it to its highest supported value.
			statCacheTTL: DefaultStatOrTypeCacheTTL,
			typeCacheTTL: DefaultStatOrTypeCacheTTL,
			ttlInSeconds: config.MaxSupportedTtlInSeconds,

			expectedMetadataCacheTTL: time.Second * time.Duration(config.MaxSupportedTtlInSeconds),
		},
		{
			// Scenario where user sets both the old flags and the metadata-cache:ttl-secs. Here ttl-secs overrides both flags. case 1.
			statCacheTTL:             5 * time.Minute,
			typeCacheTTL:             time.Hour,
			ttlInSeconds:             10800,
			expectedMetadataCacheTTL: 10800 * time.Second,
		},
		{
			// Scenario where user sets both the old flags and the metadata-cache:ttl-secs. Here ttl-secs overrides both flags. case 2.
			statCacheTTL:             5 * time.Minute,
			typeCacheTTL:             time.Hour,
			ttlInSeconds:             1800,
			expectedMetadataCacheTTL: 1800 * time.Second,
		},
		{
			// Old-scenario where user sets only stat/type-cache-ttl flag(s), and not metadata-cache:ttl-secs. Case 1.
			statCacheTTL:             0,
			typeCacheTTL:             0,
			ttlInSeconds:             config.TtlInSecsUnsetSentinel,
			expectedMetadataCacheTTL: 0,
		},
		{
			// Old-scenario where user sets only stat/type-cache-ttl flag(s), and not metadata-cache:ttl-secs. Case 2. Stat-cache enabled, but not type-cache.
			statCacheTTL:             time.Hour,
			typeCacheTTL:             0,
			ttlInSeconds:             config.TtlInSecsUnsetSentinel,
			expectedMetadataCacheTTL: 0,
		},
		{
			// Old-scenario where user sets only stat/type-cache-ttl flag(s), and not metadata-cache:ttl-secs. Case 3. Type-cache enabled, but not stat-cache.
			statCacheTTL:             0,
			typeCacheTTL:             time.Hour,
			ttlInSeconds:             config.TtlInSecsUnsetSentinel,
			expectedMetadataCacheTTL: 0,
		},
		{
			// Old-scenario where user sets only stat/type-cache-ttl flag(s), and not metadata-cache:ttl-secs. Case 4. Both Type-cache and stat-cache enabled. The lower of the two TTLs is taken.
			statCacheTTL:             time.Second,
			typeCacheTTL:             time.Minute,
			ttlInSeconds:             config.TtlInSecsUnsetSentinel,
			expectedMetadataCacheTTL: time.Second,
		},
	}
	for _, input := range inputs {
		assert.Equal(t, input.expectedMetadataCacheTTL, resolveMetadataCacheTTL(input.statCacheTTL, input.typeCacheTTL, input.ttlInSeconds))
	}
}

func TestResolveStatCacheMaxSizeMB(t *testing.T) {
	testCases := []struct {
		// Equivalent of user-setting of flag --stat-cache-capacity.
		flagStatCacheCapacity int

		// Equivalent of user-setting of metadata-cache:stat-cache-max-size-mb in --config-file.
		mountConfigStatCacheMaxSizeMB int64

		// Expected output
		expectedStatCacheMaxSizeMB uint64
	}{
		{
			// Most common scenario, when user doesn't set either the flag or the config.
			flagStatCacheCapacity:         DefaultStatCacheCapacity,
			mountConfigStatCacheMaxSizeMB: config.StatCacheMaxSizeMBUnsetSentinel,
			expectedStatCacheMaxSizeMB:    DefaultStatCacheMaxSizeMB,
		},
		{
			// Scenario where user sets only metadata-cache:stat-cache-max-size-mb and sets it to -1.
			flagStatCacheCapacity:         DefaultStatCacheCapacity,
			mountConfigStatCacheMaxSizeMB: -1,
			expectedStatCacheMaxSizeMB:    config.MaxSupportedStatCacheMaxSizeMB,
		},
		{
			// Scenario where user sets only metadata-cache:stat-cache-max-size-mb and sets it to 0.
			flagStatCacheCapacity:         DefaultStatCacheCapacity,
			mountConfigStatCacheMaxSizeMB: 0,
			expectedStatCacheMaxSizeMB:    0,
		},
		{
			// Scenario where user sets only metadata-cache:stat-cache-max-size-mb and sets it to a positive value.
			flagStatCacheCapacity:         DefaultStatCacheCapacity,
			mountConfigStatCacheMaxSizeMB: 100,
			expectedStatCacheMaxSizeMB:    100,
		},
		{
			// Scenario where user sets only metadata-cache:stat-cache-max-size-mb and sets it to its highest user-input value.
			flagStatCacheCapacity:         DefaultStatCacheCapacity,
			mountConfigStatCacheMaxSizeMB: int64(config.MaxSupportedStatCacheMaxSizeMB),
			expectedStatCacheMaxSizeMB:    config.MaxSupportedStatCacheMaxSizeMB,
		},
		{
			// Scenario where user sets both stat-cache-capacity and the metadata-cache:stat-cache-max-size-mb. Here stat-cache-max-size-mb overrides stat-cache-capacity. case 1.
			flagStatCacheCapacity:         10000,
			mountConfigStatCacheMaxSizeMB: 100,
			expectedStatCacheMaxSizeMB:    100,
		},
		{
			// Scenario where user sets both stat-cache-capacity and the metadata-cache:stat-cache-max-size-mb. Here stat-cache-max-size-mb overrides stat-cache-capacity. case 2.
			flagStatCacheCapacity:         10000,
			mountConfigStatCacheMaxSizeMB: -1,
			expectedStatCacheMaxSizeMB:    config.MaxSupportedStatCacheMaxSizeMB,
		},
		{
			// Scenario where user sets both stat-cache-capacity and the metadata-cache:stat-cache-max-size-mb. Here stat-cache-max-size-mb overrides stat-cache-capacity. case 3.
			flagStatCacheCapacity:         10000,
			mountConfigStatCacheMaxSizeMB: 0,
			expectedStatCacheMaxSizeMB:    0,
		},
		{
			// Old-scenario where user sets only stat-cache-capacity flag(s), and not metadata-cache:stat-cache-max-size-mb. Case 1: stat-cache-capacity is 0.
			flagStatCacheCapacity:         0,
			mountConfigStatCacheMaxSizeMB: config.StatCacheMaxSizeMBUnsetSentinel,
			expectedStatCacheMaxSizeMB:    0,
		},
		{
			// Old-scenario where user sets only stat-cache-capacity flag(s), and not metadata-cache:stat-cache-max-size-mb. Case 2: stat-cache-capacity is non-zero.
			flagStatCacheCapacity:         10000,
			mountConfigStatCacheMaxSizeMB: config.StatCacheMaxSizeMBUnsetSentinel,
			expectedStatCacheMaxSizeMB:    16, // 16 MiB = MiB ceiling (10k entries * 1640 bytes (AssumedSizeOfPositiveStatCacheEntry + AssumedSizeOfNegativeStatCacheEntry))
		},
	}
	for _, tc := range testCases {
		statCacheMaxSizeMB, err := resolveStatCacheMaxSizeMB(tc.mountConfigStatCacheMaxSizeMB, tc.flagStatCacheCapacity)

		if assert.Nil(t, err) {
			assert.Equal(t, tc.expectedStatCacheMaxSizeMB, statCacheMaxSizeMB)
		}
	}
}
