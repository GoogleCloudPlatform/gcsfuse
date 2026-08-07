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
	// DefaultLROPollMin is the minimum backoff sleep duration for long-running operations.
	DefaultLROPollMin = 50 * time.Millisecond

	// DefaultLROPollMax is the maximum backoff sleep duration cap for long-running operations.
	DefaultLROPollMax = 30 * time.Second

	// DefaultLROPollCapTime is the time at which the delay reaches DefaultLROPollMax.
	DefaultLROPollCapTime = 10 * time.Minute
)

// LROPollConfig holds configuration parameters for polling long-running operations.
type LROPollConfig struct {
	// Min is the minimum backoff sleep duration.
	Min time.Duration

	// Max is the maximum backoff sleep duration cap.
	Max time.Duration

	// CapTime is the elapsed time at which the delay reaches Max.
	CapTime time.Duration
}

// DefaultLROPollConfig returns the default polling configuration for long-running operations.
func DefaultLROPollConfig() LROPollConfig {
	return LROPollConfig{
		Min:     DefaultLROPollMin,
		Max:     DefaultLROPollMax,
		CapTime: DefaultLROPollCapTime,
	}
}

// LROPoller defines the interface required to poll an LRO.
type LROPoller[T any] interface {
	Poll(ctx context.Context, opts ...gax.CallOption) (T, error)
	Done() bool
}

// PollLRO polls the given LRO using a time-based linear delay schedule.
func PollLRO[T any](ctx context.Context, op LROPoller[T], cfg LROPollConfig) (T, error) {
	var zero T

	if cfg.Min < 0 {
		return zero, fmt.Errorf("min sleep duration must be non-negative")
	}
	if cfg.Max <= 0 {
		return zero, fmt.Errorf("max sleep duration must be greater than 0")
	}
	if cfg.Min > cfg.Max {
		return zero, fmt.Errorf("min sleep duration must not exceed max sleep duration")
	}
	if cfg.CapTime <= 0 {
		return zero, fmt.Errorf("cap time must be greater than 0")
	}

	// Poll #0: Immediate status check right after operation creation to handle instantaneous completions.
	result, err := op.Poll(ctx)
	if err != nil {
		if !ShouldRetryWithoutLogging(err) || op.Done() {
			return zero, err
		}
		// On transient errors (e.g. 429), fall through and let the normal polling loop take over.
	} else if op.Done() {
		return result, nil
	}

	startTime := time.Now()

	// slope represents the rate of increase of the delay interval per unit of elapsed time.
	// It is used to calculate a time-based linear delay schedule: delay(t) = slope * elapsed_t,
	// growing the delay from 0 up to Max over the duration of CapTime.
	slope := float64(cfg.Max) / float64(cfg.CapTime)

	for {
		if err := ctx.Err(); err != nil {
			return zero, err
		}

		elapsed := time.Since(startTime)
		ns := min(float64(cfg.Max), max(float64(cfg.Min), slope*float64(elapsed)))
		// Apply ±10% multiplicative jitter (0.9x to 1.1x of the computed delay)
		// to prevent synchronized retries from multiple operations hitting the API at the same time.
		jitterFactor := 0.9 + 0.2*rand.Float64()
		pause := time.Duration(ns * jitterFactor)

		timer := time.NewTimer(pause)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		case <-timer.C:
		}

		result, err = op.Poll(ctx)
		if err != nil {
			if !ShouldRetryWithoutLogging(err) || op.Done() {
				return zero, err
			}
			// On transient errors (e.g. 429), ignore and proceed to sleep for the next retry interval.
		} else if op.Done() {
			return result, nil
		}
	}
}
