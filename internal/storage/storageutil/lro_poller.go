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

package storageutil

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/googleapis/gax-go/v2"
)

const (
	// DefaultLROPollInitial is the fixed sleep duration between retry polls during the initial fast phase.
	DefaultLROPollInitial = 500 * time.Millisecond

	// DefaultLROPollFastPhase is the duration window during which backoff sleep stays fixed at DefaultLROPollInitial.
	DefaultLROPollFastPhase = 30 * time.Second

	// DefaultLROPollMultiplier is the backoff factor applied to sleep intervals after the fast phase window expires.
	DefaultLROPollMultiplier = 1.1

	// DefaultLROPollMax is the maximum backoff sleep duration cap for long-running operations.
	DefaultLROPollMax = 30 * time.Second
)

// LROPollConfig holds configuration parameters for polling long-running operations.
type LROPollConfig struct {
	// Initial sleep duration between polls during the fast phase window.
	Initial time.Duration

	// FastPhaseWindow is the elapsed time duration during which initial sleep duration remains fixed.
	FastPhaseWindow time.Duration

	// Multiplier is the factor by which the sleep duration increases after FastPhaseWindow expires.
	Multiplier float64

	// Max is the maximum backoff sleep duration cap.
	Max time.Duration
}

// DefaultLROPollConfig returns the default polling configuration for long-running operations.
func DefaultLROPollConfig() LROPollConfig {
	return LROPollConfig{
		Initial:         DefaultLROPollInitial,
		FastPhaseWindow: DefaultLROPollFastPhase,
		Multiplier:      DefaultLROPollMultiplier,
		Max:             DefaultLROPollMax,
	}
}

// LROPoller defines the interface required to poll an LRO.
type LROPoller[T any] interface {
	Poll(ctx context.Context, opts ...gax.CallOption) (T, error)
	Done() bool
}

// PollLRO polls the given LRO using a two-stage adaptive polling strategy.
func PollLRO[T any](ctx context.Context, op LROPoller[T], cfg LROPollConfig) (T, error) {
	var zero T

	if cfg.Initial <= 0 {
		return zero, fmt.Errorf("initial sleep duration must be greater than 0")
	}
	if cfg.FastPhaseWindow <= 0 {
		return zero, fmt.Errorf("fast phase window must be greater than 0")
	}
	if cfg.Multiplier < 1.0 {
		return zero, fmt.Errorf("multiplier must be greater than or equal to 1.0")
	}
	if cfg.Max <= 0 {
		return zero, fmt.Errorf("max sleep duration must be greater than 0")
	}

	// Poll #0: Immediate status check right after operation creation to handle instantaneous completions.
	result, err := op.Poll(ctx)
	if err != nil {
		if !ShouldRetryWithoutLogging(err) {
			return zero, err
		}
		// On transient errors (e.g. 429), fall through and let the normal polling loop take over.
	} else if op.Done() {
		return result, nil
	}

	startTime := time.Now()
	backoff := cfg.Initial

	timer := time.NewTimer(backoff)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-timer.C:
		}

		result, err = op.Poll(ctx)
		if err != nil {
			if !ShouldRetryWithoutLogging(err) {
				return zero, err
			}
			// On transient errors (e.g. 429), ignore and proceed to sleep for the next retry interval.
		} else if op.Done() {
			return result, nil
		}

		// Apply exponential backoff after fast phase window.
		if time.Since(startTime) >= cfg.FastPhaseWindow {
			backoff = min(cfg.Max, time.Duration(float64(backoff)*cfg.Multiplier))
		}

		// Add a random jitter (sleeping between 80% and 100% of the backoff) to prevent
		// multiple retries from other operations from hitting the API at the same time.
		pause := backoff - time.Duration(rand.Int63n(int64(backoff/5)+1))
		timer.Reset(pause)
	}
}
