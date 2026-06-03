// Copyright 2026 Google LLC
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
	"encoding/base64"
	"reflect"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg/pb"
	"google.golang.org/protobuf/proto"
)

func ToProto(c *Config) *pb.Config {
	if c == nil {
		return nil
	}
	return &pb.Config{
		AppName:                      c.AppName,
		CacheDir:                     "", // SCRUBBED
		CloudProfiler:                toCloudProfilerProto(&c.CloudProfiler),
		Debug:                        toDebugProto(&c.Debug),
		DisableAutoconfig:            c.DisableAutoconfig,
		DisableListAccessCheck:       c.DisableListAccessCheck,
		DummyIo:                      toDummyIoProto(&c.DummyIo),
		EnableAtomicRenameObject:     c.EnableAtomicRenameObject,
		EnableGoogleLibAuth:          c.EnableGoogleLibAuth,
		EnableHns:                    c.EnableHns,
		EnableNewReader:              c.EnableNewReader,
		EnableStandardSymlinks:       c.EnableStandardSymlinks,
		EnableTypeCacheDeprecation:   c.EnableTypeCacheDeprecation,
		EnableUnsupportedPathSupport: c.EnableUnsupportedPathSupport,
		FileCache:                    toFileCacheProto(&c.FileCache),
		FileSystem:                   toFileSystemProto(&c.FileSystem),
		Foreground:                   c.Foreground,
		GcsAuth:                      toGcsAuthProto(&c.GcsAuth),
		GcsConnection:                toGcsConnectionProto(&c.GcsConnection),
		GcsRetries:                   toGcsRetriesProto(&c.GcsRetries),
		ImplicitDirs:                 c.ImplicitDirs,
		List:                         toListProto(&c.List),
		Logging:                      toLoggingProto(&c.Logging),
		MachineType:                  c.MachineType,
		MetadataCache:                toMetadataCacheProto(&c.MetadataCache),
		Metrics:                      toMetricsProto(&c.Metrics),
		Mrd:                          toMrdProto(&c.Mrd),
		OnlyDir:                      c.OnlyDir,
		Profile:                      c.Profile,
		Read:                         toReadProto(&c.Read),
		Trace:                        toTraceProto(&c.Trace),
		WorkloadInsight:              toWorkloadInsightProto(&c.WorkloadInsight),
		Write:                        toWriteProto(&c.Write),
	}
}

func toCloudProfilerProto(c *CloudProfilerConfig) *pb.CloudProfilerConfig {
	if c == nil {
		return nil
	}
	return &pb.CloudProfilerConfig{
		AllocatedHeap: c.AllocatedHeap,
		Cpu:           c.Cpu,
		Enabled:       c.Enabled,
		Goroutines:    c.Goroutines,
		Heap:          c.Heap,
		Label:         c.Label,
		Mutex:         c.Mutex,
		ServiceName:   c.ServiceName,
	}
}

func toDebugProto(c *DebugConfig) *pb.DebugConfig {
	if c == nil {
		return nil
	}
	return &pb.DebugConfig{
		ExitOnInvariantViolation: c.ExitOnInvariantViolation,
		Fuse:                     c.Fuse,
		Gcs:                      c.Gcs,
		LogMutex:                 c.LogMutex,
	}
}

func toDummyIoProto(c *DummyIoConfig) *pb.DummyIoConfig {
	if c == nil {
		return nil
	}
	return &pb.DummyIoConfig{
		Enable:        c.Enable,
		PerMbLatency:  int64(c.PerMbLatency),
		ReaderLatency: int64(c.ReaderLatency),
	}
}

func toFileCacheProto(c *FileCacheConfig) *pb.FileCacheConfig {
	if c == nil {
		return nil
	}
	return &pb.FileCacheConfig{
		CacheFileForRangeRead:                  c.CacheFileForRangeRead,
		DownloadChunkSizeMb:                    c.DownloadChunkSizeMb,
		EnableCrc:                              c.EnableCrc,
		EnableExperimentalSharedChunkCache:     c.EnableExperimentalSharedChunkCache,
		EnableODirect:                          c.EnableODirect,
		EnableParallelDownloads:                c.EnableParallelDownloads,
		ExcludeRegex:                           c.ExcludeRegex,
		ExperimentalDisableSizeCalculationFix:  c.ExperimentalDisableSizeCalculationFix,
		ExperimentalEnableChunkCache:           c.ExperimentalEnableChunkCache,
		ExperimentalParallelDownloadsDefaultOn: c.ExperimentalParallelDownloadsDefaultOn,
		IncludeRegex:                           c.IncludeRegex,
		MaxParallelDownloads:                   c.MaxParallelDownloads,
		MaxSizeMb:                              c.MaxSizeMb,
		ParallelDownloadsPerFile:               c.ParallelDownloadsPerFile,
		SharedCacheChunkSizeMb:                 c.SharedCacheChunkSizeMb,
		WriteBufferSize:                        c.WriteBufferSize,
	}
}

