// Copyright 2023 Google Inc. All Rights Reserved.
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

package storage

import "net/http"

// WithUserAgent returns a ClientOption that sets the User-Agent. This option is incompatible with the WithHTTPClient option.
// As we are using http-client, we will need to add this header via RoundTripper middleware.
// https://pkg.go.dev/google.golang.org/api/option#WithUserAgent
type userAgentRoundTripper struct {
	wrapped   http.RoundTripper
	UserAgent string
}

func (ug *userAgentRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", ug.UserAgent)
	return ug.wrapped.RoundTrip(r)
}
