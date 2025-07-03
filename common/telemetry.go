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

package common

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

type ShutdownFn func(ctx context.Context) error

// The default time buckets for latency metrics.
// The unit can however change for different units i.e. for one metric the unit could be microseconds and for another it could be milliseconds.
var defaultLatencyDistribution = metric.WithExplicitBucketBoundaries(1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000)

const (
	ReadTypeSequential = "Sequential"
	ReadTypeRandom     = "Random"
	ReadTypeParallel   = "Parallel"
)

// Pair of CacheHit and ReadType attributes
type CacheHitReadType struct {
	CacheHit string
	ReadType string
}

// Pair of FSOp and ErrorCategory attributes
type FSOpsErrorCategory struct {
	FSOps         string
	ErrorCategory string
}

// JoinShutdownFunc combines the provided shutdown functions into a single function.
func JoinShutdownFunc(shutdownFns ...ShutdownFn) ShutdownFn {
	return func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFns {
			if fn == nil {
				continue
			}
			err = errors.Join(err, fn(ctx))
		}
		return err
	}
}

// MetricAttr represents the attributes associated with a metric.
type MetricAttr struct {
	Key, Value string
}

func (a *MetricAttr) String() string {
	return fmt.Sprintf("Key: %s, Value: %s", a.Key, a.Value)
}

func CaptureGCSReadMetrics(ctx context.Context, metricHandle MetricHandle, readType string, requestedDataSize int64) {
	metricHandle.GcsReadCount(1, readType)
	metricHandle.GcsDownloadBytesCount(requestedDataSize, readType)
}
