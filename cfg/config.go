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

	"github.com/googlecloudplatform/gcsfuse/v3/cfg/shared"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// AllFlagOptimizationRules is the generated map from a flag's config-path to its specific rules.
var AllFlagOptimizationRules = map[string]shared.OptimizationRules{"file-cache.cache-file-for-range-read": {
	Profiles: []shared.ProfileOptimization{
		{
			Name:  "aiml-serving",
			Value: bool(true),
		},
		{
			Name:  "aiml-checkpointing",
			Value: bool(true),
		},
	},
}, "implicit-dirs": {
	MachineBasedOptimization: []shared.MachineBasedOptimization{
		{
			Group: "high-performance",
			Value: bool(true),
		},
	},
	Profiles: []shared.ProfileOptimization{
		{
			Name:  "aiml-training",
			Value: bool(true),
		},
		{
			Name:  "aiml-serving",
			Value: bool(true),
		},
		{
			Name:  "aiml-checkpointing",
			Value: bool(true),
		},
	},
}, "file-system.kernel-list-cache-ttl-secs": {
	Profiles: []shared.ProfileOptimization{
		{
			Name:  "aiml-serving",
			Value: int64(-1),
		},
	},
}, "metadata-cache.negative-ttl-secs": {
	MachineBasedOptimization: []shared.MachineBasedOptimization{
		{
			Group: "high-performance",
			Value: int64(0),
		},
	},
	Profiles: []shared.ProfileOptimization{
		{
			Name:  "aiml-training",
			Value: int64(0),
		},
		{
			Name:  "aiml-serving",
			Value: int64(0),
		},
		{
			Name:  "aiml-checkpointing",
			Value: int64(0),
		},
	},
}, "metadata-cache.ttl-secs": {
	MachineBasedOptimization: []shared.MachineBasedOptimization{
		{
			Group: "high-performance",
			Value: int64(-1),
		},
	},
	Profiles: []shared.ProfileOptimization{
		{
			Name:  "aiml-training",
			Value: int64(-1),
		},
		{
			Name:  "aiml-serving",
			Value: int64(-1),
		},
		{
			Name:  "aiml-checkpointing",
			Value: int64(-1),
		},
	},
}, "file-system.rename-dir-limit": {
	MachineBasedOptimization: []shared.MachineBasedOptimization{
		{
			Group: "high-performance",
			Value: int64(200000),
		},
	},
	Profiles: []shared.ProfileOptimization{
		{
			Name:  "aiml-checkpointing",
			Value: int64(200000),
		},
	},
}, "metadata-cache.stat-cache-max-size-mb": {
	MachineBasedOptimization: []shared.MachineBasedOptimization{
		{
			Group: "high-performance",
			Value: int64(1024),
		},
	},
	Profiles: []shared.ProfileOptimization{
		{
			Name:  "aiml-training",
			Value: int64(-1),
		},
		{
			Name:  "aiml-serving",
			Value: int64(-1),
		},
		{
			Name:  "aiml-checkpointing",
			Value: int64(-1),
		},
	},
}, "metadata-cache.type-cache-max-size-mb": {
	MachineBasedOptimization: []shared.MachineBasedOptimization{
		{
			Group: "high-performance",
			Value: int64(128),
		},
	},
	Profiles: []shared.ProfileOptimization{
		{
			Name:  "aiml-training",
			Value: int64(-1),
		},
		{
			Name:  "aiml-serving",
			Value: int64(-1),
		},
		{
			Name:  "aiml-checkpointing",
			Value: int64(-1),
		},
	},
}, "write.global-max-blocks": {
	MachineBasedOptimization: []shared.MachineBasedOptimization{
		{
			Group: "high-performance",
			Value: int64(1600),
		},
	},
},
}

// machineTypeToGroupMap is the generated map from machine type to the group it belongs to.
var machineTypeToGroupMap = map[string]string{
	"a2-megagpu-16g":        "high-performance",
	"a2-ultragpu-8g":        "high-performance",
	"a3-edgegpu-8g":         "high-performance",
	"a3-highgpu-8g":         "high-performance",
	"a3-megagpu-8g":         "high-performance",
	"a3-ultragpu-8g":        "high-performance",
	"a4-highgpu-8g":         "high-performance",
	"a4-highgpu-8g-lowmem":  "high-performance",
	"a4-highgpu-8g-nolssd":  "high-performance",
	"a4x-highgpu-4g":        "high-performance",
	"a4x-highgpu-4g-nolssd": "high-performance",
	"ct5l-hightpu-8t":       "high-performance",
	"ct5lp-hightpu-8t":      "high-performance",
	"ct5p-hightpu-4t":       "high-performance",
	"ct5p-hightpu-4t-tpu":   "high-performance",
	"ct6e-standard-4t":      "high-performance",
	"ct6e-standard-4t-tpu":  "high-performance",
	"ct6e-standard-8t":      "high-performance",
	"ct6e-standard-8t-tpu":  "high-performance",
	"tpu7x-standard-4t":     "high-performance",
	"tpu7x-standard-4t-tpu": "high-performance",
	"tpu7x-ultranet-4t":     "high-performance",
	"tpu7x-ultranet-4t-tpu": "high-performance",
}

// ApplyOptimizations modifies the config in-place with optimized values.
func (c *Config) ApplyOptimizations(isSet isValueSet) map[string]OptimizationResult {
	var optimizedFlags = make(map[string]OptimizationResult)
	// Skip all optimizations if autoconfig is disabled.
	if c.DisableAutoconfig {
		return nil
	}

	profileName := c.Profile
	machineType, err := getMachineType(isSet, c)
	if err != nil {
		// Non-fatal, just means machine-based optimizations won't apply.
		machineType = ""
	}
	c.MachineType = machineType

	// Apply optimizations for each flag that has rules defined.
	if !isSet.IsSet("file-cache-cache-file-for-range-read") {
		rules := AllFlagOptimizationRules["file-cache.cache-file-for-range-read"]
		result := getOptimizedValue(&rules, c.FileCache.CacheFileForRangeRead, profileName, machineType, machineTypeToGroupMap)
		if result.Optimized {
			if val, ok := result.FinalValue.(bool); ok {
				if c.FileCache.CacheFileForRangeRead != val {
					c.FileCache.CacheFileForRangeRead = val
					optimizedFlags["file-cache.cache-file-for-range-read"] = result
				}
			}
		}
	}
	if !isSet.IsSet("implicit-dirs") {
		rules := AllFlagOptimizationRules["implicit-dirs"]
		result := getOptimizedValue(&rules, c.ImplicitDirs, profileName, machineType, machineTypeToGroupMap)
		if result.Optimized {
			if val, ok := result.FinalValue.(bool); ok {
				if c.ImplicitDirs != val {
					c.ImplicitDirs = val
					optimizedFlags["implicit-dirs"] = result
				}
			}
		}
	}
	if !isSet.IsSet("kernel-list-cache-ttl-secs") {
		rules := AllFlagOptimizationRules["file-system.kernel-list-cache-ttl-secs"]
		result := getOptimizedValue(&rules, c.FileSystem.KernelListCacheTtlSecs, profileName, machineType, machineTypeToGroupMap)
		if result.Optimized {
			if val, ok := result.FinalValue.(int64); ok {
				if c.FileSystem.KernelListCacheTtlSecs != val {
					c.FileSystem.KernelListCacheTtlSecs = val
					optimizedFlags["file-system.kernel-list-cache-ttl-secs"] = result
				}
			}
		}
	}
	if !isSet.IsSet("metadata-cache-negative-ttl-secs") {
		rules := AllFlagOptimizationRules["metadata-cache.negative-ttl-secs"]
		result := getOptimizedValue(&rules, c.MetadataCache.NegativeTtlSecs, profileName, machineType, machineTypeToGroupMap)
		if result.Optimized {
			if val, ok := result.FinalValue.(int64); ok {
				if c.MetadataCache.NegativeTtlSecs != val {
					c.MetadataCache.NegativeTtlSecs = val
					optimizedFlags["metadata-cache.negative-ttl-secs"] = result
				}
			}
		}
	}
	if !isSet.IsSet("metadata-cache-ttl-secs") {
		rules := AllFlagOptimizationRules["metadata-cache.ttl-secs"]
		result := getOptimizedValue(&rules, c.MetadataCache.TtlSecs, profileName, machineType, machineTypeToGroupMap)
		if result.Optimized {
			if val, ok := result.FinalValue.(int64); ok {
				if c.MetadataCache.TtlSecs != val {
					c.MetadataCache.TtlSecs = val
					optimizedFlags["metadata-cache.ttl-secs"] = result
				}
			}
		}
	}
	if !isSet.IsSet("rename-dir-limit") {
		rules := AllFlagOptimizationRules["file-system.rename-dir-limit"]
		result := getOptimizedValue(&rules, c.FileSystem.RenameDirLimit, profileName, machineType, machineTypeToGroupMap)
		if result.Optimized {
			if val, ok := result.FinalValue.(int64); ok {
				if c.FileSystem.RenameDirLimit != val {
					c.FileSystem.RenameDirLimit = val
					optimizedFlags["file-system.rename-dir-limit"] = result
				}
			}
		}
	}
	if !isSet.IsSet("stat-cache-max-size-mb") {
		rules := AllFlagOptimizationRules["metadata-cache.stat-cache-max-size-mb"]
		result := getOptimizedValue(&rules, c.MetadataCache.StatCacheMaxSizeMb, profileName, machineType, machineTypeToGroupMap)
		if result.Optimized {
			if val, ok := result.FinalValue.(int64); ok {
				if c.MetadataCache.StatCacheMaxSizeMb != val {
					c.MetadataCache.StatCacheMaxSizeMb = val
					optimizedFlags["metadata-cache.stat-cache-max-size-mb"] = result
				}
			}
		}
	}
	if !isSet.IsSet("type-cache-max-size-mb") {
		rules := AllFlagOptimizationRules["metadata-cache.type-cache-max-size-mb"]
		result := getOptimizedValue(&rules, c.MetadataCache.TypeCacheMaxSizeMb, profileName, machineType, machineTypeToGroupMap)
		if result.Optimized {
			if val, ok := result.FinalValue.(int64); ok {
				if c.MetadataCache.TypeCacheMaxSizeMb != val {
					c.MetadataCache.TypeCacheMaxSizeMb = val
					optimizedFlags["metadata-cache.type-cache-max-size-mb"] = result
				}
			}
		}
	}
	if !isSet.IsSet("write-global-max-blocks") {
		rules := AllFlagOptimizationRules["write.global-max-blocks"]
		result := getOptimizedValue(&rules, c.Write.GlobalMaxBlocks, profileName, machineType, machineTypeToGroupMap)
		if result.Optimized {
			if val, ok := result.FinalValue.(int64); ok {
				if c.Write.GlobalMaxBlocks != val {
					c.Write.GlobalMaxBlocks = val
					optimizedFlags["write.global-max-blocks"] = result
				}
			}
		}
	}
	return optimizedFlags
}