func toFileSystemProto(c *FileSystemConfig) *pb.FileSystemConfig {
	if c == nil {
		return nil
	}
	return &pb.FileSystemConfig{
		CongestionThreshold:           c.CongestionThreshold,
		DirMode:                       uint32(c.DirMode),
		DisableParallelDirops:         c.DisableParallelDirops,
		EnableKernelReader:            c.EnableKernelReader,
		ExperimentalEnableDentryCache: c.ExperimentalEnableDentryCache,
		ExperimentalEnablePirlo:       c.ExperimentalEnablePirlo,
		ExperimentalEnableReaddirplus: c.ExperimentalEnableReaddirplus,
		ExperimentalODirect:           c.ExperimentalODirect,
		FileMode:                      uint32(c.FileMode),
		FuseOptions:                   c.FuseOptions,
		Gid:                           c.Gid,
		IgnoreInterrupts:              c.IgnoreInterrupts,
		InactiveMrdCacheSize:          c.InactiveMrdCacheSize,
		KernelListCacheTtlSecs:        c.KernelListCacheTtlSecs,
		KernelParamsFile:              "", // SCRUBBED
		MaxBackground:                 c.MaxBackground,
		MaxReadAheadKb:                c.MaxReadAheadKb,
		RenameDirLimit:                c.RenameDirLimit,
		TempDir:                       "", // SCRUBBED
		Uid:                           c.Uid,
	}
}

func toGcsAuthProto(c *GcsAuthConfig) *pb.GcsAuthConfig {
	if c == nil {
		return nil
	}
	return &pb.GcsAuthConfig{
		AnonymousAccess:   c.AnonymousAccess,
		KeyFile:           "", // SCRUBBED
		ReuseTokenFromUrl: c.ReuseTokenFromUrl,
		TokenUrl:          "", // SCRUBBED
	}
}

func toGcsConnectionProto(c *GcsConnectionConfig) *pb.GcsConnectionConfig {
	if c == nil {
		return nil
	}
	return &pb.GcsConnectionConfig{
		BillingProject:                 c.BillingProject,
		ClientProtocol:                 string(c.ClientProtocol),
		CustomEndpoint:                 "", // SCRUBBED
		EnableHttpDnsCache:             c.EnableHttpDnsCache,
		ExperimentalEnableJsonRead:     c.ExperimentalEnableJsonRead,
		ExperimentalLocalSocketAddress: "", // SCRUBBED
		GrpcConnPoolSize:               c.GrpcConnPoolSize,
		GrpcPathStrategy:               string(c.GrpcPathStrategy),
		HttpClientTimeout:              int64(c.HttpClientTimeout),
		LimitBytesPerSec:               c.LimitBytesPerSec,
		LimitOpsPerSec:                 c.LimitOpsPerSec,
		MaxConnsPerHost:                c.MaxConnsPerHost,
		MaxIdleConnsPerHost:            c.MaxIdleConnsPerHost,
		SequentialReadSizeMb:           c.SequentialReadSizeMb,
	}
}

func toReadStallProto(c *ReadStallGcsRetriesConfig) *pb.ReadStallGcsRetriesConfig {
	return &pb.ReadStallGcsRetriesConfig{
		Enable:              c.Enable,
		InitialReqTimeout:   int64(c.InitialReqTimeout),
		MaxReqTimeout:       int64(c.MaxReqTimeout),
		MinReqTimeout:       int64(c.MinReqTimeout),
		ReqIncreaseRate:     c.ReqIncreaseRate,
		ReqTargetPercentile: c.ReqTargetPercentile,
	}
}

func toGcsRetriesProto(c *GcsRetriesConfig) *pb.GcsRetriesConfig {
	if c == nil {
		return nil
	}
	return &pb.GcsRetriesConfig{
		ChunkRetryDeadlineSecs:                  c.ChunkRetryDeadlineSecs,
		ChunkTransferTimeoutSecs:                c.ChunkTransferTimeoutSecs,
		EnableMountRetries:                      c.EnableMountRetries,
		ExperimentalNonrapidFolderApiStallRetry: c.ExperimentalNonrapidFolderApiStallRetry,
		MaxRetryAttempts:                        c.MaxRetryAttempts,
		MaxRetrySleep:                           int64(c.MaxRetrySleep),
		Multiplier:                              c.Multiplier,
		ReadStall:                               toReadStallProto(&c.ReadStall),
	}
}

func toListProto(c *ListConfig) *pb.ListConfig {
	if c == nil {
		return nil
	}
	return &pb.ListConfig{
		EnableEmptyManagedFolders: c.EnableEmptyManagedFolders,
	}
}

func toLoggingProto(c *LoggingConfig) *pb.LoggingConfig {
	if c == nil {
		return nil
	}
	return &pb.LoggingConfig{
		FilePath: "", // SCRUBBED
		Format:   c.Format,
		LogRotate: &pb.LogRotateLoggingConfig{
			BackupFileCount: c.LogRotate.BackupFileCount,
			Compress:        c.LogRotate.Compress,
			MaxFileSizeMb:   c.LogRotate.MaxFileSizeMb,
		},
		Severity: string(c.Severity),
		WireLog:  "", // SCRUBBED
	}
}

