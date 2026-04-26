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

package monitor

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// --- MountState ---

func TestMountState_DefaultIsUnmounted(t *testing.T) {
	var s MountState
	assert.False(t, s.IsMounted())
}

func TestMountState_SetMountedTrue(t *testing.T) {
	var s MountState
	s.SetMounted(true)
	assert.True(t, s.IsMounted())
}

func TestMountState_SetMountedFalse(t *testing.T) {
	var s MountState
	s.SetMounted(true)
	s.SetMounted(false)
	assert.False(t, s.IsMounted())
}

// --- healthzHandler ---

func TestHealthzHandler_NotMounted_Returns503(t *testing.T) {
	var state MountState // mounted = false

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	healthzHandler(&state)(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestHealthzHandler_Mounted_Returns200(t *testing.T) {
	var state MountState
	state.SetMounted(true)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	healthzHandler(&state)(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

// --- readyzHandler ---

func TestReadyzHandler_NotMounted_Returns503(t *testing.T) {
	var state MountState

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	readyzHandler(&state, 0.05, &mockGatherer{})(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestReadyzHandler_GathererError_Returns503(t *testing.T) {
	var state MountState
	state.SetMounted(true)

	g := &mockGatherer{err: errors.New("registry unavailable")}
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	readyzHandler(&state, 0.05, g)(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestReadyzHandler_ErrorRateBelowThreshold_Returns200(t *testing.T) {
	var state MountState
	state.SetMounted(true)

	// 1 error out of 100 ops = 1% < 5% threshold
	g := metricFamiliesGatherer(100, 1)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	readyzHandler(&state, 0.05, g)(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestReadyzHandler_ErrorRateAboveThreshold_Returns503(t *testing.T) {
	var state MountState
	state.SetMounted(true)

	// 10 errors out of 100 ops = 10% > 5% threshold
	g := metricFamiliesGatherer(100, 10)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	readyzHandler(&state, 0.05, g)(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestReadyzHandler_ErrorRateAtThreshold_Returns200(t *testing.T) {
	var state MountState
	state.SetMounted(true)

	// exactly at threshold: 5 errors / 100 ops = 5% == threshold (not exceeded)
	g := metricFamiliesGatherer(100, 5)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	readyzHandler(&state, 0.05, g)(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestReadyzHandler_NoOpsYet_Returns200(t *testing.T) {
	var state MountState
	state.SetMounted(true)

	// no ops recorded yet — rate is 0, always healthy
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	readyzHandler(&state, 0.05, &mockGatherer{})(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- fsErrorRate ---

func TestFsErrorRate_NoMetrics_ReturnsZero(t *testing.T) {
	rate, err := fsErrorRate(&mockGatherer{})
	require.NoError(t, err)
	assert.Equal(t, 0.0, rate)
}

func TestFsErrorRate_GathererError_ReturnsError(t *testing.T) {
	g := &mockGatherer{err: errors.New("gather failed")}
	_, err := fsErrorRate(g)
	assert.Error(t, err)
}

func TestFsErrorRate_OpsCountOnly_ReturnsZero(t *testing.T) {
	g := &mockGatherer{families: []*dto.MetricFamily{
		counterFamily("fs_ops_count", 200),
	}}
	rate, err := fsErrorRate(g)
	require.NoError(t, err)
	assert.Equal(t, 0.0, rate)
}

func TestFsErrorRate_CorrectRatio(t *testing.T) {
	g := &mockGatherer{families: []*dto.MetricFamily{
		counterFamily("fs_ops_count", 200),
		counterFamily("fs_ops_error_count", 20),
	}}
	rate, err := fsErrorRate(g)
	require.NoError(t, err)
	assert.InDelta(t, 0.1, rate, 1e-9)
}

func TestFsErrorRate_IgnoresTotalSuffixVariant(t *testing.T) {
	// WithoutCounterSuffixes() is set on the OTel exporter so the registry only
	// ever emits "fs_ops_count" (no "_total"). If a _total-suffixed family were
	// present it must be ignored to avoid double-counting.
	g := &mockGatherer{families: []*dto.MetricFamily{
		counterFamily("fs_ops_count", 100),
		counterFamily("fs_ops_error_count", 10),
		counterFamily("fs_ops_count_total", 999),
		counterFamily("fs_ops_error_count_total", 999),
	}}
	rate, err := fsErrorRate(g)
	require.NoError(t, err)
	// Only the canonical names are counted: 10/100 = 0.1, not the _total values.
	assert.InDelta(t, 0.1, rate, 1e-9)
}

func TestFsErrorRate_SumsMultipleMetricSeries(t *testing.T) {
	// Each family has two counter metrics (e.g. different label combinations)
	opsFamily := &dto.MetricFamily{
		Name: proto.String("fs_ops_count"),
		Type: dto.MetricType_COUNTER.Enum(),
		Metric: []*dto.Metric{
			{Counter: &dto.Counter{Value: proto.Float64(60)}},
			{Counter: &dto.Counter{Value: proto.Float64(40)}},
		},
	}
	errFamily := &dto.MetricFamily{
		Name: proto.String("fs_ops_error_count"),
		Type: dto.MetricType_COUNTER.Enum(),
		Metric: []*dto.Metric{
			{Counter: &dto.Counter{Value: proto.Float64(6)}},
			{Counter: &dto.Counter{Value: proto.Float64(4)}},
		},
	}
	g := &mockGatherer{families: []*dto.MetricFamily{opsFamily, errFamily}}
	rate, err := fsErrorRate(g)
	require.NoError(t, err)
	// 10 errors / 100 ops = 0.1
	assert.InDelta(t, 0.1, rate, 1e-9)
}

// --- counterVal ---

func TestCounterVal_WithCounter_ReturnsValue(t *testing.T) {
	m := &dto.Metric{Counter: &dto.Counter{Value: proto.Float64(42.5)}}
	assert.Equal(t, 42.5, counterVal(m))
}

func TestCounterVal_WithGauge_ReturnsZero(t *testing.T) {
	m := &dto.Metric{Gauge: &dto.Gauge{Value: proto.Float64(99)}}
	assert.Equal(t, 0.0, counterVal(m))
}

func TestCounterVal_NilMetric_ReturnsZero(t *testing.T) {
	assert.Equal(t, 0.0, counterVal(&dto.Metric{}))
}

// --- helpers ---

type mockGatherer struct {
	families []*dto.MetricFamily
	err      error
}

func (m *mockGatherer) Gather() ([]*dto.MetricFamily, error) {
	return m.families, m.err
}

// counterFamily creates a MetricFamily with a single counter metric.
func counterFamily(name string, value float64) *dto.MetricFamily {
	return &dto.MetricFamily{
		Name: proto.String(name),
		Type: dto.MetricType_COUNTER.Enum(),
		Metric: []*dto.Metric{
			{Counter: &dto.Counter{Value: proto.Float64(value)}},
		},
	}
}

// metricFamiliesGatherer returns a gatherer with the given total/error op counts.
func metricFamiliesGatherer(totalOps, errorOps float64) *mockGatherer {
	return &mockGatherer{families: []*dto.MetricFamily{
		counterFamily("fs_ops_count", totalOps),
		counterFamily("fs_ops_error_count", errorOps),
	}}
}
