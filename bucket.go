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

package main

import (
	"fmt"
	"os"
	"path"
	"time"

	"golang.org/x/net/context"

	"github.com/GoogleCloudPlatform/gcsfuse/internal/canned"
	"github.com/GoogleCloudPlatform/gcsfuse/internal/gcsx"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcscaching"
	"github.com/jacobsa/ratelimit"
	"github.com/jacobsa/timeutil"
)

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

// Configure a bucket based on the supplied flags.
//
// Special case: if the bucket name is canned.FakeBucketName, set up a fake
// bucket as described in that package.
func setUpBucket(
	ctx context.Context,
	flags *flagStorage,
	conn gcs.Conn,
	name string) (b gcs.Bucket, err error) {
	// Set up the appropriate backing bucket.
	if name == canned.FakeBucketName {
		b = canned.MakeFakeBucket(ctx)
	} else {
		b, err = conn.OpenBucket(ctx, &gcs.OpenBucketOptions{Name: name, BillingProject: flags.BillingProject})
		if err != nil {
			err = fmt.Errorf("OpenBucket: %v", err)
			return
		}
	}

	// Limit to a requested prefix of the bucket, if any.
	if flags.OnlyDir != "" {
		b, err = gcsx.NewPrefixBucket(path.Clean(flags.OnlyDir)+"/", b)
		if err != nil {
			err = fmt.Errorf("NewPrefixBucket: %v", err)
			return
		}
	}

	// Enable rate limiting, if requested.
	b, err = setUpRateLimiting(
		b,
		flags.OpRateLimitHz,
		flags.EgressBandwidthLimitBytesPerSecond)

	if err != nil {
		err = fmt.Errorf("setUpRateLimiting: %v", err)
		return
	}

	// Enable cached StatObject results, if appropriate.
	if flags.StatCacheTTL != 0 {
		const cacheCapacity = 4096
		b = gcscaching.NewFastStatBucket(
			flags.StatCacheTTL,
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
