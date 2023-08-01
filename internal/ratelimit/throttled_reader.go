// Copyright 2023 Google Inc. All Rights Reserved.
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

package ratelimit

import (
	"io"

	"golang.org/x/net/context"
	"golang.org/x/time/rate"
)

// Create a reader that limits the bandwidth of reads made from r according to
// the supplied throttler. Reads are assumed to be made under the supplied
// context.
func ThrottledReader(
	ctx context.Context,
	r io.Reader,
	throttle *rate.Limiter) io.Reader {
	return &throttledReader{
		ctx:      ctx,
		wrapped:  r,
		throttle: throttle,
	}
}

type throttledReader struct {
	ctx      context.Context
	wrapped  io.Reader
	throttle *rate.Limiter
}

func (tr *throttledReader) Read(p []byte) (n int, err error) {
	// We can't serve a read larger than the throttle's capacity.
	if len(p) > tr.throttle.Burst() {
		p = p[:tr.throttle.Burst()]
	}

	// Wait for permission to continue.
	err = tr.throttle.WaitN(tr.ctx, len(p))
	if err != nil {
		return
	}

	// Serve the full amount we acquired from the throttle (unless we hit an
	// early error, including EOF).
	for len(p) > 0 && err == nil {
		var tmp int
		tmp, err = tr.wrapped.Read(p)

		n += tmp
		p = p[tmp:]
	}

	return
}
