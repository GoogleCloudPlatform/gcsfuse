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

package benchmark

import (
	"context"
	"io"
	"testing"
	"time"

	storagev2 "cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// --------------------------------------------------------------------------
// Minimal in-memory mock bucket for unit tests.
// It does not implement the full gcs.Bucket, only what the engine calls.
// --------------------------------------------------------------------------

// mockBucket implements gcs.Bucket for testing. It records read calls and
// returns configurable amounts of zero data.
type mockBucket struct {
	readDelay  time.Duration
	writeDelay time.Duration // injected into CreateObject to simulate write latency
	readBytes  int64
	objectSize uint64 // size returned by CreateObject (0 → default 4096)
	createErr  error
	readCalls  int
	writeCalls int
}

// mockReader is an io.ReadCloser that returns n bytes of zeros.
type mockReader struct {
	remaining int64
	delay     time.Duration
	once      bool
}

func (r *mockReader) Read(p []byte) (n int, err error) {
	if !r.once && r.delay > 0 {
		time.Sleep(r.delay)
		r.once = true
	}
	if r.remaining <= 0 {
		return 0, io.EOF
	}
	toRead := int64(len(p))
	if toRead > r.remaining {
		toRead = r.remaining
	}
	r.remaining -= toRead
	if r.remaining == 0 {
		return int(toRead), io.EOF
	}
	return int(toRead), nil
}
func (r *mockReader) Close() error                     { return nil }
func (r *mockReader) ReadHandle() storagev2.ReadHandle { return nil }

func (b *mockBucket) Name() string { return "mock-bucket" }
func (b *mockBucket) BucketType() gcs.BucketType {
	return gcs.BucketType{Zonal: true}
}
func (b *mockBucket) NewReaderWithReadHandle(_ context.Context, req *gcs.ReadObjectRequest) (gcs.StorageReader, error) {
	b.readCalls++
	size := b.readBytes
	if req.Range != nil {
		size = int64(req.Range.Limit) - int64(req.Range.Start)
		if size < 0 {
			size = 0
		}
	}
	return &mockReader{remaining: size, delay: b.readDelay}, nil
}
func (b *mockBucket) NewMultiRangeDownloader(_ context.Context, _ *gcs.MultiRangeDownloaderRequest) (gcs.MultiRangeDownloader, error) {
	return nil, nil
}
func (b *mockBucket) CreateObject(_ context.Context, req *gcs.CreateObjectRequest) (*gcs.Object, error) {
	b.writeCalls++
	if b.writeDelay > 0 {
		time.Sleep(b.writeDelay)
	}
	if b.createErr != nil {
		return nil, b.createErr
	}
	sz := b.objectSize
	if sz == 0 {
		sz = 4096
	}
	return &gcs.Object{Name: req.Name, Size: sz}, nil
}
func (b *mockBucket) CreateObjectChunkWriter(_ context.Context, _ *gcs.CreateObjectRequest, _ int, _ func(int64)) (gcs.Writer, error) {
	return nil, nil
}
func (b *mockBucket) CreateAppendableObjectWriter(_ context.Context, _ *gcs.CreateObjectChunkWriterRequest) (gcs.Writer, error) {
	return nil, nil
}
func (b *mockBucket) FinalizeUpload(_ context.Context, _ gcs.Writer) (*gcs.MinObject, error) {
	return nil, nil
}
func (b *mockBucket) FlushPendingWrites(_ context.Context, _ gcs.Writer) (*gcs.MinObject, error) {
	return nil, nil
}
func (b *mockBucket) CopyObject(_ context.Context, _ *gcs.CopyObjectRequest) (*gcs.Object, error) {
	return nil, nil
}
func (b *mockBucket) ComposeObjects(_ context.Context, _ *gcs.ComposeObjectsRequest) (*gcs.Object, error) {
	return nil, nil
}
func (b *mockBucket) StatObject(_ context.Context, _ *gcs.StatObjectRequest) (*gcs.MinObject, *gcs.ExtendedObjectAttributes, error) {
	return nil, nil, nil
}
func (b *mockBucket) ListObjects(_ context.Context, _ *gcs.ListObjectsRequest) (*gcs.Listing, error) {
	return &gcs.Listing{}, nil
}
func (b *mockBucket) UpdateObject(_ context.Context, _ *gcs.UpdateObjectRequest) (*gcs.Object, error) {
	return nil, nil
}
func (b *mockBucket) DeleteObject(_ context.Context, _ *gcs.DeleteObjectRequest) error { return nil }
func (b *mockBucket) MoveObject(_ context.Context, _ *gcs.MoveObjectRequest) (*gcs.Object, error) {
	return nil, nil
}
func (b *mockBucket) DeleteFolder(_ context.Context, _ string) error { return nil }
func (b *mockBucket) GetFolder(_ context.Context, _ *gcs.GetFolderRequest) (*gcs.Folder, error) {
	return nil, nil
}
func (b *mockBucket) RenameFolder(_ context.Context, _, _ string) (*gcs.Folder, error) {
	return nil, nil
}
func (b *mockBucket) CreateFolder(_ context.Context, _ string) (*gcs.Folder, error) {
	return nil, nil
}
func (b *mockBucket) GCSName(_ *gcs.MinObject) string { return "" }

// --------------------------------------------------------------------------
// Engine tests using the mock bucket.
// --------------------------------------------------------------------------

func makeReadOnlyConfig(duration, warmup time.Duration, concurrency int) cfg.BenchmarkConfig {
	return cfg.BenchmarkConfig{
		Duration:         duration,
		WarmupDuration:   warmup,
		TotalConcurrency: concurrency,
		OutputFormat:     "yaml",
		Histograms:       cfg.DefaultHistogramConfig(),
		Tracks: []cfg.BenchmarkTrack{
			{
				Name:          "read-4kb",
				Weight:        1,
				OpType:        "read",
				ObjectSizeMin: 4096,
				ObjectSizeMax: 4096,
				ReadSize:      4096,
				AccessPattern: "random",
				ObjectCount:   10,
				Concurrency:   concurrency,
			},
		},
	}
}

// TestNewEngineNilBucket verifies that NewEngine rejects a nil bucket.
func TestNewEngineNilBucket(t *testing.T) {
	_, err := NewEngine(nil, makeReadOnlyConfig(time.Second, 0, 1))
	if err == nil {
		t.Fatal("expected error for nil bucket, got nil")
	}
}

// TestNewEngineNoTracks verifies that NewEngine rejects an empty track list.
func TestNewEngineNoTracks(t *testing.T) {
	_, err := NewEngine(&mockBucket{}, cfg.BenchmarkConfig{
		Duration:   time.Second,
		Histograms: cfg.DefaultHistogramConfig(),
	})
	if err == nil {
		t.Fatal("expected error for empty tracks, got nil")
	}
}

// TestEngineRunShort runs a brief real benchmark loop against the mock bucket
// and validates that the summary fields are sensible.
func TestEngineRunShort(t *testing.T) {
	mb := &mockBucket{readDelay: 0, readBytes: 4096}
	bCfg := makeReadOnlyConfig(200*time.Millisecond, 0, 2)

	engine, err := NewEngine(mb, bCfg)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	summary, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("engine.Run: %v", err)
	}

	if len(summary.Tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(summary.Tracks))
	}
	ts := summary.Tracks[0]
	if ts.TrackName != "read-4kb" {
		t.Errorf("expected track name 'read-4kb', got %q", ts.TrackName)
	}
	if ts.TotalOps <= 0 {
		t.Errorf("expected > 0 ops, got %d", ts.TotalOps)
	}
	if ts.OpsPerSec <= 0 {
		t.Errorf("expected positive ops/s, got %.2f", ts.OpsPerSec)
	}
	if ts.TotalLatency.P50 < 0 {
		t.Errorf("expected non-negative p50, got %.0f", ts.TotalLatency.P50)
	}
}

