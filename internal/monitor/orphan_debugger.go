// Copyright 2024 Google LLC
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
	"encoding/json"
	"fmt"
	"os" // Required to get file and line number
	"runtime"

	// Required for path shortening
	"sync"
	"time"

	"go.opentelemetry.io/otel/sdk/trace"
)

// SpanInfo holds minimal data for tracking an active span and its source.
type SpanInfo struct {
	SpanID     string `json:"spanID"`
	Name       string `json:"name"`
	TraceID    string `json:"traceID"`
	ParentID   string `json:"parentID"`
	StackTrace string `json:"stackTrace"`

	StartTime time.Time
}

// OrphanDebugger implements the trace.SpanProcessor interface to detect un-ended spans.
type OrphanDebugger struct {
	// We use a concurrent-safe map because OnStart and OnEnd can be called from different goroutines.
	activeSpans sync.Map // map[trace.SpanID]SpanInfo
}

// NewOrphanDebugger creates a new instance of the custom processor.
func NewOrphanDebugger() *OrphanDebugger {
	return &OrphanDebugger{}
}

// OnStart is called synchronously when a new span is started.
func (p *OrphanDebugger) OnStart(_ context.Context, s trace.ReadWriteSpan) {
	sCtx := s.SpanContext()

	// Capture the SpanID as a string immediately for easy searching later
	sid := sCtx.SpanID().String()

	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	stackStr := string(buf[:n])

	info := SpanInfo{
		SpanID:     sid,
		Name:       s.Name(),
		TraceID:    sCtx.TraceID().String(),
		ParentID:   s.Parent().SpanID().String(),
		StackTrace: stackStr,
		StartTime:  time.Now(),
	}
	p.activeSpans.Store(sCtx.SpanID(), info)
}

// OnEnd is called when a span is ended.
func (p *OrphanDebugger) OnEnd(s trace.ReadOnlySpan) {
	spanID := s.SpanContext().SpanID()

	// Remove the span from the map of active spans, confirming it was closed.
	p.activeSpans.Delete(spanID)
}

// FindOrphans iterates through all currently active spans and prints them.
func (p *OrphanDebugger) FindOrphans(filename string) error {
	orphans := []SpanInfo{}
	p.activeSpans.Range(func(key, value interface{}) bool {
		info := value.(SpanInfo)
		orphans = append(orphans, info)
		return true
	})

	if len(orphans) == 0 {
		fmt.Println("✅ SHUTDOWN AUDIT: No orphaned spans detected.")
		// Optionally delete old report file here if one exists
		return nil
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create orphan report file %s: %w", filename, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(orphans); err != nil {
		return fmt.Errorf("failed to encode orphan data to JSON: %w", err)
	}

	fmt.Printf("\n❌ SHUTDOWN AUDIT FAILED: %d orphaned spans found.\n", len(orphans))
	fmt.Printf("   Details written to: %s\n", filename)
	return nil
}

// Required interface methods (do nothing here)
func (*OrphanDebugger) Shutdown(context.Context) error   { return nil }
func (*OrphanDebugger) ForceFlush(context.Context) error { return nil }
