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

package cfg

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
)

// Octal is the datatype for params such as file-mode and dir-mode which accept a base-8 value.
type Octal int

func (o *Octal) UnmarshalText(text []byte) error {
	v, err := strconv.ParseInt(string(text) /*base=*/, 8 /*bitSize=*/, 32)
	if err != nil {
		return err
	}
	*o = Octal(v)
	return nil
}

func (o Octal) MarshalText() ([]byte, error) {
	return []byte(strconv.FormatInt(int64(o), 8)), nil
}

// Protocol is the datatype that specifies the type of connection: http1/http2/grpc.
type Protocol string

const (
	HTTP1 = "http1"
	HTTP2 = "http2"
	GRPC  = "grpc"
)

func (p *Protocol) UnmarshalText(text []byte) error {
	txtStr := string(text)
	protocol := strings.ToLower(txtStr)
	v := []string{"http1", "http2", "grpc"}
	if !slices.Contains(v, protocol) {
		return fmt.Errorf("invalid protocol value: %s. It can only accept values in the list: %v", txtStr, v)
	}
	*p = Protocol(protocol)
	return nil
}

// LogSeverity represents the logging severity and can accept the following values
// "TRACE", "DEBUG", "INFO", "WARNING", "ERROR", "OFF"
type LogSeverity string

// Constants for all supported log severities.
const (
	TraceLogSeverity   LogSeverity = "TRACE"
	DebugLogSeverity   LogSeverity = "DEBUG"
	InfoLogSeverity    LogSeverity = "INFO"
	WarningLogSeverity LogSeverity = "WARNING"
	ErrorLogSeverity   LogSeverity = "ERROR"
	OffLogSeverity     LogSeverity = "OFF"
)

// severityRanking maps each level to an integer for validation and comparison.
var severityRanking = map[LogSeverity]int{
	TraceLogSeverity:   0,
	DebugLogSeverity:   1,
	InfoLogSeverity:    2,
	WarningLogSeverity: 3,
	ErrorLogSeverity:   4,
	OffLogSeverity:     5,
}

func (l *LogSeverity) UnmarshalText(text []byte) error {
	level := LogSeverity(strings.ToUpper(string(text)))
	if _, ok := severityRanking[level]; !ok {
		return fmt.Errorf("invalid log severity level: %s. Must be one of [TRACE, DEBUG, INFO, WARNING, ERROR, OFF]", text)
	}
	*l = level
	return nil
}

// Rank returns the integer representation of the severity rank.
// Returns -1 if the severity is unknown.
func (l LogSeverity) Rank() int {
	if rank, ok := severityRanking[l]; ok {
		return rank
	}
	// This case should ideally not be reached as LogSeverity configs are validated before mounting.
	return -1
}

// ResolvedPath represents a file-path which is an absolute path and is resolved
// based on the value of GCSFUSE_PARENT_PROCESS_DIR env var.
type ResolvedPath string

func (p *ResolvedPath) UnmarshalText(text []byte) error {
	path, err := util.GetResolvedPath(string(text))
	if err != nil {
		return err
	}
	*p = ResolvedPath(path)
	return nil
}

// OptimizationInput provides runtime context for applying optimizations.
// This struct can be extended in the future to support additional optimization
// dimensions such as region, storage class, or project-specific settings.
type OptimizationInput struct {
	// BucketType specifies the GCS bucket type.
	// An empty string means no bucket-type-based optimization should be applied.
	BucketType BucketType
}

// BucketType represents the type of GCS bucket.
type BucketType string

const (
	// BucketTypeZonal represents a zonal bucket with single-zone storage.
	BucketTypeZonal BucketType = "zonal"

	// BucketTypeHierarchical represents a bucket with hierarchical namespace enabled.
	BucketTypeHierarchical BucketType = "hierarchical"

	// BucketTypeFlat represents a flat (regional or multi-regional) bucket.
	BucketTypeFlat BucketType = "flat"
)

// IsValid returns true if the BucketType is one of the defined valid types.
func (bt BucketType) IsValid() bool {
	return bt == BucketTypeZonal || bt == BucketTypeHierarchical || bt == BucketTypeFlat
}
