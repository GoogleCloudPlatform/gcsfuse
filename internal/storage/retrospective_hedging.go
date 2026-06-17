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
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	control "cloud.google.com/go/storage/control/apiv2"
	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googleapis/gax-go/v2"
	"github.com/googleapis/gax-go/v2/callctx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

const (
	// defaultTargetFraction represents the target fraction of requests to hedge.
	// In retrospective hedging, we track a target percentile of the best request attempts
	// instead of all attempts. We target the 99th percentile, which corresponds to
	// a target fraction of 0.01 (1.0 - 0.99 = 0.01).
	defaultTargetFraction = 0.01

	// defaultIncreaseRate is the rate at which the hedging delay increases.
	// It specifies how many consecutive increases are required for the delay to double.
	// A value of 5.0 means the delay doubles after 5 consecutive increases (factor = 2^(1/5)).
	defaultIncreaseRate = 5.0

	// defaultHedgingDelayMultiplier is the scaling factor applied to the tracked delay
	// to determine when to trigger a backup request. We trigger hedging after
	// currentDelay * defaultHedgingDelayMultiplier has elapsed. A multiplier of 2.0
	// balances tail latency reduction with query/request amplification.
	defaultHedgingDelayMultiplier = 2.0

	// defaultMaxTokens is the capacity of the token bucket used for overload protection.
	// It caps the maximum number of concurrent/burst hedged requests we can send.
	defaultMaxTokens = 5.0

	// defaultShortestHedgingDelay is the safety lower bound on the hedging delay.
	// Even if the tracked latency is lower, we won't hedge earlier than this duration
	// to protect the GCS backend from a storm of duplicate requests if latency is very low.
	defaultShortestHedgingDelay = 1 * time.Second

	// defaultMaxDelay is the upper bound on the hedging delay. It prevents the
	// calculated delay from growing infinitely if the backend is permanently slow.
	defaultMaxDelay = 1 * time.Hour

	// defaultTokenLimitFraction is the fraction of a token added to the token bucket
	// for every incoming request. A value of 0.055 means we add 0.055 tokens per request.
	// This ensures that at most 5.5% of total requests can be hedged over time, protecting
	// the GCS backend from overload.
	defaultTokenLimitFraction = 0.055
)

// dynamicDelay tracks the latency and dynamically adjusts the hedging delay.
type dynamicDelay struct {
	mu             sync.Mutex
	value          time.Duration
	minDelay       time.Duration
	maxDelay       time.Duration
	increaseFactor float64
	decreaseFactor float64
}

func newDynamicDelay(targetFraction, increaseRate float64, minDelay, maxDelay time.Duration) *dynamicDelay {
	// Compute increaseFactor such that increaseFactor ^ increaseRate = 2.
	increaseFactor := math.Pow(2.0, 1.0/increaseRate)
	// Compute decreaseFactor such that:
	// increaseFactor^targetFraction * decreaseFactor^(1-targetFraction) = 1.
	decreaseFactor := math.Pow(increaseFactor, -targetFraction/(1.0-targetFraction))

	if minDelay < 1*time.Microsecond {
		minDelay = 1 * time.Microsecond
	}
	if maxDelay < minDelay {
		maxDelay = minDelay
	}

	initialDelay := minDelay * 2
	if initialDelay > maxDelay {
		initialDelay = maxDelay
	}

	return &dynamicDelay{
		value:          initialDelay,
		minDelay:       minDelay,
		maxDelay:       maxDelay,
		increaseFactor: increaseFactor,
		decreaseFactor: decreaseFactor,
	}
}

func (d *dynamicDelay) getValue() time.Duration {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.value
}

func (d *dynamicDelay) increase() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.value = time.Duration(float64(d.value) * d.increaseFactor)
	if d.value > d.maxDelay {
		d.value = d.maxDelay
	}
	logger.Debugf("Retrospective hedging: Increased delay value to %v", d.value)
}

func (d *dynamicDelay) decrease() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.value = time.Duration(float64(d.value) * d.decreaseFactor)
	if d.value < d.minDelay {
		d.value = d.minDelay
	}
	logger.Debugf("Retrospective hedging: Decreased delay value to %v", d.value)
}

// fixedFractionLimit limits events to a fixed fraction of a base value.
// It is implemented as a token bucket that is not based on time.
type fixedFractionLimit struct {
	mu        sync.Mutex
	fraction  float64
	maxTokens float64
	tokens    float64
}

func newFixedFractionLimit(fraction, maxTokens float64) *fixedFractionLimit {
	return &fixedFractionLimit{
		fraction:  fraction,
		maxTokens: maxTokens,
		tokens:    0.0,
	}
}

func (l *fixedFractionLimit) increaseBase() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tokens += l.fraction
	if l.tokens > l.maxTokens {
		l.tokens = l.maxTokens
	}
}

func (l *fixedFractionLimit) tryAcquire() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.tokens < 1.0 {
		return false
	}
	l.tokens -= 1.0
	return true
}

type requestState struct {
	mu                    sync.Mutex
	registered            bool
	state                 string // "running", "success", "failed"
	fastestAttemptIndex   int
	fastestAttemptLatency time.Duration
}

