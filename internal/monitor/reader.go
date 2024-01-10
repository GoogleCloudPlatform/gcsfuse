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

package monitor

import (
	"log"
	"strconv"

	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/internal/monitor/tags"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"golang.org/x/net/context"
)

var (
	// When a first read call is made by the user, we either fetch entire file or x number of bytes from GCS based on the request.
	// Now depending on the pagesize multiple read calls will be issued by user to read the entire file. These
	// requests will be served from the downloaded data.
	// This metric captures only the requests made to GCS, not the subsequent page calls.
	gcsReadCount = stats.Int64("gcs/read_count",
		"Specifies the number of gcs reads made along with type - Sequential/Random",
		stats.UnitDimensionless)
	downloadBytesCount = stats.Int64("gcs/download_bytes_count",
		"The cumulative number of bytes downloaded from GCS along with type - Sequential/Random",
		stats.UnitBytes)
	fileCacheReadCount = stats.Int64("file_cache/read_count",
		"Specifies the number of read requests made via file cache along with type - Sequential/Random and cache hit - true/false",
		stats.UnitDimensionless)
	fileCacheReadBytesCount = stats.Int64("file_cache/read_bytes_count",
		"The cumulative number of bytes read from file cache along with read type - Sequential/Random",
		stats.UnitBytes)
	fileCacheReadLatency = stats.Float64("file_cache/read_latency",
		"Latency of read from file cache along with cache hit - true/false",
		stats.UnitMilliseconds)
)

const NanosecondsInOneMillisecond = 1000000

// Initialize the metrics.
func init() {
	// GCS related metrics
	if err := view.Register(
		&view.View{
			Name:        "gcs/read_count",
			Measure:     gcsReadCount,
			Description: "Specifies the number of gcs reads made along with type - Sequential/Random",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tags.ReadType},
		},
		&view.View{
			Name:        "gcs/download_bytes_count",
			Measure:     downloadBytesCount,
			Description: "The cumulative number of bytes downloaded from GCS along with type - Sequential/Random",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tags.ReadType},
		},
		// File cache related metrics
		&view.View{
			Name:        "file_cache/read_count",
			Measure:     fileCacheReadCount,
			Description: "Specifies the number of read requests made via file cache along with type - Sequential/Random and cache hit - true/false",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tags.ReadType, tags.CacheHit},
		},
		&view.View{
			Name:        "file_cache/read_bytes_count",
			Measure:     fileCacheReadBytesCount,
			Description: "The cumulative number of bytes read from file cache along with read type - Sequential/Random",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tags.ReadType},
		},
		&view.View{
			Name:        "file_cache/read_latencies",
			Measure:     fileCacheReadLatency,
			Description: "The cumulative distribution of the file cache read latencies along with cache hit - true/false",
			Aggregation: ochttp.DefaultLatencyDistribution,
			TagKeys:     []tag.Key{tags.CacheHit},
		},
	); err != nil {
		log.Fatalf("Failed to register the reader view: %v", err)
	}
}

func CaptureGCSReadMetrics(ctx context.Context, readType string, requestedDataSize int64) {
	if err := stats.RecordWithTags(
		ctx,
		[]tag.Mutator{
			tag.Upsert(tags.ReadType, readType),
		},
		gcsReadCount.M(1),
	); err != nil {
		// Error in recording gcsReadCount.
		logger.Errorf("Cannot record gcsReadCount %v", err)
	}

	if err := stats.RecordWithTags(
		ctx,
		[]tag.Mutator{
			tag.Upsert(tags.ReadType, readType),
		},
		downloadBytesCount.M(requestedDataSize),
	); err != nil {
		// Error in recording downloadBytesCount.
		logger.Errorf("Cannot record downloadBytesCount %v", err)
	}
}

func CaptureFileCacheMetrics(ctx context.Context, readType string, readDataSize int, cacheHit bool, readLatencyNs int64) {
	if err := stats.RecordWithTags(
		ctx,
		[]tag.Mutator{
			tag.Upsert(tags.ReadType, readType),
			tag.Upsert(tags.CacheHit, strconv.FormatBool(cacheHit)),
		},
		fileCacheReadCount.M(1),
	); err != nil {
		// Error in recording fileCacheReadCount.
		logger.Errorf("Cannot record fileCacheReadCount %v", err)
	}

	if err := stats.RecordWithTags(
		ctx,
		[]tag.Mutator{
			tag.Upsert(tags.ReadType, readType),
		},
		fileCacheReadBytesCount.M(int64(readDataSize)),
	); err != nil {
		// Error in recording fileCacheReadBytesCount.
		logger.Errorf("Cannot record fileCacheReadBytesCount %v", err)
	}

	readLatencyMs := float64(readLatencyNs) / float64(NanosecondsInOneMillisecond)
	if err := stats.RecordWithTags(
		ctx,
		[]tag.Mutator{
			tag.Upsert(tags.CacheHit, strconv.FormatBool(cacheHit)),
		},
		fileCacheReadLatency.M(readLatencyMs),
	); err != nil {
		// Error in recording fileCacheReadLatency.
		logger.Errorf("Cannot record fileCacheReadLatency %v", err)
	}
}
