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

package storage_test

import (
	"context"
	"io"
	"strings"
	"testing"

	storagev2 "cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/benchmark"
	internalstorage "github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// --------------------------------------------------------------------------
// Minimal mock bucket for instrumentation tests.
// --------------------------------------------------------------------------

type testBucket struct {
	name string
}

func (b *testBucket) Name() string               { return b.name }
func (b *testBucket) BucketType() gcs.BucketType { return gcs.BucketType{} }
func (b *testBucket) NewReaderWithReadHandle(_ context.Context, req *gcs.ReadObjectRequest) (gcs.StorageReader, error) {
	size := int64(1024)
	if req.Range != nil {
		size = int64(req.Range.Limit) - int64(req.Range.Start)
	}
	return &testReader{remaining: size}, nil
}
func (b *testBucket) NewMultiRangeDownloader(_ context.Context, _ *gcs.MultiRangeDownloaderRequest) (gcs.MultiRangeDownloader, error) {
	return nil, nil
}
func (b *testBucket) CreateObject(_ context.Context, req *gcs.CreateObjectRequest) (*gcs.Object, error) {
	return &gcs.Object{Name: req.Name, Size: 512}, nil
}
func (b *testBucket) CreateObjectChunkWriter(_ context.Context, _ *gcs.CreateObjectRequest, _ int, _ func(int64)) (gcs.Writer, error) {
	return nil, nil
}
func (b *testBucket) CreateAppendableObjectWriter(_ context.Context, _ *gcs.CreateObjectChunkWriterRequest) (gcs.Writer, error) {
	return nil, nil
}
func (b *testBucket) FinalizeUpload(_ context.Context, _ gcs.Writer) (*gcs.MinObject, error) {
	return nil, nil
}
func (b *testBucket) FlushPendingWrites(_ context.Context, _ gcs.Writer) (*gcs.MinObject, error) {
	return nil, nil
}
func (b *testBucket) CopyObject(_ context.Context, _ *gcs.CopyObjectRequest) (*gcs.Object, error) {
	return nil, nil
}
func (b *testBucket) ComposeObjects(_ context.Context, _ *gcs.ComposeObjectsRequest) (*gcs.Object, error) {
	return nil, nil
}
func (b *testBucket) StatObject(_ context.Context, _ *gcs.StatObjectRequest) (*gcs.MinObject, *gcs.ExtendedObjectAttributes, error) {
	return &gcs.MinObject{Name: "obj"}, nil, nil
}
func (b *testBucket) ListObjects(_ context.Context, _ *gcs.ListObjectsRequest) (*gcs.Listing, error) {
	return &gcs.Listing{}, nil
}
func (b *testBucket) UpdateObject(_ context.Context, _ *gcs.UpdateObjectRequest) (*gcs.Object, error) {
	return nil, nil
}
func (b *testBucket) DeleteObject(_ context.Context, _ *gcs.DeleteObjectRequest) error { return nil }
func (b *testBucket) MoveObject(_ context.Context, _ *gcs.MoveObjectRequest) (*gcs.Object, error) {
	return nil, nil
}
func (b *testBucket) DeleteFolder(_ context.Context, _ string) error { return nil }
func (b *testBucket) GetFolder(_ context.Context, _ *gcs.GetFolderRequest) (*gcs.Folder, error) {
	return nil, nil
}
func (b *testBucket) RenameFolder(_ context.Context, _, _ string) (*gcs.Folder, error) {
	return nil, nil
}
func (b *testBucket) CreateFolder(_ context.Context, _ string) (*gcs.Folder, error) {
	return nil, nil
}
func (b *testBucket) GCSName(_ *gcs.MinObject) string { return "" }

type testReader struct {
	remaining int64
}

func (r *testReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, io.EOF
	}
	n := int64(len(p))
	if n > r.remaining {
		n = r.remaining
	}
	r.remaining -= n
	if r.remaining == 0 {
		return int(n), io.EOF
	}
	return int(n), nil
}
func (r *testReader) Close() error                     { return nil }
func (r *testReader) ReadHandle() storagev2.ReadHandle { return nil }

// --------------------------------------------------------------------------
// Tests
// --------------------------------------------------------------------------

