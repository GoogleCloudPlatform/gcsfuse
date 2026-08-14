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

package metadata

import (
	"context"
	"sync/atomic"

	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
)

// Priority constants for collected cache lookup events.
// Higher numerical value indicates higher priority when aggregating multiple candidate probes.
const (
	priorityNone            uint32 = 0
	priorityNotFound        uint32 = 1 // cache_hit: false, entry_status: "", lookup_detail: "not_found"
	priorityNegativeExpired uint32 = 2 // cache_hit: false, entry_status: "negative", lookup_detail: "ttl_expired"
	priorityPositiveExpired uint32 = 3 // cache_hit: false, entry_status: "positive", lookup_detail: "ttl_expired"
	priorityNegativeFound   uint32 = 4 // cache_hit: true,  entry_status: "negative", lookup_detail: "found"
	priorityPositiveFound   uint32 = 5 // cache_hit: true,  entry_status: "positive", lookup_detail: "found"
)

// CacheReadCollector collects cache read lookup events during a compound operation
// (e.g. child inode lookup which probes multiple candidate names in stat cache)
// and evaluates the events upon Flush to record exactly one aggregated metric.
//
// It is completely lock-free and zero-allocation, tracking the highest-priority
// outcome via an atomic integer.
type CacheReadCollector struct {
	state atomic.Uint32
}

func NewCacheReadCollector() *CacheReadCollector {
	return &CacheReadCollector{}
}

// Record captures a single cache lookup event in the collector.
// It updates the internal state only if the new event has higher priority than previous events.
func (c *CacheReadCollector) Record(hit bool, entryStatus metrics.EntryStatus, lookupDetail metrics.LookupDetail) {
	if c == nil {
		return
	}
	p := computePriority(hit, entryStatus, lookupDetail)
	for {
		current := c.state.Load()
		if p <= current {
			break
		}
		if c.state.CompareAndSwap(current, p) {
			break
		}
	}
}

func computePriority(hit bool, entryStatus metrics.EntryStatus, lookupDetail metrics.LookupDetail) uint32 {
	if hit {
		if entryStatus == metrics.EntryStatusPositiveAttr {
			return priorityPositiveFound
		}
		return priorityNegativeFound
	}
	if lookupDetail == metrics.LookupDetailTtlExpiredAttr {
		if entryStatus == metrics.EntryStatusPositiveAttr {
			return priorityPositiveExpired
		}
		return priorityNegativeExpired
	}
	return priorityNotFound
}

// Flush evaluates the highest-priority recorded lookup event and emits a single
// aggregated metric to the provided MetricHandle:
// 1. Positive Found (highest priority: object exists in cache)
// 2. Negative Found (explicit negative cache hit)
// 3. Positive TTL Expired (positive cache entry was found but expired)
// 4. Negative TTL Expired (negative cache entry was found but expired)
// 5. Not Found (entry was not found in cache)
func (c *CacheReadCollector) Flush(mh metrics.MetricHandle) {
	if c == nil || !metrics.IsMonitoringEnabled(mh) {
		return
	}
	switch c.state.Load() {
	case priorityPositiveFound:
		mh.MetadataCacheReadCount(1, true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr)
	case priorityNegativeFound:
		mh.MetadataCacheReadCount(1, true, metrics.EntryStatusNegativeAttr, metrics.LookupDetailFoundAttr)
	case priorityPositiveExpired:
		mh.MetadataCacheReadCount(1, false, metrics.EntryStatusPositiveAttr, metrics.LookupDetailTtlExpiredAttr)
	case priorityNegativeExpired:
		mh.MetadataCacheReadCount(1, false, metrics.EntryStatusNegativeAttr, metrics.LookupDetailTtlExpiredAttr)
	case priorityNotFound:
		mh.MetadataCacheReadCount(1, false, metrics.EntryStatusAttr, metrics.LookupDetailNotFoundAttr)
	case priorityNone:
		// No events recorded; emit nothing.
	}
}

type collectorContextKey struct{}

// WithCollector returns a child context carrying the given CacheReadCollector.
// If the context already contains a collector, it returns the context as-is.
func WithCollector(ctx context.Context, c *CacheReadCollector) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if existing := CollectorFromContext(ctx); existing != nil {
		return ctx
	}
	if c == nil {
		c = NewCacheReadCollector()
	}
	return context.WithValue(ctx, collectorContextKey{}, c)
}

// CollectorFromContext retrieves the CacheReadCollector from the context, if present.
func CollectorFromContext(ctx context.Context) *CacheReadCollector {
	if ctx == nil {
		return nil
	}
	c, _ := ctx.Value(collectorContextKey{}).(*CacheReadCollector)
	return c
}
