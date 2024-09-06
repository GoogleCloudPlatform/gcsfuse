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

// GENERATED CODE - DO NOT EDIT MANUALLY.

package cfg

import (
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	AppName string `yaml:"app-name,omitempty" json:"app-name,omitempty"`

	CacheDir ResolvedPath `yaml:"cache-dir,omitempty" json:"cache-dir,omitempty"`

	Debug DebugConfig `yaml:"debug,omitempty" json:"debug,omitempty"`

	EnableHns bool `yaml:"enable-hns,omitempty" json:"enable-hns,omitempty"`

	FileCache FileCacheConfig `yaml:"file-cache,omitempty" json:"file-cache,omitempty"`

	FileSystem FileSystemConfig `yaml:"file-system,omitempty" json:"file-system,omitempty"`

	Foreground bool `yaml:"foreground,omitempty" json:"foreground,omitempty"`

	GcsAuth GcsAuthConfig `yaml:"gcs-auth,omitempty" json:"gcs-auth,omitempty"`

	GcsConnection GcsConnectionConfig `yaml:"gcs-connection,omitempty" json:"gcs-connection,omitempty"`

	GcsRetries GcsRetriesConfig `yaml:"gcs-retries,omitempty" json:"gcs-retries,omitempty"`

	ImplicitDirs bool `yaml:"implicit-dirs,omitempty" json:"implicit-dirs,omitempty"`

	List ListConfig `yaml:"list,omitempty" json:"list,omitempty"`

	Logging LoggingConfig `yaml:"logging,omitempty" json:"logging,omitempty"`

	MetadataCache MetadataCacheConfig `yaml:"metadata-cache,omitempty" json:"metadata-cache,omitempty"`

	Metrics MetricsConfig `yaml:"metrics,omitempty" json:"metrics,omitempty"`

	Monitoring MonitoringConfig `yaml:"monitoring,omitempty" json:"monitoring,omitempty"`

	OnlyDir string `yaml:"only-dir,omitempty" json:"only-dir,omitempty"`

	Write WriteConfig `yaml:"write,omitempty" json:"write,omitempty"`
}

type DebugConfig struct {
	ExitOnInvariantViolation bool `yaml:"exit-on-invariant-violation,omitempty" json:"exit-on-invariant-violation,omitempty"`

	Fuse bool `yaml:"fuse,omitempty" json:"fuse,omitempty"`

	Gcs bool `yaml:"gcs,omitempty" json:"gcs,omitempty"`

	LogMutex bool `yaml:"log-mutex,omitempty" json:"log-mutex,omitempty"`
}

type FileCacheConfig struct {
	CacheFileForRangeRead bool `yaml:"cache-file-for-range-read,omitempty" json:"cache-file-for-range-read,omitempty"`

	DownloadChunkSizeMb int64 `yaml:"download-chunk-size-mb,omitempty" json:"download-chunk-size-mb,omitempty"`

	EnableCrc bool `yaml:"enable-crc,omitempty" json:"enable-crc,omitempty"`

	EnableODirect bool `yaml:"enable-o-direct,omitempty" json:"enable-o-direct,omitempty"`

	EnableParallelDownloads bool `yaml:"enable-parallel-downloads,omitempty" json:"enable-parallel-downloads,omitempty"`

	MaxParallelDownloads int64 `yaml:"max-parallel-downloads,omitempty" json:"max-parallel-downloads,omitempty"`

	MaxSizeMb int64 `yaml:"max-size-mb,omitempty" json:"max-size-mb,omitempty"`

	ParallelDownloadsPerFile int64 `yaml:"parallel-downloads-per-file,omitempty" json:"parallel-downloads-per-file,omitempty"`

	WriteBufferSize int64 `yaml:"write-buffer-size,omitempty" json:"write-buffer-size,omitempty"`
}

type FileSystemConfig struct {
	DirMode Octal `yaml:"dir-mode,omitempty" json:"dir-mode,omitempty"`

	DisableParallelDirops bool `yaml:"disable-parallel-dirops,omitempty" json:"disable-parallel-dirops,omitempty"`

	FileMode Octal `yaml:"file-mode,omitempty" json:"file-mode,omitempty"`

	FuseOptions []string `yaml:"fuse-options,omitempty" json:"fuse-options,omitempty"`

	Gid int64 `yaml:"gid,omitempty" json:"gid,omitempty"`

	IgnoreInterrupts bool `yaml:"ignore-interrupts,omitempty" json:"ignore-interrupts,omitempty"`

	KernelListCacheTtlSecs int64 `yaml:"kernel-list-cache-ttl-secs,omitempty" json:"kernel-list-cache-ttl-secs,omitempty"`

	RenameDirLimit int64 `yaml:"rename-dir-limit,omitempty" json:"rename-dir-limit,omitempty"`

	TempDir ResolvedPath `yaml:"temp-dir,omitempty" json:"temp-dir,omitempty"`

	Uid int64 `yaml:"uid,omitempty" json:"uid,omitempty"`
}

