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
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// MountState tracks whether a gcsfuse mount is currently active.
// It is safe for concurrent use.
type MountState struct {
	mounted atomic.Bool
}

// SetMounted marks the mount as active (true) or inactive (false).
func (m *MountState) SetMounted(v bool) {
	m.mounted.Store(v)
}

// IsMounted reports whether the mount is currently active.
func (m *MountState) IsMounted() bool {
	return m.mounted.Load()
}

// healthzHandler returns a handler for GET /healthz.
// It reports 200 OK when the mount is active, 503 otherwise.
func healthzHandler(state *MountState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !state.IsMounted() {
			http.Error(w, "not mounted", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}
}

// readyzHandler returns a handler for GET /readyz.
// It reports 200 OK when the mount is active AND the fs error rate is below
// threshold. threshold is a fraction in [0.0, 1.0].
func readyzHandler(state *MountState, threshold float64, gatherer prometheus.Gatherer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !state.IsMounted() {
			http.Error(w, "not mounted", http.StatusServiceUnavailable)
			return
		}
		rate, err := fsErrorRate(gatherer)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not compute error rate: %v", err), http.StatusServiceUnavailable)
			return
		}
		if rate > threshold {
			http.Error(w, fmt.Sprintf("error rate %.4f exceeds threshold %.4f", rate, threshold), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}
}

// fsErrorRate computes errorOps/totalOps from the Prometheus default registry.
// Returns 0 when no ops have been recorded yet (mount is healthy by default).
// The metric names fs_ops_count and fs_ops_error_count match what the OTel
// Prometheus exporter emits for the fs/ops_count and fs/ops_error_count metrics
// when WithoutCounterSuffixes() is set.
func fsErrorRate(gatherer prometheus.Gatherer) (float64, error) {
	mfs, err := gatherer.Gather()
	if err != nil {
		return 0, err
	}

	var totalOps, errorOps float64
	for _, mf := range mfs {
		name := mf.GetName()
		// WithoutCounterSuffixes() is set on the OTel Prometheus exporter, so the
		// registry will only ever contain "fs_ops_count" (not "_total"). Matching
		// only the canonical name avoids double-counting if both variants appeared.
		switch name {
		case "fs_ops_count":
			for _, m := range mf.GetMetric() {
				totalOps += counterVal(m)
			}
		case "fs_ops_error_count":
			for _, m := range mf.GetMetric() {
				errorOps += counterVal(m)
			}
		}
	}

	if totalOps == 0 {
		return 0, nil
	}
	return errorOps / totalOps, nil
}

func counterVal(m *dto.Metric) float64 {
	if c := m.GetCounter(); c != nil {
		return c.GetValue()
	}
	return 0
}
