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
// context. Writes larger than the throttle's capacity are split into chunks of
// at most the capacity.
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
	capacity := tw.throttle.Capacity()
	for len(p) > 0 {
		// We can't acquire more than the throttle's capacity at once.
		chunk := p
		if uint64(len(chunk)) > capacity {
			chunk = p[:capacity]
		}

		// Wait for permission to continue.
		err = tw.throttle.Wait(tw.ctx, uint64(len(chunk)))
		if err != nil {
			return
		}

		var written int
		written, err = tw.wrapped.Write(chunk)
		n += written
		p = p[written:]
		if err != nil {
			return
		}
		if written < len(chunk) {
			err = io.ErrShortWrite
			return
		}
	}

	return
}

// throttledUploadPayloadReader paces reads from an upload source against the
// write throttle (ingressThrottle). It charges tokens only for bytes actually
// read so that short reads and zero-byte directory markers are not overcharged.
//
// REQUIRES: throttle.Capacity() > 0
func throttledUploadPayloadReader(
	ctx context.Context,
	r io.Reader,
	throttle Throttle) io.Reader {
	return &throttledUploadPayload{
		ctx:      ctx,
		wrapped:  r,
		throttle: throttle,
	}
}

type throttledUploadPayload struct {
	ctx      context.Context
	wrapped  io.Reader
	throttle Throttle
}

func (tr *throttledUploadPayload) Read(p []byte) (n int, err error) {
	// We can't serve a read larger than the throttle's capacity.
	if uint64(len(p)) > tr.throttle.Capacity() {
		p = p[:tr.throttle.Capacity()]
	}

	n, err = tr.wrapped.Read(p)
	if n > 0 {
		if werr := tr.throttle.Wait(tr.ctx, uint64(n)); werr != nil {
			return n, werr
		}
	}

	return
}
