// Copyright 2026 Google LLC
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

package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v4/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetGcpOtelHttpClient_TokenURLTimeout(t *testing.T) {
	for _, scope := range []string{
		"https://www.googleapis.com/auth/monitoring.write",
		"https://www.googleapis.com/auth/logging.write",
	} {
		t.Run(scope, func(t *testing.T) {
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				select {
				case <-r.Context().Done():
				case <-time.After(5 * time.Second):
					_, _ = w.Write([]byte(`{"access_token":"late-token"}`))
				}
			}))
			defer provider.Close()
			var apiRequests atomic.Int32
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				apiRequests.Add(1)
				w.WriteHeader(http.StatusNoContent)
			}))
			defer api.Close()
			client, err := getGcpOtelHttpClient(context.Background(), cfg.GcsAuthConfig{
				TokenUrl:                    provider.URL,
				ExperimentalTokenUrlTimeout: 100 * time.Millisecond,
			}, scope)
			require.NoError(t, err)

			start := time.Now()
			resp, err := client.Get(api.URL)
			if resp != nil {
				defer func() { _ = resp.Body.Close() }()
			}

			assert.ErrorIs(t, err, context.DeadlineExceeded)
			assert.Less(t, time.Since(start), 2*time.Second)
			assert.Zero(t, apiRequests.Load())
		})
	}
}
