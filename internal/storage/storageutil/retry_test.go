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

package storageutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ExponentialBackoffTest struct {
	suite.Suite
}

func TestExponentialBackoffTestSuite(t *testing.T) {
	suite.Run(t, new(ExponentialBackoffTest))
}

func (t *ExponentialBackoffTest) TestNewBackoff() {
	initial := 1 * time.Second
	max := 10 * time.Second
	multiplier := 2.0

	b := NewExponentialBackoff(&ExponentialBackoffConfig{
		Initial:    initial,
		Max:        max,
		Multiplier: multiplier,
	})

	assert.NotNil(t.T(), b)
	assert.Equal(t.T(), initial, b.next)
	assert.Equal(t.T(), initial, b.config.Initial)
	assert.Equal(t.T(), max, b.config.Max)
	assert.Equal(t.T(), multiplier, b.config.Multiplier)
}

func (t *ExponentialBackoffTest) TestNext() {
	initial := 1 * time.Second
	max := 3 * time.Second
	multiplier := 2.0
	b := NewExponentialBackoff(&ExponentialBackoffConfig{
		Initial:    initial,
		Max:        max,
		Multiplier: multiplier,
	})

	// First call to next() should return initial, and update current.
	assert.Equal(t.T(), 1*time.Second, b.nextDuration())

	// Second call.
	assert.Equal(t.T(), 2*time.Second, b.nextDuration())

	// Third call - capped at max.
	assert.Equal(t.T(), 3*time.Second, b.nextDuration())

	// Should stay capped at max.
	assert.Equal(t.T(), 3*time.Second, b.nextDuration())
}

func (t *ExponentialBackoffTest) TestWaitWithJitter_ContextCancelled() {
	initial := 100 * time.Microsecond // A long duration to ensure cancellation happens first.
	max := 5 * initial
	b := NewExponentialBackoff(&ExponentialBackoffConfig{
		Initial:    initial,
		Max:        max,
		Multiplier: 2.0,
	})
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel the context immediately.
	cancel()

	start := time.Now()
	err := b.WaitWithJitter(ctx)
	elapsed := time.Since(start)

	assert.ErrorIs(t.T(), err, context.Canceled)
	// The function should return almost immediately.
	assert.Less(t.T(), elapsed, initial, "waitWithJitter should return quickly when context is cancelled")
}

func (t *ExponentialBackoffTest) TestWaitWithJitter_NoContextCancelled() {
	initial := time.Millisecond // A short duration to ensure it waits. Making it any shorter can cause random failures
	// because context cancel itself takes about a millisecond.
	max := 5 * initial
	b := NewExponentialBackoff(&ExponentialBackoffConfig{
		Initial:    initial,
		Max:        max,
		Multiplier: 2.0,
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	start := time.Now()
	err := b.WaitWithJitter(ctx)
	elapsed := time.Since(start)

	assert.NoError(t.T(), err)
	// The function should wait for a duration close to initial.
	assert.LessOrEqual(t.T(), elapsed, initial*2, "waitWithJitter should not wait excessively long")
}
