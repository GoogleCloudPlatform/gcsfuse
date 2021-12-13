// Copyright 2021 Google Inc. All Rights Reserved.
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
	"time"
)

type logEntry struct {
	Name             string `json:"name,omitempty"`
	LevelName        string `json:"levelname,omitempty"`
	Severity         string `json:"severity,omitempty"`
	Message          string `json:"message,omitempty"`
	TimestampSeconds int64  `json:"timestampSeconds,omitempty"`
	TimestampNanos   int    `json:"timestampNanos,omitempty"`
}

// jsonWriter is an io.Writer that prints the logs to the file in
// fluentd-based json format. This is not thread-safe.
type jsonWriter struct {
	w     io.Writer
	level string
}

// Write writes log message with formatting.
func (f *jsonWriter) Write(p []byte) (n int, err error) {
	now := time.Now()

	entry := logEntry{
		Name:             "root",
		LevelName:        f.level,
		Severity:         f.level,
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

type textWriter struct {
	w     io.Writer
	level string
}

func (f *textWriter) Write(p []byte) (n int, err error) {
	now := time.Now()
	if _, err := f.w.Write([]byte{f.level[0]}); err != nil {
		return 0, err
	}
	if _, err := f.w.Write([]byte(now.Format("0102 15:04:05.000000"))); err != nil {
		return 0, err
	}
	if _, err := f.w.Write([]byte{' '}); err != nil {
		return 0, err
	}
	return f.w.Write(p)
}
