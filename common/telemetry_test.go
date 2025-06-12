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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func TestJoinShutdownFunc(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		fns          []ShutdownFn
		expectedErrs []string
	}{
		{
			name:         "normal",
			fns:          []ShutdownFn{func(_ context.Context) error { return nil }},
			expectedErrs: nil,
		},
		{
			name:         "one_err",
			fns:          []ShutdownFn{func(_ context.Context) error { return fmt.Errorf("err") }},
			expectedErrs: []string{"err"},
		},
		{
			name: "two_err",
			fns: []ShutdownFn{
				func(_ context.Context) error { return fmt.Errorf("err1") },
				func(_ context.Context) error { return fmt.Errorf("err2") },
			},
			expectedErrs: []string{"err1", "err2"},
		},
		{
			name: "two_err_one_normal",
			fns: []ShutdownFn{
				func(_ context.Context) error { return fmt.Errorf("err1") },
				func(_ context.Context) error { return nil },
				func(_ context.Context) error { return fmt.Errorf("err2") },
			},
			expectedErrs: []string{"err1", "err2"},
		},
		{
			name: "nil",
			fns: []ShutdownFn{
				func(_ context.Context) error { return fmt.Errorf("err1") },
				nil,
				func(_ context.Context) error { return fmt.Errorf("err2") },
			},
			expectedErrs: []string{"err1", "err2"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := JoinShutdownFunc(tc.fns...)(context.Background())

			if len(tc.expectedErrs) == 0 {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				for _, e := range tc.expectedErrs {
					assert.ErrorContains(t, err, e)
				}
			}
		})
	}
}

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
		attr: metric.WithAttributes(attribute.String(ReadType, readType)),
	})
}

func (f *fakeMetricHandle) GCSDownloadBytesCount(ctx context.Context, requestedDataSize int64, readType string) {
	f.GCSDownloadBytesCounter = append(f.GCSDownloadBytesCounter, int64DataPoint{
		v:    requestedDataSize,
		attr: metric.WithAttributes(attribute.String(ReadType, readType)),
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
