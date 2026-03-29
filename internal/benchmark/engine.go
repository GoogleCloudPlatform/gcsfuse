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

// Package benchmark implements a standalone GCS I/O benchmark engine.
package benchmark

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// Engine drives a benchmark workload against a GCS bucket.
type Engine struct {
	bucket gcs.Bucket
	bCfg   cfg.BenchmarkConfig

	// verbosity mirrors the -v/-vv/-vvv flag count from the CLI.
	// 0 = WARN (silent progress), 1 = INFO, 2 = DEBUG (extra per-interval detail).
	verbosity int

	// trackState holds per-track runtime state.
	trackState []*trackState

	// writeEntropy is a per-engine random seed mixed into every write block's RNG
	// seed. Combined with writeBlockSeq it guarantees that no two 1 MiB blocks
	// written by this engine share the same Xoshiro256++ state.
	writeEntropy uint64

	// writeBlockSeq is a monotonically increasing block counter. Each doWrite
	// call atomically claims a contiguous range of block indices proportional
	// to the object size, so concurrent writers never overlap in seed space.
	// Shared with DataPool.seqCounter so pool and inline fills never collide.
	writeBlockSeq atomic.Uint64

	// writePool pre-fills write buffers in a background goroutine using
	// Xoshiro256++ so that doWrite can upload without blocking on data
	// generation.  nil when disabled (e.g. no write tracks, or object size
	// exceeds poolSlotSizeCap).
	writePool *DataPool
}

// trackState holds mutable per-track counters and histograms.
type trackState struct {
	cfg        cfg.BenchmarkTrack
	hists      *TrackHistograms
	totalOps   atomic.Int64
	totalErrs  atomic.Int64
	totalBytes atomic.Int64

	// objectPaths is the pre-built list of all object names for this track.
	// Built from DirectoryStructure if set, otherwise from flat ObjectCount.
	objectPaths []string

	// lnMu and lnSigma are the precomputed lognormal parameters (log-space)
	// derived from SizeSpec.Mean and SizeSpec.StdDev.
	lnMu    float64
	lnSigma float64
}

// NewEngine creates a benchmark Engine backed by the provided gcs.Bucket.
// verbosity mirrors the CLI -v/-vv/-vvv count (0 = silent progress, 1 = INFO,
// 2 = DEBUG with extra per-interval detail).
func NewEngine(bucket gcs.Bucket, bCfg cfg.BenchmarkConfig, verbosity int) (*Engine, error) {
	if bucket == nil {
		return nil, fmt.Errorf("bucket must not be nil")
	}
	if len(bCfg.Tracks) == 0 {
		return nil, fmt.Errorf("at least one benchmark track must be configured")
	}

	states := make([]*trackState, len(bCfg.Tracks))
	for i, track := range bCfg.Tracks {
		ts := &trackState{
			cfg:         track,
			hists:       NewTrackHistograms(bCfg.Histograms),
			objectPaths: buildObjectPaths(bCfg.ObjectPrefix, track),
		}
		// Precompute lognormal distribution parameters if SizeSpec requests it.
		if track.SizeSpec != nil && strings.ToLower(track.SizeSpec.Type) == "lognormal" {
			ts.lnMu, ts.lnSigma = lognormalParams(track.SizeSpec.Mean, track.SizeSpec.StdDev)
		}
		states[i] = ts
	}

	entropy := newWriteEntropy()
	eng := &Engine{
		bucket:       bucket,
		bCfg:         bCfg,
		verbosity:    verbosity,
		trackState:   states,
		writeEntropy: entropy,
	}

	// Build write pool if any track writes fixed-size (or bounded) objects
	// that fit within poolSlotSizeCap.  Pool pre-generates data in a background
	// goroutine so doWrite never stalls on Xoshiro256++ fill.
	//
	// poolSlotSizeCap prevents the pool from consuming excessive RAM.
	// At 512 MiB × depth-32 = 16 GiB — feasible on 32 GiB machines.
	// Increase or make configurable if needed.
	const poolSlotSizeCap int64 = 512 << 20 // 512 MiB
	eng.writePool = buildWritePool(bCfg, poolSlotSizeCap, entropy, &eng.writeBlockSeq)

	return eng, nil
}

