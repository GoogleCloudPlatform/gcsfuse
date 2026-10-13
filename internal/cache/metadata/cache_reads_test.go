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
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cacheRead is the outcome of one stat-cache read, and also of one recorded
// metadata_cache/read_count event.
type cacheRead struct {
	hit    bool
	status metrics.EntryStatus
	detail metrics.LookupDetail
}

var (
	positiveHit     = cacheRead{true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr}
	negativeHit     = cacheRead{true, metrics.EntryStatusNegativeAttr, metrics.LookupDetailFoundAttr}
	absent          = cacheRead{false, metrics.EntryStatusAttr, metrics.LookupDetailNotFoundAttr}
	expiredPositive = cacheRead{false, metrics.EntryStatusPositiveAttr, metrics.LookupDetailTtlExpiredAttr}
	expiredNegative = cacheRead{false, metrics.EntryStatusNegativeAttr, metrics.LookupDetailTtlExpiredAttr}
)

// readCountRecorder records the metadata_cache/read_count events it receives
// and ignores every other metric.
type readCountRecorder struct {
	metrics.MetricHandle
	events []cacheRead
}

func newReadCountRecorder() *readCountRecorder {
	return &readCountRecorder{MetricHandle: metrics.NewNoopMetrics()}
}

func (r *readCountRecorder) MetadataCacheReadCount(inc int64, cacheHit bool, entryStatus metrics.EntryStatus, lookupDetail metrics.LookupDetail) {
	for range inc {
		r.events = append(r.events, cacheRead{cacheHit, entryStatus, lookupDetail})
	}
}

func record(ctx context.Context, reads ...cacheRead) {
	for _, read := range reads {
		metadata.RecordCacheRead(ctx, read.hit, read.status, read.detail)
	}
}

func TestCacheReads_Emit(t *testing.T) {
	testCases := []struct {
		name    string
		reads   []cacheRead
		fetched bool // Whether the op fetched from GCS after a read missed.
		want    []cacheRead
	}{
		{name: "no reads records nothing"},
		{name: "positive hit", reads: []cacheRead{positiveHit}, want: []cacheRead{positiveHit}},
		{name: "negative hit", reads: []cacheRead{negativeHit}, want: []cacheRead{negativeHit}},
		{name: "repeated hits are one event", reads: []cacheRead{positiveHit, positiveHit, positiveHit}, want: []cacheRead{positiveHit}},
		{name: "positive hit outranks negative hit", reads: []cacheRead{negativeHit, positiveHit}, want: []cacheRead{positiveHit}},
		{name: "miss answered by a hit without a GCS fetch is a hit", reads: []cacheRead{absent, positiveHit}, want: []cacheRead{positiveHit}},
		{name: "expired entry answered by a hit without a GCS fetch is a hit", reads: []cacheRead{expiredNegative, negativeHit}, want: []cacheRead{negativeHit}},
		{name: "misses without a GCS fetch or a hit record nothing", reads: []cacheRead{absent, expiredPositive}},
		{name: "a GCS fetch without any read records nothing", fetched: true},
		{name: "absent entry fetched from GCS", reads: []cacheRead{absent}, fetched: true, want: []cacheRead{absent}},
		{name: "expired positive entry fetched from GCS", reads: []cacheRead{expiredPositive}, fetched: true, want: []cacheRead{expiredPositive}},
		{name: "expired negative entry fetched from GCS", reads: []cacheRead{expiredNegative}, fetched: true, want: []cacheRead{expiredNegative}},
		{name: "a GCS fetch makes the op a miss despite hits", reads: []cacheRead{positiveHit, absent, negativeHit}, fetched: true, want: []cacheRead{absent}},
		{name: "a GCS fetch alongside only hits is a not_found miss", reads: []cacheRead{negativeHit}, fetched: true, want: []cacheRead{absent}},
		{name: "TTL expiry outranks absent entry", reads: []cacheRead{expiredNegative, absent}, fetched: true, want: []cacheRead{expiredNegative}},
		{name: "positive expiry outranks negative expiry", reads: []cacheRead{expiredNegative, expiredPositive, absent}, fetched: true, want: []cacheRead{expiredPositive}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reversed := slices.Clone(tc.reads)
			slices.Reverse(reversed)
			// The outcome must not depend on the order of the reads, nor on when
			// the fetch happens relative to them.
			for i, reads := range [][]cacheRead{tc.reads, reversed} {
				cacheReads := metadata.NewCacheReads(context.Background())
				recorder := newReadCountRecorder()
				fetchFirst := i == 1
				if tc.fetched && fetchFirst {
					metadata.RecordGCSFetch(cacheReads)
				}
				record(cacheReads, reads...)
				if tc.fetched && !fetchFirst {
					metadata.RecordGCSFetch(cacheReads)
				}

				cacheReads.Emit(recorder)

				assert.Equal(t, tc.want, recorder.events, "reads: %v", reads)
			}
		})
	}
}