type GcsAuthConfig struct {
	AnonymousAccess bool `yaml:"anonymous-access,omitempty" json:"anonymous-access,omitempty"`

	KeyFile ResolvedPath `yaml:"key-file,omitempty" json:"key-file,omitempty"`

	ReuseTokenFromUrl bool `yaml:"reuse-token-from-url,omitempty" json:"reuse-token-from-url,omitempty"`

	TokenUrl string `yaml:"token-url,omitempty" json:"token-url,omitempty"`
}

type GcsConnectionConfig struct {
	BillingProject string `yaml:"billing-project,omitempty" json:"billing-project,omitempty"`

	ClientProtocol Protocol `yaml:"client-protocol,omitempty" json:"client-protocol,omitempty"`

	CustomEndpoint string `yaml:"custom-endpoint,omitempty" json:"custom-endpoint,omitempty"`

	ExperimentalEnableJsonRead bool `yaml:"experimental-enable-json-read,omitempty" json:"experimental-enable-json-read,omitempty"`

	GrpcConnPoolSize int64 `yaml:"grpc-conn-pool-size,omitempty" json:"grpc-conn-pool-size,omitempty"`

	HttpClientTimeout time.Duration `yaml:"http-client-timeout,omitempty" json:"http-client-timeout,omitempty"`

	LimitBytesPerSec float64 `yaml:"limit-bytes-per-sec,omitempty" json:"limit-bytes-per-sec,omitempty"`

	LimitOpsPerSec float64 `yaml:"limit-ops-per-sec,omitempty" json:"limit-ops-per-sec,omitempty"`

	MaxConnsPerHost int64 `yaml:"max-conns-per-host,omitempty" json:"max-conns-per-host,omitempty"`

	MaxIdleConnsPerHost int64 `yaml:"max-idle-conns-per-host,omitempty" json:"max-idle-conns-per-host,omitempty"`

	SequentialReadSizeMb int64 `yaml:"sequential-read-size-mb,omitempty" json:"sequential-read-size-mb,omitempty"`
}

type GcsRetriesConfig struct {
	MaxRetryAttempts int64 `yaml:"max-retry-attempts,omitempty" json:"max-retry-attempts,omitempty"`

	MaxRetrySleep time.Duration `yaml:"max-retry-sleep,omitempty" json:"max-retry-sleep,omitempty"`

	Multiplier float64 `yaml:"multiplier,omitempty" json:"multiplier,omitempty"`
}

type ListConfig struct {
	EnableEmptyManagedFolders bool `yaml:"enable-empty-managed-folders,omitempty" json:"enable-empty-managed-folders,omitempty"`
}

type LogRotateLoggingConfig struct {
	BackupFileCount int64 `yaml:"backup-file-count,omitempty" json:"backup-file-count,omitempty"`

	Compress bool `yaml:"compress,omitempty" json:"compress,omitempty"`

	MaxFileSizeMb int64 `yaml:"max-file-size-mb,omitempty" json:"max-file-size-mb,omitempty"`
}

type LoggingConfig struct {
	FilePath ResolvedPath `yaml:"file-path,omitempty" json:"file-path,omitempty"`

	Format string `yaml:"format,omitempty" json:"format,omitempty"`

	LogRotate LogRotateLoggingConfig `yaml:"log-rotate,omitempty" json:"log-rotate,omitempty"`

	Severity LogSeverity `yaml:"severity,omitempty" json:"severity,omitempty"`
}

