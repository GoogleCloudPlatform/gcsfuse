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

package gcsfake

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/syncutil"
	"github.com/jacobsa/timeutil"
)

// Create an "in-memory GCS" that allows access to buckets of any name, each
// initially with empty contents. The supplied clock will be used for
// generating timestamps.
func NewConn(clock timeutil.Clock) (c gcs.Conn) {
	typed := &conn{
		clock:   clock,
		buckets: make(map[string]gcs.Bucket),
	}

	typed.mu = syncutil.NewInvariantMutex(typed.checkInvariants)

	c = typed
	return
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type conn struct {
	clock timeutil.Clock

	mu syncutil.InvariantMutex

	// INVARIANT: For each k, v: v.Name() == k
	//
	// GUARDED_BY(mu)
	buckets map[string]gcs.Bucket
}

// LOCKS_REQUIRED(c.mu)
func (c *conn) checkInvariants() {
	// INVARIANT: For each k, v: v.Name() == k
	for k, v := range c.buckets {
		if v.Name() != k {
			panic(fmt.Sprintf("Name mismatch: %q vs. %q", v.Name(), k))
		}
	}
}

// LOCKS_EXCLUDED(c.mu)
func (c *conn) OpenBucket(
	ctx context.Context,
	options *gcs.OpenBucketOptions) (b gcs.Bucket, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Do we already know this bucket name?
	b = c.buckets[options.Name]
	if b != nil {
		return
	}

	// Create it.
	b = NewFakeBucket(c.clock, options.Name)
	c.buckets[options.Name] = b

	return
}