// buildWritePool examines the benchmark config and, if any write track has
// objects small enough to pre-fill (≤ slotSizeCap), allocates and returns a
// DataPool sized to the largest such write object.  Returns nil if no write
// tracks are found or all objects exceed the cap (they use the inline path).
func buildWritePool(bCfg cfg.BenchmarkConfig, slotSizeCap int64, entropy uint64, seqCounter *atomic.Uint64) *DataPool {
	var maxWriteSize int64
	for _, track := range bCfg.Tracks {
		if strings.ToLower(track.OpType) != "write" {
			continue
		}
		var sz int64
		switch {
		case track.SizeSpec != nil && strings.ToLower(track.SizeSpec.Type) == "lognormal":
			// Conservative upper bound: Mean + 3σ covers 99.87% of lognormal draws.
			// Clamp to SizeSpec.Max if set.
			sz = int64(track.SizeSpec.Mean + 3*track.SizeSpec.StdDev)
			if track.SizeSpec.Max > 0 && sz > track.SizeSpec.Max {
				sz = track.SizeSpec.Max
			}
		case track.SizeSpec != nil:
			sz = track.SizeSpec.Max
			if sz <= 0 {
				sz = track.SizeSpec.Min
			}
		default:
			sz = track.ObjectSizeMax
		}
		if sz > maxWriteSize {
			maxWriteSize = sz
		}
	}

	if maxWriteSize <= 0 || maxWriteSize > slotSizeCap {
		// Either no write tracks or objects too large — pool disabled.
		return nil
	}

	// Ring depth: enough slots to keep the producer ahead of all concurrent writers.
	// 4 × TotalConcurrency is generous; clamped to ≥ 32.
	depth := bCfg.TotalConcurrency * 4
	if depth < 32 {
		depth = 32
	}

	return newDataPool(maxWriteSize, depth, entropy, seqCounter)
}

