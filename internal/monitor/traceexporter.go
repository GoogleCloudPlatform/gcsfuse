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
	"fmt"

	cloudtrace "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/auth"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/api/option"
)

// SetupTracing bootstraps the OpenTelemetry tracing pipeline.
func SetupTracing(ctx context.Context, c *cfg.Config, mountID string) common.ShutdownFn {
	tp, shutdown, err := newTraceProvider(ctx, c, mountID)
	if err != nil {
		logger.Errorf("error occurred while setting up tracing: %v", err)
		return nil
	}
	if tp != nil {
		otel.SetTracerProvider(tp)
		return shutdown
	}

	return nil
}

type exporterFactory func() (sdktrace.SpanExporter, error)

func newTraceProvider(ctx context.Context, c *cfg.Config, mountID string) (trace.TracerProvider, common.ShutdownFn, error) {
	var opts []sdktrace.TracerProviderOption
	exporterNames := c.Trace.Exporters

	exporterRegistry := map[string]exporterFactory{
		"stdout": func() (sdktrace.SpanExporter, error) {
			return newStdoutTraceExporter()
		},
		"gcpexporter": func() (sdktrace.SpanExporter, error) {
			return newGCPCloudTraceExporter(ctx, c)
		},
	}

	for _, name := range exporterNames {
		if expFactory, ok := exporterRegistry[name]; ok {
			exporter, err := expFactory()

			if err != nil {
				logger.Errorf("failed to initialize %s exporter: %s", name, err)
				return nil, nil, err
			}

			opts = append(opts, sdktrace.WithBatcher(exporter))
		}
	}

	res, err := getResource(ctx, mountID)
	if err != nil {
		return nil, nil, err
	}

	opts = append(opts, sdktrace.WithResource(res), sdktrace.WithSampler(sdktrace.TraceIDRatioBased(c.Trace.SamplingRatio)))

	tp := sdktrace.NewTracerProvider(opts...)

	return tp, tp.Shutdown, nil
}

func newStdoutTraceExporter() (sdktrace.SpanExporter, error) {
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint())

	if err != nil {
		return nil, err
	}

	return exporter, nil
}

// newGCPCloudTraceExporter creates a Cloud Trace exporter with fallback credential support.
//
// The function attempts to initialize the exporter with the following credential
// sources in order:
//  1. Application Default Credentials (ADC)
//  2. Token URL credentials (if configured via --token-url)
//
// Project ID is determined from:
//  1. Trace.ProjectId config (if set)
//  2. BillingProject (fallback, if tracing project ID not set)
//  3. Auto-detected from credentials (if neither is set)
func newGCPCloudTraceExporter(ctx context.Context, c *cfg.Config) (sdktrace.SpanExporter, error) {
	baseOptions := buildBaseTraceOptions(c)

	// Attempt to create exporter with default credentials.
	exporter, err := cloudtrace.New(baseOptions...)
	if err == nil {
		logger.Infof("Cloud Trace exporter initialized with default credentials")
		return exporter, nil
	}

	// Fall back to token URL credentials if configured.
	if c.GcsAuth.TokenUrl == "" {
		return nil, fmt.Errorf("default credentials unavailable and no --token-url configured: %w", err)
	}

	logger.Infof("Default credentials unavailable, using --token-url for Cloud Trace")
	return createCloudTraceExporterWithTokenURL(ctx, c, baseOptions)
}

// buildBaseTraceOptions constructs the base Cloud Trace options including
// project ID if explicitly configured.
func buildBaseTraceOptions(c *cfg.Config) []cloudtrace.Option {
	var options []cloudtrace.Option
	if c.Trace.ProjectId != "" {
		options = append(options, cloudtrace.WithProjectID(c.Trace.ProjectId))
	}
	return options
}

// createCloudTraceExporterWithTokenURL creates a Cloud Trace exporter using
// token URL credentials.
func createCloudTraceExporterWithTokenURL(
	ctx context.Context,
	c *cfg.Config,
	baseOptions []cloudtrace.Option,
) (sdktrace.SpanExporter, error) {
	tokenSrc, err := auth.NewTokenSourceFromURL(ctx, c.GcsAuth.TokenUrl, c.GcsAuth.ReuseTokenFromUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create token source: %w", err)
	}

	clientOpts := []option.ClientOption{option.WithTokenSource(tokenSrc)}
	options := append(baseOptions, cloudtrace.WithTraceClientOptions(clientOpts))

	// Try with explicit billing project ID if tracing project ID is not configured.
	if c.Trace.ProjectId == "" && c.GcsConnection.BillingProject != "" {
		optionsWithProject := append(options, cloudtrace.WithProjectID(c.GcsConnection.BillingProject))
		if exporter, err := cloudtrace.New(optionsWithProject...); err == nil {
			logger.Infof("Cloud Trace exporter initialized with --token-url and project %s", c.GcsConnection.BillingProject)
			return exporter, nil
		}
		logger.Infof("Failed to initialize with explicit project ID, retrying without project")
	}

	// Retry without explicit project ID.
	exporter, err := cloudtrace.New(options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter with --token-url: %w", err)
	}

	logger.Infof("Cloud Trace exporter initialized with --token-url credentials")
	return exporter, nil
}
