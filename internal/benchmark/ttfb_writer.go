// Copyright 2025 Google LLC
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

package benchmark

import (
	"io"
	"sync"
	"time"
)

// TTFBWriter wraps an io.Writer and fires a one-shot callback once a threshold
// of bytes has been written through it.  It is used to record time-to-first-
// buffer (TTFB) metrics in both the pull-based (new-reader) and push-based
// (multirange) read paths.
//
// Usage:
//
//	w := &TTFBWriter{
//	    Wrapped:   io.Discard,          // or any drain writer
//	    Start:     time.Now(),
//	    Threshold: 256 * 1024,          // fire after 256 KiB
//	    OnFirst:   func(d time.Duration) { ts.hists.RecordTTFB(d.Microseconds()) },
//	}
//	// ... stream data through w ...
//	w.Finalize()  // fires OnFirst if threshold never reached (small objects)
type TTFBWriter struct {
	// Wrapped is the underlying writer that actually consumes bytes.
	// Typically io.Discard for drain paths where data is intentionally thrown away.
	Wrapped io.Writer

	// Start is the timestamp from which TTFB is measured.
	Start time.Time

	// Threshold is the number of bytes that must flow through before OnFirst fires.
	// 256 KiB matches the standard drain-buffer size used in the pull path.
	Threshold int64

	// OnFirst is called exactly once with the elapsed time since Start, either
	// when Threshold bytes have been received or when Finalize() is called —
	// whichever comes first.
	OnFirst func(time.Duration)

	received int64
	once     sync.Once
}

// Write passes p through to Wrapped and fires OnFirst once Threshold bytes
// have accumulated.
func (w *TTFBWriter) Write(p []byte) (int, error) {
	n, err := w.Wrapped.Write(p)
	if n > 0 {
		w.received += int64(n)
		if w.received >= w.Threshold {
			w.once.Do(func() {
				w.OnFirst(time.Since(w.Start))
			})
		}
	}
	return n, err
}

// Finalize fires OnFirst if it has not already fired (handles objects smaller
// than Threshold).  Must be called after all data has been written.
func (w *TTFBWriter) Finalize() {
	w.once.Do(func() {
		w.OnFirst(time.Since(w.Start))
	})
}