// Run executes the full benchmark lifecycle:
//  1. Warm-up (discarded from histograms)
//  2. Measurement phase
//  3. Returns a RunSummary
//
// In "prepare" mode (bCfg.Mode == "prepare") it instead calls runPrepare,
// which writes every object exactly once and returns an empty RunSummary.
func (e *Engine) Run(ctx context.Context) (RunSummary, error) {
	// Start the write-data producer goroutine if the pool was created.
	// Use a child context so the producer stops when Run returns (and cancel
	// is deferred), regardless of whether the parent ctx is still alive.
	poolCtx, stopPool := context.WithCancel(ctx)
	defer stopPool()
	if e.writePool != nil {
		go e.writePool.RunProducer(poolCtx)
	}

	// Synchronized start: sleep until the requested wall-clock time so that
	// multiple distributed workers begin their warm-up simultaneously.
	if e.bCfg.StartAt > 0 {
		target := time.Unix(e.bCfg.StartAt, 0)
		if wait := time.Until(target); wait > 0 {
			logger.Infof("Waiting %s for synchronized start at %s...\n",
				wait.Round(time.Second), target.UTC().Format(time.RFC3339))
			select {
			case <-ctx.Done():
				return RunSummary{}, ctx.Err()
			case <-time.After(wait):
			}
		}
	}

	if strings.ToLower(e.bCfg.Mode) == "prepare" {
		prepStart := time.Now()
		prepErr := e.runPrepare(ctx)
		prepElapsed := time.Since(prepStart)
		// Build a summary so results are exported like any other run.
		prepSummary := RunSummary{
			StartTime:           prepStart,
			MeasurementDuration: prepElapsed,
			WorkerID:            e.bCfg.WorkerID,
		}
		for _, ts := range e.trackState {
			ops := ts.totalOps.Load()
			errs := ts.totalErrs.Load()
			byts := ts.totalBytes.Load()
			var opsPerSec, throughput float64
			if prepElapsed > 0 {
				opsPerSec = float64(ops) / prepElapsed.Seconds()
				throughput = float64(byts) / prepElapsed.Seconds()
			}
			successfulOps := ops - errs
			var avgOpSize float64
			if successfulOps > 0 {
				avgOpSize = float64(byts) / float64(successfulOps)
			}
			ttfb, total := ts.hists.Snapshot()
			prepSummary.Tracks = append(prepSummary.Tracks, TrackStats{
				TrackName:             ts.cfg.Name,
				OpType:                "write",
				WorkerID:              e.bCfg.WorkerID,
				TotalOps:              ops,
				Errors:                errs,
				ThroughputBytesPerSec: throughput,
				OpsPerSec:             opsPerSec,
				AvgOpSizeBytes:        avgOpSize,
				TTFB:                  ttfb,
				TotalLatency:          total,
			})
		}
		return prepSummary, prepErr
	}
	// --- Warm-up ---
	if e.bCfg.WarmupDuration > 0 {
		logger.Infof("Warming up for %s...\n", e.bCfg.WarmupDuration)
		warmupCtx, cancel := context.WithTimeout(ctx, e.bCfg.WarmupDuration)
		stopWarmupProgress := e.startProgressReporter(warmupCtx, "warmup", e.bCfg.WarmupDuration)
		if err := e.runPhase(warmupCtx); err != nil && err != context.DeadlineExceeded {
			stopWarmupProgress()
			cancel()
			return RunSummary{}, fmt.Errorf("warmup: %w", err)
		}
		stopWarmupProgress()
		cancel()

		// Reset histograms and counters so warmup data is discarded.
		for _, ts := range e.trackState {
			ts.hists.Reset()
			ts.totalOps.Store(0)
			ts.totalErrs.Store(0)
			ts.totalBytes.Store(0)
		}
	}

	// --- Measurement ---
	logger.Infof("Measuring for %s...\n", e.bCfg.Duration)
	start := time.Now()
	measCtx, cancel := context.WithTimeout(ctx, e.bCfg.Duration)
	defer cancel()

	stopMeasProgress := e.startProgressReporter(measCtx, "bench", e.bCfg.Duration)
	if err := e.runPhase(measCtx); err != nil && err != context.DeadlineExceeded {
		stopMeasProgress()
		return RunSummary{}, fmt.Errorf("measurement: %w", err)
	}
	stopMeasProgress()
	elapsed := time.Since(start)

	// Log pool pipeline final summary after measurement completes.
	if e.writePool != nil && e.verbosity >= 1 {
		logPoolFinalSummary(e.writePool, elapsed)
	}

	// --- Collect results ---
	summary := RunSummary{
		StartTime:           start,
		MeasurementDuration: elapsed,
		WorkerID:            e.bCfg.WorkerID,
	}
	for i, ts := range e.trackState {
		ttfb, total := ts.hists.Snapshot()
		ops := ts.totalOps.Load()
		errs := ts.totalErrs.Load()
		bytes := ts.totalBytes.Load()

		var opsPerSec, throughput float64
		if elapsed > 0 {
			opsPerSec = float64(ops) / elapsed.Seconds()
			throughput = float64(bytes) / elapsed.Seconds()
		}

		successfulOps := ops - errs
		var avgOpSize float64
		if successfulOps > 0 {
			avgOpSize = float64(bytes) / float64(successfulOps)
		}

		stat := TrackStats{
			TrackName:             ts.cfg.Name,
			OpType:                ts.cfg.OpType,
			WorkerID:              e.bCfg.WorkerID,
			Goroutines:            goroutinesForTrack(e.bCfg, i),
			TotalOps:              ops,
			Errors:                errs,
			ThroughputBytesPerSec: throughput,
			OpsPerSec:             opsPerSec,
			AvgOpSizeBytes:        avgOpSize,
			TTFB:                  ttfb,
			TotalLatency:          total,
		}
		// Export raw histogram snapshots when running as part of a multi-worker
		// distributed benchmark so that 'gcs-bench merge-results' can merge them.
		if e.bCfg.NumWorkers > 1 {
			rawTTFB, rawTotal, err := ts.hists.ExportBase64()
			if err != nil {
				logger.Warnf("warning: could not export histograms for track %q: %v\n", ts.cfg.Name, err)
			} else {
				stat.RawTTFB = rawTTFB
				stat.RawTotal = rawTotal
			}
		}
		summary.Tracks = append(summary.Tracks, stat)
	}
	return summary, nil
}

// runPhase runs goroutines for each track until ctx is done.
func (e *Engine) runPhase(ctx context.Context) error {
	var wg sync.WaitGroup

	for i, ts := range e.trackState {
		concurrency := goroutinesForTrack(e.bCfg, i)
		for j := 0; j < concurrency; j++ {
			wg.Add(1)
			go func(ts *trackState) {
				defer wg.Done()
				e.runWorker(ctx, ts)
			}(ts)
		}
	}

	wg.Wait()
	return ctx.Err()
}

// goroutinesForTrack computes the number of goroutines for track i.
// If the track has an explicit Concurrency > 0 that is used directly.
// Otherwise concurrency is distributed proportionally by Weight.
func goroutinesForTrack(bCfg cfg.BenchmarkConfig, idx int) int {
	track := bCfg.Tracks[idx]
	if track.Concurrency > 0 {
		return track.Concurrency
	}
	if bCfg.TotalConcurrency <= 0 || len(bCfg.Tracks) == 0 {
		return 1
	}

	// Sum weights.
	totalWeight := 0
	for _, t := range bCfg.Tracks {
		totalWeight += t.Weight
	}
	if totalWeight <= 0 {
		return bCfg.TotalConcurrency / len(bCfg.Tracks)
	}
	weight := track.Weight
	if weight <= 0 {
		weight = 1
	}
	c := (bCfg.TotalConcurrency * weight) / totalWeight
	if c < 1 {
		c = 1
	}
	return c
}

