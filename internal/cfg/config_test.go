package cfg

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var defaultConfig = Config{
	MountPoint: "/m",
	FileCache: FileCacheConfig{
		MaxSizeMB: -1,
	},
	MetadataCache: MetadataCacheConfig{
		StatCacheMaxSizeMB: 32,
		TtlInSeconds:       3600,
		TypeCacheMaxSizeMB: 4,
	},
	GCSConnection: GCSConnectionConfig{
		MaxBytesPerSec:       -1,
		MaxOpsPerSec:         -1,
		SequentialReadSizeMB: 200,
		Protocol:             "http1",
		MaxConnections:       100,
		MaxIdleConnections:   100,
		Retries: RetryConfig{
			Multiplier:    2,
			MaxRetrySleep: 30 * time.Second,
		},
		Auth: AuthConfig{ReuseTokenFromURL: true},
	},
	Logging: LogConfig{
		Severity: "INFO",
		Format:   "json",
		LogRotate: LogRotateConfig{
			MaxFileSizeMB:   512,
			BackupFileCount: 10,
			Compress:        true,
		},
	},
	Grpc: GrpcClientConfig{ConnPoolSize: 1},
	FileSystem: FileSystemConfig{
		GID:          -1,
		UID:          -1,
		FileMode:     0644,
		DirMode:      0755,
		MountOptions: []string{},
	},
}