func (s *requestState) onAttemptFinished(attemptIndex int, err error, latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != "running" {
		return
	}
	if err == nil {
		s.state = "success"
		s.fastestAttemptIndex = attemptIndex
		s.fastestAttemptLatency = latency
	} else {
		s.state = "failed"
		s.fastestAttemptIndex = attemptIndex
		s.fastestAttemptLatency = latency
	}
}

type latencyTracker struct {
	latencies [300]atomic.Int64
}

type hedgedClient interface {
	logStats()
}

var (
	globalClientsMu sync.Mutex
	globalClients   []hedgedClient
	shutdownCalled  atomic.Bool
)

func registerHedgedClient(c hedgedClient) {
	globalClientsMu.Lock()
	defer globalClientsMu.Unlock()
	globalClients = append(globalClients, c)
}

// LogHedgingLatencyStats logs latency distribution stats for all hedged clients.
func LogHedgingLatencyStats() {
	if shutdownCalled.CompareAndSwap(false, true) {
		globalClientsMu.Lock()
		defer globalClientsMu.Unlock()
		for _, c := range globalClients {
			c.logStats()
		}
	}
}

type retrospectiveHedgingEngine struct {
	delay           *dynamicDelay
	fixedLimit      *fixedFractionLimit
	multiplier      float64
	latencyTrackers map[string]*latencyTracker
}

func newRetrospectiveHedgingEngine(apiNames []string) *retrospectiveHedgingEngine {
	minDelay := time.Duration(float64(defaultShortestHedgingDelay) / defaultHedgingDelayMultiplier)
	trackers := make(map[string]*latencyTracker)
	for _, name := range apiNames {
		trackers[name] = &latencyTracker{}
	}
	return &retrospectiveHedgingEngine{
		delay:           newDynamicDelay(defaultTargetFraction, defaultIncreaseRate, minDelay, defaultMaxDelay),
		fixedLimit:      newFixedFractionLimit(defaultTokenLimitFraction, defaultMaxTokens),
		multiplier:      defaultHedgingDelayMultiplier,
		latencyTrackers: trackers,
	}
}

func (rhe *retrospectiveHedgingEngine) logStats() {
	for apiName, tracker := range rhe.latencyTrackers {
		var sb strings.Builder
		sb.WriteString(" [")
		for i := 0; i < len(tracker.latencies); i++ {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(strconv.FormatInt(tracker.latencies[i].Load(), 10))
		}
		sb.WriteByte(']')
		logger.Infof("Retrospective hedging latency stats for API %s (final): %s", apiName, sb.String())
	}
}

func (rhe *retrospectiveHedgingEngine) recordLatency(apiName string, duration time.Duration) {
	sec := int(duration / time.Second)
	if sec < 0 {
		sec = 0
	}
	if sec >= 300 {
		sec = 299
	}
	if tracker, ok := rhe.latencyTrackers[apiName]; ok {
		tracker.latencies[sec].Add(1)
	}
}

func (rhe *retrospectiveHedgingEngine) registerLatency(reqState *requestState, currentDelay time.Duration) {
	reqState.mu.Lock()
	defer reqState.mu.Unlock()

	if reqState.registered {
		return
	}
	reqState.registered = true

	if reqState.state == "failed" {
		logger.Debugf("Retrospective hedging: Request failed, skipping latency registration.")
		return
	}

	if reqState.state == "success" {
		if reqState.fastestAttemptLatency <= currentDelay {
			rhe.delay.decrease()
		} else {
			rhe.delay.increase()
		}
		return
	}

	if reqState.state == "running" {
		rhe.delay.increase()
	}
}