// runWorker runs the I/O loop for a single goroutine until ctx is cancelled.
func (e *Engine) runWorker(ctx context.Context, ts *trackState) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		objectName := e.pickObjectFromState(rng, ts)
		var err error
		switch strings.ToLower(ts.cfg.OpType) {
		case "read":
			err = e.doRead(ctx, ts, objectName)
		case "write":
			err = e.doWrite(ctx, ts, rng, objectName)
		case "stat":
			err = e.doStat(ctx, ts, objectName)
		case "list":
			err = e.doList(ctx, ts, objectName)
		default:
			// Unsupported op; skip and continue.
		}
		if err != nil {
			// Errors caused by context cancellation are expected at phase
			// transitions (warmup→measurement, end-of-run) when in-flight
			// gRPC calls are interrupted. Don't count or log them.
			if ctx.Err() != nil {
				return
			}
			n := ts.totalErrs.Add(1)
			if n <= 3 {
				logger.Warnf("[bench] track=%q error #%d: %v\n", ts.cfg.Name, n, err)
			}
		}
		ts.totalOps.Add(1)
	}
}

// pickObjectFromState returns an object name from this track's pre-built path
// list. Both random and sequential access patterns use uniform random selection
// across the goroutine pool (true per-worker sequential cursors would require
// additional synchronisation and are not yet implemented).
func (e *Engine) pickObjectFromState(rng *rand.Rand, ts *trackState) string {
	n := len(ts.objectPaths)
	if n == 0 {
		return ""
	}
	return ts.objectPaths[rng.Intn(n)]
}