type CloudProfilerConfig struct {
	AllocatedHeap bool `yaml:"allocated-heap"`

	Cpu bool `yaml:"cpu"`

	Enabled bool `yaml:"enabled"`

	Goroutines bool `yaml:"goroutines"`

	Heap bool `yaml:"heap"`

	Label string `yaml:"label"`

	Mutex bool `yaml:"mutex"`
}

type Config struct {
	AppName string `yaml:"app-name"`

	CacheDir ResolvedPath `yaml:"cache-dir"`

	CloudProfiler CloudProfilerConfig `yaml:"cloud-profiler"`

	Debug DebugConfig `yaml:"debug"`

	DisableAutoconfig bool `yaml:"disable-autoconfig"`

	DummyIo DummyIoConfig `yaml:"dummy-io"`

	EnableAtomicRenameObject bool `yaml:"enable-atomic-rename-object"`

	EnableGoogleLibAuth bool `yaml:"enable-google-lib-auth"`

	EnableHns bool `yaml:"enable-hns"`

	EnableNewReader bool `yaml:"enable-new-reader"`

	EnableUnsupportedPathSupport bool `yaml:"enable-unsupported-path-support"`

	FileCache FileCacheConfig `yaml:"file-cache"`

	FileSystem FileSystemConfig `yaml:"file-system"`

	Foreground bool `yaml:"foreground"`

	GcsAuth GcsAuthConfig `yaml:"gcs-auth"`

	GcsConnection GcsConnectionConfig `yaml:"gcs-connection"`

	GcsRetries GcsRetriesConfig `yaml:"gcs-retries"`

	ImplicitDirs bool `yaml:"implicit-dirs"`

	List ListConfig `yaml:"list"`

	Logging LoggingConfig `yaml:"logging"`

	MachineType string `yaml:"machine-type"`

	MetadataCache MetadataCacheConfig `yaml:"metadata-cache"`

	Metrics MetricsConfig `yaml:"metrics"`

	Monitoring MonitoringConfig `yaml:"monitoring"`

	OnlyDir string `yaml:"only-dir"`

	Profile string `yaml:"profile"`

	Read ReadConfig `yaml:"read"`

	WorkloadInsight WorkloadInsightConfig `yaml:"workload-insight"`

	Write WriteConfig `yaml:"write"`
}

type DebugConfig struct {
	ExitOnInvariantViolation bool `yaml:"exit-on-invariant-violation"`

	Fuse bool `yaml:"fuse"`

	Gcs bool `yaml:"gcs"`

	LogMutex bool `yaml:"log-mutex"`
}

type DummyIoConfig struct {
	Enable bool `yaml:"enable"`

	PerMbLatency time.Duration `yaml:"per-mb-latency"`

	ReaderLatency time.Duration `yaml:"reader-latency"`
}

type FileCacheConfig struct {
	CacheFileForRangeRead bool `yaml:"cache-file-for-range-read"`

	DownloadChunkSizeMb int64 `yaml:"download-chunk-size-mb"`

	EnableCrc bool `yaml:"enable-crc"`

	EnableODirect bool `yaml:"enable-o-direct"`

	EnableParallelDownloads bool `yaml:"enable-parallel-downloads"`

	ExcludeRegex string `yaml:"exclude-regex"`

	ExperimentalEnableChunkCache bool `yaml:"experimental-enable-chunk-cache"`

	ExperimentalParallelDownloadsDefaultOn bool `yaml:"experimental-parallel-downloads-default-on"`

	IncludeRegex string `yaml:"include-regex"`

	MaxParallelDownloads int64 `yaml:"max-parallel-downloads"`

	MaxSizeMb int64 `yaml:"max-size-mb"`

	ParallelDownloadsPerFile int64 `yaml:"parallel-downloads-per-file"`

	WriteBufferSize int64 `yaml:"write-buffer-size"`
}

type FileSystemConfig struct {
	DirMode Octal `yaml:"dir-mode"`

	DisableParallelDirops bool `yaml:"disable-parallel-dirops"`

	ExperimentalEnableDentryCache bool `yaml:"experimental-enable-dentry-cache"`

	ExperimentalEnableReaddirplus bool `yaml:"experimental-enable-readdirplus"`

	FileMode Octal `yaml:"file-mode"`

	FuseOptions []string `yaml:"fuse-options"`

	Gid int64 `yaml:"gid"`

	IgnoreInterrupts bool `yaml:"ignore-interrupts"`

	KernelListCacheTtlSecs int64 `yaml:"kernel-list-cache-ttl-secs"`

	MaxReadAheadKb int64 `yaml:"max-read-ahead-kb"`

	ODirect bool `yaml:"o-direct"`

	PreconditionErrors bool `yaml:"precondition-errors"`

	RenameDirLimit int64 `yaml:"rename-dir-limit"`

	TempDir ResolvedPath `yaml:"temp-dir"`

	Uid int64 `yaml:"uid"`
}

type GcsAuthConfig struct {
	AnonymousAccess bool `yaml:"anonymous-access"`

	KeyFile ResolvedPath `yaml:"key-file"`

	ReuseTokenFromUrl bool `yaml:"reuse-token-from-url"`

	TokenUrl string `yaml:"token-url"`
}

type GcsConnectionConfig struct {
	BillingProject string `yaml:"billing-project"`

	ClientProtocol Protocol `yaml:"client-protocol"`

	CustomEndpoint string `yaml:"custom-endpoint"`

	EnableHttpDnsCache bool `yaml:"enable-http-dns-cache"`

	ExperimentalEnableJsonRead bool `yaml:"experimental-enable-json-read"`

	ExperimentalLocalSocketAddress string `yaml:"experimental-local-socket-address"`

	GrpcConnPoolSize int64 `yaml:"grpc-conn-pool-size"`

	HttpClientTimeout time.Duration `yaml:"http-client-timeout"`

	LimitBytesPerSec float64 `yaml:"limit-bytes-per-sec"`

	LimitOpsPerSec float64 `yaml:"limit-ops-per-sec"`

	MaxConnsPerHost int64 `yaml:"max-conns-per-host"`

	MaxIdleConnsPerHost int64 `yaml:"max-idle-conns-per-host"`

	SequentialReadSizeMb int64 `yaml:"sequential-read-size-mb"`
}

