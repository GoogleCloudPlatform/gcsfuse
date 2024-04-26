// Copyright 2023 Google Inc. All Rights Reserved.
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

package cfg

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/mitchellh/mapstructure"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	configFileFlagName = "config-file"
)

type WriteConfig struct {
	CreateEmptyFile bool `mapstructure:"create-empty-file"`
}

// LogRotateConfig defines the parameters for log rotation. It consists of three
// configuration options:
// 1. max-file-size-mb: specifies the maximum size in megabytes that a log file
// can reach before it is rotated. The default value is 512 megabytes.
// 2. backup-file-count: determines the maximum number of backup log files to
// retain after they have been rotated. The default value is 10. When value is
// set to 0, all backup files are retained.
// 3. compress: indicates whether the rotated log files should be compressed
// using gzip. The default value is False.
type LogRotateConfig struct {
	MaxFileSizeMB   int `mapstructure:"max-file-size-mb"`
	BackupFileCount int `mapstructure:"backup-file-count"`
	Compress        bool
}

type LogSeverity string

type LogConfig struct {
	Severity  LogSeverity
	Format    string
	FilePath  string          `mapstructure:"file-path"`
	LogRotate LogRotateConfig `mapstructure:"log-rotate"`
}

type ListConfig struct {
	// This flag is specially added to handle the corner case in listing managed folders.
	// There are two corner cases (a) empty managed folder (b) nested managed folder which doesn't contain any descendent as object.
	// This flag always works in conjunction with ImplicitDirectories flag.
	//
	// (a) If only ImplicitDirectories is true, all managed folders are listed other than above two mentioned cases.
	// (b) If both ImplicitDirectories and EnableEmptyManagedFolders are true, then all the managed folders are listed including the above-mentioned corner case.
	// (c) If ImplicitDirectories is false then no managed folders are listed irrespective of EnableEmptyManagedFolders flag.
	EnableEmptyManagedFolders bool `mapstructure:"enable-empty-managed-folders"`
}

type GrpcClientConfig struct {
	// ConnPoolSize configures the number of gRPC channel in grpc client.
	ConnPoolSize int `mapstructure:"conn-pool-size"`
}

// EnableHNS enables the storage control client flow on HNS buckets to utilize new APIs.
type EnableHNS bool
type CacheDir string

type FileCacheConfig struct {
	MaxSizeMB             int64 `mapstructure:"max-size-mb"`
	CacheFileForRangeRead bool  `mapstructure:"cache-file-for-range-read"`
}

type MetadataCacheConfig struct {
	// TtlInSeconds is the ttl
	// value in seconds, to be used for stat-cache and type-cache.
	// It can be set to -1 for no-ttl, 0 for
	// no cache and > 0 for ttl-controlled metadata-cache.
	// Any value set below -1 will throw an error.
	TtlInSeconds int64 `mapstructure:"ttl-secs,omitempty"`
	// TypeCacheMaxSizeMB is the upper limit
	// on the maximum size of type-cache maps,
	// which are currently
	// maintained at per-directory level.
	TypeCacheMaxSizeMB int `mapstructure:"type-cache-max-size-mb,omitempty"`

	// StatCacheMaxSizeMB is the maximum size of stat-cache
	// in MiBs.
	// It can also be set to -1 for no-size-limit, 0 for
	// no cache. Values below -1 are not supported.
	StatCacheMaxSizeMB int64 `mapstructure:"stat-cache-max-size-mb,omitempty"`

	EnableNonExistentTypeCache bool `mapstructure:"enable-nonexistent-type-cache"`
}

type RetryConfig struct {
	Multiplier    int
	MaxRetrySleep time.Duration `mapstructure:"max-retry-sleep"`
}

type AuthConfig struct {
	KeyFile           string `mapstructure:"key-file"`
	TokenURL          string `mapstructure:"token-url"`
	ReuseTokenFromURL bool   `mapstructure:"reuse-token-from-url"`
}

type GCSConnectionConfig struct {
	BillingProject             string  `mapstructure:"billing-project"`
	MaxBytesPerSec             float64 `mapstructure:"max-bytes-per-sec"`
	MaxOpsPerSec               float64 `mapstructure:"max-ops-per-sec"`
	SequentialReadSizeMB       int     `mapstructure:"sequential-read-size-mb"`
	ExperimentalEnableJSONRead bool    `mapstructure:"experimental-enable-json-read"`
	CustomEndpoint             string  `mapstructure:"custom-endpoint"`
	Protocol                   string
	Timeout                    time.Duration
	MaxConnections             int `mapstructure:"max-connections"`
	MaxIdleConnections         int `mapstructure:"max-idle-connections"`
	ConnectionPoolSize         int `mapstructure:"connection-pool-size"`
	Retries                    RetryConfig
	Auth                       AuthConfig
}

type OctalInt int

