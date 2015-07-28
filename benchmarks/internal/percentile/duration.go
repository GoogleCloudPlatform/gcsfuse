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

package percentile

import (
	"math"
	"time"
)

// An implementation of sort.Interface for durations.
type DurationSlice []time.Duration

func (p DurationSlice) Len() int           { return len(p) }
func (p DurationSlice) Less(i, j int) bool { return p[i] < p[j] }
func (p DurationSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// Compute the pth percentile of vals.
//
// REQUIRES: vals is sorted.
// REQUIRES: len(vals) > 0
// REQUIRES: 0 <= p <= 100
func Duration(
	vals DurationSlice,
	p int) (x time.Duration) {
	// We use the NIST method:
	//
	//     https://en.wikipedia.org/wiki/Percentile#NIST_method
	//
	// Begin by computing the rank.
	N := len(vals)
	rank := (float64(p) / 100) * float64(N+1)
	kFloat, d := math.Modf(rank)
	k := int(kFloat)

	// Handle each case.
	switch {
	case k == 0:
		x = vals[0]
		return

	case k >= N:
		x = vals[N-1]
		return

	case 0 < k && k < N:
		xFloat := float64(vals[k-1]) + d*float64(vals[k]-vals[k-1])
		x = time.Duration(xFloat)
		return

	default:
		panic("Invalid input")
	}
}
