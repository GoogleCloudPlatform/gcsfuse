// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package monitor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockExporter struct {
	metric.Exporter
	exportFunc func(context.Context, *metricdata.ResourceMetrics) error
}

func (m *mockExporter) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	if m.exportFunc != nil {
		return m.exportFunc(ctx, rm)
	}
	return nil
}

func (m *mockExporter) ForceFlush(ctx context.Context) error {
	return nil
}

func (m *mockExporter) Shutdown(ctx context.Context) error {
	return nil
}

func TestPermissionAwareExporter_ExportSuccess(t *testing.T) {
	mock := &mockExporter{}
	exporter := &permissionAwareExporter{Exporter: mock}

	err := exporter.Export(context.Background(), &metricdata.ResourceMetrics{})

	assert.NoError(t, err)
	assert.False(t, exporter.disabled.Load())
}

func TestPermissionAwareExporter_ExportPermissionDenied(t *testing.T) {
	mock := &mockExporter{
		exportFunc: func(ctx context.Context, rm *metricdata.ResourceMetrics) error {
			return status.Error(codes.PermissionDenied, "permission denied")
		},
	}
	exporter := &permissionAwareExporter{Exporter: mock}
	// First call fails and disables
	err := exporter.Export(context.Background(), &metricdata.ResourceMetrics{})
	require.Error(t, err)
	require.Equal(t, codes.PermissionDenied, status.Code(err))
	require.True(t, exporter.disabled.Load())

	// Second call should be skipped (return nil)
	err = exporter.Export(context.Background(), &metricdata.ResourceMetrics{})

	assert.NoError(t, err)
}

func TestPermissionAwareExporter_ExportOtherError(t *testing.T) {
	mock := &mockExporter{
		exportFunc: func(ctx context.Context, rm *metricdata.ResourceMetrics) error {
			return errors.New("some other error")
		},
	}
	exporter := &permissionAwareExporter{Exporter: mock}

	err := exporter.Export(context.Background(), &metricdata.ResourceMetrics{})

	assert.Error(t, err)
	assert.False(t, exporter.disabled.Load())
}