func (oi OctalInt) String() string {
	return fmt.Sprintf("%o", oi)
}

type FileSystemConfig struct {
	TempDir        string `mapstructure:"temp-dir"`
	RenameDirLimit int    `mapstructure:"rename-dir-limit"`
	GID            int
	UID            int
	FileMode       OctalInt `mapstructure:"file-mode"`
	DirMode        OctalInt `mapstructure:"dir-mode"`
	MountOptions   []string `mapstructure:"mount-options"`
}

type DebugConfig struct {
	ExitOnInvariantViolation bool `mapstructure:"exit-on-invariant-violation"`
	LogMutex                 bool `mapstructure:"log-mutex"`
}

type MonitoringConfig struct {
	MetricsExportInterval                     time.Duration `mapstructure:"metrics-export-interval"`
	ExperimentalOpenTelemetryCollectorAddress string        `mapstructure:"experimental-opentelemetry-collector-address"`
}
type Config struct {
	AppName       string `mapstructure:"app-name"`
	Bucket        string
	MountPoint    string `mapstructure:"mount-point"`
	Foreground    bool
	OnlyDir       string              `mapstructure:"only-dir"`
	GCSConnection GCSConnectionConfig `mapstructure:"gcs-connection"`
	Write         WriteConfig
	Logging       LogConfig
	FileCache     FileCacheConfig     `mapstructure:"file-cache"`
	CacheDir      string              `mapstructure:"cache-dir"`
	MetadataCache MetadataCacheConfig `mapstructure:"metadata-cache"`
	List          ListConfig
	Grpc          GrpcClientConfig `mapstructure:"grpc"`
	EnableHNS     bool             `mapstructure:"enable-hns"`
	FileSystem    FileSystemConfig `mapstructure:"file-system"`
	Debug         DebugConfig
	Monitoring    MonitoringConfig
}

// octalIntHookFunc converts string flag to octal while flag parsing.
func octalIntHookFunc() mapstructure.DecodeHookFuncType {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}

		if t != reflect.TypeOf(OctalInt(0)) {
			return data, nil
		}

		val, err := strconv.ParseInt(data.(string), 8, 32)
		if err != nil {
			err = fmt.Errorf("parsing as octal: %w", err)
			return 0, err
		}
		return val, nil
	}
}

var Cfg Config

var flagNames = make([]string, 0, 100)

func addStringParam(flagSet *flag.FlagSet, flagName, defaultVal, helpDoc string) {
	flagNames = append(flagNames, flagName)
	flagSet.String(flagName, defaultVal, helpDoc)
}

func addStringParamP(flagSet *flag.FlagSet, flagName, shorthand, defaultVal, helpDoc string) {
	flagNames = append(flagNames, flagName)
	flagSet.StringP(flagName, shorthand, defaultVal, helpDoc)
}

func addBoolParam(flagSet *flag.FlagSet, flagName string, defaultVal bool, helpDoc string) {
	flagNames = append(flagNames, flagName)
	flagSet.Bool(flagName, defaultVal, helpDoc)
}

func addIntParam(flagSet *flag.FlagSet, flagName string, defaultVal int, helpDoc string) {
	flagNames = append(flagNames, flagName)
	flagSet.Int(flagName, defaultVal, helpDoc)
}

func addFloat64Param(flagSet *flag.FlagSet, flagName string, defaultVal float64, helpDoc string) {
	flagNames = append(flagNames, flagName)
	flagSet.Float64(flagName, defaultVal, helpDoc)
}

func addDurationParam(flagSet *flag.FlagSet, flagName string, defaultVal time.Duration, helpDoc string) {
	flagNames = append(flagNames, flagName)
	flagSet.Duration(flagName, defaultVal, helpDoc)
}

func addStringSliceParamP(flagSet *flag.FlagSet, flagName string, shorthand string, defaultVal []string, helpDoc string) {
	flagNames = append(flagNames, flagName)
	flagSet.StringSliceP(flagName, shorthand, defaultVal, helpDoc)
}

