// Copyright 2023 Google LLC
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
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/monitor/tags"
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
	gcsReadCountOC = stats.Int64("gcs/read_count",
		"Specifies the number of gcs reads made along with type - Sequential/Random",
		UnitDimensionless)
	downloadBytesCountOC = stats.Int64("gcs/download_bytes_count",
		"The cumulative number of bytes downloaded from GCS along with type - Sequential/Random",
		UnitBytes)
	fileCacheReadCountOC = stats.Int64("file_cache/read_count",
		"Specifies the number of read requests made via file cache along with type - Sequential/Random and cache hit - true/false",
		UnitDimensionless)
	fileCacheReadBytesCountOC = stats.Int64("file_cache/read_bytes_count",
		"The cumulative number of bytes read from file cache along with read type - Sequential/Random",
		UnitBytes)
	fileCacheReadLatencyOC = stats.Int64("file_cache/read_latency",
		"Latency of read from file cache along with cache hit - true/false",
		UnitMicroseconds)
)

const NanosecondsInOneMillisecond = 1000000

// Initialize the metrics.
func init() {
	// GCS related metrics
	if err := view.Register(
		&view.View{
			Name:        "gcs/read_count",
			Measure:     gcsReadCountOC,
			Description: "Specifies the number of gcs reads made along with type - Sequential/Random",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tags.ReadType},
		},
		&view.View{
			Name:        "gcs/download_bytes_count",
			Measure:     downloadBytesCountOC,
			Description: "The cumulative number of bytes downloaded from GCS along with type - Sequential/Random",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tags.ReadType},
		},
		// File cache related metrics
		&view.View{
			Name:        "file_cache/read_count",
			Measure:     fileCacheReadCountOC,
			Description: "Specifies the number of read requests made via file cache along with type - Sequential/Random and cache hit - true/false",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tags.ReadType, tags.CacheHit},
		},
		&view.View{
			Name:        "file_cache/read_bytes_count",
			Measure:     fileCacheReadBytesCountOC,
			Description: "The cumulative number of bytes read from file cache along with read type - Sequential/Random",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tags.ReadType},
		},
		&view.View{
			Name:        "file_cache/read_latencies",
			Measure:     fileCacheReadLatencyOC,
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
		gcsReadCountOC.M(1),
	); err != nil {
		// Error in recording gcsReadCountOC.
		logger.Errorf("Cannot record gcsReadCountOC %v", err)
	}

	if err := stats.RecordWithTags(
		ctx,
		[]tag.Mutator{
			tag.Upsert(tags.ReadType, readType),
		},
		downloadBytesCountOC.M(requestedDataSize),
	); err != nil {
		// Error in recording downloadBytesCountOC.
		logger.Errorf("Cannot record downloadBytesCountOC %v", err)
	}
}

func CaptureFileCacheMetrics(ctx context.Context, readType string, readDataSize int, cacheHit bool, readLatency time.Duration) {
	if err := stats.RecordWithTags(
		ctx,
		[]tag.Mutator{
			tag.Upsert(tags.ReadType, readType),
			tag.Upsert(tags.CacheHit, strconv.FormatBool(cacheHit)),
		},
		fileCacheReadCountOC.M(1),
	); err != nil {
		// Error in recording fileCacheReadCountOC.
		logger.Errorf("Cannot record fileCacheReadCountOC %v", err)
	}

	if err := stats.RecordWithTags(
		ctx,
		[]tag.Mutator{
			tag.Upsert(tags.ReadType, readType),
		},
		fileCacheReadBytesCountOC.M(int64(readDataSize)),
	); err != nil {
		// Error in recording fileCacheReadBytesCountOC.
		logger.Errorf("Cannot record fileCacheReadBytesCountOC %v", err)
	}

	if err := stats.RecordWithTags(
		ctx,
		[]tag.Mutator{
			tag.Upsert(tags.CacheHit, strconv.FormatBool(cacheHit)),
		},
		fileCacheReadLatencyOC.M(readLatency.Microseconds()),
	); err != nil {
		// Error in recording fileCacheReadLatencyOC.
		logger.Errorf("Cannot record fileCacheReadLatencyOC %v", err)
	}
}
