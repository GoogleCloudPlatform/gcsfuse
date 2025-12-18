// Copyright 2025 Google LLC
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
	"os"
	"runtime"
	"sync"
	"time"

	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

type SpanInfo struct {
	SpanID     string `json:"spanID"`
	Name       string `json:"name"`
	TraceID    string `json:"traceID"`
	ParentID   string `json:"parentID"`
	StackTrace string `json:"stackTrace"`

	StartTime time.Time
}

type OrphanDebugger struct {
	activeSpans sync.Map
}

func NewOrphanDebugger() *OrphanDebugger {
	return &OrphanDebugger{}
}

func (p *OrphanDebugger) OnStart(_ context.Context, s trace.ReadWriteSpan) {
	sCtx := s.SpanContext()

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

func (p *OrphanDebugger) OnEnd(s trace.ReadOnlySpan) {
	spanID := s.SpanContext().SpanID()

	p.activeSpans.Delete(spanID)
}

func (p *OrphanDebugger) FindOrphans(filename string) error {
	orphans := []SpanInfo{}
	p.activeSpans.Range(func(key, value interface{}) bool {
		info := value.(SpanInfo)
		orphans = append(orphans, info)
		return true
	})

	if len(orphans) == 0 {
		logger.Info("✅ SHUTDOWN AUDIT: No orphaned spans detected.\n")
		return nil
	}

	file, err := os.Create(filename)
	if err != nil {
		logger.Errorf("Failed to create orphan report file %s: %w\n", filename, err)
		return nil
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(orphans); err != nil {
		logger.Errorf("Failed to encode orphan data to JSON: %w\n", err)
		return nil
	}

	logger.Errorf("❌ SHUTDOWN AUDIT FAILED: %d orphaned spans found.\n", len(orphans))
	logger.Infof("Details written to: %s\n", filename)
	return nil
}

func (*OrphanDebugger) Shutdown(context.Context) error   { return nil }
func (*OrphanDebugger) ForceFlush(context.Context) error { return nil }
