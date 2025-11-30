// Copyright 2015 Google LLC
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

package gcsx

import (
	"context"
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/canned"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/monitor"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/ratelimit"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/caching"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/timeutil"
)

type BucketConfig struct {
	BillingProject                     string
	OnlyDir                            string
	EgressBandwidthLimitBytesPerSecond float64
	OpRateLimitHz                      float64
	StatCacheMaxSizeMB                 uint64
	// Config for TTL of entries for existing file in stat cache
	StatCacheTTL time.Duration
	// Config for TTL of entries for non-existing file in stat cache
	NegativeStatCacheTTL time.Duration
	EnableMonitoring     bool
	LogSeverity          cfg.LogSeverity

	// Files backed by on object of length at least AppendThreshold that have
	// only been appended to (i.e. none of the object's contents have been
	// dirtied) will be written out by "appending" to the object in GCS with this
	// process:
	//
	// 1. Write out a temporary object containing the appended contents whose
	//    name begins with TmpObjectPrefix.
	//
	// 2. Compose the original object and the temporary object on top of the
	//    original object.
	//
	// 3. Delete the temporary object.
	//
	// Note that if the process fails or is interrupted the temporary object will
	// not be cleaned up, so the user must ensure that TmpObjectPrefix is
	// periodically garbage collected.
	AppendThreshold          int64
	ChunkTransferTimeoutSecs int64
	TmpObjectPrefix          string
	// Used in Zonal buckets to determine if objects should be finalized or not.
	FinalizeFileForRapid bool

	// Enable dummy I/O mode for testing purposes. In this mode all reads and writes are
	// simulated and no actual data is transferred to or from Cloud Storage.
	// All the metadata operations like object listing and stats are real.
	DummyIOCfg cfg.DummyIoConfig
}

// BucketManager manages the lifecycle of buckets.
type BucketManager interface {
	SetUpBucket(
		ctx context.Context,
		name string, isMultibucketMount bool, metricHandle metrics.MetricHandle) (b SyncerBucket, err error)

	// Shuts down the bucket manager and its buckets
	ShutDown()
}

type bucketManager struct {
	config          BucketConfig
	storageHandle   storage.StorageHandle
	sharedStatCache *lru.Cache

	// Garbage collector
	gcCtx                 context.Context
	stopGarbageCollecting func()
}

func NewBucketManager(config BucketConfig, storageHandle storage.StorageHandle) BucketManager {
	var c *lru.Cache
	if config.StatCacheMaxSizeMB > 0 {
		c = lru.NewCache(util.MiBsToBytes(config.StatCacheMaxSizeMB))
	}

	bm := &bucketManager{
		config:          config,
		storageHandle:   storageHandle,
		sharedStatCache: c,
	}
	bm.gcCtx, bm.stopGarbageCollecting = context.WithCancel(context.Background())
	return bm
}

func setUpRateLimiting(
	in gcs.Bucket,
	opRateLimitHz float64,
	egressBandwidthLimit float64) (out gcs.Bucket, err error) {
	// If no rate limiting has been requested, just return the bucket.
	if !(opRateLimitHz > 0 || egressBandwidthLimit > 0) {
		out = in
		return
	}

	// Treat a disabled limit as a very large one.
	if !(opRateLimitHz > 0) {
		opRateLimitHz = 1e15
	}

	if !(egressBandwidthLimit > 0) {
		egressBandwidthLimit = 1e15
	}

	// Choose token bucket capacities, targeting only a few percent error in each
	// window of the given size.
	const window = 8 * time.Hour

	opCapacity, err := ratelimit.ChooseLimiterCapacity(
		opRateLimitHz,
		window)

	if err != nil {
		err = fmt.Errorf("choosing operation token bucket capacity: %w", err)
		return
	}

	egressCapacity, err := ratelimit.ChooseLimiterCapacity(
		egressBandwidthLimit,
		window)

	if err != nil {
		err = fmt.Errorf("choosing egress bandwidth token bucket capacity: %w", err)
		return
	}

	// Create the throttles.
	opThrottle := ratelimit.NewThrottle(opRateLimitHz, opCapacity)
	egressThrottle := ratelimit.NewThrottle(egressBandwidthLimit, egressCapacity)

	// And the bucket.
	out = ratelimit.NewThrottledBucket(
		opThrottle,
		egressThrottle,
		in)

	return
}