type GcsRetriesConfig struct {
	ChunkTransferTimeoutSecs int64 `yaml:"chunk-transfer-timeout-secs"`

	MaxRetryAttempts int64 `yaml:"max-retry-attempts"`

	MaxRetrySleep time.Duration `yaml:"max-retry-sleep"`

	Multiplier float64 `yaml:"multiplier"`

	ReadStall ReadStallGcsRetriesConfig `yaml:"read-stall"`
}

type ListConfig struct {
	EnableEmptyManagedFolders bool `yaml:"enable-empty-managed-folders"`
}

type LogRotateLoggingConfig struct {
	BackupFileCount int64 `yaml:"backup-file-count"`

	Compress bool `yaml:"compress"`

	MaxFileSizeMb int64 `yaml:"max-file-size-mb"`
}

type LoggingConfig struct {
	FilePath ResolvedPath `yaml:"file-path"`

	Format string `yaml:"format"`

	LogRotate LogRotateLoggingConfig `yaml:"log-rotate"`

	Severity LogSeverity `yaml:"severity"`
}

type MetadataCacheConfig struct {
	DeprecatedStatCacheCapacity int64 `yaml:"deprecated-stat-cache-capacity"`

	DeprecatedStatCacheTtl time.Duration `yaml:"deprecated-stat-cache-ttl"`

	DeprecatedTypeCacheTtl time.Duration `yaml:"deprecated-type-cache-ttl"`

	EnableNonexistentTypeCache bool `yaml:"enable-nonexistent-type-cache"`

	ExperimentalMetadataPrefetchOnMount string `yaml:"experimental-metadata-prefetch-on-mount"`

	NegativeTtlSecs int64 `yaml:"negative-ttl-secs"`

	StatCacheMaxSizeMb int64 `yaml:"stat-cache-max-size-mb"`

	TtlSecs int64 `yaml:"ttl-secs"`

	TypeCacheMaxSizeMb int64 `yaml:"type-cache-max-size-mb"`
}

type MetricsConfig struct {
	BufferSize int64 `yaml:"buffer-size"`

	CloudMetricsExportIntervalSecs int64 `yaml:"cloud-metrics-export-interval-secs"`

	EnableGrpcMetrics bool `yaml:"enable-grpc-metrics"`

	PrometheusPort int64 `yaml:"prometheus-port"`

	StackdriverExportInterval time.Duration `yaml:"stackdriver-export-interval"`

	UseNewNames bool `yaml:"use-new-names"`

	Workers int64 `yaml:"workers"`
}

type MonitoringConfig struct {
	ExperimentalTracingMode string `yaml:"experimental-tracing-mode"`

	ExperimentalTracingProjectId string `yaml:"experimental-tracing-project-id"`

	ExperimentalTracingSamplingRatio float64 `yaml:"experimental-tracing-sampling-ratio"`
}

type ReadConfig struct {
	BlockSizeMb int64 `yaml:"block-size-mb"`

	EnableBufferedRead bool `yaml:"enable-buffered-read"`

	GlobalMaxBlocks int64 `yaml:"global-max-blocks"`

	InactiveStreamTimeout time.Duration `yaml:"inactive-stream-timeout"`

	MaxBlocksPerHandle int64 `yaml:"max-blocks-per-handle"`

	MinBlocksPerHandle int64 `yaml:"min-blocks-per-handle"`

	RandomSeekThreshold int64 `yaml:"random-seek-threshold"`

	StartBlocksPerHandle int64 `yaml:"start-blocks-per-handle"`
}

type ReadStallGcsRetriesConfig struct {
	Enable bool `yaml:"enable"`

	InitialReqTimeout time.Duration `yaml:"initial-req-timeout"`

	MaxReqTimeout time.Duration `yaml:"max-req-timeout"`

	MinReqTimeout time.Duration `yaml:"min-req-timeout"`

	ReqIncreaseRate float64 `yaml:"req-increase-rate"`

	ReqTargetPercentile float64 `yaml:"req-target-percentile"`
}

type WorkloadInsightConfig struct {
	ForwardMergeThresholdMb int64 `yaml:"forward-merge-threshold-mb"`

	OutputFile string `yaml:"output-file"`

	Visualize bool `yaml:"visualize"`
}

type WriteConfig struct {
	BlockSizeMb int64 `yaml:"block-size-mb"`

	CreateEmptyFile bool `yaml:"create-empty-file"`

	EnableRapidAppends bool `yaml:"enable-rapid-appends"`

	EnableStreamingWrites bool `yaml:"enable-streaming-writes"`

	FinalizeFileForRapid bool `yaml:"finalize-file-for-rapid"`

	GlobalMaxBlocks int64 `yaml:"global-max-blocks"`

	MaxBlocksPerFile int64 `yaml:"max-blocks-per-file"`
}