type MetadataCacheConfig struct {
	DeprecatedStatCacheCapacity int64 `yaml:"deprecated-stat-cache-capacity,omitempty" json:"deprecated-stat-cache-capacity,omitempty"`

	DeprecatedStatCacheTtl time.Duration `yaml:"deprecated-stat-cache-ttl,omitempty" json:"deprecated-stat-cache-ttl,omitempty"`

	DeprecatedTypeCacheTtl time.Duration `yaml:"deprecated-type-cache-ttl,omitempty" json:"deprecated-type-cache-ttl,omitempty"`

	EnableNonexistentTypeCache bool `yaml:"enable-nonexistent-type-cache,omitempty" json:"enable-nonexistent-type-cache,omitempty"`

	ExperimentalMetadataPrefetchOnMount string `yaml:"experimental-metadata-prefetch-on-mount,omitempty" json:"experimental-metadata-prefetch-on-mount,omitempty"`

	StatCacheMaxSizeMb int64 `yaml:"stat-cache-max-size-mb,omitempty" json:"stat-cache-max-size-mb,omitempty"`

	TtlSecs int64 `yaml:"ttl-secs,omitempty" json:"ttl-secs,omitempty"`

	TypeCacheMaxSizeMb int64 `yaml:"type-cache-max-size-mb,omitempty" json:"type-cache-max-size-mb,omitempty"`
}

type MetricsConfig struct {
	PrometheusPort int64 `yaml:"prometheus-port,omitempty" json:"prometheus-port,omitempty"`

	StackdriverExportInterval time.Duration `yaml:"stackdriver-export-interval,omitempty" json:"stackdriver-export-interval,omitempty"`
}

type MonitoringConfig struct {
	ExperimentalOpentelemetryCollectorAddress string `yaml:"experimental-opentelemetry-collector-address,omitempty" json:"experimental-opentelemetry-collector-address,omitempty"`

	ExperimentalTracingMode string `yaml:"experimental-tracing-mode,omitempty" json:"experimental-tracing-mode,omitempty"`

	ExperimentalTracingSamplingRatio float64 `yaml:"experimental-tracing-sampling-ratio,omitempty" json:"experimental-tracing-sampling-ratio,omitempty"`
}

type WriteConfig struct {
	CreateEmptyFile bool `yaml:"create-empty-file,omitempty" json:"create-empty-file,omitempty"`
}