func (bm *bucketManager) SetUpBucket(
	ctx context.Context,
	name string,
	isMultibucketMount bool,
	metricHandle metrics.MetricHandle,
) (sb SyncerBucket, err error) {
	var b gcs.Bucket
	// Set up the appropriate backing bucket.
	if name == canned.FakeBucketName {
		b = canned.MakeFakeBucket(ctx)
	} else {
		b, err = bm.storageHandle.BucketHandle(ctx, name, bm.config.BillingProject, bm.config.FinalizeFileForRapid)
		if err != nil {
			err = fmt.Errorf("BucketHandle: %w", err)
			return
		}
	}

	if bm.config.DummyIOCfg.Enable {
		logger.Infof("Enabling dummy I/O mode for bucket %q\n", name)
		// Wrap in a dummy I/O bucket, which serves the data without actually going to network (GCS).
		b = storage.NewDummyIOBucket(b, storage.DummyIOBucketParams{
			ReaderLatency: bm.config.DummyIOCfg.ReaderLatency,
			PerMBLatency:  bm.config.DummyIOCfg.PerMbLatency,
		})
	}

	// Enable monitoring.
	b = monitor.NewMonitoringBucket(b, metricHandle)

	if bm.config.LogSeverity == cfg.TraceLogSeverity {
		// Enable gcs logs.
		b = storage.NewDebugBucket(b)
	}

	// Limit to a requested prefix of the bucket, if any.
	if bm.config.OnlyDir != "" {
		b, err = NewPrefixBucket(path.Clean(bm.config.OnlyDir)+"/", b)
		if err != nil {
			err = fmt.Errorf("NewPrefixBucket: %w", err)
			return
		}
	}

	// Enable rate limiting, if requested.
	b, err = setUpRateLimiting(
		b,
		bm.config.OpRateLimitHz,
		bm.config.EgressBandwidthLimitBytesPerSecond)

	if err != nil {
		err = fmt.Errorf("setUpRateLimiting: %w", err)
		return
	}

	// Enable cached StatObject results based on stat cache config.
	// Disabling stat cache with below config also disables negative stat cache.
	if bm.config.StatCacheTTL != 0 && bm.sharedStatCache != nil {
		var statCache metadata.StatCache
		if isMultibucketMount {
			statCache = metadata.NewStatCacheBucketView(bm.sharedStatCache, name)
		} else {
			statCache = metadata.NewStatCacheBucketView(bm.sharedStatCache, "")
		}

		b = caching.NewFastStatBucket(
			bm.config.StatCacheTTL,
			statCache,
			timeutil.RealClock(),
			b,
			bm.config.NegativeStatCacheTTL)
	}

	// Enable content type awareness
	b = NewContentTypeBucket(b)

	// Enable Syncer
	if bm.config.TmpObjectPrefix == "" {
		err = errors.New("you must set TmpObjectPrefix")
		return
	}
	sb = NewSyncerBucket(
		bm.config.AppendThreshold,
		bm.config.ChunkTransferTimeoutSecs,
		bm.config.TmpObjectPrefix,
		b)

	// Fetch bucket type from storage layout api and set bucket type.
	b.BucketType()

	// Check whether this bucket works, giving the user a warning early if there
	// is some problem.
	{
		_, err = b.ListObjects(ctx, &gcs.ListObjectsRequest{MaxResults: 1})
		if err != nil {
			return
		}
	}

	// Periodically garbage collect temporary objects
	go garbageCollect(bm.gcCtx, bm.config.TmpObjectPrefix, sb)

	return
}

func (bm *bucketManager) ShutDown() {
	bm.stopGarbageCollecting()
}