func toMetadataCacheProto(c *MetadataCacheConfig) *pb.MetadataCacheConfig {
	if c == nil {
		return nil
	}
	return &pb.MetadataCacheConfig{
		DeprecatedStatCacheCapacity:         c.DeprecatedStatCacheCapacity,
		DeprecatedStatCacheTtl:              int64(c.DeprecatedStatCacheTtl),
		DeprecatedTypeCacheTtl:              int64(c.DeprecatedTypeCacheTtl),
		EnableMetadataPrefetch:              c.EnableMetadataPrefetch,
		EnableNonexistentTypeCache:          c.EnableNonexistentTypeCache,
		ExperimentalMetadataPrefetchOnMount: c.ExperimentalMetadataPrefetchOnMount,
		MetadataPrefetchEntriesLimit:        c.MetadataPrefetchEntriesLimit,
		MetadataPrefetchMaxWorkers:          c.MetadataPrefetchMaxWorkers,
		NegativeTtlSecs:                     c.NegativeTtlSecs,
		StatCacheMaxSizeMb:                  c.StatCacheMaxSizeMb,
		TtlSecs:                             c.TtlSecs,
		TypeCacheMaxSizeMb:                  c.TypeCacheMaxSizeMb,
	}
}

func toMetricsProto(c *MetricsConfig) *pb.MetricsConfig {
	if c == nil {
		return nil
	}
	return &pb.MetricsConfig{
		BufferSize:                     c.BufferSize,
		CloudMetricsExportIntervalSecs: c.CloudMetricsExportIntervalSecs,
		ExperimentalEnableGrpcMetrics:  c.ExperimentalEnableGrpcMetrics,
		PrometheusPort:                 c.PrometheusPort,
		StackdriverExportInterval:      int64(c.StackdriverExportInterval),
		UseNewNames:                    c.UseNewNames,
		Workers:                        c.Workers,
	}
}

func toMrdProto(c *MrdConfig) *pb.MrdConfig {
	if c == nil {
		return nil
	}
	return &pb.MrdConfig{
		PoolSize: c.PoolSize,
	}
}

func toReadProto(c *ReadConfig) *pb.ReadConfig {
	if c == nil {
		return nil
	}
	return &pb.ReadConfig{
		BlockSizeMb:             c.BlockSizeMb,
		EnableBufferedRead:      c.EnableBufferedRead,
		GlobalMaxBlocks:         c.GlobalMaxBlocks,
		InactiveStreamTimeout:   int64(c.InactiveStreamTimeout),
		MaxBlocksPerHandle:      c.MaxBlocksPerHandle,
		MinBlocksPerHandle:      c.MinBlocksPerHandle,
		RandomSeekThreshold:     c.RandomSeekThreshold,
		StartBlocksPerHandle:    c.StartBlocksPerHandle,
	}
}

func toTraceProto(c *TraceConfig) *pb.TraceConfig {
	if c == nil {
		return nil
	}
	return &pb.TraceConfig{
		Exporters:     c.Exporters,
		ProjectId:     c.ProjectId,
		SamplingRatio: c.SamplingRatio,
	}
}

func toWorkloadInsightProto(c *WorkloadInsightConfig) *pb.WorkloadInsightConfig {
	if c == nil {
		return nil
	}
	return &pb.WorkloadInsightConfig{
		ForwardMergeThresholdMb: c.ForwardMergeThresholdMb,
		OutputFile:              c.OutputFile,
		Visualize:               c.Visualize,
	}
}

func toWriteProto(c *WriteConfig) *pb.WriteConfig {
	if c == nil {
		return nil
	}
	return &pb.WriteConfig{
		BlockSizeMb:           c.BlockSizeMb,
		CreateEmptyFile:       c.CreateEmptyFile,
		EnableRapidAppends:    c.EnableRapidAppends,
		EnableRapidWrites:     c.EnableRapidWrites,
		EnableStreamingWrites: c.EnableStreamingWrites,
		FinalizeFileOnClose:   c.FinalizeFileOnClose,
		GlobalMaxBlocks:       c.GlobalMaxBlocks,
		MaxBlocksPerFile:      c.MaxBlocksPerFile,
	}
}

func SerializeConfigToProtoBase64(c *Config) (string, error) {
	p := ToProto(c)
	truncateStrings(p)
	bytes, err := proto.Marshal(p)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func truncateStrings(p any) {
	v := reflect.ValueOf(p)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.Kind() == reflect.String && f.CanSet() && f.Len() > 50 {
			f.SetString(f.String()[:49] + "+")
		} else if f.Kind() == reflect.Ptr && !f.IsNil() {
			truncateStrings(f.Interface())
		}
	}
}