func ParseConfig() (Config, error) {
	flagSet := flag.NewFlagSet("flagSet", flag.ExitOnError)
	addStringParamP(flagSet, configFileFlagName, "c", "", "The path to the config file where all gcsfuse related config needs to be specified. "+
		"Refer to 'https://cloud.google.com/storage/docs/gcsfuse-cli#config-file' for possible configurations.")

	addFlags(flagSet)

	var cfg Config
	err := flagSet.Parse(os.Args[1:])
	if err != nil {
		return cfg, err
	}
	v := viper.New()
	for _, f := range flagNames {
		err = v.BindPFlag(f, flagSet.Lookup(f))
		if err != nil {
			return cfg, err
		}
	}

	if cfgFile := v.GetString(configFileFlagName); cfgFile != "" {
		// Use config file from the flag.
		v.SetConfigFile(cfgFile)
		if err := v.ReadInConfig(); err != nil {
			err = fmt.Errorf("error while reading the config file: %w", err)
			return cfg, err
		}
	}
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	err = v.Unmarshal(&cfg, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		mapstructure.TextUnmarshallerHookFunc(),
		octalIntHookFunc(),
		mapstructure.StringToTimeDurationHookFunc(), // default hook
		mapstructure.StringToSliceHookFunc(","),     // default hook
	)))
	if err != nil {
		return cfg, err
	}
	cfg.Bucket, cfg.MountPoint, err = populateArgs(flagSet.Args())
	if err != nil {
		return cfg, err
	}
	cfg.MountPoint, err = util.GetResolvedPath(cfg.MountPoint)
	if err != nil {
		err = fmt.Errorf("canonicalizing mount point: %w", err)
		return cfg, err
	}
	err = validateConfig(cfg)
	return cfg, err
}

