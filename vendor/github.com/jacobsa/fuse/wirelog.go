// Copyright 2025 Google Inc. All Rights Reserved.
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

package fuse

import (
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"syscall"
	"time"

	"github.com/jacobsa/fuse/fuseops"
)

// NewWireLogRecord creates a new empty WireLogRecord.
func NewWireLogRecord() *WireLogRecord {
	return &WireLogRecord{
		StartTime: time.Now(),
		Args:      make(map[string]any),
		Extra:     make(map[string]any),
	}
}

// A WireLogRecord is created for each FUSE operation when WireLogger is
// non-nil. Fields are filled in by jacobsa/fuse; file system implementations
// can add their own fields by writing to the Extra map.
type WireLogRecord struct {
	Operation string
	StartTime time.Time
	Duration  time.Duration
	Status    int
	Context   *fuseops.OpContext
	Args      map[string]any // Serialized representation of the fuseops.*Op struct
	Extra     map[string]any // Custom fields added by file system implementation
}

var ignoredParams = []string{"OpContext", "Dst", "Data"}

func formatWireLogEntry(op any, opErr error, wlog *WireLogRecord) ([]byte, error) {
	v := reflect.ValueOf(op).Elem()
	t := v.Type()

	// Operation name and duration
	wlog.Operation = t.Name()
	wlog.Duration = time.Since(wlog.StartTime)

	// Result of the operation
	var errno syscall.Errno
	if opErr == nil {
		wlog.Status = 0
	} else if errors.As(opErr, &errno) {
		wlog.Status = int(errno)
	}

	// Separate section for the operation context
	if f := v.FieldByName("OpContext"); f.IsValid() {
		if ctx, ok := f.Interface().(fuseops.OpContext); ok {
			wlog.Context = &ctx
		}
	}

	// Copy the the rest of the fields to the "Args" section
	args := map[string]any{}
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.Kind() == reflect.Ptr && f.IsNil() {
			continue
		}
		if f.Kind() == reflect.Func {
			continue
		}
		fieldName := t.Field(i).Name
		if slices.Contains(ignoredParams, fieldName) {
			continue
		}
		args[fieldName] = f.Interface()
	}

	switch typed := op.(type) {
	case *fuseops.ReadFileOp:
		args["BytesRead"] = typed.BytesRead

	case *fuseops.WriteFileOp:
		args["Size"] = len(typed.Data)
	}

	wlog.Args = args

	// Serialize as pretty-printed JSON
	buf, err := json.MarshalIndent(wlog, "", "  ")
	if err == nil {
		buf = append(buf, '\n')
	}
	return buf, err
}
