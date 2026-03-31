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
	"fmt"
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
	_, err := NewEngine(nil, makeReadOnlyConfig(time.Second, 0, 1), 0, nil)
	if err == nil {
		t.Fatal("expected error for nil bucket, got nil")
	}
}

// TestNewEngineNoTracks verifies that NewEngine rejects an empty track list.
func TestNewEngineNoTracks(t *testing.T) {
	_, err := NewEngine(&mockBucket{}, cfg.BenchmarkConfig{
		Duration:   time.Second,
		Histograms: cfg.DefaultHistogramConfig(),
	}, 0, nil)
	if err == nil {
		t.Fatal("expected error for empty tracks, got nil")
	}
}

// TestEngineRunShort runs a brief real benchmark loop against the mock bucket
// and validates that the summary fields are sensible.
func TestEngineRunShort(t *testing.T) {
	mb := &mockBucket{readDelay: 0, readBytes: 4096}
	bCfg := makeReadOnlyConfig(200*time.Millisecond, 0, 2)

	engine, err := NewEngine(mb, bCfg, 0, nil)
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

	engine, err := NewEngine(mb, bCfg, 0, nil)
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

// TestReadBufPoolInitialized verifies that NewEngine initialises readBufPool
// and that it vends correctly sized buffers (256 KiB).
func TestReadBufPoolInitialized(t *testing.T) {
	mb := &mockBucket{readBytes: 4096}
	bCfg := makeReadOnlyConfig(100*time.Millisecond, 0, 1)

	engine, err := NewEngine(mb, bCfg, 0, nil)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	// GetPool a buffer, check size, then return it.
	bufPtr := engine.readBufPool.Get().(*[]byte)
	if bufPtr == nil {
		t.Fatal("readBufPool.Get() returned nil")
	}
	const wantSize = 256 * 1024
	if len(*bufPtr) != wantSize {
		t.Errorf("pool buffer len = %d, want %d", len(*bufPtr), wantSize)
	}
	engine.readBufPool.Put(bufPtr)
}

// TestRuntimeStatsPopulated verifies that RunSummary.Runtime is populated with
// sensible values after a normal measurement run.
func TestRuntimeStatsPopulated(t *testing.T) {
	mb := &mockBucket{readDelay: 0, readBytes: 4096}
	bCfg := makeReadOnlyConfig(200*time.Millisecond, 0, 2)

	engine, err := NewEngine(mb, bCfg, 0, nil)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	summary, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("engine.Run: %v", err)
	}

	rt := summary.Runtime

	if rt.GoHeapAllocBytes == 0 {
		t.Error("expected non-zero GoHeapAllocBytes after run")
	}
	if rt.GoHeapSysBytes == 0 {
		t.Error("expected non-zero GoHeapSysBytes after run")
	}
	if rt.PeakRSSKiB <= 0 {
		t.Errorf("expected positive PeakRSSKiB, got %d", rt.PeakRSSKiB)
	}
	if rt.ProcessUserCPUPct < 0 {
		t.Errorf("expected non-negative ProcessUserCPUPct, got %.2f", rt.ProcessUserCPUPct)
	}
	if rt.ProcessSysCPUPct < 0 {
		t.Errorf("expected non-negative ProcessSysCPUPct, got %.2f", rt.ProcessSysCPUPct)
	}
	if rt.SystemCPUPercent < 0 {
		t.Errorf("expected non-negative SystemCPUPercent, got %.2f", rt.SystemCPUPercent)
	}
}

// TestDoReadFullObjectWhenReadSizeIsZero verifies that a read-size of 0 (or
// negative) issues a full-object read (no Range restriction) rather than
// silently defaulting to 4 MiB. The mock bucket uses b.readBytes when
// req.Range is nil, so we confirm the correct number of bytes is accumulated.
func TestDoReadFullObjectWhenReadSizeIsZero(t *testing.T) {
	const fullObjectSize = 8 * 1024 * 1024 // 8 MiB

	for _, readSize := range []int64{0, -1} {
		t.Run(fmt.Sprintf("read-size=%d", readSize), func(t *testing.T) {
			mb := &mockBucket{readBytes: fullObjectSize}
			bCfg := cfg.BenchmarkConfig{
				Duration:         200 * time.Millisecond,
				TotalConcurrency: 1,
				Histograms:       cfg.DefaultHistogramConfig(),
				Tracks: []cfg.BenchmarkTrack{
					{
						Name:        "full-read",
						OpType:      "read",
						Weight:      1,
						ReadSize:    readSize,
						ObjectCount: 1,
						Concurrency: 1,
					},
				},
			}

			eng, err := NewEngine(mb, bCfg, 0, nil)
			if err != nil {
				t.Fatalf("NewEngine: %v", err)
			}
			summary, err := eng.Run(context.Background())
			if err != nil {
				t.Fatalf("engine.Run: %v", err)
			}
			if len(summary.Tracks) == 0 {
				t.Fatal("expected at least 1 track in summary")
			}
			ts := summary.Tracks[0]

			// Each successful op should have read the full 8 MiB.
			successfulOps := ts.TotalOps - ts.Errors
			if successfulOps <= 0 {
				t.Fatalf("expected > 0 successful ops, got %d ops / %d errs", ts.TotalOps, ts.Errors)
			}
			expectedBytes := successfulOps * fullObjectSize
			if ts.AvgOpSizeBytes < float64(fullObjectSize)*0.9 {
				t.Errorf("read-size=%d: avg op size %.0f bytes is much less than full object size %d — "+
					"full-object read may not be working; expected each op to read %d bytes (total: %d)",
					readSize, ts.AvgOpSizeBytes, fullObjectSize, fullObjectSize, expectedBytes)
			}
		})
	}
}

