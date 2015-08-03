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

package httputil

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"

	"golang.org/x/net/context"
)

// Wait for the allDone channel to be closed. If the context is cancelled
// before then, cancel the HTTP request via the canceller repeatedly until
// allDone is closed.
//
// We must cancel repeatedly because the http package's design for cancellation
// is flawed: the canceller must naturally race with the transport receiving
// the request (CancelRequest may be called before RoundTrip, and in this case
// the cancellation will be lost). This is especially a problem with wrapper
// transports like the oauth2 one, which may do extra work before calling
// through to http.Transport.
func propagateCancellation(
	ctx context.Context,
	allDone chan struct{},
	c CancellableRoundTripper,
	req *http.Request) {
	// If the context is not cancellable, there is nothing interesting we can do.
	cancelChan := ctx.Done()
	if cancelChan == nil {
		return
	}

	// If the user closes allDone before the context is cancelled, we can return
	// immediately.
	select {
	case <-allDone:
		return

	case <-cancelChan:
	}

	// The context has been cancelled. Repeatedly cancel the HTTP request as
	// described above until the user closes allDone.
	for {
		c.CancelRequest(req)

		select {
		case <-allDone:
			return

		case <-time.After(10 * time.Millisecond):
		}
	}
}

// An io.ReadCloser that closes a channel when it is closed. Used to clean up
// the propagateCancellation helper.
type closeChannelReadCloser struct {
	wrapped io.ReadCloser
	c       chan struct{}
}

func (rc *closeChannelReadCloser) Read(p []byte) (n int, err error) {
	n, err = rc.wrapped.Read(p)
	return
}

func (rc *closeChannelReadCloser) Close() (err error) {
	if rc.c != nil {
		close(rc.c)
		rc.c = nil
	}

	err = rc.wrapped.Close()
	return
}

// Call client.Do with the supplied request, cancelling the request if the
// context is cancelled. Return an error if the client does not support
// cancellation.
func Do(
	ctx context.Context,
	client *http.Client,
	req *http.Request) (resp *http.Response, err error) {
	// Make sure the transport supports cancellation.
	c, ok := client.Transport.(CancellableRoundTripper)
	if !ok {
		err = fmt.Errorf(
			"Transport of type %v doesn't support cancellation",
			reflect.TypeOf(client.Transport))
		return
	}

	// Set up a goroutine that will watch the context for cancellation, and a
	// channel that tells it that it can return.
	allDone := make(chan struct{})
	go propagateCancellation(ctx, allDone, c, req)

	// Call through. On error, clean up the helper function.
	resp, err = client.Do(req)
	if err != nil {
		close(allDone)
		return
	}

	// Otherwise, wrap the body in a layer that will tell the helper function
	// that it can return when it is closed.
	resp.Body = &closeChannelReadCloser{
		wrapped: resp.Body,
		c:       allDone,
	}

	return
}
