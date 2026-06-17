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

package storage

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	control "cloud.google.com/go/storage/control/apiv2"
	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeCall struct {
	delay time.Duration
	val   any
	err   error
}

type fakeStorageControlClient struct {
	mu        sync.Mutex
	callCount int
	calls     []*fakeCall
}

func (f *fakeStorageControlClient) getCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.callCount
}

func (f *fakeStorageControlClient) runCall(ctx context.Context) (any, error) {
	f.mu.Lock()
	idx := f.callCount
	f.callCount++
	f.mu.Unlock()

	if idx >= len(f.calls) {
		return nil, errors.New("no mock calls defined")
	}
	c := f.calls[idx]

	if c.delay > 0 {
		select {
		case <-time.After(c.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return c.val, c.err
}

func (f *fakeStorageControlClient) GetStorageLayout(ctx context.Context, req *controlpb.GetStorageLayoutRequest, opts ...gax.CallOption) (*controlpb.StorageLayout, error) {
	val, err := f.runCall(ctx)
	if err != nil {
		return nil, err
	}
	return val.(*controlpb.StorageLayout), nil
}

func (f *fakeStorageControlClient) DeleteFolder(ctx context.Context, req *controlpb.DeleteFolderRequest, opts ...gax.CallOption) error {
	_, err := f.runCall(ctx)
	return err
}

func (f *fakeStorageControlClient) GetFolder(ctx context.Context, req *controlpb.GetFolderRequest, opts ...gax.CallOption) (*controlpb.Folder, error) {
	val, err := f.runCall(ctx)
	if err != nil {
		return nil, err
	}
	return val.(*controlpb.Folder), nil
}

func (f *fakeStorageControlClient) RenameFolder(ctx context.Context, req *controlpb.RenameFolderRequest, opts ...gax.CallOption) (*control.RenameFolderOperation, error) {
	val, err := f.runCall(ctx)
	if err != nil {
		return nil, err
	}
	return val.(*control.RenameFolderOperation), nil
}

func (f *fakeStorageControlClient) CreateFolder(ctx context.Context, req *controlpb.CreateFolderRequest, opts ...gax.CallOption) (*controlpb.Folder, error) {
	val, err := f.runCall(ctx)
	if err != nil {
		return nil, err
	}
	return val.(*controlpb.Folder), nil
}

func (f *fakeStorageControlClient) PollRenameFolder(ctx context.Context, op *control.RenameFolderOperation, requestID string) (*controlpb.Folder, error) {
	val, err := f.runCall(ctx)
	if err != nil {
		return nil, err
	}
	return val.(*controlpb.Folder), nil
}

func TestHedgedCall_FastSuccess(t *testing.T) {
	fakeClient := &fakeStorageControlClient{
		calls: []*fakeCall{
			{delay: 1 * time.Millisecond, val: &controlpb.StorageLayout{}},
		},
	}
	// Make shortest_hedging_delay large enough so hedging doesn't trigger
	hedgedClient := &hedgedControlClient{
		raw: fakeClient,
		retrospectiveHedgingEngine: &retrospectiveHedgingEngine{
			delay:      newDynamicDelay(defaultTargetFraction, defaultIncreaseRate, 50*time.Millisecond, defaultMaxDelay),
			fixedLimit: newFixedFractionLimit(defaultTokenLimitFraction, defaultMaxTokens),
			multiplier: 2.0,
			latencyTrackers: map[string]*latencyTracker{
				"GetStorageLayout": {},
			},
		},
	}

	ctx := context.Background()
	req := &controlpb.GetStorageLayoutRequest{}
	layout, err := hedgedClient.GetStorageLayout(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, layout)
	assert.Equal(t, 1, fakeClient.getCallCount())
}

func TestHedgedCall_HedgesSlowCall(t *testing.T) {
	fakeClient := &fakeStorageControlClient{
		calls: []*fakeCall{
			{delay: 50 * time.Millisecond, val: &controlpb.StorageLayout{}}, // Primary attempt is slow
			{delay: 1 * time.Millisecond, val: &controlpb.StorageLayout{}},  // Hedged attempt is fast
		},
	}
	// Set min_delay to 1ms, hedging delay multiplier to 2x (so hedging triggers at 20ms since initial delay is 10ms)
	hedgedClient := &hedgedControlClient{
		raw: fakeClient,
		retrospectiveHedgingEngine: &retrospectiveHedgingEngine{
			delay:      newDynamicDelay(defaultTargetFraction, defaultIncreaseRate, 1*time.Millisecond, defaultMaxDelay),
			fixedLimit: newFixedFractionLimit(defaultTokenLimitFraction, defaultMaxTokens),
			multiplier: 2.0,
			latencyTrackers: map[string]*latencyTracker{
				"GetStorageLayout": {},
			},
		},
	}
	// Grant tokens to allow hedging
	hedgedClient.fixedLimit.tokens = 5.0

	ctx := context.Background()
	req := &controlpb.GetStorageLayoutRequest{}
	layout, err := hedgedClient.GetStorageLayout(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, layout)
	// Both primary and hedged calls should be executed
	assert.Equal(t, 2, fakeClient.getCallCount())
}

func TestHedgedCall_HedgesSlowCallButTokensExhausted(t *testing.T) {
	fakeClient := &fakeStorageControlClient{
		calls: []*fakeCall{
			{delay: 50 * time.Millisecond, val: &controlpb.StorageLayout{}}, // Primary attempt is slow
		},
	}
	hedgedClient := &hedgedControlClient{
		raw: fakeClient,
		retrospectiveHedgingEngine: &retrospectiveHedgingEngine{
			delay:      newDynamicDelay(defaultTargetFraction, defaultIncreaseRate, 1*time.Millisecond, defaultMaxDelay),
			fixedLimit: newFixedFractionLimit(defaultTokenLimitFraction, defaultMaxTokens),
			multiplier: 2.0,
			latencyTrackers: map[string]*latencyTracker{
				"GetStorageLayout": {},
			},
		},
	}
	// Token bucket starts empty, and executing one call adds defaultTokenLimitFraction (0.055) which is < 1.0.
	// So tryAcquire will fail, and no hedging attempt will be made.

	ctx := context.Background()
	req := &controlpb.GetStorageLayoutRequest{}
	layout, err := hedgedClient.GetStorageLayout(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, layout)
	// Only primary call should be executed
	assert.Equal(t, 1, fakeClient.getCallCount())
}

func TestHedgedCall_RetryableErrorReturnedImmediately(t *testing.T) {
	fakeClient := &fakeStorageControlClient{
		calls: []*fakeCall{
			{delay: 0, err: status.Error(codes.Unavailable, "temporary failure")}, // Primary fails immediately with retryable error
		},
	}
	hedgedClient := &hedgedControlClient{
		raw: fakeClient,
		retrospectiveHedgingEngine: &retrospectiveHedgingEngine{
			delay:      newDynamicDelay(defaultTargetFraction, defaultIncreaseRate, 50*time.Millisecond, defaultMaxDelay),
			fixedLimit: newFixedFractionLimit(defaultTokenLimitFraction, defaultMaxTokens),
			multiplier: 2.0,
			latencyTrackers: map[string]*latencyTracker{
				"GetStorageLayout": {},
			},
		},
	}
	// Grant tokens to allow hedging
	hedgedClient.fixedLimit.tokens = 5.0

	ctx := context.Background()
	req := &controlpb.GetStorageLayoutRequest{}
	_, err := hedgedClient.GetStorageLayout(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
	// Only 1 call should be executed, error returned immediately
	assert.Equal(t, 1, fakeClient.getCallCount())
}

func TestHedgedCall_FastFailureNoLatencyUpdate(t *testing.T) {
	fakeClient := &fakeStorageControlClient{
		calls: []*fakeCall{
			{delay: 0, err: status.Error(codes.InvalidArgument, "invalid arg")}, // Non-retryable error
		},
	}
	hedgedClient := &hedgedControlClient{
		raw: fakeClient,
		retrospectiveHedgingEngine: &retrospectiveHedgingEngine{
			delay:      newDynamicDelay(defaultTargetFraction, defaultIncreaseRate, 5*time.Millisecond, defaultMaxDelay),
			fixedLimit: newFixedFractionLimit(defaultTokenLimitFraction, defaultMaxTokens),
			multiplier: 2.0,
			latencyTrackers: map[string]*latencyTracker{
				"GetStorageLayout": {},
			},
		},
	}

	initialDelay := hedgedClient.delay.getValue()

	ctx := context.Background()
	req := &controlpb.GetStorageLayoutRequest{}
	_, err := hedgedClient.GetStorageLayout(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, 1, fakeClient.getCallCount())

	// Wait to make sure if any timer was scheduled it would have executed
	time.Sleep(20 * time.Millisecond)
	// Delay should NOT have changed because the request failed
	assert.Equal(t, initialDelay, hedgedClient.delay.getValue())
}

func TestHedgedCall_LatencyTracking(t *testing.T) {
	fakeClient := &fakeStorageControlClient{
		calls: []*fakeCall{
			{delay: 10 * time.Millisecond, val: &controlpb.StorageLayout{}},
		},
	}

	client := withRetrospectiveHedging(fakeClient)
	hedgedClient, ok := client.(*hedgedControlClient)
	require.True(t, ok)

	ctx := context.Background()
	req := &controlpb.GetStorageLayoutRequest{}
	_, err := hedgedClient.GetStorageLayout(ctx, req)

	assert.NoError(t, err)
	// Verify that the 0s latency bucket has been incremented to 1
	tracker := hedgedClient.latencyTrackers["GetStorageLayout"]
	assert.Equal(t, int64(1), tracker.latencies[0].Load())
}
