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

package metadata_test

import (
	"context"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/stretchr/testify/assert"
)

type collectorMockMetrics struct {
	metrics.MetricHandle
	readCounts []recordedReadCount
}

func (m *collectorMockMetrics) MetadataCacheReadCount(inc int64, cacheHit bool, entryStatus metrics.EntryStatus, lookupDetail metrics.LookupDetail) {
	m.readCounts = append(m.readCounts, recordedReadCount{
		inc:          inc,
		cacheHit:     cacheHit,
		entryStatus:  entryStatus,
		lookupDetail: lookupDetail,
	})
}

func TestCacheReadCollector_Empty(t *testing.T) {
	collector := metadata.NewCacheReadCollector()
	mockMetrics := &collectorMockMetrics{MetricHandle: metrics.NewNoopMetrics()}

	collector.Flush(mockMetrics)

	assert.Empty(t, mockMetrics.readCounts)
}

func TestCacheReadCollector_ContextHelpers(t *testing.T) {
	ctx := context.Background()
	assert.Nil(t, metadata.CollectorFromContext(ctx))

	collector := metadata.NewCacheReadCollector()
	ctxWithCollector := metadata.WithCollector(ctx, collector)

	assert.Equal(t, collector, metadata.CollectorFromContext(ctxWithCollector))
}

func TestCacheReadCollector_SingleNotFound(t *testing.T) {
	collector := metadata.NewCacheReadCollector()
	mockMetrics := &collectorMockMetrics{MetricHandle: metrics.NewNoopMetrics()}

	collector.Record(false, "", metrics.LookupDetailNotFoundAttr)
	collector.Flush(mockMetrics)

	assert.Len(t, mockMetrics.readCounts, 1)
	assert.Equal(t, recordedReadCount{
		inc:          1,
		cacheHit:     false,
		entryStatus:  "",
		lookupDetail: metrics.LookupDetailNotFoundAttr,
	}, mockMetrics.readCounts[0])
}

func TestCacheReadCollector_DoubleNotFound(t *testing.T) {
	// Candidate dir + candidate file both not found -> exactly one not_found recorded
	collector := metadata.NewCacheReadCollector()
	mockMetrics := &collectorMockMetrics{MetricHandle: metrics.NewNoopMetrics()}

	collector.Record(false, "", metrics.LookupDetailNotFoundAttr)
	collector.Record(false, "", metrics.LookupDetailNotFoundAttr)
	collector.Flush(mockMetrics)

	assert.Len(t, mockMetrics.readCounts, 1)
	assert.Equal(t, recordedReadCount{
		inc:          1,
		cacheHit:     false,
		entryStatus:  "",
		lookupDetail: metrics.LookupDetailNotFoundAttr,
	}, mockMetrics.readCounts[0])
}

func TestCacheReadCollector_NegativeHitPriorityOverNotFound(t *testing.T) {
	// Warm stat non-existent: dir probe returns negative hit, file probe returns negative hit (or not found)
	collector := metadata.NewCacheReadCollector()
	mockMetrics := &collectorMockMetrics{MetricHandle: metrics.NewNoopMetrics()}

	collector.Record(false, "", metrics.LookupDetailNotFoundAttr)
	collector.Record(true, metrics.EntryStatusNegativeAttr, metrics.LookupDetailFoundAttr)
	collector.Flush(mockMetrics)

	assert.Len(t, mockMetrics.readCounts, 1)
	assert.Equal(t, recordedReadCount{
		inc:          1,
		cacheHit:     true,
		entryStatus:  metrics.EntryStatusNegativeAttr,
		lookupDetail: metrics.LookupDetailFoundAttr,
	}, mockMetrics.readCounts[0])
}

func TestCacheReadCollector_PositiveHitPriorityOverNegativeHit(t *testing.T) {
	// Warm stat existent: dir probe returns negative hit, file probe returns positive hit
	collector := metadata.NewCacheReadCollector()
	mockMetrics := &collectorMockMetrics{MetricHandle: metrics.NewNoopMetrics()}

	collector.Record(true, metrics.EntryStatusNegativeAttr, metrics.LookupDetailFoundAttr)
	collector.Record(true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr)
	collector.Flush(mockMetrics)

	assert.Len(t, mockMetrics.readCounts, 1)
	assert.Equal(t, recordedReadCount{
		inc:          1,
		cacheHit:     true,
		entryStatus:  metrics.EntryStatusPositiveAttr,
		lookupDetail: metrics.LookupDetailFoundAttr,
	}, mockMetrics.readCounts[0])
}

func TestCacheReadCollector_NegativeExpiredPriorityOverNotFound(t *testing.T) {
	// Negative TTL Expired: dir probe returns not found, file probe returns negative expired
	collector := metadata.NewCacheReadCollector()
	mockMetrics := &collectorMockMetrics{MetricHandle: metrics.NewNoopMetrics()}

	collector.Record(false, "", metrics.LookupDetailNotFoundAttr)
	collector.Record(false, metrics.EntryStatusNegativeAttr, metrics.LookupDetailTtlExpiredAttr)
	collector.Flush(mockMetrics)

	assert.Len(t, mockMetrics.readCounts, 1)
	assert.Equal(t, recordedReadCount{
		inc:          1,
		cacheHit:     false,
		entryStatus:  metrics.EntryStatusNegativeAttr,
		lookupDetail: metrics.LookupDetailTtlExpiredAttr,
	}, mockMetrics.readCounts[0])
}

func TestCacheReadCollector_PositiveExpiredPriorityOverNegativeExpired(t *testing.T) {
	// Positive TTL Expired: dir probe returns negative expired (or not found), file probe returns positive expired
	collector := metadata.NewCacheReadCollector()
	mockMetrics := &collectorMockMetrics{MetricHandle: metrics.NewNoopMetrics()}

	collector.Record(false, metrics.EntryStatusNegativeAttr, metrics.LookupDetailTtlExpiredAttr)
	collector.Record(false, metrics.EntryStatusPositiveAttr, metrics.LookupDetailTtlExpiredAttr)
	collector.Flush(mockMetrics)

	assert.Len(t, mockMetrics.readCounts, 1)
	assert.Equal(t, recordedReadCount{
		inc:          1,
		cacheHit:     false,
		entryStatus:  metrics.EntryStatusPositiveAttr,
		lookupDetail: metrics.LookupDetailTtlExpiredAttr,
	}, mockMetrics.readCounts[0])
}

func BenchmarkCacheReadCollector(b *testing.B) {
	mockMetrics := metrics.NewNoopMetrics()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		collector := metadata.NewCacheReadCollector()
		collector.Record(false, "", metrics.LookupDetailNotFoundAttr)
		collector.Record(true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr)
		collector.Flush(mockMetrics)
	}
}

func BenchmarkCacheReadCollector_Parallel(b *testing.B) {
	mockMetrics := metrics.NewNoopMetrics()
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			collector := metadata.NewCacheReadCollector()
			collector.Record(false, "", metrics.LookupDetailNotFoundAttr)
			collector.Record(true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr)
			collector.Flush(mockMetrics)
		}
	})
}
