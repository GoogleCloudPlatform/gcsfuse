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
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/net/context"
)

var (
	gcsMeter       = otel.Meter("gcs")
	fileCacheMeter = otel.Meter("file_cache")
	// When a first read call is made by the user, we either fetch entire file or x number of bytes from GCS based on the request.
	// Now depending on the pagesize multiple read calls will be issued by user to read the entire file. These
	// requests will be served from the downloaded data.
	// This metric captures only the requests made to GCS, not the subsequent page calls.
	gcsReadCount, _       = gcsMeter.Int64Counter("gcs/read_count", metric.WithDescription("Specifies the number of gcs reads made along with type - Sequential/Random"))
	downloadBytesCount, _ = gcsMeter.Int64Counter("gcs/download_bytes_count",
		metric.WithDescription("The cumulative number of bytes downloaded from GCS along with type - Sequential/Random"),
		metric.WithUnit("By"))
	fileCacheReadCount, _ = fileCacheMeter.Int64Counter("file_cache/read_count",
		metric.WithDescription("Specifies the number of read requests made via file cache along with type - Sequential/Random and cache hit - true/false"))
	fileCacheReadBytesCount, _ = fileCacheMeter.Int64Counter("file_cache/read_bytes_count",
		metric.WithDescription("The cumulative number of bytes read from file cache along with read type - Sequential/Random"),
		metric.WithUnit("By"))
	fileCacheReadLatency, _ = fileCacheMeter.Int64Histogram("file_cache/read_latencies",
		metric.WithDescription("Latency of read from file cache along with cache hit - true/false"),
		metric.WithUnit("us"),
		common.DefaultLatencyDistribution)
)

func CaptureGCSReadMetrics(ctx context.Context, readType string, requestedDataSize int64) {
	gcsReadCount.Add(ctx, 1, metric.WithAttributes(attribute.String("read_type", readType)))
	downloadBytesCount.Add(ctx, requestedDataSize, metric.WithAttributes(attribute.String("read_type", readType)))
}

func CaptureFileCacheMetrics(ctx context.Context, readType string, readDataSize int, cacheHit bool, readLatency time.Duration) {
	fileCacheReadCount.Add(ctx, 1, metric.WithAttributes(attribute.String("read_type", readType), attribute.Bool("cache_hit", cacheHit)))
	fileCacheReadBytesCount.Add(ctx, int64(readDataSize), metric.WithAttributes(attribute.String("read_type", readType)))
	fileCacheReadLatency.Record(ctx, readLatency.Microseconds(), metric.WithAttributes(attribute.Bool("cache_hit", cacheHit)))
}
