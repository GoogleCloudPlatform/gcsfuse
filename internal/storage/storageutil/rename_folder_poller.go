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
	"time"

	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googleapis/gax-go/v2"
)

const (
	// DefaultRenamePollInitial is the fixed sleep duration between retry polls during the initial fast phase.
	DefaultRenamePollInitial = 500 * time.Millisecond

	// DefaultRenamePollFastPhase is the duration window during which backoff sleep stays fixed at DefaultRenamePollInitial.
	DefaultRenamePollFastPhase = 30 * time.Second

	// DefaultRenamePollMultiplier is the backoff factor applied to sleep intervals after the fast phase window expires.
	DefaultRenamePollMultiplier = 1.1

	// DefaultRenamePollMax is the maximum backoff sleep duration cap for long-running folder rename operations.
	DefaultRenamePollMax = 30 * time.Second
)

// RenamePollConfig holds configuration parameters for polling long-running operations.
type RenamePollConfig struct {
	// Initial sleep duration between polls during the fast phase window.
	Initial time.Duration

	// FastPhaseWindow is the elapsed time duration during which initial sleep duration remains fixed.
	FastPhaseWindow time.Duration

	// Multiplier is the factor by which the sleep duration increases after FastPhaseWindow expires.
	Multiplier float64

	// Max is the maximum backoff sleep duration cap.
	Max time.Duration
}

// DefaultRenamePollConfig returns the default polling configuration for folder rename operations.
func DefaultRenamePollConfig() RenamePollConfig {
	return RenamePollConfig{
		Initial:         DefaultRenamePollInitial,
		FastPhaseWindow: DefaultRenamePollFastPhase,
		Multiplier:      DefaultRenamePollMultiplier,
		Max:             DefaultRenamePollMax,
	}
}

// RenameFolderOperationPoller defines the interface required to poll a RenameFolder LRO.
type RenameFolderOperationPoller interface {
	Poll(ctx context.Context, opts ...gax.CallOption) (*controlpb.Folder, error)
	Done() bool
}

// PollRenameFolderOperation polls the given RenameFolder LRO using a two-stage adaptive polling strategy.
// It maintains a fixed initial sleep (500ms) for the first 30 seconds to provide low latency for fast/medium renames.
// For long-running operations exceeding 30s, backoff sleep grows by 1.1x per retry up to a 30s max cap to protect Control API quota.
func PollRenameFolderOperation(ctx context.Context, op RenameFolderOperationPoller, cfg RenamePollConfig) (*controlpb.Folder, error) {
	// Poll #0: Immediate status check right after operation creation to handle instantaneous completions.
	folder, err := op.Poll(ctx)
	if err != nil {
		return nil, err
	}
	if op.Done() {
		return folder, nil
	}

	initial := cfg.Initial
	if initial <= 0 {
		initial = DefaultRenamePollInitial
	}
	fastPhaseWindow := cfg.FastPhaseWindow
	if fastPhaseWindow <= 0 {
		fastPhaseWindow = DefaultRenamePollFastPhase
	}
	multiplier := cfg.Multiplier
	if multiplier < 1.0 {
		multiplier = DefaultRenamePollMultiplier
	}
	maxInterval := cfg.Max
	if maxInterval <= 0 {
		maxInterval = DefaultRenamePollMax
	}

	startTime := time.Now()
	interval := initial

	timer := time.NewTimer(interval)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}

		folder, err = op.Poll(ctx)
		if err != nil {
			return nil, err
		}
		if op.Done() {
			return folder, nil
		}

		// Keep sleep interval fixed at initial (500ms) for the first fastPhaseWindow (30s).
		// Afterwards, increase sleep interval smoothly by multiplier (1.1x) up to maxInterval (30s).
		if time.Since(startTime) >= fastPhaseWindow {
			interval = min(maxInterval, time.Duration(float64(interval)*multiplier))
		}
		timer.Reset(interval)
	}
}
