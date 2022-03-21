// Copyright 2021 Google Inc. All Rights Reserved.
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

// Package monitor collects metrics and exports them to Stackdriver if enabled.
package monitor

import (
	"fmt"
	"net/http"

	"github.com/jacobsa/gcloud/httputil"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
)

// EnableHTTPMonitoring enables the metrics
//   - "custom.googleapis.com/http_sent_bytes",
//   - "custom.googleapis.com/http_received_bytes",
//   - "custom.googleapis.com/http_roundtrip_latency",
// in the HTTP transport. It returns the transport being monitored.
func EnableHTTPMonitoring(r http.RoundTripper) (httputil.CancellableRoundTripper, error) {
	if err := view.Register(
		&view.View{
			Name:        "http/bytes/sent",
			Measure:     ochttp.ClientSentBytes,
			Aggregation: view.Sum(),
			Description: "Total bytes sent in request body (not including headers), by HTTP method and response status",
		},
		&view.View{
			Name:        "http/bytes/received",
			Measure:     ochttp.ClientReceivedBytes,
			Aggregation: view.Sum(),
			Description: "Total bytes received in response bodies (not including headers but including error responses with bodies), by HTTP method and response status",
		},
	); err != nil {
		return nil, fmt.Errorf("register views: %w", err)
	}
	return &ochttp.Transport{Base: r}, nil
}
