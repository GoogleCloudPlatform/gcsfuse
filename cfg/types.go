// Copyright 2024 Google Inc. All Rights Reserved.
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

	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
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

func (o *Octal) String() string {
	return fmt.Sprintf("%o", *o)
}

// Protocol is the datatype that specifies the type of connection: http1/http2/grpc.
type Protocol string

const (
	HTTP1 Protocol = "http1"
	HTTP2 Protocol = "http2"
	GRPC  Protocol = "grpc"
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

func (l *LogSeverity) UnmarshalText(text []byte) error {
	textStr := string(text)
	level := strings.ToUpper(textStr)
	v := []string{"TRACE", "DEBUG", "INFO", "WARNING", "ERROR", "OFF"}
	if !slices.Contains(v, level) {
		return fmt.Errorf("invalid logseverity value: %s. It can only assume values in the list: %v", textStr, v)
	}
	*l = LogSeverity(level)
	return nil
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
