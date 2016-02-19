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
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
)

// An interface for transports that support the signature of
// http.Transport.CancelRequest.
type CancellableRoundTripper interface {
	http.RoundTripper
	CancelRequest(*http.Request)
}

// Wrap the supplied round tripper in a layer that dumps information about HTTP
// requests. unmodified.
func DebuggingRoundTripper(
	in CancellableRoundTripper,
	logger *log.Logger) (out CancellableRoundTripper) {
	out = &debuggingRoundTripper{
		wrapped: in,
		logger:  logger,
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// Ensure that the supplied request has a correct ContentLength field set.
func fillInContentLength(req *http.Request) (err error) {
	body := req.Body
	if body == nil {
		req.ContentLength = 0
		return
	}

	// Make a copy.
	contents, err := ioutil.ReadAll(req.Body)
	if err != nil {
		err = fmt.Errorf("ReadAll: %v", err)
		return
	}

	req.Body.Close()

	// Fill in the content length and restore the body.
	req.ContentLength = int64(len(contents))
	req.Body = ioutil.NopCloser(bytes.NewReader(contents))

	return
}

////////////////////////////////////////////////////////////////////////
// debuggingRoundTripper
////////////////////////////////////////////////////////////////////////

type debuggingRoundTripper struct {
	wrapped CancellableRoundTripper
	logger  *log.Logger
}

func (t *debuggingRoundTripper) RoundTrip(
	req *http.Request) (resp *http.Response, err error) {
	// Ensure that the request has a ContentLength field, so that it doesn't need
	// to be transmitted with chunked encoding. This improves debugging output.
	err = fillInContentLength(req)
	if err != nil {
		err = fmt.Errorf("fillInContentLength: %v", err)
		return
	}

	// Dump the request.
	{
		var dumped []byte
		dumped, err = httputil.DumpRequestOut(req, true)
		if err != nil {
			err = fmt.Errorf("DumpRequestOut: %v", err)
			return
		}

		t.logger.Printf("========== REQUEST:\n%s", dumped)
	}

	// Execute the request.
	resp, err = t.wrapped.RoundTrip(req)
	if err != nil {
		return
	}

	// Dump the response.
	{
		var dumped []byte
		dumped, err = httputil.DumpResponse(resp, true)
		if err != nil {
			err = fmt.Errorf("DumpResponse: %v", err)
			return
		}

		t.logger.Printf("========== RESPONSE:\n%s", dumped)
		t.logger.Println("====================")
	}

	return
}

func (t *debuggingRoundTripper) CancelRequest(req *http.Request) {
	t.wrapped.CancelRequest(req)
}