func BindFlags(v *viper.Viper, flagSet *pflag.FlagSet) error {

	flagSet.BoolP("anonymous-access", "", false, "Authentication is enabled by default. This flag disables authentication")

	if err := v.BindPFlag("gcs-auth.anonymous-access", flagSet.Lookup("anonymous-access")); err != nil {
		return err
	}

	flagSet.StringP("app-name", "", "", "The application name of this mount.")

	if err := v.BindPFlag("app-name", flagSet.Lookup("app-name")); err != nil {
		return err
	}

	flagSet.StringP("billing-project", "", "", "Project to use for billing when accessing a bucket enabled with \"Requester Pays\". (The default is none)")

	if err := v.BindPFlag("gcs-connection.billing-project", flagSet.Lookup("billing-project")); err != nil {
		return err
	}

	flagSet.StringP("cache-dir", "", "", "Enables file-caching. Specifies the directory to use for file-cache.")

	if err := v.BindPFlag("cache-dir", flagSet.Lookup("cache-dir")); err != nil {
		return err
	}

	flagSet.BoolP("cache-file-for-range-read", "", false, "Whether to cache file for range reads.")

	if err := v.BindPFlag("file-cache.cache-file-for-range-read", flagSet.Lookup("cache-file-for-range-read")); err != nil {
		return err
	}

	flagSet.StringP("client-protocol", "", "http1", "The protocol used for communicating with the GCS backend. Value can be 'http1' (HTTP/1.1), 'http2' (HTTP/2) or 'grpc'.")

	if err := v.BindPFlag("gcs-connection.client-protocol", flagSet.Lookup("client-protocol")); err != nil {
		return err
	}

	flagSet.BoolP("create-empty-file", "", false, "For a new file, it creates an empty file in Cloud Storage bucket as a hold.")

	if err := flagSet.MarkDeprecated("create-empty-file", "This flag will be deleted soon."); err != nil {
		return err
	}

	if err := v.BindPFlag("write.create-empty-file", flagSet.Lookup("create-empty-file")); err != nil {
		return err
	}

	flagSet.StringP("custom-endpoint", "", "", "Specifies an alternative custom endpoint for fetching data. Should only be used for testing.  The custom endpoint must support the equivalent resources and operations as the GCS  JSON endpoint, https://storage.googleapis.com/storage/v1. If a custom endpoint is not specified,  GCSFuse uses the global GCS JSON API endpoint, https://storage.googleapis.com/storage/v1.")

	if err := v.BindPFlag("gcs-connection.custom-endpoint", flagSet.Lookup("custom-endpoint")); err != nil {
		return err
	}

	flagSet.BoolP("debug_fs", "", false, "This flag is unused.")

	if err := flagSet.MarkDeprecated("debug_fs", "Debug fuse logs are now controlled by log-severity flag, please use log-severity trace to view the logs."); err != nil {
		return err
	}

	flagSet.BoolP("debug_fuse", "", false, "Set logging threshold to trace in order to view the debug logs")

	if err := flagSet.MarkDeprecated("debug_fuse", "This flag has been deprecated. Please use log-severity instead."); err != nil {
		return err
	}

	if err := v.BindPFlag("debug.fuse", flagSet.Lookup("debug_fuse")); err != nil {
		return err
	}

	flagSet.BoolP("debug_fuse_errors", "", true, "This flag is currently unused.")

	if err := flagSet.MarkDeprecated("debug_fuse_errors", "This flag is currently unused."); err != nil {
		return err
	}

	flagSet.BoolP("debug_gcs", "", false, "Set logging threshold to trace in order to view the debug logs")

	if err := flagSet.MarkDeprecated("debug_gcs", "This flag has been deprecated. Please use log-severity instead."); err != nil {
		return err
	}

	if err := v.BindPFlag("debug.gcs", flagSet.Lookup("debug_gcs")); err != nil {
		return err
	}

	flagSet.BoolP("debug_http", "", false, "This flag is currently unused.")

	if err := flagSet.MarkDeprecated("debug_http", "This flag is currently unused."); err != nil {
		return err
	}

	flagSet.BoolP("debug_invariants", "", false, "Exit when internal invariants are violated.")

	if err := v.BindPFlag("debug.exit-on-invariant-violation", flagSet.Lookup("debug_invariants")); err != nil {
		return err
	}

	flagSet.BoolP("debug_mutex", "", false, "Print debug messages when a mutex is held too long.")

	if err := v.BindPFlag("debug.log-mutex", flagSet.Lookup("debug_mutex")); err != nil {
		return err
	}

	flagSet.StringP("dir-mode", "", "0755", "Permissions bits for directories, in octal.")

	if err := v.BindPFlag("file-system.dir-mode", flagSet.Lookup("dir-mode")); err != nil {
		return err
	}

	flagSet.BoolP("disable-parallel-dirops", "", false, "Specifies whether to allow parallel dir operations (lookups and readers)")

	if err := flagSet.MarkHidden("disable-parallel-dirops"); err != nil {
		return err
	}

	if err := v.BindPFlag("file-system.disable-parallel-dirops", flagSet.Lookup("disable-parallel-dirops")); err != nil {
		return err
	}

	flagSet.IntP("download-chunk-size-mb", "", 50, "Size of chunks in MiB that each concurrent request downloads.")

	if err := v.BindPFlag("file-cache.download-chunk-size-mb", flagSet.Lookup("download-chunk-size-mb")); err != nil {
		return err
	}

	flagSet.BoolP("enable-crc", "", false, "Performs CRC to ensure that file is correctly downloaded into cache.")

	if err := v.BindPFlag("file-cache.enable-crc", flagSet.Lookup("enable-crc")); err != nil {
		return err
	}

	flagSet.BoolP("enable-empty-managed-folders", "", false, "This handles the corner case in listing managed folders. There are two corner cases (a) empty managed folder (b) nested managed folder which doesn't contain any descendent as object. This flag always works in conjunction with --implicit-dirs flag. (a) If only ImplicitDirectories is true, all managed folders are listed other than above two mentioned cases. (b) If both ImplicitDirectories and EnableEmptyManagedFolders are true, then all the managed folders are listed including the above-mentioned corner case. (c) If ImplicitDirectories is false then no managed folders are listed irrespective of enable-empty-managed-folders flag.")

	if err := v.BindPFlag("list.enable-empty-managed-folders", flagSet.Lookup("enable-empty-managed-folders")); err != nil {
		return err
	}

	flagSet.BoolP("enable-hns", "", true, "Enables support for HNS buckets")

	if err := v.BindPFlag("enable-hns", flagSet.Lookup("enable-hns")); err != nil {
		return err
	}

	flagSet.BoolP("enable-nonexistent-type-cache", "", false, "Once set, if an inode is not found in GCS, a type cache entry with type NonexistentType will be created. This also means new file/dir created might not be seen. For example, if this flag is set, and metadata-cache-ttl-secs is set, then if we create the same file/node in the meantime using the same mount, since we are not refreshing the cache, it will still return nil.")

	if err := v.BindPFlag("metadata-cache.enable-nonexistent-type-cache", flagSet.Lookup("enable-nonexistent-type-cache")); err != nil {
		return err
	}

	flagSet.BoolP("enable-o-direct", "", false, "Whether to use O_DIRECT while writing to file-cache in case of parallel downloads.")

	if err := flagSet.MarkHidden("enable-o-direct"); err != nil {
		return err
	}

	if err := v.BindPFlag("file-cache.enable-o-direct", flagSet.Lookup("enable-o-direct")); err != nil {
		return err
	}

	flagSet.BoolP("enable-parallel-downloads", "", false, "Enable parallel downloads.")

	if err := v.BindPFlag("file-cache.enable-parallel-downloads", flagSet.Lookup("enable-parallel-downloads")); err != nil {
		return err
	}

	flagSet.BoolP("experimental-enable-json-read", "", false, "By default, GCSFuse uses the GCS XML API to get and read objects. When this flag is specified, GCSFuse uses the GCS JSON API instead.\"")

	if err := flagSet.MarkDeprecated("experimental-enable-json-read", "Experimental flag: could be dropped even in a minor release."); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-connection.experimental-enable-json-read", flagSet.Lookup("experimental-enable-json-read")); err != nil {
		return err
	}

	flagSet.IntP("experimental-grpc-conn-pool-size", "", 1, "The number of gRPC channel in grpc client.")

	if err := flagSet.MarkDeprecated("experimental-grpc-conn-pool-size", "Experimental flag: can be removed in a minor release."); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-connection.grpc-conn-pool-size", flagSet.Lookup("experimental-grpc-conn-pool-size")); err != nil {
		return err
	}

	flagSet.StringP("experimental-metadata-prefetch-on-mount", "", "disabled", "Experimental: This indicates whether or not to prefetch the metadata (prefilling of metadata caches and creation of inodes) of the mounted bucket at the time of mounting the bucket. Supported values: \"disabled\", \"sync\" and \"async\". Any other values will return error on mounting. This is applicable only to static mounting, and not to dynamic mounting.")

	if err := flagSet.MarkDeprecated("experimental-metadata-prefetch-on-mount", "Experimental flag: could be removed even in a minor release."); err != nil {
		return err
	}

	if err := v.BindPFlag("metadata-cache.experimental-metadata-prefetch-on-mount", flagSet.Lookup("experimental-metadata-prefetch-on-mount")); err != nil {
		return err
	}

	flagSet.StringP("experimental-opentelemetry-collector-address", "", "", "Experimental: Export metrics to the OpenTelemetry collector at this address.")

	if err := flagSet.MarkDeprecated("experimental-opentelemetry-collector-address", "Experimental flag: could be dropped even in a minor release."); err != nil {
		return err
	}

	if err := v.BindPFlag("monitoring.experimental-opentelemetry-collector-address", flagSet.Lookup("experimental-opentelemetry-collector-address")); err != nil {
		return err
	}

	flagSet.StringP("experimental-tracing-mode", "", "", "Experimental: specify tracing mode")

	if err := flagSet.MarkHidden("experimental-tracing-mode"); err != nil {
		return err
	}

	if err := v.BindPFlag("monitoring.experimental-tracing-mode", flagSet.Lookup("experimental-tracing-mode")); err != nil {
		return err
	}

	flagSet.Float64P("experimental-tracing-sampling-ratio", "", 0, "Experimental: Trace sampling ratio")

	if err := flagSet.MarkHidden("experimental-tracing-sampling-ratio"); err != nil {
		return err
	}

	if err := v.BindPFlag("monitoring.experimental-tracing-sampling-ratio", flagSet.Lookup("experimental-tracing-sampling-ratio")); err != nil {
		return err
	}

	flagSet.IntP("file-cache-max-size-mb", "", -1, "Maximum size of the file-cache in MiBs")

	if err := v.BindPFlag("file-cache.max-size-mb", flagSet.Lookup("file-cache-max-size-mb")); err != nil {
		return err
	}

	flagSet.StringP("file-mode", "", "0644", "Permissions bits for files, in octal.")

	if err := v.BindPFlag("file-system.file-mode", flagSet.Lookup("file-mode")); err != nil {
		return err
	}

	flagSet.BoolP("foreground", "", false, "Stay in the foreground after mounting.")

	if err := v.BindPFlag("foreground", flagSet.Lookup("foreground")); err != nil {
		return err
	}

	flagSet.IntP("gid", "", -1, "GID owner of all inodes.")

	if err := v.BindPFlag("file-system.gid", flagSet.Lookup("gid")); err != nil {
		return err
	}

	flagSet.DurationP("http-client-timeout", "", 0*time.Nanosecond, "The time duration that http client will wait to get response from the server. The default value 0 indicates no timeout.")

	if err := v.BindPFlag("gcs-connection.http-client-timeout", flagSet.Lookup("http-client-timeout")); err != nil {
		return err
	}

	flagSet.BoolP("ignore-interrupts", "", true, "Instructs gcsfuse to ignore system interrupt signals (like SIGINT, triggered by Ctrl+C). This prevents those signals from immediately terminating gcsfuse inflight operations. (default: true)")

	if err := v.BindPFlag("file-system.ignore-interrupts", flagSet.Lookup("ignore-interrupts")); err != nil {
		return err
	}

	flagSet.BoolP("implicit-dirs", "", false, "Implicitly define directories based on content. See files and directories in docs/semantics for more information")

	if err := v.BindPFlag("implicit-dirs", flagSet.Lookup("implicit-dirs")); err != nil {
		return err
	}

	flagSet.IntP("kernel-list-cache-ttl-secs", "", 0, "How long the directory listing (output of ls <dir>) should be cached in the kernel page cache. If a particular directory cache entry is kept by kernel for longer than TTL, then it will be sent for invalidation by gcsfuse on next opendir (comes in the start, as part of next listing) call. 0 means no caching. Use -1 to cache for lifetime (no ttl). Negative value other than -1 will throw error.")

	if err := v.BindPFlag("file-system.kernel-list-cache-ttl-secs", flagSet.Lookup("kernel-list-cache-ttl-secs")); err != nil {
		return err
	}

	flagSet.StringP("key-file", "", "", "Absolute path to JSON key file for use with GCS. (The default is none, Google application default credentials used)")

	if err := v.BindPFlag("gcs-auth.key-file", flagSet.Lookup("key-file")); err != nil {
		return err
	}

	flagSet.Float64P("limit-bytes-per-sec", "", -1, "Bandwidth limit for reading data, measured over a 30-second window. (use -1 for no limit)")

	if err := v.BindPFlag("gcs-connection.limit-bytes-per-sec", flagSet.Lookup("limit-bytes-per-sec")); err != nil {
		return err
	}

	flagSet.Float64P("limit-ops-per-sec", "", -1, "Operations per second limit, measured over a 30-second window (use -1 for no limit)")

	if err := v.BindPFlag("gcs-connection.limit-ops-per-sec", flagSet.Lookup("limit-ops-per-sec")); err != nil {
		return err
	}

	flagSet.StringP("log-file", "", "", "The file for storing logs that can be parsed by fluentd. When not provided, plain text logs are printed to stdout when Cloud Storage FUSE is run  in the foreground, or to syslog when Cloud Storage FUSE is run in the  background.")

	if err := v.BindPFlag("logging.file-path", flagSet.Lookup("log-file")); err != nil {
		return err
	}

	flagSet.StringP("log-format", "", "json", "The format of the log file: 'text' or 'json'.")

	if err := v.BindPFlag("logging.format", flagSet.Lookup("log-format")); err != nil {
		return err
	}

	flagSet.IntP("log-rotate-backup-file-count", "", 10, "The maximum number of backup log files to retain after they have been rotated. The default value is 10. When value is set to 0, all backup files are retained.")

	if err := v.BindPFlag("logging.log-rotate.backup-file-count", flagSet.Lookup("log-rotate-backup-file-count")); err != nil {
		return err
	}

	flagSet.BoolP("log-rotate-compress", "", true, "Controls whether the rotated log files should be compressed using gzip.")

	if err := v.BindPFlag("logging.log-rotate.compress", flagSet.Lookup("log-rotate-compress")); err != nil {
		return err
	}

	flagSet.IntP("log-rotate-max-log-file-size-mb", "", 512, "The maximum size in megabytes that a log file can reach before it is rotated.")

	if err := v.BindPFlag("logging.log-rotate.max-file-size-mb", flagSet.Lookup("log-rotate-max-log-file-size-mb")); err != nil {
		return err
	}

	flagSet.StringP("log-severity", "", "info", "Specifies the logging severity expressed as one of [trace, debug, info, warning, error, off]")

	if err := v.BindPFlag("logging.severity", flagSet.Lookup("log-severity")); err != nil {
		return err
	}

	flagSet.IntP("max-conns-per-host", "", 0, "The max number of TCP connections allowed per server. This is effective when client-protocol is set to 'http1'. The default value 0 indicates no limit on TCP connections (limited by the machine specifications).")

	if err := v.BindPFlag("gcs-connection.max-conns-per-host", flagSet.Lookup("max-conns-per-host")); err != nil {
		return err
	}

	flagSet.IntP("max-idle-conns-per-host", "", 100, "The number of maximum idle connections allowed per server.")

	if err := v.BindPFlag("gcs-connection.max-idle-conns-per-host", flagSet.Lookup("max-idle-conns-per-host")); err != nil {
		return err
	}

	flagSet.IntP("max-parallel-downloads", "", DefaultMaxParallelDownloads(), "Sets an uber limit of number of concurrent file download requests that are made across all files.")

	if err := v.BindPFlag("file-cache.max-parallel-downloads", flagSet.Lookup("max-parallel-downloads")); err != nil {
		return err
	}

	flagSet.IntP("max-retry-attempts", "", 0, "It sets a limit on the number of times an operation will be retried if it fails, preventing endless retry loops. The default value 0 indicates no limit.")

	if err := v.BindPFlag("gcs-retries.max-retry-attempts", flagSet.Lookup("max-retry-attempts")); err != nil {
		return err
	}

	flagSet.DurationP("max-retry-duration", "", 0*time.Nanosecond, "This is currently unused.")

	if err := flagSet.MarkDeprecated("max-retry-duration", "This is currently unused."); err != nil {
		return err
	}

	flagSet.DurationP("max-retry-sleep", "", 30000000000*time.Nanosecond, "The maximum duration allowed to sleep in a retry loop with exponential backoff for failed requests to GCS backend. Once the backoff duration exceeds this limit, the retry continues with this specified maximum value.")

	if err := v.BindPFlag("gcs-retries.max-retry-sleep", flagSet.Lookup("max-retry-sleep")); err != nil {
		return err
	}

	flagSet.IntP("metadata-cache-ttl-secs", "", 60, "The ttl value in seconds to be used for expiring items in metadata-cache. It can be set to -1 for no-ttl, 0 for no cache and > 0 for ttl-controlled metadata-cache. Any value set below -1 will throw an error.\"")

	if err := v.BindPFlag("metadata-cache.ttl-secs", flagSet.Lookup("metadata-cache-ttl-secs")); err != nil {
		return err
	}

	flagSet.StringSliceP("o", "", []string{}, "Additional system-specific mount options. Multiple options can be passed as comma separated. For readonly, use --o ro")

	if err := v.BindPFlag("file-system.fuse-options", flagSet.Lookup("o")); err != nil {
		return err
	}

	flagSet.StringP("only-dir", "", "", "Mount only a specific directory within the bucket. See docs/mounting for more information")

	if err := v.BindPFlag("only-dir", flagSet.Lookup("only-dir")); err != nil {
		return err
	}

	flagSet.IntP("parallel-downloads-per-file", "", 16, "Number of concurrent download requests per file.")

	if err := v.BindPFlag("file-cache.parallel-downloads-per-file", flagSet.Lookup("parallel-downloads-per-file")); err != nil {
		return err
	}

	flagSet.IntP("prometheus-port", "", 0, "Expose Prometheus metrics endpoint on this port and a path of /metrics.")

	if err := v.BindPFlag("metrics.prometheus-port", flagSet.Lookup("prometheus-port")); err != nil {
		return err
	}

	flagSet.IntP("rename-dir-limit", "", 0, "Allow rename a directory containing fewer descendants than this limit.")

	if err := v.BindPFlag("file-system.rename-dir-limit", flagSet.Lookup("rename-dir-limit")); err != nil {
		return err
	}

	flagSet.Float64P("retry-multiplier", "", 2, "Param for exponential backoff algorithm, which is used to increase waiting time b/w two consecutive retries.")

	if err := v.BindPFlag("gcs-retries.multiplier", flagSet.Lookup("retry-multiplier")); err != nil {
		return err
	}

	flagSet.BoolP("reuse-token-from-url", "", true, "If false, the token acquired from token-url is not reused.")

	if err := v.BindPFlag("gcs-auth.reuse-token-from-url", flagSet.Lookup("reuse-token-from-url")); err != nil {
		return err
	}

	flagSet.IntP("sequential-read-size-mb", "", 200, "File chunk size to read from GCS in one call. Need to specify the value in MB. ChunkSize less than 1MB is not supported")

	if err := v.BindPFlag("gcs-connection.sequential-read-size-mb", flagSet.Lookup("sequential-read-size-mb")); err != nil {
		return err
	}

	flagSet.DurationP("stackdriver-export-interval", "", 0*time.Nanosecond, "Export metrics to stackdriver with this interval. The default value 0 indicates no exporting.")

	if err := v.BindPFlag("metrics.stackdriver-export-interval", flagSet.Lookup("stackdriver-export-interval")); err != nil {
		return err
	}

	flagSet.IntP("stat-cache-capacity", "", 20460, "How many entries can the stat-cache hold (impacts memory consumption). This flag has been deprecated (starting v2.0) and in favor of stat-cache-max-size-mb. For now, the value of stat-cache-capacity will be translated to the next higher corresponding value of stat-cache-max-size-mb (assuming stat-cache entry-size ~= 1640 bytes, including 1400 for positive entry and 240 for corresponding negative entry), if stat-cache-max-size-mb is not set.\"")

	if err := flagSet.MarkDeprecated("stat-cache-capacity", "This flag has been deprecated (starting v2.0) in favor of stat-cache-max-size-mb."); err != nil {
		return err
	}

	if err := v.BindPFlag("metadata-cache.deprecated-stat-cache-capacity", flagSet.Lookup("stat-cache-capacity")); err != nil {
		return err
	}

	flagSet.IntP("stat-cache-max-size-mb", "", 32, "The maximum size of stat-cache in MiBs. It can also be set to -1 for no-size-limit, 0 for no cache. Values below -1 are not supported.")

	if err := v.BindPFlag("metadata-cache.stat-cache-max-size-mb", flagSet.Lookup("stat-cache-max-size-mb")); err != nil {
		return err
	}

	flagSet.DurationP("stat-cache-ttl", "", 60000000000*time.Nanosecond, "How long to cache StatObject results and inode attributes. This flag has been deprecated (starting v2.0) in favor of metadata-cache-ttl-secs. For now, the minimum of stat-cache-ttl and type-cache-ttl values, rounded up to the next higher multiple of a second is used as ttl for both stat-cache and type-cache, when metadata-cache-ttl-secs is not set.")

	if err := flagSet.MarkDeprecated("stat-cache-ttl", "This flag has been deprecated (starting v2.0) in favor of metadata-cache-ttl-secs."); err != nil {
		return err
	}

	if err := v.BindPFlag("metadata-cache.deprecated-stat-cache-ttl", flagSet.Lookup("stat-cache-ttl")); err != nil {
		return err
	}

	flagSet.StringP("temp-dir", "", "", "Path to the temporary directory where writes are staged prior to upload to Cloud Storage. (default: system default, likely /tmp)\"")

	if err := v.BindPFlag("file-system.temp-dir", flagSet.Lookup("temp-dir")); err != nil {
		return err
	}

	flagSet.StringP("token-url", "", "", "A url for getting an access token when the key-file is absent.")

	if err := v.BindPFlag("gcs-auth.token-url", flagSet.Lookup("token-url")); err != nil {
		return err
	}

	flagSet.IntP("type-cache-max-size-mb", "", 4, "Max size of type-cache maps which are maintained at a per-directory level.")

	if err := v.BindPFlag("metadata-cache.type-cache-max-size-mb", flagSet.Lookup("type-cache-max-size-mb")); err != nil {
		return err
	}

	flagSet.DurationP("type-cache-ttl", "", 60000000000*time.Nanosecond, "Usage: How long to cache StatObject results and inode attributes. This flag has been deprecated (starting v2.0) in favor of metadata-cache-ttl-secs. For now, the minimum of stat-cache-ttl and type-cache-ttl values, rounded up to the next higher multiple of a second is used as ttl for both stat-cache and type-cache, when metadata-cache-ttl-secs is not set.")

	if err := flagSet.MarkDeprecated("type-cache-ttl", "This flag has been deprecated (starting v2.0) in favor of metadata-cache-ttl-secs."); err != nil {
		return err
	}

	if err := v.BindPFlag("metadata-cache.deprecated-type-cache-ttl", flagSet.Lookup("type-cache-ttl")); err != nil {
		return err
	}

	flagSet.IntP("uid", "", -1, "UID owner of all inodes.")

	if err := v.BindPFlag("file-system.uid", flagSet.Lookup("uid")); err != nil {
		return err
	}

	flagSet.IntP("write-buffer-size", "", 4194304, "Size of in-memory buffer that is used per goroutine in parallel downloads while writing to file-cache.")

	if err := flagSet.MarkHidden("write-buffer-size"); err != nil {
		return err
	}

	if err := v.BindPFlag("file-cache.write-buffer-size", flagSet.Lookup("write-buffer-size")); err != nil {
		return err
	}

	return nil
}