// doRead issues a GCS read and drains the response to measure throughput.
// When ReadSize <= 0, the entire object is read (no Range restriction).
// When ReadSize > 0, a range-read of exactly ReadSize bytes is performed.
func (e *Engine) doRead(ctx context.Context, ts *trackState, objectName string) error {
	readSize := ts.cfg.ReadSize

	req := &gcs.ReadObjectRequest{
		Name: objectName,
	}
	if readSize > 0 {
		req.Range = &gcs.ByteRange{
			Start: 0,
			Limit: uint64(readSize),
		}
	}
	// readSize <= 0: Range left nil → GCS returns the entire object.

	start := time.Now()
	reader, err := e.bucket.NewReaderWithReadHandle(ctx, req)
	if err != nil {
		return fmt.Errorf("NewReaderWithReadHandle: %w", err)
	}
	defer reader.Close()

	buf := make([]byte, 256*1024) // 256 KiB read buffer
	var bytesRead int64
	ttfbRecorded := false
	for {
		n, readErr := reader.Read(buf)
		// Record TTFB on the first non-empty result (or on any first call).
		if !ttfbRecorded {
			ts.hists.RecordTTFB(time.Since(start).Microseconds())
			ttfbRecorded = true
		}
		bytesRead += int64(n)
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			ts.totalErrs.Add(1)
			return fmt.Errorf("read: %w", readErr)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	ts.hists.RecordTotal(time.Since(start).Microseconds())
	ts.totalBytes.Add(bytesRead)
	return nil
}

// ---------------------------------------------------------------------------
// Object path building helpers
// ---------------------------------------------------------------------------

// buildObjectPaths returns the complete list of object GCS names for a track.
// When DirectoryStructure is set it generates a nested tree path list;
// otherwise it falls back to the flat "prefix+trackName-NNNNNNNN" model.
func buildObjectPaths(prefix string, track cfg.BenchmarkTrack) []string {
	if track.DirectoryStructure != nil {
		ds := track.DirectoryStructure
		dirPat := ds.DirPattern
		if dirPat == "" {
			dirPat = "dir-%04d"
		}
		filePat := ds.FilePattern
		if filePat == "" {
			filePat = "obj-%06d"
		}
		leaves := buildLeafDirs(prefix, ds.Width, ds.Depth, dirPat)
		paths := make([]string, 0, len(leaves)*ds.FilesPerDir)
		for _, dir := range leaves {
			for f := 0; f < ds.FilesPerDir; f++ {
				paths = append(paths, dir+fmt.Sprintf(filePat, f))
			}
		}
		return paths
	}
	// Flat model: one object per index.
	count := track.ObjectCount
	if count <= 0 {
		count = 1
	}
	paths := make([]string, count)
	for i := range paths {
		paths[i] = fmt.Sprintf("%s%s-%08d", prefix, track.Name, i)
	}
	return paths
}

// buildLeafDirs recursively builds the set of leaf directory paths.
// Each path ends with a trailing slash and is ready to have file names appended.
func buildLeafDirs(prefix string, width, depth int, pattern string) []string {
	if depth == 0 {
		return []string{prefix}
	}
	result := make([]string, 0)
	for i := 0; i < width; i++ {
		seg := fmt.Sprintf(pattern, i) + "/"
		sub := buildLeafDirs(prefix+seg, width, depth-1, pattern)
		result = append(result, sub...)
	}
	return result
}

// ---------------------------------------------------------------------------
// Size sampling helpers
// ---------------------------------------------------------------------------

// lognormalParams converts real-space mean m and standard deviation s into
// the log-space μ and σ parameters of the underlying normal distribution.
// Formula:
//
//	σ_ln = sqrt(log(1 + (s/m)²))
//	μ_ln = log(m) − 0.5·σ_ln²
func lognormalParams(mean, stdDev float64) (mu, sigma float64) {
	if mean <= 0 {
		mean = 1
	}
	if stdDev < 0 {
		stdDev = 0
	}
	cv := stdDev / mean
	sigma = math.Sqrt(math.Log(1 + cv*cv))
	mu = math.Log(mean) - 0.5*sigma*sigma
	return mu, sigma
}

// sampleObjectSize returns an object size in bytes sampled from the track's
// configured distribution. Falls back to ObjectSizeMin/Max if no SizeSpec.
func sampleObjectSize(rng *rand.Rand, ts *trackState) int64 {
	spec := ts.cfg.SizeSpec
	if spec == nil {
		// Uniform distribution between ObjectSizeMin and ObjectSizeMax.
		min := ts.cfg.ObjectSizeMin
		max := ts.cfg.ObjectSizeMax
		if max <= min {
			if min > 0 {
				return min
			}
			return 4096
		}
		return min + rng.Int63n(max-min+1)
	}
	switch strings.ToLower(spec.Type) {
	case "fixed":
		return spec.Min
	case "lognormal":
		z := rng.NormFloat64()
		sz := int64(math.Round(math.Exp(ts.lnMu + ts.lnSigma*z)))
		if spec.Min > 0 && sz < spec.Min {
			sz = spec.Min
		}
		if spec.Max > 0 && sz > spec.Max {
			sz = spec.Max
		}
		return sz
	default: // "uniform"
		min := spec.Min
		max := spec.Max
		if max <= min {
			return min
		}
		return min + rng.Int63n(max-min+1)
	}
}

// doWrite creates an object to benchmark write latency.
// Object size is sampled from SizeSpec (if set) or the ObjectSizeMin/Max range.
// Every byte written is unique incompressible data generated by Xoshiro256++ —
// never zeroes, never a static buffer.
//
// When the write pool is active and the object fits within a pre-filled slot,
// doWrite acquires a slot (zero heap allocation, no inline fill) and streams
// the pre-generated bytes directly to GCS.  Larger objects that exceed the
// pool's slot capacity use the inline fill path (make+fillRandom) as before.
func (e *Engine) doWrite(ctx context.Context, ts *trackState, rng *rand.Rand, objectName string) error {
	size := sampleObjectSize(rng, ts)
	if size <= 0 {
		size = 4096
	}

	// Pool fast path: pre-filled slot available and large enough.
	// Zero heap allocation, zero inline fill in the critical path.
	if e.writePool != nil && size <= e.writePool.slotSize {
		slotIdx, data, err := e.writePool.AcquireSlot(ctx, size)
		if err != nil {
			return err // context cancelled
		}
		req := &gcs.CreateObjectRequest{
			Name:     objectName,
			Contents: io.NopCloser(bytes.NewReader(data)),
		}
		start := time.Now()
		obj, err := e.bucket.CreateObject(ctx, req)
		elapsed := time.Since(start)
		// Release the slot back to the producer immediately — GCS has already
		// copied the data from the io.Reader; we no longer need it.
		e.writePool.ReleaseSlot(slotIdx)
		if err != nil {
			return fmt.Errorf("CreateObject: %w", err)
		}
		if obj != nil {
			ts.totalBytes.Add(int64(obj.Size))
		}
		ts.hists.RecordTotal(elapsed.Microseconds())
		return nil
	}

	// Inline fallback: object too large for pool (or pool disabled).
	// Claim a contiguous range of block indices for this write so that no two
	// concurrent writers share a block seed. nblocks is rounded up to 1 MiB
	// boundaries, matching genBlockSize in datagen.go.
	nblocks := uint64((size + genBlockSize - 1) / genBlockSize)
	startSeq := e.writeBlockSeq.Add(nblocks) - nblocks

	// Allocate buffer and fill with Xoshiro256++ pseudo-random bytes.
	// Parallel goroutines fill independent 1 MiB sub-slices directly (zero intermediary copy).
	data := make([]byte, size)
	fillRandom(data, e.writeEntropy, startSeq)

	req := &gcs.CreateObjectRequest{
		Name:     objectName,
		Contents: io.NopCloser(bytes.NewReader(data)),
	}

	start := time.Now()
	obj, err := e.bucket.CreateObject(ctx, req)
	elapsed := time.Since(start)
	if err != nil {
		return fmt.Errorf("CreateObject: %w", err)
	}
	if obj != nil {
		ts.totalBytes.Add(int64(obj.Size))
	}
	// Record write total latency. Writes have no TTFB in the read sense.
	ts.hists.RecordTotal(elapsed.Microseconds())
	return nil
}

// doStat issues a GCS metadata-only HEAD request for objectName.
func (e *Engine) doStat(ctx context.Context, ts *trackState, objectName string) error {
	req := &gcs.StatObjectRequest{
		Name:              objectName,
		ForceFetchFromGcs: true,
	}
	start := time.Now()
	minObj, _, err := e.bucket.StatObject(ctx, req)
	elapsed := time.Since(start)
	if err != nil {
		return fmt.Errorf("StatObject: %w", err)
	}
	if minObj != nil {
		ts.totalBytes.Add(int64(minObj.Size))
	}
	ts.hists.RecordTotal(elapsed.Microseconds())
	return nil
}

// doList issues a GCS list request scoped to the leaf directory that contains
// objectName. It counts the returned objects for throughput accounting but does
// not download their contents.
func (e *Engine) doList(ctx context.Context, ts *trackState, objectName string) error {
	// Derive the list prefix from the object's directory (strip the filename).
	prefix := e.bCfg.ObjectPrefix
	if idx := strings.LastIndex(objectName, "/"); idx >= 0 {
		prefix = objectName[:idx+1]
	}
	req := &gcs.ListObjectsRequest{
		Prefix:     prefix,
		MaxResults: 1000,
	}
	start := time.Now()
	listing, err := e.bucket.ListObjects(ctx, req)
	elapsed := time.Since(start)
	if err != nil {
		return fmt.Errorf("ListObjects: %w", err)
	}
	if listing != nil {
		ts.totalBytes.Add(int64(len(listing.MinObjects)))
	}
	ts.hists.RecordTotal(elapsed.Microseconds())
	return nil
}

// runPrepare writes every object in each track's path list exactly once, then
// returns. It is the implementation for Mode == "prepare".
//
// Host-level sharding: when NumWorkers > 1, each host processes only the
// objects at indices where (i % NumWorkers) == WorkerID. This ensures each
// object is written by exactly one host with no coordination required.
//
// Within the host's shard, paths are distributed across goroutines using a
// round-robin stride so that writes are interleaved (goroutine g handles the
// shard's g-th, (g+concurrency)-th, ... entries).
func (e *Engine) runPrepare(ctx context.Context) error {
	for tIdx, ts := range e.trackState {
		opType := strings.ToLower(ts.cfg.OpType)
		if opType != "write" {
			logger.Infof("[prepare] Track %q: op-type=%q — skipping (only 'write' runs in prepare mode)\n",
				ts.cfg.Name, ts.cfg.OpType)
			continue
		}

		// --- Host-level shard selection ---
		numWorkers := e.bCfg.NumWorkers
		workerID := e.bCfg.WorkerID
		if numWorkers <= 1 {
			numWorkers = 1
			workerID = 0
		}

		// Collect the subset of paths that belong to this worker.
		allPaths := ts.objectPaths
		var shardPaths []string
		for i, p := range allPaths {
			if i%numWorkers == workerID {
				shardPaths = append(shardPaths, p)
			}
		}

		total := len(shardPaths)
		if total == 0 {
			logger.Infof("[prepare] Track %q: no objects assigned to worker %d/%d — skipping\n",
				ts.cfg.Name, workerID, numWorkers)
			continue
		}

		concurrency := goroutinesForTrack(e.bCfg, tIdx)
		if numWorkers > 1 {
			logger.Infof("[prepare] Track %q: worker %d/%d — writing %d/%d objects with %d goroutines...\n",
				ts.cfg.Name, workerID, numWorkers, total, len(allPaths), concurrency)
		} else {
			logger.Infof("[prepare] Track %q: writing %d objects with %d goroutines...\n",
				ts.cfg.Name, total, concurrency)
		}

		var written, writeErrs atomic.Int64
		start := time.Now()

		// Progress reporter — cancelled explicitly after wg.Wait().
		progressCtx, progressCancel := context.WithCancel(ctx)
		go func(total int) {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					n := written.Load()
					errs := writeErrs.Load()
					elapsed := time.Since(start).Seconds()
					rate := float64(n) / elapsed
					pct := float64(n) / float64(total) * 100
					logger.Infof("  [prepare] %d/%d (%.0f%%)  %.0f obj/s  %d errors\n",
						n, total, pct, rate, errs)
				case <-progressCtx.Done():
					return
				}
			}
		}(total)

		// Distribute shard paths across goroutines via round-robin stride.
		var wg sync.WaitGroup
		for g := 0; g < concurrency; g++ {
			wg.Add(1)
			go func(goroutineIdx int) {
				defer wg.Done()
				rng := rand.New(rand.NewSource(int64(workerID*1000 + goroutineIdx + 1)))
				for i := goroutineIdx; i < len(shardPaths); i += concurrency {
					if ctx.Err() != nil {
						return
					}
					if err := e.doWrite(ctx, ts, rng, shardPaths[i]); err != nil {
						n := writeErrs.Add(1)
						ts.totalErrs.Add(1)
						if n <= 3 {
							logger.Warnf("[prepare] write error #%d on %q: %v\n", n, shardPaths[i], err)
						}
					} else {
						written.Add(1)
					}
					ts.totalOps.Add(1)
				}
			}(g)
		}
		wg.Wait()
		progressCancel()

		elapsed := time.Since(start)
		n := written.Load()
		errs := writeErrs.Load()
		logger.Infof("[prepare] Track %q: complete — %d/%d written in %s (%.0f obj/s, %d errors)\n",
			ts.cfg.Name, n, total, elapsed.Round(time.Second),
			float64(n)/elapsed.Seconds(), errs)
	}
	return nil
}

