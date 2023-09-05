// Copyright 2015 Google Inc. All Rights Reserved.
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

	"github.com/googlecloudplatform/gcsfuse/internal/canned"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/internal/monitor"
	"github.com/googlecloudplatform/gcsfuse/internal/ratelimit"
	"github.com/googlecloudplatform/gcsfuse/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/bucket"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/requests"
	"github.com/jacobsa/timeutil"
)

type BucketConfig struct {
	BillingProject                     string
	OnlyDir                            string
	EgressBandwidthLimitBytesPerSecond float64
	OpRateLimitHz                      float64
	StatCacheCapacity                  int
	StatCacheTTL                       time.Duration
	EnableMonitoring                   bool
	DebugGCS                           bool

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
	AppendThreshold int64
	TmpObjectPrefix string
}

// BucketManager manages the lifecycle of buckets.
type BucketManager interface {
	SetUpBucket(
		ctx context.Context,
		name string) (b SyncerBucket, err error)

	// Shuts down the bucket manager and its buckets
	ShutDown()
}

type bucketManager struct {
	config        BucketConfig
	storageHandle storage.StorageHandle

	// Garbage collector
	gcCtx                 context.Context
	stopGarbageCollecting func()
}

func NewBucketManager(config BucketConfig, storageHandle storage.StorageHandle) BucketManager {
	bm := &bucketManager{
		config:        config,
		storageHandle: storageHandle,
	}
	bm.gcCtx, bm.stopGarbageCollecting = context.WithCancel(context.Background())
	return bm
}

func setUpRateLimiting(
	in bucket.Bucket,
	opRateLimitHz float64,
	egressBandwidthLimit float64) (out bucket.Bucket, err error) {
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
		err = fmt.Errorf("Choosing operation token bucket capacity: %w", err)
		return
	}

	egressCapacity, err := ratelimit.ChooseLimiterCapacity(
		egressBandwidthLimit,
		window)

	if err != nil {
		err = fmt.Errorf("Choosing egress bandwidth token bucket capacity: %w", err)
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

// Configure a bucket based on the supplied config.
//
// Special case: if the bucket name is canned.FakeBucketName, set up a fake
// bucket as described in that package.
func (bm *bucketManager) SetUpGcsBucket(name string) (b bucket.Bucket, err error) {
	b = bm.storageHandle.BucketHandle(name, bm.config.BillingProject)

	if bm.config.DebugGCS {
		b = bucket.NewDebugBucket(b, logger.NewDebug("gcs: "))
	}
	return
}

func (bm *bucketManager) SetUpBucket(
	ctx context.Context,
	name string) (sb SyncerBucket, err error) {
	var b bucket.Bucket
	// Set up the appropriate backing bucket.
	if name == canned.FakeBucketName {
		b = canned.MakeFakeBucket(ctx)
	} else {
		b, err = bm.SetUpGcsBucket(name)
		if err != nil {
			err = fmt.Errorf("OpenBucket: %w", err)
			return
		}
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

	// Enable cached StatObject results, if appropriate.
	if bm.config.StatCacheTTL != 0 {
		cacheCapacity := bm.config.StatCacheCapacity
		b = caching.NewFastStatBucket(
			bm.config.StatCacheTTL,
			caching.NewStatCache(cacheCapacity),
			timeutil.RealClock(),
			b)
	}

	// Enable content type awareness
	b = NewContentTypeBucket(b)

	// Enable monitoring
	if bm.config.EnableMonitoring {
		b = monitor.NewMonitoringBucket(b)
	}

	// Enable Syncer
	if bm.config.TmpObjectPrefix == "" {
		err = errors.New("You must set TmpObjectPrefix.")
		return
	}
	sb = NewSyncerBucket(
		bm.config.AppendThreshold,
		bm.config.TmpObjectPrefix,
		b)

	// Check whether this bucket works, giving the user a warning early if there
	// is some problem.
	{
		_, err = b.ListObjects(ctx, &requests.ListObjectsRequest{MaxResults: 1})
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
