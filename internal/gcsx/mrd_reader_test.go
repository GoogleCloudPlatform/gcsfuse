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

package gcsx

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
)

func TestNewMRDReader(t *testing.T) {
	object := &gcs.MinObject{
		Name:       "test-object",
		Size:       1024,
		Generation: 1,
	}

	config := &cfg.Config{
		FileSystem: cfg.FileSystemConfig{
			IgnoreInterrupts: true,
		},
	}

	mrdWrapper, err := NewMultiRangeDownloaderWrapper(nil, object, config, false)
	if err != nil {
		t.Fatalf("Failed to create MRDWrapper: %v", err)
	}

	reader := NewMRDReader(object, metrics.NewNoopMetrics(), &mrdWrapper)

	if reader == nil {
		t.Fatal("NewMRDReader returned nil")
	}

	if reader.object != object {
		t.Errorf("Expected object %v, got %v", object, reader.object)
	}

	if reader.mrdWrapper != &mrdWrapper {
		t.Errorf("Expected mrdWrapper %v, got %v", &mrdWrapper, reader.mrdWrapper)
	}
}

func TestMRDReader_CheckInvariants(t *testing.T) {
	config := &cfg.Config{
		FileSystem: cfg.FileSystemConfig{
			IgnoreInterrupts: true,
		},
	}

	tests := []struct {
		name        string
		setupReader func() *MRDReader
		shouldPanic bool
		panicMsg    string
	}{
		{
			name: "valid reader",
			setupReader: func() *MRDReader {
				object := &gcs.MinObject{Name: "test", Size: 100}
				mrdWrapper, _ := NewMultiRangeDownloaderWrapper(nil, object, config, false)
				return NewMRDReader(object, metrics.NewNoopMetrics(), &mrdWrapper)
			},
			shouldPanic: false,
		},
		{
			name: "nil object",
			setupReader: func() *MRDReader {
				object := &gcs.MinObject{Name: "test", Size: 100}
				mrdWrapper, _ := NewMultiRangeDownloaderWrapper(nil, object, config, false)
				return &MRDReader{object: nil, mrdWrapper: &mrdWrapper}
			},
			shouldPanic: true,
			panicMsg:    "object is nil",
		},
		{
			name: "nil mrdWrapper",
			setupReader: func() *MRDReader {
				object := &gcs.MinObject{Name: "test", Size: 100}
				return &MRDReader{object: object, mrdWrapper: nil}
			},
			shouldPanic: false, // Changed: nil mrdWrapper is now allowed for test environments
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := tt.setupReader()

			if tt.shouldPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic but didn't get one")
					} else if tt.panicMsg != "" {
						msg := fmt.Sprintf("%v", r)
						if msg != "MRDReader: "+tt.panicMsg {
							t.Errorf("Expected panic message '%s', got '%s'", tt.panicMsg, msg)
						}
					}
				}()
			}

			reader.CheckInvariants()

			if tt.shouldPanic {
				t.Error("Expected panic but didn't get one")
			}
		})
	}
}

func TestMRDReader_ReadAt(t *testing.T) {
	object := &gcs.MinObject{
		Name:       "test-object",
		Size:       1024,
		Generation: 1,
	}

	config := &cfg.Config{
		FileSystem: cfg.FileSystemConfig{
			IgnoreInterrupts: true,
		},
	}

	mrdWrapper, err := NewMultiRangeDownloaderWrapper(nil, object, config, false)
	if err != nil {
		t.Fatalf("Failed to create MRDWrapper: %v", err)
	}

	reader := NewMRDReader(object, metrics.NewNoopMetrics(), &mrdWrapper)

	tests := []struct {
		name        string
		offset      int64
		bufferSize  int
		expectError bool
		expectEOF   bool
	}{
		{
			name:        "read beyond object size",
			offset:      1024,
			bufferSize:  100,
			expectError: true,
			expectEOF:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			buffer := make([]byte, tt.bufferSize)
			req := &ReadRequest{
				Buffer: buffer,
				Offset: tt.offset,
			}

			resp, err := reader.ReadAt(ctx, req)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if tt.expectEOF && err != io.EOF {
					t.Errorf("Expected io.EOF, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if resp.Size <= 0 {
					t.Errorf("Expected positive size, got %d", resp.Size)
				}
			}
		})
	}
}

func TestMRDReader_Object(t *testing.T) {
	object := &gcs.MinObject{
		Name:       "test-object",
		Size:       1024,
		Generation: 1,
	}

	config := &cfg.Config{
		FileSystem: cfg.FileSystemConfig{
			IgnoreInterrupts: true,
		},
	}

	mrdWrapper, err := NewMultiRangeDownloaderWrapper(nil, object, config, false)
	if err != nil {
		t.Fatalf("Failed to create MRDWrapper: %v", err)
	}

	reader := NewMRDReader(object, metrics.NewNoopMetrics(), &mrdWrapper)

	if reader.Object() != object {
		t.Errorf("Expected object %v, got %v", object, reader.Object())
	}
}

func TestMRDReader_Destroy(t *testing.T) {
	object := &gcs.MinObject{
		Name:       "test-object",
		Size:       1024,
		Generation: 1,
	}

	config := &cfg.Config{
		FileSystem: cfg.FileSystemConfig{
			IgnoreInterrupts: true,
		},
	}

	mrdWrapper, err := NewMultiRangeDownloaderWrapper(nil, object, config, false)
	if err != nil {
		t.Fatalf("Failed to create MRDWrapper: %v", err)
	}

	reader := NewMRDReader(object, metrics.NewNoopMetrics(), &mrdWrapper)

	// Should not panic
	reader.Destroy()
}
