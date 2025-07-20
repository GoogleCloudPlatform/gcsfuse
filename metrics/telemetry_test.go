// Copyright 2025 Google LLC
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

package metrics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type int64DataPoint struct {
	v    int64
	attr metric.MeasurementOption
}

type fakeMetricHandle struct {
	noopMetrics
	GCSReadBytesCounter     []int64DataPoint
	GCSDownloadBytesCounter []int64DataPoint
}

func (f *fakeMetricHandle) GCSReadCount(ctx context.Context, inc int64, readType string) {
	f.GCSReadBytesCounter = append(f.GCSReadBytesCounter, int64DataPoint{
		v:    inc,
		attr: metric.WithAttributes(attribute.String("read_type", readType)),
	})
}

func (f *fakeMetricHandle) GCSDownloadBytesCount(ctx context.Context, requestedDataSize int64, readType string) {
	f.GCSDownloadBytesCounter = append(f.GCSDownloadBytesCounter, int64DataPoint{
		v:    requestedDataSize,
		attr: metric.WithAttributes(attribute.String("read_type", readType)),
	})
}

func TestCaptureGCSReadMetrics(t *testing.T) {
	t.Parallel()
	metricHandle := fakeMetricHandle{}

	CaptureGCSReadMetrics(context.Background(), &metricHandle, "Sequential", 64)

	require.Len(t, metricHandle.GCSReadBytesCounter, 1)
	require.Len(t, metricHandle.GCSDownloadBytesCounter, 1)
	assert.Equal(t, metricHandle.GCSReadBytesCounter[0], int64DataPoint{
		v:    1,
		attr: metric.WithAttributes(attribute.String("read_type", "Sequential")),
	})
	assert.Equal(t, metricHandle.GCSDownloadBytesCounter[0], int64DataPoint{
		v:    64,
		attr: metric.WithAttributes(attribute.String("read_type", "Sequential")),
	})
}
