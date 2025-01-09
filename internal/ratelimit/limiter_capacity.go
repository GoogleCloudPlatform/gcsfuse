// Copyright 2023 Google LLC
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
	"fmt"
	"math"
	"time"
)

// Choose a limiter capacity that ensures that the action gated by the
// limiter will be limited to within a few percent of `rateHz * window`
// for any window of the given size.
//
// This is not be possible for all rates and windows. In that case, an error
// will be returned.
func ChooseLimiterCapacity(
	rateHz float64,
	window time.Duration) (capacity uint64, err error) {
	// Check that the input is reasonable.
	if rateHz <= 0 || math.IsInf(rateHz, 0) {
		err = fmt.Errorf("illegal rate: %f", rateHz)
		return
	}

	if window <= 0 {
		err = fmt.Errorf("illegal window: %v", window)
		return
	}

	// We cannot help but allow the rate to exceed the configured maximum by some
	// factor in an arbitrary window, no matter how small we scale the max
	// accumulated credit -- the bucket may be full at the start of the window,
	// be immediately exhausted, then be repeatedly exhausted just before filling
	// throughout the window.
	//
	// For example: let the window W = 10 seconds, and the bandwidth B = 20 MiB/s.
	// Set the max accumulated credit C = W*B/2 = 100 MiB. Then this
	// sequence of events is allowed:
	//
	//  *  T=0:        Allow through 100 MiB.
	//  *  T=4.999999: Allow through nearly 100 MiB.
	//  *  T=9.999999: Allow through nearly 100 MiB.
	//
	// Above we allow through nearly 300 MiB, exceeding the allowed bytes for the
	// window by nearly 50%. Note however that this trend cannot continue into
	// the next window, so this must be a transient spike.
	//
	// In general if we set C <= W*B/N, then we're off by no more than a factor
	// of (N+1)/N within any window of size W.
	//
	// Choose a reasonable N.
	const N = 50 // At most 2% error

	w := float64(window) / float64(time.Second)
	capacityFloat := math.Floor(w * rateHz / N)
	if !(capacityFloat >= 1 && capacityFloat < float64(math.MaxUint64)) {
		err = fmt.Errorf(
			"can't use a token bucket to limit to %f Hz over a window of %v "+
				"(result is a capacity of %f)",
			rateHz,
			window,
			capacityFloat)

		return
	}

	capacity = uint64(capacityFloat)

	return
}
