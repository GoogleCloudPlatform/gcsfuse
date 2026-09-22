// Copyright 2026 Google LLC
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

package inode

import (
	"context"

	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
)

// CacheOutcome records how the metadata cache answered a single child lookup.
//
// It is carried on the context rather than returned from LookUpChild so that
// the emission point can live at the file-system layer, where there is exactly
// one outcome per FUSE operation, without widening the DirInode interface.
// This mirrors the way net/http/httptrace carries per-request observation
// hooks.
type CacheOutcome struct {
	// Recorded reports whether a lookup classified the cache at all. It stays
	// false when LookUpChild returned before probing the cache, for example for
	// conflict-marker names or when the legacy type cache is in use.
	Recorded     bool
	CacheHit     bool
	EntryStatus  metrics.EntryStatus
	LookupDetail metrics.LookupDetail
}

type cacheOutcomeKey struct{}

// WithCacheOutcome returns a context carrying a fresh CacheOutcome, along with
// the outcome itself for the caller to read once the lookup has completed.
func WithCacheOutcome(ctx context.Context) (context.Context, *CacheOutcome) {
	outcome := &CacheOutcome{}
	return context.WithValue(ctx, cacheOutcomeKey{}, outcome), outcome
}

// recordCacheOutcome stores how the cache answered on the outcome installed by
// WithCacheOutcome.
//
// Only the first call has any effect. lookUpOrCreateChildInode may run
// LookUpChild up to three times when it races with a concurrent generation
// bump, and the first result is the one describing the cache state the
// operation actually encountered; later probes observe a cache that the retry
// itself has already warmed.
//
// This is a no-op when the context carries no outcome, which is the case for
// unit tests that drive LookUpChild directly.
//
// Not safe for concurrent use. All call sites are on LookUpChild's own
// goroutine; in particular none are inside the errgroup workers that
// lookUpUnknownType spawns.
func recordCacheOutcome(ctx context.Context, cacheHit bool, entryStatus metrics.EntryStatus, lookupDetail metrics.LookupDetail) {
	outcome, ok := ctx.Value(cacheOutcomeKey{}).(*CacheOutcome)
	if !ok || outcome.Recorded {
		return
	}

	outcome.Recorded = true
	outcome.CacheHit = cacheHit
	outcome.EntryStatus = entryStatus
	outcome.LookupDetail = lookupDetail
}
