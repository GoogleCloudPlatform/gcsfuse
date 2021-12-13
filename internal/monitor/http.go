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
	"log"
	"net/http"

	"github.com/jacobsa/gcloud/httputil"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// EnableHTTPMonitoring enables the metrics
//   - "opencensus.io/http/client/sent_bytes",
//   - "opencensus.io/http/client/received_bytes",
//   - "opencensus.io/http/client/roundtrip_latency",
// in the HTTP transport. It returns the transport being monitored.
func EnableHTTPMonitoring(r http.RoundTripper) httputil.CancellableRoundTripper {
	if err := view.Register(
		&view.View{
			Name:        "http_sent_bytes",
			Measure:     ochttp.ClientSentBytes,
			Aggregation: ochttp.DefaultSizeDistribution,
			Description: "Total bytes sent in request body (not including headers), by HTTP method and response status",
			TagKeys:     []tag.Key{ochttp.KeyClientMethod, ochttp.KeyClientStatus},
		},
		&view.View{
			Name:        "http_received_bytes",
			Measure:     ochttp.ClientSentBytes,
			Aggregation: ochttp.DefaultSizeDistribution,
			Description: "Total bytes received in response bodies (not including headers but including error responses with bodies), by HTTP method and response status",
			TagKeys:     []tag.Key{ochttp.KeyClientMethod, ochttp.KeyClientStatus},
		},
		&view.View{
			Name:        "http_roundtrip_latency",
			Measure:     ochttp.ClientRoundtripLatency,
			Aggregation: ochttp.DefaultLatencyDistribution,
			Description: "End-to-end latency, by HTTP method and response status",
			TagKeys:     []tag.Key{ochttp.KeyClientMethod, ochttp.KeyClientStatus},
		},
	); err != nil {
		log.Fatalf("cannot register opencensus views for http: ", err)
	}
	return &ochttp.Transport{
		Base: r,
	}
}
