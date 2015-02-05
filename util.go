// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

// Adapt the cancellation mechansim used by fuse to context.Context's
// cancellation mechanism. Return a context that is cancelled when the intr
// channel is closed, and a cancellation function that can be used to cancel
// the context under other conditions.
//
// The caller must arrange for the cancellation function to be called
// eventually, e.g. when the request has completed.
func withIntr(parent context.Context, intr fs.Intr) (
	context.Context, context.CancelFunc) {
	// Create a cancellable context.
	ctx, cancel := context.WithCancel(parent)

	// Set up the cancellation function to be called if the interrupt channel is
	// ever closed.
	go func() {
		select {
		case <-intr:
			cancel()
			return

		case <-ctx.Done():
			// Somebody else has cancelled the context; we have nothing further to
			// contribute. Note that this is guaranteed to happen eventually, because
			// we require the caller to eventually call the cancellation function.
			return
		}
	}()

	return ctx, cancel
}
