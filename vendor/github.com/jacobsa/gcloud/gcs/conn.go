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

package gcs

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"github.com/jacobsa/gcloud/httputil"
	"github.com/jacobsa/reqtrace"

	"google.golang.org/api/googleapi"
	storagev1 "google.golang.org/api/storage/v1"
)

// OAuth scopes for GCS. For use with e.g. google.DefaultTokenSource.
const (
	Scope_FullControl = storagev1.DevstorageFullControlScope
	Scope_ReadOnly    = storagev1.DevstorageReadOnlyScope
	Scope_ReadWrite   = storagev1.DevstorageReadWriteScope
)

// Conn represents a connection to GCS, pre-bound with a project ID and
// information required for authorization.
type Conn interface {
	// Return a Bucket object representing the GCS bucket with the given name.
	// Attempt to fail early in the case of bad credentials.
	OpenBucket(
		ctx context.Context,
		name string) (b Bucket, err error)
}

// Configuration accepted by NewConn.
type ConnConfig struct {
	// An oauth2 token source to use for authenticating to GCS.
	//
	// You probably want this one:
	//     http://godoc.org/golang.org/x/oauth2/google#DefaultTokenSource
	TokenSource oauth2.TokenSource

	// The value to set in User-Agent headers for outgoing HTTP requests. If
	// empty, a default will be used.
	UserAgent string

	// The HTTP transport to use for communication with GCS. If not supplied,
	// http.DefaultTransport will be used.
	Transport httputil.CancellableRoundTripper

	// The maximum amount of time to spend sleeping in a retry loop with
	// exponential backoff for failed requests. The default of zero disables
	// automatic retries.
	//
	// If you enable automatic retries, beware of the following:
	//
	//  *  Bucket.CreateObject will buffer the entire object contents in memory,
	//     so your object contents must not be too large to fit.
	//
	//  *  Bucket.NewReader needs to perform an additional round trip to GCS in
	//     order to find the latest object generation if you don't specify a
	//     particular generation.
	//
	//  *  Make sure your operations are idempotent, or that your application can
	//     tolerate it if not.
	//
	MaxBackoffSleep time.Duration

	// Loggers for GCS events, and (much more verbose) HTTP requests and
	// responses. If nil, no logging is performed.
	GCSDebugLogger  *log.Logger
	HTTPDebugLogger *log.Logger
}

// Open a connection to GCS.
func NewConn(cfg *ConnConfig) (c Conn, err error) {
	// Fix the user agent if there is none.
	userAgent := cfg.UserAgent
	if userAgent == "" {
		const defaultUserAgent = "github.com-jacobsa-gloud-gcs"
		userAgent = defaultUserAgent
	}

	// Choose the basic transport.
	transport := cfg.Transport
	if cfg.HTTPDebugLogger != nil {
		transport = http.DefaultTransport.(httputil.CancellableRoundTripper)
	}

	// Enable HTTP debugging if requested.
	if cfg.HTTPDebugLogger != nil {
		transport = httputil.DebuggingRoundTripper(transport, cfg.HTTPDebugLogger)
	}

	// Wrap the HTTP transport in an oauth layer.
	if cfg.TokenSource == nil {
		err = errors.New("You must set TokenSource.")
		return
	}

	transport = &oauth2.Transport{
		Source: cfg.TokenSource,
		Base:   transport,
	}

	// Set up the connection.
	c = &conn{
		client:          &http.Client{Transport: transport},
		userAgent:       userAgent,
		maxBackoffSleep: cfg.MaxBackoffSleep,
		debugLogger:     cfg.GCSDebugLogger,
	}

	return
}

type conn struct {
	client          *http.Client
	userAgent       string
	maxBackoffSleep time.Duration
	debugLogger     *log.Logger
}

func (c *conn) OpenBucket(
	ctx context.Context,
	name string) (b Bucket, err error) {
	b = newBucket(c.client, c.userAgent, name)

	// Enable retry loops if requested.
	if c.maxBackoffSleep > 0 {
		// TODO(jacobsa): Show the retries as distinct spans in the trace.
		b = newRetryBucket(c.maxBackoffSleep, b)
	}

	// Enable tracing if appropriate.
	if reqtrace.Enabled() {
		b = &reqtraceBucket{
			Wrapped: b,
		}
	}

	// Print debug output if requested.
	if c.debugLogger != nil {
		b = newDebugBucket(b, c.debugLogger)
	}

	// Attempt to make an innocuous request to the bucket, snooping for HTTP 403
	// errors that indicate bad credentials. This lets us warn the user early in
	// the latter case, with a more helpful message than just "HTTP 403
	// Forbidden". Similarly for bad bucket names that don't collide with another
	// bucket.
	_, err = b.ListObjects(ctx, &ListObjectsRequest{MaxResults: 1})

	if typed, ok := err.(*googleapi.Error); ok {
		switch typed.Code {
		case http.StatusForbidden:
			err = fmt.Errorf(
				"Bad credentials for bucket %q. Check the bucket name and your "+
					"credentials.",
				b.Name())

			return

		case http.StatusNotFound:
			err = fmt.Errorf("Unknown bucket %q", b.Name())
			return
		}
	}

	// Otherwise, don't interfere.
	err = nil

	return
}