// ── Write-pool Pipeline tests ─────────────────────────────────────────────

// runPrepareFullSummary is like runPrepare but returns the full RunSummary so
// tests can inspect Pipeline and Runtime fields.
func runPrepareFullSummary(t *testing.T, mb *mockBucket, objectCount int) RunSummary {
	t.Helper()
	engine, err := NewEngine(mb, makePrepareConfig(objectCount), 0, nil)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	summary, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("engine.Run (prepare): %v", err)
	}
	return summary
}

// runBenchWriteFullSummary is like runBenchWrite but returns the full RunSummary.
func runBenchWriteFullSummary(t *testing.T, mb *mockBucket, duration time.Duration, concurrency int) RunSummary {
	t.Helper()
	engine, err := NewEngine(mb, makeWriteBenchConfig(duration, concurrency), 0, nil)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	summary, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("engine.Run (benchmark write): %v", err)
	}
	return summary
}

// TestPipelineStatsPopulatedForPrepare verifies that RunSummary.Pipeline is
// non-nil and contains plausible values after a prepare run whose objects fit
// within the pool slot cap (objects ≤ 512 MiB always use the pool path).
func TestPipelineStatsPopulatedForPrepare(t *testing.T) {
	mb := &mockBucket{objectSize: mockObjectSize}
	summary := runPrepareFullSummary(t, mb, 20)

	if summary.Pipeline == nil {
		t.Fatal("expected Pipeline to be non-nil after prepare with write pool active")
	}
	p := summary.Pipeline

	if p.ProducerRateGiBps <= 0 {
		t.Errorf("ProducerRateGiBps: want > 0, got %.4f", p.ProducerRateGiBps)
	}
	if p.ConsumerRateGiBps <= 0 {
		t.Errorf("ConsumerRateGiBps: want > 0, got %.4f", p.ConsumerRateGiBps)
	}
	if p.HeadroomRatio <= 0 {
		t.Errorf("HeadroomRatio: want > 0, got %.4f", p.HeadroomRatio)
	}
	// Producer stall fraction must be non-negative (can be zero in fast tests).
	if p.ProducerStallSec < 0 {
		t.Errorf("ProducerStallSec: want >= 0, got %.6f", p.ProducerStallSec)
	}
	if p.ProducerStallPct < 0 {
		t.Errorf("ProducerStallPct: want >= 0, got %.4f", p.ProducerStallPct)
	}
	if p.ConsumerStallSec < 0 {
		t.Errorf("ConsumerStallSec: want >= 0, got %.6f", p.ConsumerStallSec)
	}
}

// TestPipelineStatsPopulatedForBenchWrite verifies that RunSummary.Pipeline is
// non-nil and contains plausible values after a time-bounded write benchmark.
func TestPipelineStatsPopulatedForBenchWrite(t *testing.T) {
	mb := &mockBucket{objectSize: mockObjectSize}
	summary := runBenchWriteFullSummary(t, mb, 200*time.Millisecond, 2)

	if summary.Pipeline == nil {
		t.Fatal("expected Pipeline to be non-nil after benchmark write with write pool active")
	}
	p := summary.Pipeline

	if p.ProducerRateGiBps <= 0 {
		t.Errorf("ProducerRateGiBps: want > 0, got %.4f", p.ProducerRateGiBps)
	}
	if p.ConsumerRateGiBps <= 0 {
		t.Errorf("ConsumerRateGiBps: want > 0, got %.4f", p.ConsumerRateGiBps)
	}
	if p.HeadroomRatio <= 0 {
		t.Errorf("HeadroomRatio: want > 0, got %.4f", p.HeadroomRatio)
	}
}

// TestPipelineNilForReadOnlyWorkload verifies that RunSummary.Pipeline is nil
// when the workload contains no write tracks (no pool is created).
func TestPipelineNilForReadOnlyWorkload(t *testing.T) {
	mb := &mockBucket{readBytes: 4096}
	bCfg := makeReadOnlyConfig(200*time.Millisecond, 0, 2)

	engine, err := NewEngine(mb, bCfg, 0, nil)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	summary, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("engine.Run: %v", err)
	}

	if summary.Pipeline != nil {
		t.Errorf("expected Pipeline to be nil for read-only workload, got %+v", summary.Pipeline)
	}
}

// TestPipelineProducerFasterThanConsumer verifies the fundamental invariant:
// in normal operation the producer (pure CPU/RAM) should generate data faster
// than the consumer can upload to GCS.  We inject a write delay to simulate
// a slow network so the producer easily stays ahead.
func TestPipelineProducerFasterThanConsumer(t *testing.T) {
	mb := &mockBucket{
		objectSize: mockObjectSize,
		writeDelay: 5 * time.Millisecond, // simulate network latency
	}
	summary := runBenchWriteFullSummary(t, mb, 300*time.Millisecond, 4)

	if summary.Pipeline == nil {
		t.Fatal("expected non-nil Pipeline")
	}
	p := summary.Pipeline

	// With an artificial write delay the producer should comfortably outpace
	// the consumer — headroom ratio should be well above 1.
	if p.HeadroomRatio < 1.0 {
		t.Errorf("HeadroomRatio: want >= 1.0 (producer faster than consumer), got %.4f", p.HeadroomRatio)
	}
	if p.ProducerRateGiBps < p.ConsumerRateGiBps {
		t.Errorf("expected ProducerRateGiBps (%.4f) >= ConsumerRateGiBps (%.4f)",
			p.ProducerRateGiBps, p.ConsumerRateGiBps)
	}
}
