package ratelimit

import (
	"golang.org/x/net/context"
	"golang.org/x/time/rate"
)

// A simple interface for limiting the rate of some event. Unlike TokenBucket,
// does not allow the user control over what time means.
//
// Safe for concurrent access.
type Throttle interface {
	// Return the maximum number of tokens that can be requested in a call to
	// Wait.
	Capacity() (c uint64)

	// Acquire the given number of tokens from the underlying token bucket, then
	// sleep until when it says to wake. If the context is cancelled before then,
	// return early with an error.
	//
	// REQUIRES: tokens <= capacity
	Wait(ctx context.Context, tokens uint64) (err error)
}

type limiter struct {
	*rate.Limiter
}

func NewThrottle(
	rateHz float64,
	capacity int) (t Throttle) {
	typed := &limiter{rate.NewLimiter(rate.Limit(rateHz), capacity)}
	t = typed
	return
}

func (l *limiter) Capacity() (c uint64) {
	return uint64(l.Burst())
}

func (l *limiter) Wait(
	ctx context.Context,
	tokens uint64) (err error) {
	return l.WaitN(ctx, int(tokens))
}