func BuildFlagSet(flagSet *pflag.FlagSet) error {

	flagSet.BoolP("anonymous-access", "", false, "This flag disables authentication.")

	flagSet.StringP("app-name", "", "", "The application name of this mount.")

	flagSet.StringP("billing-project", "", "", "Project to use for billing when accessing a bucket enabled with \"Requester Pays\".")

	flagSet.StringP("cache-dir", "", "", "Enables file-caching. Specifies the directory to use for file-cache.")

	flagSet.IntP("chunk-transfer-timeout-secs", "", 10, "We send larger file uploads in 16 MiB chunks. This flag controls the duration that the HTTP client will wait for a response after making a request to upload a chunk. As an example, a value of 10 indicates that the client will wait 10 seconds for upload completion; otherwise, it cancels the request and retries for that chunk till chunkRetryDeadline(32s). 0 means no timeout.")

	if err := flagSet.MarkHidden("chunk-transfer-timeout-secs"); err != nil {
		return err
	}

	flagSet.StringP("client-protocol", "", "http1", "The protocol used for communicating with the GCS backend. Value can be 'http1' (HTTP/1.1), 'http2' (HTTP/2) or 'grpc'.")

	flagSet.IntP("cloud-metrics-export-interval-secs", "", 0, "Specifies the interval at which the metrics are uploaded to cloud monitoring")

	flagSet.BoolP("cloud-profiler-allocated-heap", "", true, "Enables allocated heap (HeapProfileAllocs) profiling. This only works when --enable-cloud-profiler is set to true.")

	if err := flagSet.MarkHidden("cloud-profiler-allocated-heap"); err != nil {
		return err
	}

	flagSet.BoolP("cloud-profiler-cpu", "", true, "Enables cpu profiling. This only works when --enable-cloud-profiler is set to true.")

	if err := flagSet.MarkHidden("cloud-profiler-cpu"); err != nil {
		return err
	}

	flagSet.BoolP("cloud-profiler-goroutines", "", false, "Enables goroutines cloud-profiler. This only works when --enable-cloud-profiler is set to true.")

	if err := flagSet.MarkHidden("cloud-profiler-goroutines"); err != nil {
		return err
	}

	flagSet.BoolP("cloud-profiler-heap", "", true, "Enables heap cloud-profiler. This only works when --enable-cloud-profiler is set to true.")

	if err := flagSet.MarkHidden("cloud-profiler-heap"); err != nil {
		return err
	}

	flagSet.StringP("cloud-profiler-label", "", "gcsfuse-0.0.0", "Allow setting a profile label to uniquely identify and compare cloud-profiler data with other profiles. This only works when --enable-cloud-profiler is set to true.")

	if err := flagSet.MarkHidden("cloud-profiler-label"); err != nil {
		return err
	}

	flagSet.BoolP("cloud-profiler-mutex", "", false, "Enables mutex cloud-profiler. This only works when --enable-cloud-profiler is set to true.")

	if err := flagSet.MarkHidden("cloud-profiler-mutex"); err != nil {
		return err
	}

	flagSet.BoolP("create-empty-file", "", false, "For a new file, it creates an empty file in Cloud Storage bucket as a hold.")

	if err := flagSet.MarkHidden("create-empty-file"); err != nil {
		return err
	}

	flagSet.StringP("custom-endpoint", "", "", "To specify a custom storage endpoint, ensure it supports the same resources as the default storage.googleapis.com:443 and includes the port number.")

	flagSet.BoolP("debug_fs", "", false, "This flag is unused.")

	if err := flagSet.MarkDeprecated("debug_fs", "This flag is currently unused."); err != nil {
		return err
	}

	flagSet.BoolP("debug_fuse", "", false, "Enables debug logs.")

	if err := flagSet.MarkDeprecated("debug_fuse", "Please set log-severity to TRACE instead."); err != nil {
		return err
	}

	flagSet.BoolP("debug_fuse_errors", "", true, "This flag is currently unused.")

	if err := flagSet.MarkDeprecated("debug_fuse_errors", "This flag is currently unused."); err != nil {
		return err
	}

	flagSet.BoolP("debug_gcs", "", false, "Enables debug logs.")

	if err := flagSet.MarkDeprecated("debug_gcs", "Please set log-severity to TRACE instead."); err != nil {
		return err
	}

	flagSet.BoolP("debug_http", "", false, "This flag is currently unused.")

	if err := flagSet.MarkDeprecated("debug_http", "This flag is currently unused."); err != nil {
		return err
	}

	flagSet.BoolP("debug_invariants", "", false, "Exit when internal invariants are violated.")

	flagSet.BoolP("debug_mutex", "", false, "Print debug messages when a mutex is held too long.")

	flagSet.StringP("dir-mode", "", "0755", "Permissions bits for directories, in octal.")

	flagSet.BoolP("disable-autoconfig", "", false, "Disable optimizing configuration automatically for a machine")

	if err := flagSet.MarkHidden("disable-autoconfig"); err != nil {
		return err
	}

	flagSet.BoolP("disable-parallel-dirops", "", false, "Specifies whether to allow parallel dir operations (lookups and readers)")

	if err := flagSet.MarkHidden("disable-parallel-dirops"); err != nil {
		return err
	}

	flagSet.DurationP("dummy-io-per-mb-latency", "", 0*time.Nanosecond, "Simulates reading from the reader latency in dummy I/O mode. This value is only used when dummy I/O mode is enabled.")

	if err := flagSet.MarkHidden("dummy-io-per-mb-latency"); err != nil {
		return err
	}

	flagSet.DurationP("dummy-io-reader-latency", "", 0*time.Nanosecond, "Simulates reader creation latency in dummy I/O mode. This value is only used when dummy I/O mode is enabled.")

	if err := flagSet.MarkHidden("dummy-io-reader-latency"); err != nil {
		return err
	}

	flagSet.BoolP("enable-atomic-rename-object", "", true, "Enables support for atomic rename object operation on HNS bucket.")

	if err := flagSet.MarkHidden("enable-atomic-rename-object"); err != nil {
		return err
	}

	flagSet.BoolP("enable-buffered-read", "", false, "When enabled, read starts using buffer to prefetch (asynchronous and in parallel) data from GCS. This improves performance for large file sequential reads. Note: Enabling this flag can increase the memory usage significantly.")

	flagSet.BoolP("enable-cloud-profiler", "", false, "Enables cloud-profiler, by default disabled.")

	if err := flagSet.MarkHidden("enable-cloud-profiler"); err != nil {
		return err
	}

	flagSet.BoolP("enable-dummy-io", "", false, "Enable dummy I/O mode for testing purposes. In this mode all reads and writes are simulated and no actual data is transferred to or from Cloud Storage. All the metadata operations like object listing and stats are real.")

	if err := flagSet.MarkHidden("enable-dummy-io"); err != nil {
		return err
	}

	flagSet.BoolP("enable-empty-managed-folders", "", false, "This handles the corner case in listing managed folders. There are two corner cases (a) empty managed folder (b) nested managed folder which doesn't contain any descendent as object. This flag always works in conjunction with --implicit-dirs flag. (a) If only ImplicitDirectories is true, all managed folders are listed other than above two mentioned cases. (b) If both ImplicitDirectories and EnableEmptyManagedFolders are true, then all the managed folders are listed including the above-mentioned corner case. (c) If ImplicitDirectories is false then no managed folders are listed irrespective of enable-empty-managed-folders flag.")

	if err := flagSet.MarkHidden("enable-empty-managed-folders"); err != nil {
		return err
	}

	flagSet.BoolP("enable-google-lib-auth", "", true, "Enable google library authentication method to fetch the credentials")

	if err := flagSet.MarkHidden("enable-google-lib-auth"); err != nil {
		return err
	}

	flagSet.BoolP("enable-grpc-metrics", "", false, "Enables support for gRPC metrics")

	flagSet.BoolP("enable-hns", "", true, "Enables support for HNS buckets")

	if err := flagSet.MarkHidden("enable-hns"); err != nil {
		return err
	}

	flagSet.BoolP("enable-http-dns-cache", "", true, "Enables DNS cache for HTTP/1 connections")

	if err := flagSet.MarkHidden("enable-http-dns-cache"); err != nil {
		return err
	}

	flagSet.BoolP("enable-new-reader", "", true, "Enables support for new reader implementation.")

	if err := flagSet.MarkHidden("enable-new-reader"); err != nil {
		return err
	}

	flagSet.BoolP("enable-nonexistent-type-cache", "", false, "Once set, if an inode is not found in GCS, a type cache entry with type NonexistentType will be created. This also means new file/dir created might not be seen. For example, if this flag is set, and metadata-cache-ttl-secs is set, then if we create the same file/node in the meantime using the same mount, since we are not refreshing the cache, it will still return nil.")

	flagSet.BoolP("enable-rapid-appends", "", true, "Enables support for appends to unfinalized object using streaming writes")

	flagSet.BoolP("enable-read-stall-retry", "", true, "To turn on/off retries for stalled read requests. This is based on a timeout that changes depending on how long similar requests took in the past.")

	if err := flagSet.MarkHidden("enable-read-stall-retry"); err != nil {
		return err
	}

	flagSet.BoolP("enable-streaming-writes", "", true, "Enables streaming uploads during write file operation.")

	flagSet.BoolP("enable-unsupported-path-support", "", true, "Enables support for file system paths with unsupported GCS names (e.g., names containing '//' or starting with /).  When set, GCSFuse will ignore these objects during listing and copying operations.  For rename and delete operations, the flag allows the action to proceed for all specified objects, including those with unsupported names.")

	if err := flagSet.MarkHidden("enable-unsupported-path-support"); err != nil {
		return err
	}

	flagSet.BoolP("experimental-enable-dentry-cache", "", false, "When enabled, it sets the Dentry cache entry timeout same as metadata-cache-ttl. This enables kernel to use cached entry to map the file paths to inodes, instead of making LookUpInode calls to GCSFuse.")

	if err := flagSet.MarkHidden("experimental-enable-dentry-cache"); err != nil {
		return err
	}

	flagSet.BoolP("experimental-enable-json-read", "", false, "By default, GCSFuse uses the GCS XML API to get and read objects. When this flag is specified, GCSFuse uses the GCS JSON API instead.\"")

	if err := flagSet.MarkDeprecated("experimental-enable-json-read", "Experimental flag: could be dropped even in a minor release."); err != nil {
		return err
	}

	flagSet.BoolP("experimental-enable-readdirplus", "", false, "Enables ReadDirPlus capability")

	if err := flagSet.MarkHidden("experimental-enable-readdirplus"); err != nil {
		return err
	}

	flagSet.IntP("experimental-grpc-conn-pool-size", "", 1, "The number of gRPC channel in grpc client.")

	if err := flagSet.MarkDeprecated("experimental-grpc-conn-pool-size", "Experimental flag: can be removed in a minor release."); err != nil {
		return err
	}

	flagSet.StringP("experimental-local-socket-address", "", "", "The local socket address to bind to. This is useful in multi-NIC scenarios. This is an experimental flag.")

	if err := flagSet.MarkHidden("experimental-local-socket-address"); err != nil {
		return err
	}

	flagSet.StringP("experimental-metadata-prefetch-on-mount", "", "disabled", "Experimental: This indicates whether or not to prefetch the metadata (prefilling of metadata caches and creation of inodes) of the mounted bucket at the time of mounting the bucket. Supported values: \"disabled\", \"sync\" and \"async\". Any other values will return error on mounting. This is applicable only to static mounting, and not to dynamic mounting.")

	if err := flagSet.MarkDeprecated("experimental-metadata-prefetch-on-mount", "Experimental flag: could be removed even in a minor release."); err != nil {
		return err
	}

	flagSet.StringP("experimental-tracing-mode", "", "", "Experimental: specify tracing mode")

	if err := flagSet.MarkHidden("experimental-tracing-mode"); err != nil {
		return err
	}

	flagSet.StringP("experimental-tracing-project-id", "", "", "Experimental: specify the GCP project-id to which traces will be exported. When unset, a project-id will be inferred as per the default credential detection process")

	if err := flagSet.MarkHidden("experimental-tracing-project-id"); err != nil {
		return err
	}

	flagSet.Float64P("experimental-tracing-sampling-ratio", "", 0, "Experimental: Trace sampling ratio")

	if err := flagSet.MarkHidden("experimental-tracing-sampling-ratio"); err != nil {
		return err
	}

	flagSet.BoolP("file-cache-cache-file-for-range-read", "", false, "Whether to cache file for range reads.")

	flagSet.IntP("file-cache-download-chunk-size-mb", "", 200, "Size of chunks in MiB that each concurrent request downloads.")

	flagSet.BoolP("file-cache-enable-crc", "", false, "Performs CRC to ensure that file is correctly downloaded into cache. No op for rapid storage.")

	if err := flagSet.MarkHidden("file-cache-enable-crc"); err != nil {
		return err
	}

	flagSet.BoolP("file-cache-enable-o-direct", "", false, "Whether to use O_DIRECT while writing to file-cache in case of parallel downloads.")

	if err := flagSet.MarkHidden("file-cache-enable-o-direct"); err != nil {
		return err
	}

	flagSet.BoolP("file-cache-enable-parallel-downloads", "", false, "Enable parallel downloads.")

	flagSet.StringP("file-cache-exclude-regex", "", "", "Exclude file paths (in the format bucket_name/object_key) specified by this regex from file caching.")

	if err := flagSet.MarkHidden("file-cache-exclude-regex"); err != nil {
		return err
	}

	flagSet.BoolP("file-cache-experimental-enable-chunk-cache", "", false, "Enable chunk cache mode for random I/O optimization that downloads only requested blocks.")

	if err := flagSet.MarkHidden("file-cache-experimental-enable-chunk-cache"); err != nil {
		return err
	}

	flagSet.BoolP("file-cache-experimental-parallel-downloads-default-on", "", true, "Enable parallel downloads by default on experimental basis.")

	if err := flagSet.MarkHidden("file-cache-experimental-parallel-downloads-default-on"); err != nil {
		return err
	}

	flagSet.StringP("file-cache-include-regex", "", "", "Include file paths (in the format bucket_name/object_key) specified by this regex for file caching.")

	if err := flagSet.MarkHidden("file-cache-include-regex"); err != nil {
		return err
	}

	flagSet.IntP("file-cache-max-parallel-downloads", "", DefaultMaxParallelDownloads(), "Sets an uber limit of number of concurrent file download requests that are made across all files.")

	flagSet.IntP("file-cache-max-size-mb", "", -1, "Maximum size of the file-cache in MiBs")

	flagSet.IntP("file-cache-parallel-downloads-per-file", "", 16, "Number of concurrent download requests per file.")

	flagSet.IntP("file-cache-write-buffer-size", "", 4194304, "Size of in-memory buffer that is used per goroutine in parallel downloads while writing to file-cache.")

	if err := flagSet.MarkHidden("file-cache-write-buffer-size"); err != nil {
		return err
	}

	flagSet.StringP("file-mode", "", "0644", "Permissions bits for files, in octal.")

	flagSet.BoolP("finalize-file-for-rapid", "", false, "Finalizes the files on close for Rapid storage. Appends will be slower on finalized files.")

	if err := flagSet.MarkHidden("finalize-file-for-rapid"); err != nil {
		return err
	}

	flagSet.BoolP("foreground", "", false, "Stay in the foreground after mounting.")

	flagSet.IntP("gid", "", -1, "GID owner of all inodes.")

	flagSet.DurationP("http-client-timeout", "", 0*time.Nanosecond, "The time duration that http client will wait to get response from the server. A value of 0 indicates no timeout.")

	flagSet.BoolP("ignore-interrupts", "", true, "Instructs gcsfuse to ignore system interrupt signals (like SIGINT, triggered by Ctrl+C). This prevents those signals from immediately terminating gcsfuse inflight operations.")

	flagSet.BoolP("implicit-dirs", "", false, "Implicitly define directories based on content. See files and directories in docs/semantics for more information")

	flagSet.IntP("kernel-list-cache-ttl-secs", "", 0, "How long the directory listing (output of ls <dir>) should be cached in the kernel page cache. If a particular directory cache entry is kept by kernel for longer than TTL, then it will be sent for invalidation by gcsfuse on next opendir (comes in the start, as part of next listing) call. 0 means no caching. Use -1 to cache for lifetime (no ttl). Negative value other than -1 will throw error.")

	flagSet.StringP("key-file", "", "", "Absolute path to JSON key file for use with GCS. If this flag is left unset, Google application default credentials are used.")

	flagSet.Float64P("limit-bytes-per-sec", "", -1, "Bandwidth limit for reading data, measured over a 30-second window. (use -1 for no limit)")

	flagSet.Float64P("limit-ops-per-sec", "", -1, "Operations per second limit, measured over a 30-second window (use -1 for no limit)")

	flagSet.StringP("log-file", "", "", "The file for storing logs that can be parsed by fluentd. When not provided, plain text logs are printed to stdout when Cloud Storage FUSE is run in the foreground, or to syslog when Cloud Storage FUSE is run in the background.")

	flagSet.StringP("log-format", "", "json", "The format of the log file: 'text' or 'json'.")

	flagSet.IntP("log-rotate-backup-file-count", "", 10, "The maximum number of backup log files to retain after they have been rotated. A value of 0 indicates all backup files are retained.")

	flagSet.BoolP("log-rotate-compress", "", true, "Controls whether the rotated log files should be compressed using gzip.")

	flagSet.IntP("log-rotate-max-file-size-mb", "", 512, "The maximum size in megabytes that a log file can reach before it is rotated.")

	flagSet.StringP("log-severity", "", "info", "Specifies the logging severity expressed as one of [trace, debug, info, warning, error, off]")

	flagSet.StringP("machine-type", "", "", "Type of the machine on which gcsfuse is being run e.g. a3-highgpu-4g")

	if err := flagSet.MarkHidden("machine-type"); err != nil {
		return err
	}

	flagSet.IntP("max-conns-per-host", "", 0, "The max number of TCP connections allowed per server. This is effective when client-protocol is set to 'http1'. A value of 0 indicates no limit on TCP connections (limited by the machine specifications).")

	flagSet.IntP("max-idle-conns-per-host", "", 100, "The number of maximum idle connections allowed per server.")

	flagSet.IntP("max-read-ahead-kb", "", 0, "Sets max kernel-read-ahead for the mount in KiB. 0 means system default. Requires sudo permission to set this value, otherwise the value will be ignored and system default will be used.")

	if err := flagSet.MarkHidden("max-read-ahead-kb"); err != nil {
		return err
	}

	flagSet.IntP("max-retry-attempts", "", 0, "It sets a limit on the number of times an operation will be retried if it fails, preventing endless retry loops. A value of 0 indicates no limit.")

	flagSet.DurationP("max-retry-duration", "", 0*time.Nanosecond, "This is currently unused.")

	if err := flagSet.MarkDeprecated("max-retry-duration", "This is currently unused."); err != nil {
		return err
	}

	flagSet.DurationP("max-retry-sleep", "", 30000000000*time.Nanosecond, "The maximum duration allowed to sleep in a retry loop with exponential backoff for failed requests to GCS backend. Once the backoff duration exceeds this limit, the retry continues with this specified maximum value.")

	flagSet.IntP("metadata-cache-negative-ttl-secs", "", 5, "The negative-ttl-secs value in seconds to be used for expiring negative entries in metadata-cache. It can be set to -1 for no-ttl, 0 for no cache and > 0 for ttl-controlled negative entries in metadata-cache. Any value set below -1 will throw an error.")

	flagSet.IntP("metadata-cache-ttl-secs", "", 60, "The ttl value in seconds to be used for expiring items in metadata-cache. It can be set to -1 for no-ttl, 0 for no cache and > 0 for ttl-controlled metadata-cache. Any value set below -1 will throw an error.")

	flagSet.IntP("metrics-buffer-size", "", 256, "The maximum number of histogram metric updates in the queue.")

	if err := flagSet.MarkHidden("metrics-buffer-size"); err != nil {
		return err
	}

	flagSet.BoolP("metrics-use-new-names", "", false, "Use the new metric names.")

	if err := flagSet.MarkHidden("metrics-use-new-names"); err != nil {
		return err
	}

	flagSet.IntP("metrics-workers", "", 3, "The number of workers that update histogram metrics concurrently.")

	if err := flagSet.MarkHidden("metrics-workers"); err != nil {
		return err
	}

	flagSet.StringSliceP("o", "", []string{}, "Additional system-specific mount options. Multiple options can be passed as comma separated. For readonly, use --o ro")

	flagSet.BoolP("o-direct", "", false, "Bypasses the kernel's page cache for file reads and writes. When enabled, all I/O operations are sent directly to the GCSFuse daemon. ")

	if err := flagSet.MarkHidden("o-direct"); err != nil {
		return err
	}

	flagSet.StringP("only-dir", "", "", "Mount only a specific directory within the bucket. See docs/mounting for more information")

	flagSet.BoolP("precondition-errors", "", true, "Throw Stale NFS file handle error in case the object being synced or read from is modified by some other concurrent process. This helps prevent silent data loss or data corruption.")

	if err := flagSet.MarkHidden("precondition-errors"); err != nil {
		return err
	}

	flagSet.StringP("profile", "", "", "The name of the profile to apply. e.g. aiml-training, aiml-serving, aiml-checkpointing")

	flagSet.IntP("prometheus-port", "", 0, "Expose Prometheus metrics endpoint on this port and a path of /metrics.")

	flagSet.IntP("read-block-size-mb", "", 16, "Specifies the block size for buffered reads. The value should be more than 0. This is used to read data in chunks from GCS.")

	if err := flagSet.MarkHidden("read-block-size-mb"); err != nil {
		return err
	}

	flagSet.IntP("read-global-max-blocks", "", 40, "Specifies the maximum number of blocks available for buffered reads across all file-handles. The value should be >= 0 or -1 (for infinite blocks). A value of 0 disables buffered reads.")

	flagSet.DurationP("read-inactive-stream-timeout", "", 10000000000*time.Nanosecond, "Duration of inactivity after which an open GCS read stream is automatically closed. This helps conserve resources when a file handle remains open without active Read calls. A value of '0s' disables this timeout.")

	if err := flagSet.MarkHidden("read-inactive-stream-timeout"); err != nil {
		return err
	}

	flagSet.IntP("read-max-blocks-per-handle", "", 20, "Specifies the maximum number of blocks to be used by a single file handle for buffered reads. The value should be >= 0 or -1 (for infinite blocks). A value of 0 disables buffered reads.")

	if err := flagSet.MarkHidden("read-max-blocks-per-handle"); err != nil {
		return err
	}

	flagSet.IntP("read-min-blocks-per-handle", "", 4, "Specifies the minimum number of blocks required by a file-handle to start reading via buffered reads. The value should be >= 1 or \"read-max-blocks-per-handle\".")

	if err := flagSet.MarkHidden("read-min-blocks-per-handle"); err != nil {
		return err
	}

	flagSet.IntP("read-random-seek-threshold", "", 3, "Specifies the random seek threshold to switch to another reader when random reads are detected.")

	if err := flagSet.MarkHidden("read-random-seek-threshold"); err != nil {
		return err
	}

	flagSet.DurationP("read-stall-initial-req-timeout", "", 20000000000*time.Nanosecond, "Initial value of the read-request dynamic timeout.")

	if err := flagSet.MarkHidden("read-stall-initial-req-timeout"); err != nil {
		return err
	}

	flagSet.DurationP("read-stall-max-req-timeout", "", 1200000000000*time.Nanosecond, "Upper bound of the read-request dynamic timeout.")

	if err := flagSet.MarkHidden("read-stall-max-req-timeout"); err != nil {
		return err
	}

	flagSet.DurationP("read-stall-min-req-timeout", "", 1500000000*time.Nanosecond, "Lower bound of the read request dynamic timeout.")

	if err := flagSet.MarkHidden("read-stall-min-req-timeout"); err != nil {
		return err
	}

	flagSet.Float64P("read-stall-req-increase-rate", "", 15, "Determines how many increase calls it takes for dynamic timeout to double.")

	if err := flagSet.MarkHidden("read-stall-req-increase-rate"); err != nil {
		return err
	}

	flagSet.Float64P("read-stall-req-target-percentile", "", 0.99, "Retry the request which take more than p(targetPercentile * 100) of past similar request.")

	if err := flagSet.MarkHidden("read-stall-req-target-percentile"); err != nil {
		return err
	}

	flagSet.IntP("read-start-blocks-per-handle", "", 1, "Specifies the number of blocks to be prefetched on the first read.")

	if err := flagSet.MarkHidden("read-start-blocks-per-handle"); err != nil {
		return err
	}

	flagSet.IntP("rename-dir-limit", "", 0, "Allow rename a directory containing fewer descendants than this limit.")

	flagSet.Float64P("retry-multiplier", "", 2, "Param for exponential backoff algorithm, which is used to increase waiting time b/w two consecutive retries.")

	flagSet.BoolP("reuse-token-from-url", "", true, "If false, the token acquired from token-url is not reused.")

	flagSet.IntP("sequential-read-size-mb", "", 200, "File chunk size to read from GCS in one call. Need to specify the value in MB. ChunkSize less than 1MB is not supported")

	flagSet.DurationP("stackdriver-export-interval", "", 0*time.Nanosecond, "Export metrics to stackdriver with this interval. A value of 0 indicates no exporting.")

	if err := flagSet.MarkDeprecated("stackdriver-export-interval", "Please use --cloud-metrics-export-interval-secs instead."); err != nil {
		return err
	}

	flagSet.IntP("stat-cache-capacity", "", 20460, "How many entries can the stat-cache hold (impacts memory consumption). This flag has been deprecated (starting v2.0) and in favor of stat-cache-max-size-mb. For now, the value of stat-cache-capacity will be translated to the next higher corresponding value of stat-cache-max-size-mb (assuming stat-cache entry-size ~= 1688 bytes, including 1448 for positive entry and 240 for corresponding negative entry), if stat-cache-max-size-mb is not set.\"")

	if err := flagSet.MarkDeprecated("stat-cache-capacity", "Please use --stat-cache-max-size-mb instead."); err != nil {
		return err
	}

	flagSet.IntP("stat-cache-max-size-mb", "", 33, "The maximum size of stat-cache in MiBs. It can also be set to -1 for no-size-limit, 0 for no cache. Values below -1 are not supported.")

	flagSet.DurationP("stat-cache-ttl", "", 60000000000*time.Nanosecond, "How long to cache StatObject results and inode attributes. This flag has been deprecated (starting v2.0) in favor of metadata-cache-ttl-secs. For now, the minimum of stat-cache-ttl and type-cache-ttl values, rounded up to the next higher multiple of a second is used as ttl for both stat-cache and type-cache, when metadata-cache-ttl-secs is not set.")

	if err := flagSet.MarkDeprecated("stat-cache-ttl", "This flag has been deprecated (starting v2.0) in favor of metadata-cache-ttl-secs."); err != nil {
		return err
	}

	flagSet.StringP("temp-dir", "", "", "Path to the temporary directory where writes are staged prior to upload to Cloud Storage. (default: system default, likely /tmp)")

	flagSet.StringP("token-url", "", "", "A url for getting an access token when the key-file is absent.")

	flagSet.IntP("type-cache-max-size-mb", "", 4, "Max size of type-cache maps which are maintained at a per-directory level.")

	flagSet.DurationP("type-cache-ttl", "", 60000000000*time.Nanosecond, "Usage: How long to cache StatObject results and inode attributes. This flag has been deprecated (starting v2.0) in favor of metadata-cache-ttl-secs. For now, the minimum of stat-cache-ttl and type-cache-ttl values, rounded up to the next higher multiple of a second is used as ttl for both stat-cache and type-cache, when metadata-cache-ttl-secs is not set.")

	if err := flagSet.MarkDeprecated("type-cache-ttl", "This flag has been deprecated (starting v2.0) in favor of metadata-cache-ttl-secs."); err != nil {
		return err
	}

	flagSet.IntP("uid", "", -1, "UID owner of all inodes.")

	flagSet.BoolP("visualize-workload-insight", "", false, "A flag to enable workload visualization. When enabled, workload insights will include visualizations to help understand access patterns. Insights will be written to the file specified by --workload-insight-output-file.")

	if err := flagSet.MarkHidden("visualize-workload-insight"); err != nil {
		return err
	}

	flagSet.IntP("workload-insight-forward-merge-threshold-mb", "", 0, "The threshold in MB for merging forward sequential reads for workload insights visualization.Reads within this threshold will be merged into a single read operation. Applicable only when --visualize-workload-insight is enabled.")

	if err := flagSet.MarkHidden("workload-insight-forward-merge-threshold-mb"); err != nil {
		return err
	}

	flagSet.StringP("workload-insight-output-file", "", "", "The file path where the workload insights will be written. If not specified, insights will be written to stdout")

	if err := flagSet.MarkHidden("workload-insight-output-file"); err != nil {
		return err
	}

	flagSet.IntP("write-block-size-mb", "", 32, "Specifies the block size for streaming writes. The value should be more than 0.")

	if err := flagSet.MarkHidden("write-block-size-mb"); err != nil {
		return err
	}

	flagSet.IntP("write-global-max-blocks", "", 4, "Specifies the maximum number of blocks available for streaming writes across all files. The value should be >= 0 or -1 (for infinite blocks). A value of 0 disables streaming writes.")

	flagSet.IntP("write-max-blocks-per-file", "", 1, "Specifies the maximum number of blocks to be used by a single file for streaming writes. The value should be >= 1 or -1 (for infinite blocks).")

	if err := flagSet.MarkHidden("write-max-blocks-per-file"); err != nil {
		return err
	}

	return nil
}

