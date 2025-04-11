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

	"github.com/stretchr/testify/mock"
)

type MockMetricHandle struct {
	mock.Mock
}

func (m *MockMetricHandle) GCSReadBytesCount(ctx context.Context, inc int64) {
	m.Called(ctx, inc)
}

func (m *MockMetricHandle) GCSReaderCount(ctx context.Context, inc int64, attrs []MetricAttr) {
	m.Called(ctx, inc, attrs)
}

func (m *MockMetricHandle) GCSRequestCount(ctx context.Context, inc int64, attrs []MetricAttr) {
	m.Called(ctx, inc, attrs)
}

func (m *MockMetricHandle) GCSRequestLatency(ctx context.Context, value float64, attrs []MetricAttr) {
	m.Called(ctx, value, attrs)
}

func (m *MockMetricHandle) GCSReadCount(ctx context.Context, inc int64, attrs []MetricAttr) {
	m.Called(ctx, inc, attrs)
}

func (m *MockMetricHandle) GCSDownloadBytesCount(ctx context.Context, inc int64, attrs []MetricAttr) {
	m.Called(ctx, inc, attrs)
}

func (m *MockMetricHandle) OpsCount(ctx context.Context, inc int64, attrs []MetricAttr) {
	m.Called(ctx, inc, attrs)
}

func (m *MockMetricHandle) OpsLatency(ctx context.Context, value float64, attrs []MetricAttr) {
	m.Called(ctx, value, attrs)
}

func (m *MockMetricHandle) OpsErrorCount(ctx context.Context, inc int64, attrs []MetricAttr) {
	m.Called(ctx, inc, attrs)
}

func (m *MockMetricHandle) FileCacheReadCount(ctx context.Context, inc int64, attrs []MetricAttr) {
	m.Called(ctx, inc, attrs)
}

func (m *MockMetricHandle) FileCacheReadBytesCount(ctx context.Context, inc int64, attrs []MetricAttr) {
	m.Called(ctx, inc, attrs)
}

func (m *MockMetricHandle) FileCacheReadLatency(ctx context.Context, value float64, attrs []MetricAttr) {
	m.Called(ctx, value, attrs)
}
