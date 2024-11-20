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
)

type ShutdownFn func(ctx context.Context) error

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

type Attr struct {
	Key, Value string
}

func (a *Attr) String() string {
	return fmt.Sprintf("Key: %s, Value: %s", a.Key, a.Value)
}

type GCSMetricHandle interface {
	GCSReadBytesCount(ctx context.Context, inc int64, attrs []Attr)
	GCSReaderCount(ctx context.Context, inc int64, attrs []Attr)
	GCSRequestCount(ctx context.Context, inc int64, attrs []Attr)
	GCSRequestLatency(ctx context.Context, value float64, attrs []Attr)
	GCSReadCount(ctx context.Context, inc int64, attrs []Attr)
	GCSDownloadBytesCount(ctx context.Context, inc int64, attrs []Attr)
}

type OpsMetricHandle interface {
	OpsCount(ctx context.Context, inc int64, attrs []Attr)
	OpsLatency(ctx context.Context, value float64, attrs []Attr)
	OpsErrorCount(ctx context.Context, inc int64, attrs []Attr)
}

type FileCacheMetricHandle interface {
	FileCacheReadCount(ctx context.Context, inc int64, attrs []Attr)
	FileCacheReadBytesCount(ctx context.Context, inc int64, attrs []Attr)
	FileCacheReadLatency(ctx context.Context, value float64, attrs []Attr)
}
type MetricHandle interface {
	GCSMetricHandle
	OpsMetricHandle
	FileCacheMetricHandle
}

func CaptureGCSReadMetrics(ctx context.Context, metricHandle MetricHandle, readType string, requestedDataSize int64) {
	metricHandle.GCSReadCount(ctx, 1, []Attr{{Key: ReadType, Value: readType}})
	metricHandle.GCSDownloadBytesCount(ctx, requestedDataSize, []Attr{{Key: ReadType, Value: readType}})
}