// TestEngineRunWithWarmup verifies that warmup is discarded and measurement
// still produces a valid summary.
func TestEngineRunWithWarmup(t *testing.T) {
	mb := &mockBucket{readBytes: 4096}
	bCfg := makeReadOnlyConfig(300*time.Millisecond, 100*time.Millisecond, 1)

	engine, err := NewEngine(mb, bCfg)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	summary, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("engine.Run: %v", err)
	}
	if len(summary.Tracks) == 0 {
		t.Fatal("expected at least one track in summary")
	}
}

// TestGoroutinesForTrack verifies concurrency distribution logic.
func TestGoroutinesForTrack(t *testing.T) {
	bCfg := cfg.BenchmarkConfig{
		TotalConcurrency: 10,
		Tracks: []cfg.BenchmarkTrack{
			{Name: "a", Weight: 3},
			{Name: "b", Weight: 7},
		},
	}

	g0 := goroutinesForTrack(bCfg, 0) // weight 3 out of 10 → 3
	g1 := goroutinesForTrack(bCfg, 1) // weight 7 out of 10 → 7
	if g0 != 3 {
		t.Errorf("expected goroutines(0)=3, got %d", g0)
	}
	if g1 != 7 {
		t.Errorf("expected goroutines(1)=7, got %d", g1)
	}
}

// TestGoroutinesForTrackDirectOverride verifies per-track Concurrency override.
func TestGoroutinesForTrackDirectOverride(t *testing.T) {
	bCfg := cfg.BenchmarkConfig{
		TotalConcurrency: 10,
		Tracks: []cfg.BenchmarkTrack{
			{Name: "x", Weight: 1, Concurrency: 4},
		},
	}
	g := goroutinesForTrack(bCfg, 0)
	if g != 4 {
		t.Errorf("expected concurrency=4 (direct override), got %d", g)
	}
}