func (rhe *retrospectiveHedgingEngine) executeHedgedCall(ctx context.Context, apiName, reqDescription, requestID string, call func(context.Context) (any, error)) (any, error) {
	startTime := time.Now()
	defer func() {
		rhe.recordLatency(apiName, time.Since(startTime))
	}()
	rhe.fixedLimit.increaseBase()

	startTimeVal := time.Now()
	currentDelay := rhe.delay.getValue()
	hedgingDelay := time.Duration(float64(currentDelay) * rhe.multiplier)

	attemptCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	type attemptResult struct {
		val          any
		err          error
		latency      time.Duration
		attemptIndex int
	}

	resultChan := make(chan attemptResult, 2)
	var secondAttemptStart time.Time

	reqState := &requestState{
		state:               "running",
		fastestAttemptIndex: -1,
	}

	// Schedule the latency registration timer.
	// Total registration delay is currentDelay * (multiplier + 1)
	registrationTimer := time.AfterFunc(time.Duration(float64(currentDelay)*(rhe.multiplier+1.0)), func() {
		rhe.registerLatency(reqState, currentDelay)
	})
	defer registrationTimer.Stop()

	// Launch first attempt
	go func() {
		ctxWithHeader := callctx.SetHeaders(attemptCtx, "x-goog-client-request-id", requestID+"-0")
		val, err := call(ctxWithHeader)
		latency := time.Since(startTimeVal)
		resultChan <- attemptResult{val: val, err: err, latency: latency, attemptIndex: 0}
	}()

	hedgeTimer := time.NewTimer(hedgingDelay)
	defer hedgeTimer.Stop()

	attemptsSent := 1
	attemptsFinished := 0

	for {
		select {
		case res := <-resultChan:
			attemptsFinished++

			if res.err == nil {
				reqState.onAttemptFinished(res.attemptIndex, nil, res.latency)
				cancel()
				return res.val, nil
			}

			// Failed with an error. If no other backup attempt is running, fail immediately.
			if attemptsFinished >= attemptsSent {
				reqState.onAttemptFinished(res.attemptIndex, res.err, res.latency)
				cancel()
				return nil, res.err
			}

		case <-hedgeTimer.C:
			if attemptsSent < 2 && rhe.fixedLimit.tryAcquire() {
				attemptsSent++
				logger.Infof("Retrospective hedging: Retrying %s for %q (attempt 2) (request_id=%s) with hedging delay=%v", apiName, reqDescription, requestID, hedgingDelay)
				secondAttemptStart = time.Now()
				go func() {
					ctxWithHeader := callctx.SetHeaders(attemptCtx, "x-goog-client-request-id", requestID+"-1")
					val, err := call(ctxWithHeader)
					latency := time.Since(secondAttemptStart)
					resultChan <- attemptResult{val: val, err: err, latency: latency, attemptIndex: 1}
				}()
			}

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

type hedgedControlClient struct {
	raw StorageControlClient
	*retrospectiveHedgingEngine
}

func withRetrospectiveHedging(raw StorageControlClient) StorageControlClient {
	apis := []string{"GetStorageLayout", "DeleteFolder", "GetFolder", "RenameFolder", "CreateFolder", "PollRenameFolder"}
	c := &hedgedControlClient{
		raw:                        raw,
		retrospectiveHedgingEngine: newRetrospectiveHedgingEngine(apis),
	}
	registerHedgedClient(c)
	return c
}

// StorageControlClient interface implementation.

func (c *hedgedControlClient) GetStorageLayout(ctx context.Context, req *controlpb.GetStorageLayoutRequest, opts ...gax.CallOption) (*controlpb.StorageLayout, error) {
	call := func(attemptCtx context.Context) (any, error) {
		return c.raw.GetStorageLayout(attemptCtx, req, opts...)
	}
	val, err := c.executeHedgedCall(ctx, "GetStorageLayout", req.Name, req.RequestId, call)
	if err != nil {
		return nil, err
	}
	return val.(*controlpb.StorageLayout), nil
}

func (c *hedgedControlClient) DeleteFolder(ctx context.Context, req *controlpb.DeleteFolderRequest, opts ...gax.CallOption) error {
	call := func(attemptCtx context.Context) (any, error) {
		err := c.raw.DeleteFolder(attemptCtx, req, opts...)
		return struct{}{}, err
	}
	_, err := c.executeHedgedCall(ctx, "DeleteFolder", req.Name, req.RequestId, call)
	return err
}

func (c *hedgedControlClient) GetFolder(ctx context.Context, req *controlpb.GetFolderRequest, opts ...gax.CallOption) (*controlpb.Folder, error) {
	call := func(attemptCtx context.Context) (any, error) {
		return c.raw.GetFolder(attemptCtx, req, opts...)
	}
	val, err := c.executeHedgedCall(ctx, "GetFolder", req.Name, req.RequestId, call)
	if err != nil {
		return nil, err
	}
	return val.(*controlpb.Folder), nil
}

func (c *hedgedControlClient) RenameFolder(ctx context.Context, req *controlpb.RenameFolderRequest, opts ...gax.CallOption) (*control.RenameFolderOperation, error) {
	call := func(attemptCtx context.Context) (any, error) {
		return c.raw.RenameFolder(attemptCtx, req, opts...)
	}
	val, err := c.executeHedgedCall(ctx, "RenameFolder", fmt.Sprintf("%q to %q", req.Name, req.DestinationFolderId), req.RequestId, call)
	if err != nil {
		return nil, err
	}
	return val.(*control.RenameFolderOperation), nil
}

func (c *hedgedControlClient) CreateFolder(ctx context.Context, req *controlpb.CreateFolderRequest, opts ...gax.CallOption) (*controlpb.Folder, error) {
	call := func(attemptCtx context.Context) (any, error) {
		return c.raw.CreateFolder(attemptCtx, req, opts...)
	}
	val, err := c.executeHedgedCall(ctx, "CreateFolder", fmt.Sprintf("%q in %q", req.FolderId, req.Parent), req.RequestId, call)
	if err != nil {
		return nil, err
	}
	return val.(*controlpb.Folder), nil
}

func (c *hedgedControlClient) PollRenameFolder(ctx context.Context, op *control.RenameFolderOperation, requestID string) (*controlpb.Folder, error) {
	call := func(attemptCtx context.Context) (any, error) {
		return c.raw.PollRenameFolder(attemptCtx, op, requestID)
	}
	val, err := c.executeHedgedCall(ctx, "PollRenameFolder", op.Name(), requestID, call)
	if err != nil {
		return nil, err
	}
	return val.(*controlpb.Folder), nil
}