func TestFlags(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	tests := []struct {
		name       string
		osArgs     []string
		updateFunc func(Config) Config
		err        error
	}{
		{
			name:   "Bucket and mount-point populated",
			osArgs: []string{"mybkt", "/m"},
			updateFunc: func(cfg Config) Config {
				cfg.Bucket = "mybkt"
				return cfg
			},
		},
		{
			name:   "Mountpoint populated using arg",
			osArgs: []string{"~/m"},
			updateFunc: func(config Config) Config {
				config.MountPoint = filepath.Join(homeDir, "m")
				return config
			},
		},
		{
			name: "Command fails when no arguments passed",
			err:  fmt.Errorf("error"),
		},
		{
			name:   "AppName",
			osArgs: []string{"/m", "--app-name=abc"},
			updateFunc: func(config Config) Config {
				config.AppName = "abc"
				return config
			},
		},
		{
			name:   "Foreground true explicitly",
			osArgs: []string{"/m", "--foreground=true"},
			updateFunc: func(config Config) Config {
				config.Foreground = true
				return config
			},
		},
		{
			name:   "Foreground true implicitly",
			osArgs: []string{"/m", "--foreground"},
			updateFunc: func(config Config) Config {
				config.Foreground = true
				return config
			},
		}, {
			name:   "Foreground true implicitly case-insensitive",
			osArgs: []string{"/m", "--foreground=True"},
			updateFunc: func(config Config) Config {
				config.Foreground = true
				return config
			},
		},
		{
			name:   "Enable HNS explicit true",
			osArgs: []string{"/m", "--enable-hns=True"},
			updateFunc: func(config Config) Config {
				config.EnableHNS = true
				return config
			},
		},
		{
			name:   "Enable HNS implicit true",
			osArgs: []string{"/m", "--enable-hns"},
			updateFunc: func(config Config) Config {
				config.EnableHNS = true
				return config
			},
		},
		{
			name:   "File-cache MaxSizeMB is set",
			osArgs: []string{"/m", "--file-cache.max-size-mb=3"},
			updateFunc: func(config Config) Config {
				config.FileCache.MaxSizeMB = 3
				return config
			},
		},
		{
			name:   "File-cache cache-file-for-range-read is set",
			osArgs: []string{"/m", "--file-cache.cache-file-for-range-read"},
			updateFunc: func(config Config) Config {
				config.FileCache.CacheFileForRangeRead = true
				return config
			},
		},
		{
			name:   "Metadata-cache stat-cache-max-size-mb is set",
			osArgs: []string{"/m", "--metadata-cache.stat-cache-max-size-mb=54"},
			updateFunc: func(config Config) Config {
				config.MetadataCache.StatCacheMaxSizeMB = 54
				return config
			},
		},
		{
			name:   "Metadata-cache ttl-secs is set",
			osArgs: []string{"/m", "--metadata-cache.ttl-secs=2"},
			updateFunc: func(config Config) Config {
				config.MetadataCache.TtlInSeconds = 2
				return config
			},
		},
		{
			name:   "Type-cache max-size is set",
			osArgs: []string{"/m", "--metadata-cache.type-cache-max-size-mb=54"},
			updateFunc: func(config Config) Config {
				config.MetadataCache.TypeCacheMaxSizeMB = 54
				return config
			},
		},
		{
			name:   "Enable non-existent-type-cache",
			osArgs: []string{"/m", "--metadata-cache.enable-nonexistent-type-cache"},
			updateFunc: func(config Config) Config {
				config.MetadataCache.EnableNonExistentTypeCache = true
				return config
			},
		},
		{
			name:   "Set billing project",
			osArgs: []string{"/m", "--gcs-connection.billing-project=bp"},
			updateFunc: func(config Config) Config {
				config.GCSConnection.BillingProject = "bp"
				return config
			},
		},
		{
			name:   "Set max-ops-per-sec",
			osArgs: []string{"/m", "--gcs-connection.max-bytes-per-sec=2.5"},
			updateFunc: func(config Config) Config {
				config.GCSConnection.MaxBytesPerSec = 2.5
				return config
			},
		},
		{
			name:   "Set max-ops-per-sec",
			osArgs: []string{"/m", "--gcs-connection.max-ops-per-sec=5.6"},
			updateFunc: func(config Config) Config {
				config.GCSConnection.MaxOpsPerSec = 5.6
				return config
			},
		},
		{
			name:   "Set sequential-read-size-mb",
			osArgs: []string{"/m", "--gcs-connection.sequential-read-size-mb=45"},
			updateFunc: func(config Config) Config {
				config.GCSConnection.SequentialReadSizeMB = 45
				return config
			},
		},
		{
			name:   "Set experimental-enable-json-read",
			osArgs: []string{"/m", "--gcs-connection.experimental-enable-json-read"},
			updateFunc: func(config Config) Config {
				config.GCSConnection.ExperimentalEnableJSONRead = true
				return config
			},
		},
		{
			name:   "Set custom-endpoint",
			osArgs: []string{"/m", "--gcs-connection.custom-endpoint=custom-endpoint"},
			updateFunc: func(config Config) Config {
				config.GCSConnection.CustomEndpoint = "custom-endpoint"
				return config
			},
		},
		{
			name:   "Set custom-endpoint",
			osArgs: []string{"/m", "--gcs-connection.protocol=http2"},
			updateFunc: func(config Config) Config {
				config.GCSConnection.Protocol = "http2"
				return config
			},
		},
		{
			name:   "Set client timeout",
			osArgs: []string{"/m", "--gcs-connection.timeout=15s"},
			updateFunc: func(config Config) Config {
				config.GCSConnection.Timeout = 15 * time.Second
				return config
			},
		},
		{
			name:   "Set retries multipler",
			osArgs: []string{"/m", "--gcs-connection.retries.multiplier=4"},
			updateFunc: func(config Config) Config {
				config.GCSConnection.Retries.Multiplier = 4
				return config
			},
		},
		{
			name:   "Set retry sleep",
			osArgs: []string{"/m", "--gcs-connection.retries.max-retry-sleep=2s"},
			updateFunc: func(config Config) Config {
				config.GCSConnection.Retries.MaxRetrySleep = 2 * time.Second
				return config
			},
		},
		{
			name:   "Set key-file",
			osArgs: []string{"/m", "--gcs-connection.auth.key-file=/home/key.json"},
			updateFunc: func(config Config) Config {
				config.GCSConnection.Auth.KeyFile = "/home/key.json"
				return config
			},
		},
		{
			name:   "Set token-url",
			osArgs: []string{"/m", "--gcs-connection.auth.token-url=http://some-token-url.com/"},
			updateFunc: func(config Config) Config {
				config.GCSConnection.Auth.TokenURL = "http://some-token-url.com/"
				return config
			},
		},
		{
			name:   "Disable reusing token from url",
			osArgs: []string{"/m", "--gcs-connection.auth.reuse-token-from-url=false"},
			updateFunc: func(config Config) Config {
				config.GCSConnection.Auth.ReuseTokenFromURL = false
				return config
			},
		},
		{
			name:   "Set logging file",
			osArgs: []string{"/m", "--logging.file-path=/home/abc.json"},
			updateFunc: func(config Config) Config {
				config.Logging.FilePath = "/home/abc.json"
				return config
			},
		},
		{
			name:   "Set logging format",
			osArgs: []string{"/m", "--logging.format=xml"},
			updateFunc: func(config Config) Config {
				config.Logging.Format = "xml"
				return config
			},
		},
		{
			name:   "Set logging severity",
			osArgs: []string{"/m", "--logging.severity=WARN"},
			updateFunc: func(config Config) Config {
				config.Logging.Severity = "WARN"
				return config
			},
		},
		{
			name:   "Set max log file size",
			osArgs: []string{"/m", "--logging.log-rotate.max-file-size-mb=2"},
			updateFunc: func(config Config) Config {
				config.Logging.LogRotate.MaxFileSizeMB = 2
				return config
			},
		},
		{
			name:   "Set number of backup log files",
			osArgs: []string{"/m", "--logging.log-rotate.backup-file-count=22"},
			updateFunc: func(config Config) Config {
				config.Logging.LogRotate.BackupFileCount = 22
				return config
			},
		},
		{
			name:   "Disable compression",
			osArgs: []string{"/m", "--logging.log-rotate.compress=false"},
			updateFunc: func(config Config) Config {
				config.Logging.LogRotate.Compress = false
				return config
			},
		},
		{
			name:   "Set gRPC connection-pool size",
			osArgs: []string{"/m", "--grpc.conn-pool-size=153"},
			updateFunc: func(config Config) Config {
				config.Grpc.ConnPoolSize = 153
				return config
			},
		},
		{
			name:   "Set temp-dir",
			osArgs: []string{"/m", "--file-system.temp-dir=/tmp/pqr"},
			updateFunc: func(config Config) Config {
				config.FileSystem.TempDir = "/tmp/pqr"
				return config
			},
		},
		{
			name:   "Set rename-dir-limit",
			osArgs: []string{"/m", "--file-system.rename-dir-limit=222"},
			updateFunc: func(config Config) Config {
				config.FileSystem.RenameDirLimit = 222
				return config
			},
		},
		{
			name:   "Set GID",
			osArgs: []string{"/m", "--file-system.gid=11"},
			updateFunc: func(config Config) Config {
				config.FileSystem.GID = 11
				return config
			},
		},
		{
			name:   "Set UID",
			osArgs: []string{"/m", "--file-system.uid=16"},
			updateFunc: func(config Config) Config {
				config.FileSystem.UID = 16
				return config
			},
		},
		{
			name:   "Set file-mode",
			osArgs: []string{"/m", "--file-system.file-mode=0721"},
			updateFunc: func(config Config) Config {
				config.FileSystem.FileMode = 0721
				return config
			},
		},
		{
			name:   "Set file-mode without 0 prefix",
			osArgs: []string{"/m", "--file-system.file-mode=721"},
			updateFunc: func(config Config) Config {
				config.FileSystem.FileMode = 0721
				return config
			},
		},
		{
			name:   "Set dir-mode",
			osArgs: []string{"/m", "--file-system.dir-mode=0751"},
			updateFunc: func(config Config) Config {
				config.FileSystem.DirMode = 0751
				return config
			},
		},
		{
			name:   "Set dir-mode without 0 prefix",
			osArgs: []string{"/m", "--file-system.dir-mode=751"},
			updateFunc: func(config Config) Config {
				config.FileSystem.DirMode = 0751
				return config
			},
		},
		{
			name:   "Set mount-options with a single option",
			osArgs: []string{"/m", "--file-system.mount-options=a"},
			updateFunc: func(config Config) Config {
				config.FileSystem.MountOptions = []string{"a"}
				return config
			},
		},
		{
			name:   "Set mount-options with multiple options",
			osArgs: []string{"/m", "--file-system.mount-options=a", "--file-system.mount-options=b=c,d=e=f"},
			updateFunc: func(config Config) Config {
				config.FileSystem.MountOptions = []string{"a", "b=c", "d=e=f"}
				return config
			},
		},
		{
			name:   "Set mount-options using shorthand",
			osArgs: []string{"/m", "-o=a", "-o=b=c,d=e=f"},
			updateFunc: func(config Config) Config {
				config.FileSystem.MountOptions = []string{"a", "b=c", "d=e=f"}
				return config
			},
		},
		{
			name:   "Set create-empty-file",
			osArgs: []string{"/m", "--write.create-empty-file"},
			updateFunc: func(config Config) Config {
				config.Write.CreateEmptyFile = true
				return config
			},
		},
		{
			name:   "Set enable-empty-managed-folders",
			osArgs: []string{"/m", "--list.enable-empty-managed-folders"},
			updateFunc: func(config Config) Config {
				config.List.EnableEmptyManagedFolders = true
				return config
			},
		},
		{
			name:   "Set log-mutex",
			osArgs: []string{"/m", "--debug.log-mutex"},
			updateFunc: func(config Config) Config {
				config.Debug.LogMutex = true
				return config
			},
		},
		{
			name:   "Set metrics-export-interval",
			osArgs: []string{"/m", "--monitoring.metrics-export-interval=15s"},
			updateFunc: func(config Config) Config {
				config.Monitoring.MetricsExportInterval = 15 * time.Second
				return config
			},
		},
		{
			name:   "Set experimental-open-telemetry-collector-address",
			osArgs: []string{"/m", "--monitoring.experimental-opentelemetry-collector-address=http://open.com/"},
			updateFunc: func(config Config) Config {
				config.Monitoring.ExperimentalOpenTelemetryCollectorAddress = "http://open.com/"
				return config
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			oldArgs := os.Args
			defer func() {
				os.Args = oldArgs
			}()

			os.Args = append([]string{"gcsfuse"}, tt.osArgs...)

			got, err := ParseConfig()

			// Assert
			if tt.err == nil {
				if assert.Nil(t, err) {
					assert.Equal(t, tt.updateFunc(defaultConfig), got)
				} else {
					assert.Error(t, err)
				}
			}

		})
	}
}