// startProgressReporter launches a background goroutine that logs throughput
// progress every 10 seconds for the given phase ("warmup" or "bench").
// The verbosity level controls detail:
//   - verbosity >= 1 (–v):  INFO line every 10 s with interval ops + GiB/s
//   - verbosity >= 2 (–vv): also logs elapsed/remaining and per-track error count
//
// Returns a stop function; call it after the phase completes to ensure the
// goroutine is torn down cleanly (in addition to ctx cancellation).
func (e *Engine) startProgressReporter(ctx context.Context, phase string, total time.Duration) func() {
	stopCh := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		type snap struct{ bytes, ops, errs int64 }
		last := make([]snap, len(e.trackState))
		lastAt := time.Now()
		phaseStart := time.Now()
		var lastPool PoolStats // zero on first tick; first interval gives cumulative-since-start rates
		for {
			select {
			case <-ctx.Done():
				return
			case <-stopCh:
				return
			case t := <-ticker.C:
				interval := t.Sub(lastAt).Seconds()
				lastAt = t
				elapsed := t.Sub(phaseStart)
				remaining := total - elapsed
				if remaining < 0 {
					remaining = 0
				}
				for i, ts := range e.trackState {
					cur := snap{
						bytes: ts.totalBytes.Load(),
						ops:   ts.totalOps.Load(),
						errs:  ts.totalErrs.Load(),
					}
					dBytes := float64(cur.bytes - last[i].bytes)
					dOps := cur.ops - last[i].ops
					last[i] = cur
					tputGiB := (dBytes / interval) / (1024 * 1024 * 1024)

					if e.verbosity >= 2 {
						// -vv: extended format with elapsed/remaining via logger
						logger.Infof("[%s] track=%q  elapsed=%s  remaining=%s  interval-ops=%d  interval-throughput=%.2f GiB/s  total-ops=%d  total-errs=%d\n",
							phase, ts.cfg.Name,
							elapsed.Round(time.Second), remaining.Round(time.Second),
							dOps, tputGiB, cur.ops, cur.errs)
					} else {
						// Default: always print compact line directly to stderr, bypassing
						// the logger severity filter so progress is visible without -v.
						fmt.Fprintf(os.Stderr, "[%s] track=%q  interval-ops=%d  interval-throughput=%.2f GiB/s  total-ops=%d\n",
							phase, ts.cfg.Name, dOps, tputGiB, cur.ops)
					}
				}
				// Pool pipeline health: produce/consume rates and coordination overhead.
				// Reported every tick at verbosity ≥ 1 (-v) when the write pool is active.
				// Producer-stall%: fraction of wall time the producer waited for an EMPTY slot
				//   (ring too shallow — consumers fill all slots before producer refills them).
				// Consumer-stall%: cumulative goroutine-% consumers spent waiting for READY slots
				//   (producer too slow — the actual bottleneck condition to avoid).
				if e.writePool != nil && e.verbosity >= 1 {
					pCur := e.writePool.Stats()
					dProd := float64(pCur.BytesProduced - lastPool.BytesProduced)
					dCons := float64(pCur.BytesConsumed - lastPool.BytesConsumed)
					prodGiB := (dProd / interval) / (1024 * 1024 * 1024)
					consGiB := (dCons / interval) / (1024 * 1024 * 1024)
					var ratio float64
					if dCons > 0 {
						ratio = dProd / dCons
					}
					intervalNs := interval * 1e9
					producerStallPct := float64(pCur.ProducerStallNs-lastPool.ProducerStallNs) / intervalNs * 100
					consumerStallPct := float64(pCur.ConsumerStallNs-lastPool.ConsumerStallNs) / intervalNs * 100
					lastPool = pCur
					if ratio > 0 && ratio < 1.05 {
						logger.Warnf("[pool] WARNING produce/consume ratio=%.3f (<5%% headroom) — generation may bottleneck writes!  produce=%.2f GiB/s  consume=%.2f GiB/s  producer-stall=%.2f%%  consumer-stall=%.2f%%\n",
							ratio, prodGiB, consGiB, producerStallPct, consumerStallPct)
					} else {
						logger.Infof("[pool] produce=%.2f GiB/s  consume=%.2f GiB/s  ratio=%.3f  producer-stall=%.2f%%  consumer-stall=%.2f%%\n",
							prodGiB, consGiB, ratio, producerStallPct, consumerStallPct)
					}
				}
			}
		}
	}()
	return func() { close(stopCh) }
}

