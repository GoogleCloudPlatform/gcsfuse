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

package tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestNewOTELTracer(t *testing.T) {
	tracer := NewOTELTracer()

	assert.NotNil(t, tracer)
	tracerImpl, ok := tracer.(*otelTracer)
	assert.True(t, ok)
	assert.NotNil(t, tracerImpl.tracer)
	assert.NotNil(t, tracerImpl.slicePool)
}

func TestOtelTracer_StartEndSpan(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otel.SetTracerProvider(provider)
	tracer := NewOTELTracer()
	spanName := "test-span"

	ctx, span := tracer.StartSpan(context.Background(), spanName)
	tracer.EndSpan(span)

	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
	spans := recorder.Ended()
	assert.Len(t, spans, 1)
	assert.Equal(t, spanName, spans[0].Name())
}

func TestOtelTracer_StartServerSpan(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otel.SetTracerProvider(provider)
	tracer := NewOTELTracer()
	spanName := "test-server-span"

	ctx, span := tracer.StartServerSpan(context.Background(), spanName)
	tracer.EndSpan(span)

	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
	spans := recorder.Ended()
	assert.Len(t, spans, 1)
	assert.Equal(t, spanName, spans[0].Name())
	assert.Equal(t, trace.SpanKindServer, spans[0].SpanKind())
}

func TestOtelTracer_RecordError(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otel.SetTracerProvider(provider)
	tracer := NewOTELTracer()
	spanName := "test-error-span"
	errMsg := "test-error"
	err := errors.New(errMsg)

	_, span := tracer.StartSpan(context.Background(), spanName)
	tracer.RecordError(span, err)
	tracer.EndSpan(span)

	spans := recorder.Ended()
	assert.Len(t, spans, 1)
	assert.Len(t, spans[0].Events(), 1)
	assert.Equal(t, "exception", spans[0].Events()[0].Name)
	assert.Contains(t, spans[0].Events()[0].Attributes, attribute.String("exception.message", errMsg))
	assert.Equal(t, codes.Error, spans[0].Status().Code)
	assert.Equal(t, errMsg, spans[0].Status().Description)
}

func TestOtelTracer_SetCacheReadAttributes(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otel.SetTracerProvider(provider)
	tracer := NewOTELTracer()
	spanName := "test-cache-read-span"
	bytesRead := 123

	t.Run("cache_hit", func(t *testing.T) {
		recorder.Reset()

		_, span := tracer.StartSpan(context.Background(), spanName)
		tracer.SetCacheReadAttributes(span, true, bytesRead)
		tracer.EndSpan(span)

		spans := recorder.Ended()
		assert.Len(t, spans, 1)
		assert.Len(t, spans[0].Attributes(), 2)
		assert.Contains(t, spans[0].Attributes(), attribute.Int(BYTES_READ, bytesRead))
		assert.Contains(t, spans[0].Attributes(), attribute.Bool(IS_CACHE_HIT, true))
	})

	t.Run("cache_miss", func(t *testing.T) {
		recorder.Reset()

		_, span := tracer.StartSpan(context.Background(), spanName)
		tracer.SetCacheReadAttributes(span, false, bytesRead)
		tracer.EndSpan(span)

		spans := recorder.Ended()
		assert.Len(t, spans, 1)
		assert.Len(t, spans[0].Attributes(), 2)
		assert.Contains(t, spans[0].Attributes(), attribute.Int(BYTES_READ, bytesRead))
		assert.Contains(t, spans[0].Attributes(), attribute.Bool(IS_CACHE_HIT, false))
	})
}

func TestOtelTracer_PropagateTraceContext(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otel.SetTracerProvider(provider)
	tracer := NewOTELTracer()
	spanName := "test-propagation-span"
	oldCtx, oldSpan := tracer.StartSpan(context.Background(), spanName)

	newCtx := tracer.PropagateTraceContext(context.Background(), oldCtx)
	tracer.EndSpan(oldSpan)

	newSpan := trace.SpanFromContext(newCtx)
	assert.Equal(t, oldSpan, newSpan)
	spans := recorder.Ended()
	assert.Len(t, spans, 1)
}
