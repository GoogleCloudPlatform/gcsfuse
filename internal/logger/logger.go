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

package logger

import (
	"encoding/json"
	"io"
	"os"
	"time"
)

// Init creates or opens a log file.
func Init(filename string) (f io.Writer, err error) {
	f, err = os.OpenFile(
		filename,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		0644,
	)
	f = &fluentdWriter{
		w: f,
	}
	return
}

type logEntry struct {
	Name             string `json:"name,omitempty"`
	LevelName        string `json:"levelname,omitempty"`
	Severity         string `json:"severity,omitempty"`
	Message          string `json:"message,omitempty"`
	TimestampSeconds int64  `json:"timestampSeconds,omitempty"`
	TimestampNanos   int    `json:"timestampNanos,omitempty"`
}

// FluentdLogFile is an io.Writer that prints the logs to the file in
// fluentd-based json format. This is not thread-safe.
type fluentdWriter struct {
	w io.Writer
}

// Write writes log message with formatting.
func (f *fluentdWriter) Write(p []byte) (n int, err error) {
	now := time.Now()

	entry := logEntry{
		Name:             "root",
		LevelName:        "INFO",
		Severity:         "INFO",
		Message:          string(p),
		TimestampSeconds: now.Unix(),
		TimestampNanos:   now.Nanosecond(),
	}

	var buf []byte
	buf, err = json.Marshal(entry)
	if err != nil {
		return
	}

	buf = append(buf, '\n')
	_, err = f.w.Write(buf)
	if err != nil {
		return
	}

	n = len(p)
	return
}