// logPoolFinalSummary logs overall produce/consume rates, the ratio, and
// cumulative stall times for the DataPool pipeline at the end of a benchmark
// measurement phase.  A WARNING is emitted when the ratio is within 5% of 1:1,
// indicating that data generation may have been the write bottleneck.
func logPoolFinalSummary(pool *DataPool, elapsed time.Duration) {
	s := pool.Stats()
	prodGiB := float64(s.BytesProduced) / (1024 * 1024 * 1024)
	consGiB := float64(s.BytesConsumed) / (1024 * 1024 * 1024)
	elapsedS := elapsed.Seconds()
	prodRate := prodGiB / elapsedS
	consRate := consGiB / elapsedS
	var ratio float64
	if s.BytesConsumed > 0 {
		ratio = float64(s.BytesProduced) / float64(s.BytesConsumed)
	}
	producerStallS := float64(s.ProducerStallNs) / 1e9
	consumerStallS := float64(s.ConsumerStallNs) / 1e9
	producerStallPct := producerStallS / elapsedS * 100
	logger.Infof("[pool] final: produced=%.2f GiB (%.2f GiB/s)  consumed=%.2f GiB (%.2f GiB/s)  ratio=%.3f  producer-stall=%.3fs (%.2f%%)  consumer-stall=%.3fs\n",
		prodGiB, prodRate, consGiB, consRate, ratio, producerStallS, producerStallPct, consumerStallS)
	if ratio > 0 && ratio < 1.05 {
		logger.Warnf("[pool] WARNING: overall produce/consume ratio=%.3f is within 5%% of 1:1 — data generation was likely the write bottleneck during this run\n", ratio)
	}
}

// Histograms returns the per-track histogram objects after a completed Run().
// The returned slice is in the same order as RunSummary.Tracks.
// Call only after Run() has returned.
func (e *Engine) Histograms() []*TrackHistograms {
	result := make([]*TrackHistograms, len(e.trackState))
	for i, ts := range e.trackState {
		result[i] = ts.hists
	}
	return result
}
