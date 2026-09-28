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

// Outcomes of individual stat-cache reads, OR-ed into CacheReads.reads.
const (
	readHitPositive uint32 = 1 << iota
	readHitNegative
	readMissAbsent
	readMissExpiredPositive
	readMissExpiredNegative
)

// cacheReadsKey is the context key under which a CacheReads is found.
type cacheReadsKey struct{}

// CacheReads collects what one FUSE op does with the stat cache, so that the
// op records a single metadata_cache/read_count event however many reads,
// retries or concurrent lookups it performs.
//
// A CacheReads is itself the context that carries it, so collecting costs one
// small allocation per op and none per read. Recording is atomic, which makes
// concurrent recording safe and the outcome independent of the order of the
// reads.
type CacheReads struct {
	context.Context
	// gcsFetched is whether the op fetched metadata from GCS rather than from
	// the stat cache. It alone decides cache_hit.
	gcsFetched atomic.Bool
	// reads holds the outcomes of all the op's reads. They decide entry_status
	// and lookup_detail.
	reads atomic.Uint32
}

// NewCacheReads returns a context derived from parent that collects the
// stat-cache reads made with it, or with any context derived from it.
func NewCacheReads(parent context.Context) *CacheReads {
	return &CacheReads{Context: parent}
}

// Value returns r for its own key and defers every other key to the parent.
func (r *CacheReads) Value(key any) any {
	if key == (cacheReadsKey{}) {
		return r
	}
	return r.Context.Value(key)
}

// WithCacheReadsFrom returns dst carrying the CacheReads of src, if src has
// one. Use it when an op continues on a context that is not derived from its
// own, so that the op's reads are still collected.
func WithCacheReadsFrom(dst, src context.Context) context.Context {
	if r := cacheReadsFrom(src); r != nil {
		return context.WithValue(dst, cacheReadsKey{}, r)
	}
	return dst
}

// RecordCacheRead records the outcome of one stat-cache read with the
// CacheReads carried by ctx. It is a no-op when ctx carries none. A read that
// misses does not make the op a miss by itself: see RecordGCSFetch.
func RecordCacheRead(ctx context.Context, hit bool, entryStatus metrics.EntryStatus, lookupDetail metrics.LookupDetail) {
	r := cacheReadsFrom(ctx)
	if r == nil {
		return
	}
	switch {
	case hit && entryStatus == metrics.EntryStatusNegativeAttr:
		r.reads.Or(readHitNegative)
	case hit:
		r.reads.Or(readHitPositive)
	case lookupDetail != metrics.LookupDetailTtlExpiredAttr:
		r.reads.Or(readMissAbsent)
	case entryStatus == metrics.EntryStatusPositiveAttr:
		r.reads.Or(readMissExpiredPositive)
	default:
		r.reads.Or(readMissExpiredNegative)
	}
}

// RecordGCSFetch records, with the CacheReads carried by ctx, that the op is
// fetching metadata from GCS rather than from the stat cache. Only such a fetch
// makes the op a miss: a cache-only read that misses costs no GCS call. It is a
// no-op when ctx carries no CacheReads.
func RecordGCSFetch(ctx context.Context) {
	if r := cacheReadsFrom(ctx); r != nil {
		r.gcsFetched.Store(true)
	}
}

// Emit records one metadata_cache/read_count event summarising the op. It
// records nothing if the op never read the stat cache, for instance because
// the cache is disabled, or if the op neither fetched from GCS nor hit the
// stat cache.
//
// cache_hit is false if the op fetched from GCS, and true otherwise.
// entry_status and lookup_detail are consolidated from all the op's reads:
//   - A hit is positive if any read found a positive entry, and negative
//     otherwise.
//   - A miss is ttl_expired if any read found an expired entry, positive
//     before negative, and not_found otherwise. An expired entry is erased
//     when read, so when a cache-only read finds an entry expired, the read
//     that then fetches it from GCS finds no entry.
func (r *CacheReads) Emit(metricHandle metrics.MetricHandle) {
	// Load gcsFetched before reads. A read is recorded before the fetch it
	// leads to, so seeing a fetch guarantees seeing that read, even if a
	// goroutine of the op is still running.
	gcsFetched := r.gcsFetched.Load()
	reads := r.reads.Load()
	if reads == 0 {
		return
	}
	if !gcsFetched {
		switch {
		case reads&readHitPositive != 0:
			metricHandle.MetadataCacheReadCount(1, true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr)
		case reads&readHitNegative != 0:
			metricHandle.MetadataCacheReadCount(1, true, metrics.EntryStatusNegativeAttr, metrics.LookupDetailFoundAttr)
		}
		return
	}
	switch {
	case reads&readMissExpiredPositive != 0:
		metricHandle.MetadataCacheReadCount(1, false, metrics.EntryStatusPositiveAttr, metrics.LookupDetailTtlExpiredAttr)
	case reads&readMissExpiredNegative != 0:
		metricHandle.MetadataCacheReadCount(1, false, metrics.EntryStatusNegativeAttr, metrics.LookupDetailTtlExpiredAttr)
	default:
		metricHandle.MetadataCacheReadCount(1, false, metrics.EntryStatusAttr, metrics.LookupDetailNotFoundAttr)
	}
}

func cacheReadsFrom(ctx context.Context) *CacheReads {
	if ctx == nil {
		return nil
	}
	r, _ := ctx.Value(cacheReadsKey{}).(*CacheReads)
	return r
}