func TestRecording_WithoutCacheReadsIsNoOp(t *testing.T) {
	assert.NotPanics(t, func() {
		record(context.Background(), absent, positiveHit)
		metadata.RecordGCSFetch(context.Background())
	})
}

func TestCacheReads_CollectsReadsOfDerivedContexts(t *testing.T) {
	type key struct{}
	cacheReads := metadata.NewCacheReads(context.Background())
	derived, cancel := context.WithCancel(context.WithValue(cacheReads, key{}, "value"))
	defer cancel()
	recorder := newReadCountRecorder()

	record(derived, expiredPositive)
	metadata.RecordGCSFetch(derived)
	cacheReads.Emit(recorder)

	assert.Equal(t, []cacheRead{expiredPositive}, recorder.events)
}

func TestWithCacheReadsFrom_CarriesCacheReadsToUnrelatedContext(t *testing.T) {
	cacheReads := metadata.NewCacheReads(context.Background())
	detached := metadata.WithCacheReadsFrom(context.Background(), cacheReads)
	recorder := newReadCountRecorder()

	record(detached, expiredNegative)
	metadata.RecordGCSFetch(detached)
	cacheReads.Emit(recorder)

	assert.Equal(t, []cacheRead{expiredNegative}, recorder.events)
}

func TestWithCacheReadsFrom_NoCacheReadsReturnsDst(t *testing.T) {
	type key struct{}
	dst := context.WithValue(context.Background(), key{}, "dst")

	got := metadata.WithCacheReadsFrom(dst, context.Background())

	assert.Equal(t, dst, got)
}

func TestCacheReads_BehavesAsParentContext(t *testing.T) {
	type key struct{}
	deadline := time.Now().Add(time.Hour)
	parent, cancel := context.WithDeadline(context.WithValue(context.Background(), key{}, "value"), deadline)

	cacheReads := metadata.NewCacheReads(parent)

	assert.Equal(t, "value", cacheReads.Value(key{}))
	gotDeadline, ok := cacheReads.Deadline()
	assert.True(t, ok)
	assert.Equal(t, deadline, gotDeadline)
	assert.NoError(t, cacheReads.Err())
	cancel()
	<-cacheReads.Done()
	assert.ErrorIs(t, cacheReads.Err(), context.Canceled)
}

func TestCacheReads_ConcurrentReads(t *testing.T) {
	cacheReads := metadata.NewCacheReads(context.Background())
	recorder := newReadCountRecorder()
	var wg sync.WaitGroup
	for i := range 64 {
		wg.Go(func() {
			if i%2 == 0 {
				record(cacheReads, positiveHit)
			} else {
				record(cacheReads, expiredNegative)
				metadata.RecordGCSFetch(cacheReads)
			}
		})
	}
	wg.Wait()

	cacheReads.Emit(recorder)

	assert.Equal(t, []cacheRead{expiredNegative}, recorder.events)
}

// ctxSink makes NewCacheReads escape to the heap, as it does when an op passes
// it down the file system.
var ctxSink context.Context

func TestCacheReads_Allocations(t *testing.T) {
	parent := context.Background()
	cacheReads := metadata.NewCacheReads(parent)
	derived, cancel := context.WithCancel(cacheReads)
	defer cancel()
	noop := metrics.NewNoopMetrics()

	require.Equal(t, 1.0, testing.AllocsPerRun(100, func() { ctxSink = metadata.NewCacheReads(parent) }),
		"collecting an op's reads must cost a single allocation")
	assert.Zero(t, testing.AllocsPerRun(100, func() {
		record(derived, positiveHit, absent, expiredPositive)
		metadata.RecordGCSFetch(derived)
	}), "recording must not allocate")
	assert.Zero(t, testing.AllocsPerRun(100, func() {
		record(parent, positiveHit)
		metadata.RecordGCSFetch(parent)
	}), "recording without a collector must not allocate")
	assert.Zero(t, testing.AllocsPerRun(100, func() { cacheReads.Emit(noop) }),
		"emitting must not allocate")
}

// BenchmarkCacheReads measures the per-op cost: collect, record a cold
// lookup's two missed reads and GCS fetch, and emit.
func BenchmarkCacheReads(b *testing.B) {
	parent := context.Background()
	noop := metrics.NewNoopMetrics()
	b.ReportAllocs()
	for b.Loop() {
		cacheReads := metadata.NewCacheReads(parent)
		ctxSink = cacheReads
		record(cacheReads, absent, absent)
		metadata.RecordGCSFetch(cacheReads)
		cacheReads.Emit(noop)
	}
}