func addFlags(flagSet *flag.FlagSet) {
	// Top level params;
	addStringParam(flagSet, "app-name", "", "The application name of this mount.")
	addBoolParam(flagSet, "foreground", false, "Stay in the foreground after mounting.")
	addStringParam(flagSet, "only-dir", "", "Mount only a specific directory within the bucket. See docs/mounting for more information.")
	addBoolParam(flagSet, "enable-hns", false, "Enable HNS-specific features")

	// File-cache
	addIntParam(flagSet, "file-cache.max-size-mb", -1, "Max size of the file-cache")
	addBoolParam(flagSet, "file-cache.cache-file-for-range-read", false, "Max size of the file-cache")

	// Metadata-cache
	addIntParam(flagSet, "metadata-cache.stat-cache-max-size-mb", 32, "Max size of the stat-cache.")
	addIntParam(flagSet, "metadata-cache.ttl-secs", 3600, "TTL in seconds of the metadata-cache.")
	addIntParam(flagSet, "metadata-cache.type-cache-max-size-mb", 4, "Max size of the type-cache.")
	addBoolParam(flagSet, "metadata-cache.enable-nonexistent-type-cache", false, "Once set, if an inode is not found in GCS, a type cache entry with type NonexistentType"+
		" will be created. This also means new file/dir created might not be seen. For example, if this"+
		" flag is set, and metadata-cache:ttl-secs (in config-file) or flag type-cache-ttl are set, then if we create the same file/node"+
		" in the meantime using the same mount, since we are not refreshing the cache, it will still return nil.")

	// GCS-Connection
	addStringParam(flagSet, "gcs-connection.billing-project", "", "Project to use for billing when accessing a bucket enabled with “Requester Pays” (default: none)")
	addFloat64Param(flagSet, "gcs-connection.max-bytes-per-sec", -1.0, "Bandwidth limit for reading data, measured over a 30-second window. (use -1 for no limit)")
	addFloat64Param(flagSet, "gcs-connection.max-ops-per-sec", -1.0, "Operations per second limit, measured over a 30-second window (use -1 for no limit)")
	addIntParam(flagSet, "gcs-connection.sequential-read-size-mb", 200, "File chunk size to read from GCS in one call. Need to specify the value in MB. ChunkSize less than 1MB is not supported")
	addBoolParam(flagSet, "gcs-connection.experimental-enable-json-read", false, "By default, GCSFuse uses the GCS XML API to get and read objects.  +")
	addStringParam(flagSet, "gcs-connection.custom-endpoint", "", "Specifies an alternative custom endpoint for fetching data. Should only be used for testing. "+
		"The custom endpoint must support the equivalent resources and operations as the GCS "+
		"JSON endpoint, https://storage.googleapis.com/storage/v1. If a custom endpoint is not specified, "+
		"GCSFuse uses the global GCS JSON API endpoint, https://storage.googleapis.com/storage/v1. "+
		"If a custom endpoint is specified, authentication is disabled on the endpoint.")
	addStringParam(flagSet, "gcs-connection.protocol", "http1", "The protocol used for communicating with the GCS backend. Value can be 'http1' (HTTP/1.1) or 'http2' (HTTP/2) or grpc.")
	addDurationParam(flagSet, "gcs-connection.timeout", 0, "The time duration that http client will wait to get response from the server. The default value 0 indicates no timeout.")
	addIntParam(flagSet, "gcs-connection.max-connections", 100, "The max number of TCP connections allowed per server. This is effective when protocol is set to 'http1'.")
	addIntParam(flagSet, "gcs-connection.max-idle-connections", 100, "The number of maximum idle connections allowed per server.")
	addIntParam(flagSet, "gcs-connection.retries.multiplier", 2, "Param for exponential backoff algorithm, which is used to increase waiting time b/w two consecutive retries.")
	addDurationParam(flagSet, "gcs-connection.retries.max-retry-sleep", 30*time.Second, "The maximum duration allowed to sleep in a retry loop with exponential backoff for failed requests to GCS backend. Once the backoff duration exceeds this limit, the retry continues with this specified maximum value.")
	addStringParam(flagSet, "gcs-connection.auth.key-file", "", "Absolute path to JSON key file for use with GCS. (default: none, Google application default credentials used)")
	addStringParam(flagSet, "gcs-connection.auth.token-url", "", "A url for getting an access token when the key-file is absent.")
	addBoolParam(flagSet, "gcs-connection.auth.reuse-token-from-url", true, "If false, the token acquired from token-url is not reused.")

	// Logging
	addStringParam(flagSet, "logging.file-path", "", "The file for storing logs that can be parsed by fluentd. When not provided, plain text logs are printed to stdout.")
	addStringParam(flagSet, "logging.format", "json", "The log format.")
	addStringParam(flagSet, "logging.severity", "INFO", "The log level. By default INFO and above will be logged.")
	addIntParam(flagSet, "logging.log-rotate.max-file-size-mb", 512, "Max size of the log file.")
	addIntParam(flagSet, "logging.log-rotate.backup-file-count", 10, "The number of log backups that should be maintained.")
	addBoolParam(flagSet, "logging.log-rotate.compress", true, "If false, the log files will not be compressed.")

	// gRPC
	addIntParam(flagSet, "grpc.conn-pool-size", 1, "The number of maximum idle connections allowed per server.")

	// FileSystem
	addStringParam(flagSet, "file-system.temp-dir", "", "Path to the temporary directory where writes are staged prior to upload to Cloud Storage. (default: system default, likely /tmp)")
	addIntParam(flagSet, "file-system.rename-dir-limit", 0, "Allow rename a directory containing fewer descendants than this limit.")
	addIntParam(flagSet, "file-system.gid", -1, "GID owner of all inodes.")
	addIntParam(flagSet, "file-system.uid", -1, "UID owner of all inodes.")
	addStringParam(flagSet, "file-system.file-mode", "0644", "Permission bits for files, in octal.")
	addStringParam(flagSet, "file-system.dir-mode", "0755", "Permissions bits for directories, in octal.")
	addStringSliceParamP(flagSet, "file-system.mount-options", "o", []string{}, "Additional system-specific mount options. Multiple options can be passed as comma separated. For readonly, use -o ro")

	// Write
	addBoolParam(flagSet, "write.create-empty-file", false, "Create an empty file was created in the Cloud Storage bucket as a hold.")
	addBoolParam(flagSet, "list.enable-empty-managed-folders", false,
		"This flag handles the corner case in listing managed folders. "+
			"There are two corner cases:"+
			"(a) empty managed folder"+
			"(b) nested managed folder which doesn't contain any descendent as object.\n"+
			"This flag always works in conjunction with ImplicitDirectories flag. "+
			"(a) If only ImplicitDirectories is true, all managed folders are listed other than above two mentioned cases.\n"+
			"(b) If both ImplicitDirectories and EnableEmptyManagedFolders are true, then all the managed folders are listed including the above-mentioned corner case.\n"+
			"(c) If ImplicitDirectories is false then no managed folders are listed irrespective of EnableEmptyManagedFolders flag.")

	// Debug
	addBoolParam(flagSet, "debug.exit-on-invariant-violation", false, "Panic when internal invariants are violated.")
	addBoolParam(flagSet, "debug.log-mutex", false, "Print debug messages when a mutex is held too long.")

	// Monitoring
	addDurationParam(flagSet, "monitoring.metrics-export-interval", 0, "Export metrics to stackdriver with this interval. The default value 0 indicates no exporting.")
	addStringParam(flagSet, "monitoring.experimental-opentelemetry-collector-address", "", "Experimental: Export metrics to the OpenTelemetry collector at this address.")

}

func validateConfig(cfg Config) error {
	if cfg.MountPoint == "" {
		return fmt.Errorf("mountpoint is empty")
	}
	return nil
}

func populateArgs(args []string) (
	bucketName, mountPoint string,
	err error) {
	switch len(args) {
	case 1:
		bucketName = ""
		mountPoint = args[0]

	case 2:
		bucketName = args[0]
		mountPoint = args[1]

	default:
		err = fmt.Errorf(
			"%s takes one or two arguments. Run `%s --help` for more info",
			path.Base(os.Args[0]),
			path.Base(os.Args[0]))

		return bucketName, mountPoint, err

	}
	// Canonicalize the mount point, making it absolute. This is important when
	// daemonizing below, since the daemon will change its working directory
	// before running this code again.
	mountPoint, err = util.GetResolvedPath(mountPoint)
	if err != nil {
		err = fmt.Errorf("canonicalizing mount point: %w", err)
		return bucketName, mountPoint, err
	}
	return bucketName, mountPoint, err
}
