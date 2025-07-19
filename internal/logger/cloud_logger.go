// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"cloud.google.com/go/logging"
)

// NewCloudLogWriter creates a new io.Writer that sends logs to Google Cloud Logging.
// It requires the GCP project ID. The logID is the name of the log stream in Cloud Logging.
func NewCloudLogWriter(projectID, logID string) (*cloudLogWriter, error) {
	ctx := context.Background()
	// NewClient uses Application Default Credentials to authenticate.
	client, err := logging.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud logging client: %w", err)
	}

	// Set a handler for errors that may occur within the Cloud Logging client.
	client.OnError = (func(err error) {
		// This error handler should not block and should be lightweight.
		// It's a good place to log to a fallback (e.g., stderr) if Cloud Logging fails.
		fmt.Fprintf(os.Stderr, "gcsfuse: Cloud Logging client error: %v\n", err)
	})

	logger := client.Logger(logID)

	return &cloudLogWriter{
		logger: logger,
		client: client,
	}, nil
}

// cloudLogWriter implements io.Writer and io.Closer, and sends writes to a Cloud Logging stream.
type cloudLogWriter struct {
	logger *logging.Logger
	client *logging.Client
}

// Write parses the JSON log entry and sends it to Cloud Logging.
// gcsfuse is configured to produce JSON logs when this writer is active.
func (w *cloudLogWriter) Write(p []byte) (n int, err error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(p, &payload); err != nil {
		// Fallback for non-JSON messages, though this shouldn't happen with correct config.
		w.logger.Log(logging.Entry{Payload: string(p)})
		return len(p), nil
	}

	// Map known fields to the Cloud Logging Entry struct for better integration.
	entry := logging.Entry{
		Payload: payload,
	}

	if severity, ok := payload["severity"].(string); ok {
		entry.Severity = mapSeverity(severity)
		// Remove from payload to avoid duplication.
		delete(payload, "severity")
	}

	w.logger.Log(entry)
	return len(p), nil
}

// Close flushes any buffered logs and closes the client.
func (w *cloudLogWriter) Close() error {
	if w.client != nil {
		// The client.Close() call also flushes any buffered entries.
		if err := w.client.Close(); err != nil {
			return fmt.Errorf("failed to close cloud logging client: %w", err)
		}
	}
	return nil
}

// mapSeverity converts a text severity level to a Cloud Logging severity.
func mapSeverity(level string) logging.Severity {
	switch level {
	case "TRACE", "DEBUG":
		return logging.Debug
	case "INFO":
		return logging.Info
	case "WARNING":
		return logging.Warning
	case "ERROR":
		return logging.Error
	case "FATAL", "PANIC":
		return logging.Critical
	default:
		return logging.Default
	}
}
