// Copyright 2020 Google Inc. All Rights Reserved.
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

package util

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/jacobsa/gcloud/httputil"
)

// NewFlakyTransport return a flaky transport that can have hiccups (service
// unavailable) at a given rate between [0, 1].
// This should be used for tests only.
func NewFlakyTransport(hiccupRate float64) httputil.CancellableRoundTripper {
	rand.Seed(int64(time.Now().Nanosecond()))
	transport := http.DefaultTransport.(httputil.CancellableRoundTripper)
	return &flakyTransport{
		reliable:   transport,
		hiccupRate: hiccupRate,
	}
}

type flakyTransport struct {
	reliable   httputil.CancellableRoundTripper
	hiccupRate float64
}

func (t *flakyTransport) unavailable() bool {
	return rand.Float64() < t.hiccupRate
}

// RoundTrip sends a request.
func (t *flakyTransport) RoundTrip(
	req *http.Request) (resp *http.Response, err error) {
	if t.unavailable() {
		err = fmt.Errorf("Service Unavailable")
		fmt.Println("Hiccup injected")
	} else {
		resp, err = t.reliable.RoundTrip(req)
	}
	return
}

// CancelRequest cancels the request.
func (t *flakyTransport) CancelRequest(req *http.Request) {
	t.reliable.CancelRequest(req)
}
