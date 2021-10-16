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

package readers

import (
	"crypto/tls"
	"fmt"
	"net/http"
)

const userAgent = "gcsfuse/dev Benchmark (concurrent_read)"

func getTransport(protocol string, connections int) *http.Transport {
	switch protocol {
	case "HTTP/1.1":
		return &http.Transport{
			MaxConnsPerHost: connections,
			// This disables HTTP/2 in the transport.
			TLSNextProto: make(
				map[string]func(string, *tls.Conn) http.RoundTripper,
			),
		}
	case "HTTP/2":
		return http.DefaultTransport.(*http.Transport).Clone()
	default:
		panic(fmt.Errorf("Unsupported protocol: %q", protocol))
	}
}
