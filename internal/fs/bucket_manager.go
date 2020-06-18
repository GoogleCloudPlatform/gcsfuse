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

package fs

import (
	"fmt"
	"os"
	"path"
	"time"

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/internal/canned"
	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcscaching"
	"github.com/jacobsa/ratelimit"
	"github.com/jacobsa/timeutil"
)

type BucketConfig struct {
	BillingProject                     string
	OnlyDir                            string
	EgressBandwidthLimitBytesPerSecond float64
	OpRateLimitHz                      float64
	StatCacheCapacity                  int
	StatCacheTTL                       time.Duration
}

// BucketManager manages the lifecycle of buckets.
type BucketManager interface {
	// Sets up a gcs bucket by its name
	SetUpBucket(
		ctx context.Context,
		name string) (b gcs.Bucket, err error)
}

type bucketManager struct {
	config BucketConfig
	conn   gcs.Conn
}

func NewBucketManager(config BucketConfig, conn gcs.Conn) BucketManager {
	return &bucketManager{
		config: config,
		conn:   conn,
	}
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

	opCapacity, err := ratelimit.ChooseTokenBucketCapacity(
		opRateLimitHz,
		window)

	if err != nil {
		err = fmt.Errorf("Choosing operation token bucket capacity: %v", err)
		return
	}

	egressCapacity, err := ratelimit.ChooseTokenBucketCapacity(
		egressBandwidthLimit,
		window)

	if err != nil {
		err = fmt.Errorf("Choosing egress bandwidth token bucket capacity: %v", err)
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
func (bm *bucketManager) SetUpBucket(
	ctx context.Context,
	name string) (b gcs.Bucket, err error) {
	// Set up the appropriate backing bucket.
	if name == canned.FakeBucketName {
		b = canned.MakeFakeBucket(ctx)
	} else {
		b, err = bm.conn.OpenBucket(
			ctx,
			&gcs.OpenBucketOptions{
				Name:           name,
				BillingProject: bm.config.BillingProject,
			},
		)
		if err != nil {
			err = fmt.Errorf("OpenBucket: %v", err)
			return
		}
	}

	// Limit to a requested prefix of the bucket, if any.
	if bm.config.OnlyDir != "" {
		b, err = gcsx.NewPrefixBucket(path.Clean(bm.config.OnlyDir)+"/", b)
		if err != nil {
			err = fmt.Errorf("NewPrefixBucket: %v", err)
			return
		}
	}

	// Enable rate limiting, if requested.
	b, err = setUpRateLimiting(
		b,
		bm.config.OpRateLimitHz,
		bm.config.EgressBandwidthLimitBytesPerSecond)

	if err != nil {
		err = fmt.Errorf("setUpRateLimiting: %v", err)
		return
	}

	// Enable cached StatObject results, if appropriate.
	if bm.config.StatCacheTTL != 0 {
		cacheCapacity := bm.config.StatCacheCapacity
		b = gcscaching.NewFastStatBucket(
			bm.config.StatCacheTTL,
			gcscaching.NewStatCache(cacheCapacity),
			timeutil.RealClock(),
			b)
	}

	// Check whether this bucket works, giving the user a warning early if there
	// is some problem.
	{
		_, err := b.ListObjects(ctx, &gcs.ListObjectsRequest{MaxResults: 1})
		if err != nil {
			fmt.Fprintln(os.Stdout, "WARNING, bucket doesn't appear to work: ", err)
		}
	}

	return
}
