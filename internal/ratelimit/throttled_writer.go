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

package ratelimit

import (
	"io"

	"context"
)

// Create a writer that limits the bandwidth of writes made to w according to
// the supplied throttler. Writes are assumed to be made under the supplied
// context.
//
// REQUIRES: throttle.Capacity() > 0
func ThrottledWriter(
	ctx context.Context,
	w io.Writer,
	throttle Throttle) io.Writer {
	return &throttledWriter{
		ctx:      ctx,
		wrapped:  w,
		throttle: throttle,
	}
}

type throttledWriter struct {
	ctx      context.Context
	wrapped  io.Writer
	throttle Throttle
}

func (tw *throttledWriter) Write(p []byte) (n int, err error) {
	// Serve the full amount in chunks (unless we hit an early error).
	for len(p) > 0 {
		// We can't serve a write larger than the throttle's capacity.
		chunk := p
		if uint64(len(chunk)) > tw.throttle.Capacity() {
			chunk = p[:tw.throttle.Capacity()]
		}

		// Wait for permission to continue.
		err = tw.throttle.Wait(tw.ctx, uint64(len(chunk)))
		if err != nil {
			return
		}

		var tmp int
		tmp, err = tw.wrapped.Write(chunk)

		n += tmp
		p = p[tmp:]
		if err != nil {
			return
		}
		if tmp < len(chunk) {
			err = io.ErrShortWrite
			return
		}
	}

	return
}
