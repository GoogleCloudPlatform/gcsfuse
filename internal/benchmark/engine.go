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

	// trackState holds per-track runtime state.
	trackState []*trackState
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
func NewEngine(bucket gcs.Bucket, bCfg cfg.BenchmarkConfig) (*Engine, error) {
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

	return &Engine{
		bucket:     bucket,
		bCfg:       bCfg,
		trackState: states,
	}, nil
}

// Run executes the full benchmark lifecycle:
//  1. Warm-up (discarded from histograms)
//  2. Measurement phase
//  3. Returns a RunSummary
//
// In "prepare" mode (bCfg.Mode == "prepare") it instead calls runPrepare,
// which writes every object exactly once and returns an empty RunSummary.
func (e *Engine) Run(ctx context.Context) (RunSummary, error) {
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
		if err := e.runPhase(warmupCtx); err != nil && err != context.DeadlineExceeded {
			cancel()
			return RunSummary{}, fmt.Errorf("warmup: %w", err)
		}
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

	if err := e.runPhase(measCtx); err != nil && err != context.DeadlineExceeded {
		return RunSummary{}, fmt.Errorf("measurement: %w", err)
	}
	elapsed := time.Since(start)

	// --- Collect results ---
	summary := RunSummary{
		StartTime:           start,
		MeasurementDuration: elapsed,
		WorkerID:            e.bCfg.WorkerID,
	}
	for _, ts := range e.trackState {
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

// doRead issues a GCS range read and drains the response to measure throughput.
func (e *Engine) doRead(ctx context.Context, ts *trackState, objectName string) error {
	readSize := ts.cfg.ReadSize
	if readSize <= 0 {
		// Default: read 4 MiB per call (a common checkpoint shard size).
		readSize = 4 * 1024 * 1024
	}

	req := &gcs.ReadObjectRequest{
		Name: objectName,
		Range: &gcs.ByteRange{
			Start: 0,
			Limit: uint64(readSize),
		},
	}

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
func (e *Engine) doWrite(ctx context.Context, ts *trackState, rng *rand.Rand, objectName string) error {
	size := sampleObjectSize(rng, ts)
	if size <= 0 {
		size = 4096
	}

	data := make([]byte, size)
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