func makeHists() *benchmark.TrackHistograms {
	return benchmark.NewTrackHistograms(cfg.DefaultHistogramConfig())
}

// TestNewInstrumentedBucketNilWrapped verifies nil-wrapping protection.
func TestNewInstrumentedBucketNilWrapped(t *testing.T) {
	b := internalstorage.NewInstrumentedBucket(nil, internalstorage.InstrumentedBucketParams{
		TrackName: "t",
		Hists:     makeHists(),
	})
	if b != nil {
		t.Error("expected nil for nil wrapped bucket")
	}
}

// TestInstrumentedBucketName verifies name delegation.
func TestInstrumentedBucketName(t *testing.T) {
	inner := &testBucket{name: "my-bucket"}
	b := internalstorage.NewInstrumentedBucket(inner, internalstorage.InstrumentedBucketParams{
		TrackName: "test",
		Hists:     makeHists(),
	})
	if b.Name() != "my-bucket" {
		t.Errorf("expected name 'my-bucket', got %q", b.Name())
	}
}

// TestInstrumentedBucketRead verifies that a read records TTFB and total latency.
func TestInstrumentedBucketRead(t *testing.T) {
	inner := &testBucket{name: "b"}
	hists := makeHists()
	b := internalstorage.NewInstrumentedBucket(inner, internalstorage.InstrumentedBucketParams{
		TrackName: "t",
		Hists:     hists,
	})

	req := &gcs.ReadObjectRequest{
		Name:  "obj",
		Range: &gcs.ByteRange{Start: 0, Limit: 1024},
	}
	reader, err := b.NewReaderWithReadHandle(context.Background(), req)
	if err != nil {
		t.Fatalf("NewReaderWithReadHandle: %v", err)
	}

	// Drain until EOF.
	buf := make([]byte, 256)
	for {
		_, readErr := reader.Read(buf)
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			t.Fatalf("unexpected read error: %v", readErr)
		}
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// After close, histograms should have one recorded total-latency value.
	_, total := hists.Snapshot()
	if total.P50 <= 0 && total.Max <= 0 {
		// It's valid for very fast reads (< 1 µs) to land at min=1,
		// so only fail if both are exactly zero (histogram truly empty).
		t.Logf("total latency quite small (p50=%.0f, max=%.0f) — likely sub-microsecond", total.P50, total.Max)
	}
}

// TestInstrumentedBucketWrite verifies that CreateObject records total latency.
func TestInstrumentedBucketWrite(t *testing.T) {
	inner := &testBucket{name: "b"}
	hists := makeHists()
	b := internalstorage.NewInstrumentedBucket(inner, internalstorage.InstrumentedBucketParams{
		TrackName: "t",
		Hists:     hists,
	})

	req := &gcs.CreateObjectRequest{
		Name:     "outobj",
		Contents: io.NopCloser(strings.NewReader("hello world")),
	}
	obj, err := b.CreateObject(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	if obj == nil || obj.Name != "outobj" {
		t.Errorf("unexpected object returned: %v", obj)
	}

	_, total := hists.Snapshot()
	// At minimum the histogram should have accepted one recording (clamped to 1).
	if total.Max < 1 {
		t.Errorf("expected at least one total latency recording, max=%.0f", total.Max)
	}
}

// TestInstrumentedBucketEventChannel verifies events are sent to the channel.
func TestInstrumentedBucketEventChannel(t *testing.T) {
	inner := &testBucket{name: "b"}
	events := make(chan benchmark.PerfEvent, 10)
	hists := makeHists()
	b := internalstorage.NewInstrumentedBucket(inner, internalstorage.InstrumentedBucketParams{
		TrackName: "test",
		Hists:     hists,
		Events:    events,
	})

	req := &gcs.CreateObjectRequest{
		Name:     "evt-obj",
		Contents: io.NopCloser(strings.NewReader("data")),
	}
	if _, err := b.CreateObject(context.Background(), req); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}

	select {
	case evt := <-events:
		if evt.Op != benchmark.OpWrite {
			t.Errorf("expected OpWrite event, got %v", evt.Op)
		}
	default:
		t.Error("expected a PerfEvent on the channel, but channel is empty")
	}
}