func TestConfigFile(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name     string
		cfgFile  string
		expected Config
		err      error
	}{
		{
			name:    "All config params",
			cfgFile: "testdata/all_config.yaml",
			expected: Config{
				AppName:    "A GCSFuse app",
				Foreground: true,
				OnlyDir:    "some-dir",
				Bucket:     "bkt",
				MountPoint: "/m",
				EnableHNS:  true,
				FileCache: FileCacheConfig{
					MaxSizeMB:             20,
					CacheFileForRangeRead: true,
				},
				MetadataCache: MetadataCacheConfig{
					StatCacheMaxSizeMB:         22,
					TtlInSeconds:               4,
					TypeCacheMaxSizeMB:         11,
					EnableNonExistentTypeCache: true,
				},
				GCSConnection: GCSConnectionConfig{
					BillingProject:             "bp",
					MaxOpsPerSec:               3.4,
					MaxBytesPerSec:             4.5,
					SequentialReadSizeMB:       50,
					ExperimentalEnableJSONRead: true,
					CustomEndpoint:             "ce",
					Protocol:                   "http2",
					Timeout:                    10 * time.Second,
					MaxConnections:             23,
					MaxIdleConnections:         2,
					Retries: RetryConfig{
						Multiplier:    5,
						MaxRetrySleep: 10 * time.Second,
					},
					Auth: AuthConfig{KeyFile: "/key.json",
						TokenURL:          "http://some-token-url/",
						ReuseTokenFromURL: false},
				},
				Logging: LogConfig{
					Severity: "ERROR",
					Format:   "xml",
					LogRotate: LogRotateConfig{
						MaxFileSizeMB:   100,
						BackupFileCount: 9,
						Compress:        false,
					}},
				Grpc: GrpcClientConfig{ConnPoolSize: 12},
				FileSystem: FileSystemConfig{
					TempDir:        "/tmp/abc",
					RenameDirLimit: 123,
					GID:            24,
					UID:            52,
					FileMode:       0614,
					DirMode:        0725,
					MountOptions:   []string{"a", "b=c", "d=e=f"},
				},
				Write: WriteConfig{CreateEmptyFile: true},
				List:  ListConfig{EnableEmptyManagedFolders: true},
				Debug: DebugConfig{
					ExitOnInvariantViolation: true,
					LogMutex:                 true,
				},
				Monitoring: MonitoringConfig{
					MetricsExportInterval:                     3600 * time.Second,
					ExperimentalOpenTelemetryCollectorAddress: "http://opentelemetry.com",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			oldArgs := os.Args
			defer func() {
				os.Args = oldArgs
			}()

			os.Args = append([]string{"gcsfuse"}, "bkt", "/m", "--config-file="+tt.cfgFile)

			got, err := ParseConfig()

			// Assert
			if tt.err == nil {
				if assert.Nil(t, err) {
					assert.Equal(t, tt.expected, got)
				} else {
					assert.Error(t, err)
				}
			}
		})
	}
}