func BindFlags(v *viper.Viper, flagSet *pflag.FlagSet) error {

	if err := v.BindPFlag("gcs-auth.anonymous-access", flagSet.Lookup("anonymous-access")); err != nil {
		return err
	}

	if err := v.BindPFlag("app-name", flagSet.Lookup("app-name")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-connection.billing-project", flagSet.Lookup("billing-project")); err != nil {
		return err
	}

	if err := v.BindPFlag("cache-dir", flagSet.Lookup("cache-dir")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-retries.chunk-transfer-timeout-secs", flagSet.Lookup("chunk-transfer-timeout-secs")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-connection.client-protocol", flagSet.Lookup("client-protocol")); err != nil {
		return err
	}

	if err := v.BindPFlag("metrics.cloud-metrics-export-interval-secs", flagSet.Lookup("cloud-metrics-export-interval-secs")); err != nil {
		return err
	}

	if err := v.BindPFlag("cloud-profiler.allocated-heap", flagSet.Lookup("cloud-profiler-allocated-heap")); err != nil {
		return err
	}

	if err := v.BindPFlag("cloud-profiler.cpu", flagSet.Lookup("cloud-profiler-cpu")); err != nil {
		return err
	}

	if err := v.BindPFlag("cloud-profiler.goroutines", flagSet.Lookup("cloud-profiler-goroutines")); err != nil {
		return err
	}

	if err := v.BindPFlag("cloud-profiler.heap", flagSet.Lookup("cloud-profiler-heap")); err != nil {
		return err
	}

	if err := v.BindPFlag("cloud-profiler.label", flagSet.Lookup("cloud-profiler-label")); err != nil {
		return err
	}

	if err := v.BindPFlag("cloud-profiler.mutex", flagSet.Lookup("cloud-profiler-mutex")); err != nil {
		return err
	}

	if err := v.BindPFlag("write.create-empty-file", flagSet.Lookup("create-empty-file")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-connection.custom-endpoint", flagSet.Lookup("custom-endpoint")); err != nil {
		return err
	}

	if err := v.BindPFlag("debug.fuse", flagSet.Lookup("debug_fuse")); err != nil {
		return err
	}

	if err := v.BindPFlag("debug.gcs", flagSet.Lookup("debug_gcs")); err != nil {
		return err
	}

	if err := v.BindPFlag("debug.exit-on-invariant-violation", flagSet.Lookup("debug_invariants")); err != nil {
		return err
	}

	if err := v.BindPFlag("debug.log-mutex", flagSet.Lookup("debug_mutex")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-system.dir-mode", flagSet.Lookup("dir-mode")); err != nil {
		return err
	}

	if err := v.BindPFlag("disable-autoconfig", flagSet.Lookup("disable-autoconfig")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-system.disable-parallel-dirops", flagSet.Lookup("disable-parallel-dirops")); err != nil {
		return err
	}

	if err := v.BindPFlag("dummy-io.per-mb-latency", flagSet.Lookup("dummy-io-per-mb-latency")); err != nil {
		return err
	}

	if err := v.BindPFlag("dummy-io.reader-latency", flagSet.Lookup("dummy-io-reader-latency")); err != nil {
		return err
	}

	if err := v.BindPFlag("enable-atomic-rename-object", flagSet.Lookup("enable-atomic-rename-object")); err != nil {
		return err
	}

	if err := v.BindPFlag("read.enable-buffered-read", flagSet.Lookup("enable-buffered-read")); err != nil {
		return err
	}

	if err := v.BindPFlag("cloud-profiler.enabled", flagSet.Lookup("enable-cloud-profiler")); err != nil {
		return err
	}

	if err := v.BindPFlag("dummy-io.enable", flagSet.Lookup("enable-dummy-io")); err != nil {
		return err
	}

	if err := v.BindPFlag("list.enable-empty-managed-folders", flagSet.Lookup("enable-empty-managed-folders")); err != nil {
		return err
	}

	if err := v.BindPFlag("enable-google-lib-auth", flagSet.Lookup("enable-google-lib-auth")); err != nil {
		return err
	}

	if err := v.BindPFlag("metrics.enable-grpc-metrics", flagSet.Lookup("enable-grpc-metrics")); err != nil {
		return err
	}

	if err := v.BindPFlag("enable-hns", flagSet.Lookup("enable-hns")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-connection.enable-http-dns-cache", flagSet.Lookup("enable-http-dns-cache")); err != nil {
		return err
	}

	if err := v.BindPFlag("enable-new-reader", flagSet.Lookup("enable-new-reader")); err != nil {
		return err
	}

	if err := v.BindPFlag("metadata-cache.enable-nonexistent-type-cache", flagSet.Lookup("enable-nonexistent-type-cache")); err != nil {
		return err
	}

	if err := v.BindPFlag("write.enable-rapid-appends", flagSet.Lookup("enable-rapid-appends")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-retries.read-stall.enable", flagSet.Lookup("enable-read-stall-retry")); err != nil {
		return err
	}

	if err := v.BindPFlag("write.enable-streaming-writes", flagSet.Lookup("enable-streaming-writes")); err != nil {
		return err
	}

	if err := v.BindPFlag("enable-unsupported-path-support", flagSet.Lookup("enable-unsupported-path-support")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-system.experimental-enable-dentry-cache", flagSet.Lookup("experimental-enable-dentry-cache")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-connection.experimental-enable-json-read", flagSet.Lookup("experimental-enable-json-read")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-system.experimental-enable-readdirplus", flagSet.Lookup("experimental-enable-readdirplus")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-connection.grpc-conn-pool-size", flagSet.Lookup("experimental-grpc-conn-pool-size")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-connection.experimental-local-socket-address", flagSet.Lookup("experimental-local-socket-address")); err != nil {
		return err
	}

	if err := v.BindPFlag("metadata-cache.experimental-metadata-prefetch-on-mount", flagSet.Lookup("experimental-metadata-prefetch-on-mount")); err != nil {
		return err
	}

	if err := v.BindPFlag("monitoring.experimental-tracing-mode", flagSet.Lookup("experimental-tracing-mode")); err != nil {
		return err
	}

	if err := v.BindPFlag("monitoring.experimental-tracing-project-id", flagSet.Lookup("experimental-tracing-project-id")); err != nil {
		return err
	}

	if err := v.BindPFlag("monitoring.experimental-tracing-sampling-ratio", flagSet.Lookup("experimental-tracing-sampling-ratio")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-cache.cache-file-for-range-read", flagSet.Lookup("file-cache-cache-file-for-range-read")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-cache.download-chunk-size-mb", flagSet.Lookup("file-cache-download-chunk-size-mb")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-cache.enable-crc", flagSet.Lookup("file-cache-enable-crc")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-cache.enable-o-direct", flagSet.Lookup("file-cache-enable-o-direct")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-cache.enable-parallel-downloads", flagSet.Lookup("file-cache-enable-parallel-downloads")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-cache.exclude-regex", flagSet.Lookup("file-cache-exclude-regex")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-cache.experimental-enable-chunk-cache", flagSet.Lookup("file-cache-experimental-enable-chunk-cache")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-cache.experimental-parallel-downloads-default-on", flagSet.Lookup("file-cache-experimental-parallel-downloads-default-on")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-cache.include-regex", flagSet.Lookup("file-cache-include-regex")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-cache.max-parallel-downloads", flagSet.Lookup("file-cache-max-parallel-downloads")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-cache.max-size-mb", flagSet.Lookup("file-cache-max-size-mb")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-cache.parallel-downloads-per-file", flagSet.Lookup("file-cache-parallel-downloads-per-file")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-cache.write-buffer-size", flagSet.Lookup("file-cache-write-buffer-size")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-system.file-mode", flagSet.Lookup("file-mode")); err != nil {
		return err
	}

	if err := v.BindPFlag("write.finalize-file-for-rapid", flagSet.Lookup("finalize-file-for-rapid")); err != nil {
		return err
	}

	if err := v.BindPFlag("foreground", flagSet.Lookup("foreground")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-system.gid", flagSet.Lookup("gid")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-connection.http-client-timeout", flagSet.Lookup("http-client-timeout")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-system.ignore-interrupts", flagSet.Lookup("ignore-interrupts")); err != nil {
		return err
	}

	if err := v.BindPFlag("implicit-dirs", flagSet.Lookup("implicit-dirs")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-system.kernel-list-cache-ttl-secs", flagSet.Lookup("kernel-list-cache-ttl-secs")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-auth.key-file", flagSet.Lookup("key-file")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-connection.limit-bytes-per-sec", flagSet.Lookup("limit-bytes-per-sec")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-connection.limit-ops-per-sec", flagSet.Lookup("limit-ops-per-sec")); err != nil {
		return err
	}

	if err := v.BindPFlag("logging.file-path", flagSet.Lookup("log-file")); err != nil {
		return err
	}

	if err := v.BindPFlag("logging.format", flagSet.Lookup("log-format")); err != nil {
		return err
	}

	if err := v.BindPFlag("logging.log-rotate.backup-file-count", flagSet.Lookup("log-rotate-backup-file-count")); err != nil {
		return err
	}

	if err := v.BindPFlag("logging.log-rotate.compress", flagSet.Lookup("log-rotate-compress")); err != nil {
		return err
	}

	if err := v.BindPFlag("logging.log-rotate.max-file-size-mb", flagSet.Lookup("log-rotate-max-file-size-mb")); err != nil {
		return err
	}

	if err := v.BindPFlag("logging.severity", flagSet.Lookup("log-severity")); err != nil {
		return err
	}

	if err := v.BindPFlag("machine-type", flagSet.Lookup("machine-type")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-connection.max-conns-per-host", flagSet.Lookup("max-conns-per-host")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-connection.max-idle-conns-per-host", flagSet.Lookup("max-idle-conns-per-host")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-system.max-read-ahead-kb", flagSet.Lookup("max-read-ahead-kb")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-retries.max-retry-attempts", flagSet.Lookup("max-retry-attempts")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-retries.max-retry-sleep", flagSet.Lookup("max-retry-sleep")); err != nil {
		return err
	}

	if err := v.BindPFlag("metadata-cache.negative-ttl-secs", flagSet.Lookup("metadata-cache-negative-ttl-secs")); err != nil {
		return err
	}

	if err := v.BindPFlag("metadata-cache.ttl-secs", flagSet.Lookup("metadata-cache-ttl-secs")); err != nil {
		return err
	}

	if err := v.BindPFlag("metrics.buffer-size", flagSet.Lookup("metrics-buffer-size")); err != nil {
		return err
	}

	if err := v.BindPFlag("metrics.use-new-names", flagSet.Lookup("metrics-use-new-names")); err != nil {
		return err
	}

	if err := v.BindPFlag("metrics.workers", flagSet.Lookup("metrics-workers")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-system.fuse-options", flagSet.Lookup("o")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-system.o-direct", flagSet.Lookup("o-direct")); err != nil {
		return err
	}

	if err := v.BindPFlag("only-dir", flagSet.Lookup("only-dir")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-system.precondition-errors", flagSet.Lookup("precondition-errors")); err != nil {
		return err
	}

	if err := v.BindPFlag("profile", flagSet.Lookup("profile")); err != nil {
		return err
	}

	if err := v.BindPFlag("metrics.prometheus-port", flagSet.Lookup("prometheus-port")); err != nil {
		return err
	}

	if err := v.BindPFlag("read.block-size-mb", flagSet.Lookup("read-block-size-mb")); err != nil {
		return err
	}

	if err := v.BindPFlag("read.global-max-blocks", flagSet.Lookup("read-global-max-blocks")); err != nil {
		return err
	}

	if err := v.BindPFlag("read.inactive-stream-timeout", flagSet.Lookup("read-inactive-stream-timeout")); err != nil {
		return err
	}

	if err := v.BindPFlag("read.max-blocks-per-handle", flagSet.Lookup("read-max-blocks-per-handle")); err != nil {
		return err
	}

	if err := v.BindPFlag("read.min-blocks-per-handle", flagSet.Lookup("read-min-blocks-per-handle")); err != nil {
		return err
	}

	if err := v.BindPFlag("read.random-seek-threshold", flagSet.Lookup("read-random-seek-threshold")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-retries.read-stall.initial-req-timeout", flagSet.Lookup("read-stall-initial-req-timeout")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-retries.read-stall.max-req-timeout", flagSet.Lookup("read-stall-max-req-timeout")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-retries.read-stall.min-req-timeout", flagSet.Lookup("read-stall-min-req-timeout")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-retries.read-stall.req-increase-rate", flagSet.Lookup("read-stall-req-increase-rate")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-retries.read-stall.req-target-percentile", flagSet.Lookup("read-stall-req-target-percentile")); err != nil {
		return err
	}

	if err := v.BindPFlag("read.start-blocks-per-handle", flagSet.Lookup("read-start-blocks-per-handle")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-system.rename-dir-limit", flagSet.Lookup("rename-dir-limit")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-retries.multiplier", flagSet.Lookup("retry-multiplier")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-auth.reuse-token-from-url", flagSet.Lookup("reuse-token-from-url")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-connection.sequential-read-size-mb", flagSet.Lookup("sequential-read-size-mb")); err != nil {
		return err
	}

	if err := v.BindPFlag("metrics.stackdriver-export-interval", flagSet.Lookup("stackdriver-export-interval")); err != nil {
		return err
	}

	if err := v.BindPFlag("metadata-cache.deprecated-stat-cache-capacity", flagSet.Lookup("stat-cache-capacity")); err != nil {
		return err
	}

	if err := v.BindPFlag("metadata-cache.stat-cache-max-size-mb", flagSet.Lookup("stat-cache-max-size-mb")); err != nil {
		return err
	}

	if err := v.BindPFlag("metadata-cache.deprecated-stat-cache-ttl", flagSet.Lookup("stat-cache-ttl")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-system.temp-dir", flagSet.Lookup("temp-dir")); err != nil {
		return err
	}

	if err := v.BindPFlag("gcs-auth.token-url", flagSet.Lookup("token-url")); err != nil {
		return err
	}

	if err := v.BindPFlag("metadata-cache.type-cache-max-size-mb", flagSet.Lookup("type-cache-max-size-mb")); err != nil {
		return err
	}

	if err := v.BindPFlag("metadata-cache.deprecated-type-cache-ttl", flagSet.Lookup("type-cache-ttl")); err != nil {
		return err
	}

	if err := v.BindPFlag("file-system.uid", flagSet.Lookup("uid")); err != nil {
		return err
	}

	if err := v.BindPFlag("workload-insight.visualize", flagSet.Lookup("visualize-workload-insight")); err != nil {
		return err
	}

	if err := v.BindPFlag("workload-insight.forward-merge-threshold-mb", flagSet.Lookup("workload-insight-forward-merge-threshold-mb")); err != nil {
		return err
	}

	if err := v.BindPFlag("workload-insight.output-file", flagSet.Lookup("workload-insight-output-file")); err != nil {
		return err
	}

	if err := v.BindPFlag("write.block-size-mb", flagSet.Lookup("write-block-size-mb")); err != nil {
		return err
	}

	if err := v.BindPFlag("write.global-max-blocks", flagSet.Lookup("write-global-max-blocks")); err != nil {
		return err
	}

	if err := v.BindPFlag("write.max-blocks-per-file", flagSet.Lookup("write-max-blocks-per-file")); err != nil {
		return err
	}

	return nil
}
