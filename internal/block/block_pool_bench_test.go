// Copyright 2026 Google LLC
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

package block

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"

	"golang.org/x/sync/semaphore"
)

const (
	mib = int64(1 << 20)
	// maxBenchBlockSize is 32 MiB, matching GCSFuse's default write-block-size-mb.
	maxBenchBlockSize = 32 * mib
	pageStride        = 4096
)

var (
	benchSrcOnce    sync.Once
	benchSrcPayload []byte
	// benchSink prevents the compiler from eliding block Read and checksum computations.
	benchSink atomic.Uint64
)

func getBenchSrcPayload() []byte {
	benchSrcOnce.Do(func() {
		benchSrcPayload = make([]byte, maxBenchBlockSize)
		for i := range benchSrcPayload {
			benchSrcPayload[i] = byte(i*31 + 17)
		}
	})
	return benchSrcPayload
}

type poolVariant struct {
	name    string
	newPool func(blockSize, maxBlocks, reservedBlocks int64, globalSem *semaphore.Weighted) (BlockPoolInterface[Block], error)
}

func getBenchVariants() []poolVariant {
	return []poolVariant{
		{
			name: "Mmap",
			newPool: func(blockSize, maxBlocks, reservedBlocks int64, globalSem *semaphore.Weighted) (BlockPoolInterface[Block], error) {
				return NewBlockPool(blockSize, maxBlocks, reservedBlocks, globalSem)
			},
		},
		{
			name: "VariantA",
			newPool: func(blockSize, maxBlocks, reservedBlocks int64, globalSem *semaphore.Weighted) (BlockPoolInterface[Block], error) {
				return NewSyncPoolBlockPool(blockSize, maxBlocks, reservedBlocks, globalSem)
			},
		},
		{
			name: "VariantB",
			newPool: func(blockSize, maxBlocks, reservedBlocks int64, globalSem *semaphore.Weighted) (BlockPoolInterface[Block], error) {
				return NewDirectSyncBlockPool(blockSize, maxBlocks, reservedBlocks, globalSem)
			},
		},
	}
}

// touchAndVerifyBlock writes the full blockSize into blk, seeks to 0, reads the full
// blockSize into readBuf, and accumulates a 4 KiB page-strided checksum.
// This guarantees that OS demand-zero page faults (for mmap) and full DRAM write+read
// memory bandwidth are genuinely exercised on every operation.
func touchAndVerifyBlock(b *testing.B, blk Block, src []byte, readBuf []byte) uint64 {
	n, err := blk.Write(src)
	if err != nil || n != len(src) {
		b.Fatalf("blk.Write failed: n=%d, want=%d, err=%v", n, len(src), err)
	}
	if _, err := blk.Seek(0, io.SeekStart); err != nil {
		b.Fatalf("blk.Seek failed: %v", err)
	}
	rn, rErr := blk.Read(readBuf)
	if (rErr != nil && !errors.Is(rErr, io.EOF)) || rn != len(readBuf) {
		b.Fatalf("blk.Read failed: rn=%d, want=%d, err=%v", rn, len(readBuf), rErr)
	}
	var sum uint64
	for off := 0; off < len(readBuf); off += pageStride {
		sum += uint64(readBuf[off])
	}
	return sum
}

type telemetrySnapshot struct {
	rusage    syscall.Rusage
	memStats  runtime.MemStats
	rssBytes  int64
	anonBytes int64
}

func readProcRSSBytes() (rssBytes int64, anonBytes int64) {
	if data, err := os.ReadFile("/proc/self/statm"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 2 {
			if pages, pErr := strconv.ParseInt(fields[1], 10, 64); pErr == nil {
				rssBytes = pages * int64(os.Getpagesize())
			}
		}
	}
	if data, err := os.ReadFile("/proc/self/status"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "RssAnon:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if kb, pErr := strconv.ParseInt(fields[1], 10, 64); pErr == nil {
						anonBytes = kb * 1024
					}
				}
				break
			}
		}
	}
	return rssBytes, anonBytes
}

func takeTelemetrySnapshot() telemetrySnapshot {
	var snap telemetrySnapshot
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &snap.rusage)
	runtime.ReadMemStats(&snap.memStats)
	snap.rssBytes, snap.anonBytes = readProcRSSBytes()
	return snap
}

func timevalToNs(tv syscall.Timeval) int64 {
	return int64(tv.Sec)*1e9 + int64(tv.Usec)*1e3
}

func reportTelemetry(b *testing.B, before, after telemetrySnapshot) {
	ops := float64(b.N)
	if ops <= 0 {
		ops = 1
	}
	minFaults := float64(after.rusage.Minflt-before.rusage.Minflt) / ops
	sysNs := float64(timevalToNs(after.rusage.Stime)-timevalToNs(before.rusage.Stime)) / ops
	gcPauseNs := float64(after.memStats.PauseTotalNs-before.memStats.PauseTotalNs) / ops
	gcCycles := float64(after.memStats.NumGC - before.memStats.NumGC)
	heapInuseMB := float64(after.memStats.HeapInuse) / (1024 * 1024)
	rssMB := float64(after.rssBytes) / (1024 * 1024)
	anonMB := float64(after.anonBytes) / (1024 * 1024)

	b.ReportMetric(minFaults, "minor-faults/op")
	b.ReportMetric(sysNs, "sys-ns/op")
	b.ReportMetric(gcPauseNs, "gc-pause-ns/op")
	b.ReportMetric(gcCycles, "gc-cycles")
	b.ReportMetric(heapInuseMB, "heap-inuse-MB")
	b.ReportMetric(rssMB, "rss-MB")
	b.ReportMetric(anonMB, "anon-rss-MB")
}

func formatBlockSize(size int64) string {
	return fmt.Sprintf("%dMiB", size/mib)
}

func prepareCleanBaseline() {
	ResetSyncPools()
	runtime.GC()
	runtime.GC()
}

// runFileCloseChurnOp executes one complete file lifecycle holding K=2 blocks simultaneously:
// NewPool -> Get K blocks -> Write+Seek+Read each -> Release K blocks -> ClearFreeBlockChannel(true).
func runFileCloseChurnOp(
	b *testing.B,
	v poolVariant,
	blockSize int64,
	globalSem *semaphore.Weighted,
	src []byte,
	readBuf []byte,
) uint64 {
	const blocksPerFile = 2
	const reservedBlocks = 1
	bp, err := v.newPool(blockSize, blocksPerFile, reservedBlocks, globalSem)
	if err != nil {
		b.Fatalf("newPool failed: %v", err)
	}
	var held [blocksPerFile]Block
	var sum uint64
	for k := range blocksPerFile {
		blk, getErr := bp.Get()
		if getErr != nil {
			b.Fatalf("Get failed: %v", getErr)
		}
		sum += touchAndVerifyBlock(b, blk, src, readBuf)
		held[k] = blk
	}
	for k := range blocksPerFile {
		bp.Release(held[k])
	}
	if err := bp.ClearFreeBlockChannel(true); err != nil {
		b.Fatalf("ClearFreeBlockChannel(true) failed: %v", err)
	}
	return sum
}

// runFsyncChurnOp models BufferedWriteHandler.Sync() (ClearFreeBlockChannel(false)) mid-file
// followed by a second wave of writes and Flush() (ClearFreeBlockChannel(true)).
func runFsyncChurnOp(
	b *testing.B,
	v poolVariant,
	blockSize int64,
	globalSem *semaphore.Weighted,
	src []byte,
	readBuf []byte,
) uint64 {
	const blocksPerFile = 2
	const reservedBlocks = 1
	bp, err := v.newPool(blockSize, blocksPerFile, reservedBlocks, globalSem)
	if err != nil {
		b.Fatalf("newPool failed: %v", err)
	}
	var held [blocksPerFile]Block
	var sum uint64

	// Wave 1 prior to fsync
	for k := range blocksPerFile {
		blk, getErr := bp.Get()
		if getErr != nil {
			b.Fatalf("Wave 1 Get failed: %v", getErr)
		}
		sum += touchAndVerifyBlock(b, blk, src, readBuf)
		held[k] = blk
	}
	for k := range blocksPerFile {
		bp.Release(held[k])
	}
	if err := bp.ClearFreeBlockChannel(false); err != nil {
		b.Fatalf("ClearFreeBlockChannel(false) failed: %v", err)
	}

	// Wave 2 after fsync, followed by file close
	for k := range blocksPerFile {
		blk, getErr := bp.Get()
		if getErr != nil {
			b.Fatalf("Wave 2 Get failed: %v", getErr)
		}
		sum += touchAndVerifyBlock(b, blk, src, readBuf)
		held[k] = blk
	}
	for k := range blocksPerFile {
		bp.Release(held[k])
	}
	if err := bp.ClearFreeBlockChannel(true); err != nil {
		b.Fatalf("ClearFreeBlockChannel(true) failed: %v", err)
	}
	return sum
}

// BenchmarkBlockPool_HighFileChurn evaluates Regime 1: repeated file-lifecycle pool creation,
// multi-block acquisition, full-block write/read, and teardown across both FileCloseChurn
// (ClearFreeBlockChannel(true)) and FsyncChurn (ClearFreeBlockChannel(false) + ClearFreeBlockChannel(true)).
func BenchmarkBlockPool_HighFileChurn(b *testing.B) {
	srcFull := getBenchSrcPayload()
	sizes := []int64{1 * mib, 8 * mib, 16 * mib, 32 * mib}
	modes := []struct {
		name       string
		blocksDone int64
		opFn       func(b *testing.B, v poolVariant, blockSize int64, globalSem *semaphore.Weighted, src, readBuf []byte) uint64
	}{
		{name: "FileCloseChurn", blocksDone: 2, opFn: runFileCloseChurnOp},
		{name: "FsyncChurn", blocksDone: 4, opFn: runFsyncChurnOp},
	}

	for _, mode := range modes {
		for _, blockSize := range sizes {
			for _, variant := range getBenchVariants() {
				prepareCleanBaseline()
				subName := fmt.Sprintf("mode=%s/size=%s/variant=%s", mode.name, formatBlockSize(blockSize), variant.name)
				b.Run(subName, func(b *testing.B) {
					src := srcFull[:blockSize]
					readBuf := make([]byte, blockSize)
					globalSem := semaphore.NewWeighted(16)

					// Warmup 1 lifecycle before timing so cross-file pools are in steady churn state
					_ = mode.opFn(b, variant, blockSize, globalSem, src, readBuf)

					b.SetBytes(blockSize * mode.blocksDone)
					before := takeTelemetrySnapshot()
					b.ResetTimer()

					var localSum uint64
					for i := 0; i < b.N; i++ {
						localSum += mode.opFn(b, variant, blockSize, globalSem, src, readBuf)
					}

					b.StopTimer()
					after := takeTelemetrySnapshot()
					benchSink.Add(localSum)
					reportTelemetry(b, before, after)
				})
			}
		}
	}
}

// BenchmarkBlockPool_SteadyStateReuse evaluates Regime 2: continuous Get -> Write -> Seek -> Read -> Release
// loops within an already-warmed pool where blocks are recycled without deallocation.
func BenchmarkBlockPool_SteadyStateReuse(b *testing.B) {
	srcFull := getBenchSrcPayload()
	sizes := []int64{1 * mib, 8 * mib, 16 * mib, 32 * mib}

	for _, blockSize := range sizes {
		for _, variant := range getBenchVariants() {
			prepareCleanBaseline()
			subName := fmt.Sprintf("size=%s/variant=%s", formatBlockSize(blockSize), variant.name)
			b.Run(subName, func(b *testing.B) {
				src := srcFull[:blockSize]
				readBuf := make([]byte, blockSize)
				globalSem := semaphore.NewWeighted(16)

				bp, err := variant.newPool(blockSize, 4, 1, globalSem)
				if err != nil {
					b.Fatalf("newPool failed: %v", err)
				}

				// Pre-warm 2 blocks in the pool outside the timed region
				w1, err := bp.Get()
				if err != nil {
					b.Fatalf("warmup Get 1 failed: %v", err)
				}
				w2, err := bp.Get()
				if err != nil {
					b.Fatalf("warmup Get 2 failed: %v", err)
				}
				_ = touchAndVerifyBlock(b, w1, src, readBuf)
				_ = touchAndVerifyBlock(b, w2, src, readBuf)
				bp.Release(w1)
				bp.Release(w2)

				b.SetBytes(blockSize)
				before := takeTelemetrySnapshot()
				b.ResetTimer()

				var localSum uint64
				for i := 0; i < b.N; i++ {
					blk, getErr := bp.Get()
					if getErr != nil {
						b.Fatalf("Get failed: %v", getErr)
					}
					localSum += touchAndVerifyBlock(b, blk, src, readBuf)
					bp.Release(blk)
				}

				b.StopTimer()
				after := takeTelemetrySnapshot()
				benchSink.Add(localSum)
				_ = bp.ClearFreeBlockChannel(true)
				reportTelemetry(b, before, after)
			})
		}
	}
}

// BenchmarkBlockPool_GCPressure evaluates Regime 3 across three sub-regimes:
// - 3A (ActiveChurn_HeapPressure): File churn alongside concurrent heap allocations and GOGC/GOMEMLIMIT policies.
// - 3B (BurstIdleGCEviction): Burst of blocks followed by ClearFreeBlockChannel(true) and 0, 1, or 2 GC cycles.
// - 3C (PostBurstMemoryRetention): Active burst vs post-ClearFreeBlockChannel(true) RSS/Heap retention.
func BenchmarkBlockPool_GCPressure(b *testing.B) {
	srcFull := getBenchSrcPayload()
	const blockSize = 8 * mib
	src := srcFull[:blockSize]

	// Sub-regime 3A: Active Churn under Concurrent Heap Pressure & GC Policy Sweep
	gcModes := []struct {
		name      string
		gcPercent int
		memLimit  int64
		gcEvery   int
	}{
		{name: "GOGC100", gcPercent: 100, memLimit: -1, gcEvery: 8},
		{name: "GOGC20", gcPercent: 20, memLimit: -1, gcEvery: 4},
		{name: "MemLimit128MiB", gcPercent: 100, memLimit: 128 * mib, gcEvery: 4},
	}

	for _, gm := range gcModes {
		for _, variant := range getBenchVariants() {
			prepareCleanBaseline()
			subName := fmt.Sprintf("subregime=ActiveChurn_HeapPressure/gcMode=%s/size=%s/variant=%s",
				gm.name, formatBlockSize(blockSize), variant.name)
			b.Run(subName, func(b *testing.B) {
				prevGC := debug.SetGCPercent(gm.gcPercent)
				defer debug.SetGCPercent(prevGC)
				if gm.memLimit > 0 {
					prevLimit := debug.SetMemoryLimit(gm.memLimit)
					defer debug.SetMemoryLimit(prevLimit)
				}

				readBuf := make([]byte, blockSize)
				globalSem := semaphore.NewWeighted(16)
				// Rotating live heap ring simulating GCSFuse metadata/stat cache activity
				heapRing := make([][]byte, 16)
				for idx := range heapRing {
					heapRing[idx] = make([]byte, 256*1024)
				}

				_ = runFileCloseChurnOp(b, variant, blockSize, globalSem, src, readBuf)

				b.SetBytes(blockSize * 2)
				before := takeTelemetrySnapshot()
				b.ResetTimer()

				var localSum uint64
				for i := 0; i < b.N; i++ {
					// Allocate 256 KiB of transient heap data per file lifecycle
					scratch := make([]byte, 256*1024)
					scratch[0] = byte(i)
					scratch[len(scratch)-1] = byte(i >> 8)
					heapRing[i%len(heapRing)] = scratch
					localSum += uint64(scratch[0])

					localSum += runFileCloseChurnOp(b, variant, blockSize, globalSem, src, readBuf)

					if (i+1)%gm.gcEvery == 0 {
						runtime.GC()
					}
				}

				b.StopTimer()
				after := takeTelemetrySnapshot()
				benchSink.Add(localSum + uint64(len(heapRing[0])))
				reportTelemetry(b, before, after)
			})
		}
	}

	// Sub-regime 3B: Burst-Idle-Burst GC Eviction Cliff (0 GC vs 1 GC victim rescue vs 2 GC full eviction)
	// Uses K=4 x 8 MiB = 32 MiB bursts per DISPATCH.md.
	runBurst4Op := func(b *testing.B, v poolVariant, blockSize int64, globalSem *semaphore.Weighted, src, readBuf []byte) uint64 {
		const burstBlocks = 4
		bp, err := v.newPool(blockSize, burstBlocks, 1, globalSem)
		if err != nil {
			b.Fatalf("newPool failed: %v", err)
		}
		var held [burstBlocks]Block
		var sum uint64
		for k := range burstBlocks {
			blk, getErr := bp.Get()
			if getErr != nil {
				b.Fatalf("Get failed: %v", getErr)
			}
			sum += touchAndVerifyBlock(b, blk, src, readBuf)
			held[k] = blk
		}
		for k := range burstBlocks {
			bp.Release(held[k])
		}
		if err := bp.ClearFreeBlockChannel(true); err != nil {
			b.Fatalf("ClearFreeBlockChannel(true) failed: %v", err)
		}
		return sum
	}

	for _, gcCount := range []int{0, 1, 2} {
		for _, variant := range getBenchVariants() {
			prepareCleanBaseline()
			subName := fmt.Sprintf("subregime=BurstIdleGCEviction/gcCount=%d/size=%s/variant=%s",
				gcCount, formatBlockSize(blockSize), variant.name)
			b.Run(subName, func(b *testing.B) {
				// Use single-P execution to isolate sync.Pool's exact 1-GC victim cache rescue
				// vs 2-GC victim eviction cliff without cross-P victim.private migration noise.
				prevProcs := runtime.GOMAXPROCS(1)
				defer runtime.GOMAXPROCS(prevProcs)
				prevGC := debug.SetGCPercent(1000)
				defer debug.SetGCPercent(prevGC)

				readBuf := make([]byte, blockSize)
				globalSem := semaphore.NewWeighted(16)

				// Warmup 1 burst + gcCount GCs so the pool state before op 0 matches steady burst-idle state
				_ = runBurst4Op(b, variant, blockSize, globalSem, src, readBuf)
				for g := 0; g < gcCount; g++ {
					runtime.GC()
				}

				b.SetBytes(blockSize * 4)
				before := takeTelemetrySnapshot()
				b.ResetTimer()

				var localSum uint64
				for i := 0; i < b.N; i++ {
					localSum += runBurst4Op(b, variant, blockSize, globalSem, src, readBuf)
					for g := 0; g < gcCount; g++ {
						runtime.GC()
					}
				}

				b.StopTimer()
				after := takeTelemetrySnapshot()
				benchSink.Add(localSum)
				reportTelemetry(b, before, after)
			})
		}
	}

	// Sub-regime 3C: Post-Burst Memory Retention (RSS & HeapInuse during burst vs after ClearFreeBlockChannel)
	retentionPhases := []string{"DuringActiveBurst", "PostClear_NoGC", "PostClear_After2GC"}
	for _, phase := range retentionPhases {
		for _, variant := range getBenchVariants() {
			prepareCleanBaseline()
			subName := fmt.Sprintf("subregime=PostBurstMemoryRetention/phase=%s/size=%s/variant=%s",
				phase, formatBlockSize(blockSize), variant.name)
			b.Run(subName, func(b *testing.B) {
				const burstBlocks = 8 // 8 * 8 MiB = 64 MiB burst
				readBuf := make([]byte, blockSize)
				globalSem := semaphore.NewWeighted(burstBlocks)

				b.SetBytes(blockSize * burstBlocks)
				before := takeTelemetrySnapshot()
				b.ResetTimer()
				var after telemetrySnapshot
				var localSum uint64

				for i := 0; i < b.N; i++ {
					bp, err := variant.newPool(blockSize, burstBlocks, 1, globalSem)
					if err != nil {
						b.Fatalf("newPool failed: %v", err)
					}
					var held [burstBlocks]Block
					for k := range burstBlocks {
						blk, getErr := bp.Get()
						if getErr != nil {
							b.Fatalf("Get failed: %v", getErr)
						}
						localSum += touchAndVerifyBlock(b, blk, src, readBuf)
						held[k] = blk
					}

					if phase == "DuringActiveBurst" && i == b.N-1 {
						b.StopTimer()
						after = takeTelemetrySnapshot()
						b.StartTimer()
					}

					for k := range burstBlocks {
						bp.Release(held[k])
					}
					if err := bp.ClearFreeBlockChannel(true); err != nil {
						b.Fatalf("ClearFreeBlockChannel failed: %v", err)
					}

					if phase == "PostClear_NoGC" && i == b.N-1 {
						b.StopTimer()
						after = takeTelemetrySnapshot()
						b.StartTimer()
					} else if phase == "PostClear_After2GC" && i == b.N-1 {
						runtime.GC()
						runtime.GC()
						debug.FreeOSMemory()
						b.StopTimer()
						after = takeTelemetrySnapshot()
						b.StartTimer()
					}
				}

				b.StopTimer()
				benchSink.Add(localSum)
				reportTelemetry(b, before, after)
			})
		}
	}
}

// BenchmarkBlockPool_Sweep evaluates Regime 4: systematic parameter sweep across
// block sizes (1 MiB, 8 MiB, 16 MiB, 32 MiB), concurrency levels (1, 4, 8, 16),
// workloads (FileChurn, SteadyReuse), and variants (Mmap, VariantA, VariantB).
func BenchmarkBlockPool_Sweep(b *testing.B) {
	srcFull := getBenchSrcPayload()
	sizes := []int64{1 * mib, 8 * mib, 16 * mib, 32 * mib}
	concurrencies := []int{1, 4, 8, 16}
	workloads := []string{"FileChurn", "SteadyReuse"}

	for _, wl := range workloads {
		for _, blockSize := range sizes {
			for _, concurrency := range concurrencies {
				for _, variant := range getBenchVariants() {
					prepareCleanBaseline()
					subName := fmt.Sprintf("workload=%s/size=%s/concurrency=%d/variant=%s",
						wl, formatBlockSize(blockSize), concurrency, variant.name)
					b.Run(subName, func(b *testing.B) {
						src := srcFull[:blockSize]
						// Allow up to 2 blocks per concurrent worker in global semaphore
						globalSem := semaphore.NewWeighted(int64(concurrency * 2))

						if wl == "FileChurn" {
							b.SetBytes(blockSize * 2)
							runSweepFileChurn(b, variant, blockSize, concurrency, globalSem, src)
						} else {
							b.SetBytes(blockSize)
							runSweepSteadyReuse(b, variant, blockSize, concurrency, globalSem, src)
						}
					})
				}
			}
		}
	}
}

func runSweepFileChurn(
	b *testing.B,
	variant poolVariant,
	blockSize int64,
	concurrency int,
	globalSem *semaphore.Weighted,
	src []byte,
) {
	if concurrency == 1 {
		readBuf := make([]byte, blockSize)
		_ = runFileCloseChurnOp(b, variant, blockSize, globalSem, src, readBuf)

		before := takeTelemetrySnapshot()
		b.ResetTimer()

		var localSum uint64
		for i := 0; i < b.N; i++ {
			localSum += runFileCloseChurnOp(b, variant, blockSize, globalSem, src, readBuf)
		}

		b.StopTimer()
		after := takeTelemetrySnapshot()
		benchSink.Add(localSum)
		reportTelemetry(b, before, after)
		return
	}

	var readyWg sync.WaitGroup
	var doneWg sync.WaitGroup
	startCh := make(chan struct{})
	var opIdx atomic.Int64
	totalOps := int64(b.N)

	readyWg.Add(concurrency)
	doneWg.Add(concurrency)

	for range concurrency {
		go func() {
			defer doneWg.Done()
			readBuf := make([]byte, blockSize)
			// Warmup 1 lifecycle on this worker before signalling ready
			_ = runFileCloseChurnOp(b, variant, blockSize, globalSem, src, readBuf)
			readyWg.Done()

			<-startCh
			var workerSum uint64
			for opIdx.Add(1) <= totalOps {
				workerSum += runFileCloseChurnOp(b, variant, blockSize, globalSem, src, readBuf)
			}
			benchSink.Add(workerSum)
		}()
	}

	readyWg.Wait()
	before := takeTelemetrySnapshot()
	b.ResetTimer()
	close(startCh)
	doneWg.Wait()
	b.StopTimer()
	after := takeTelemetrySnapshot()
	reportTelemetry(b, before, after)
}

func runSweepSteadyReuse(
	b *testing.B,
	variant poolVariant,
	blockSize int64,
	concurrency int,
	globalSem *semaphore.Weighted,
	src []byte,
) {
	if concurrency == 1 {
		readBuf := make([]byte, blockSize)
		bp, err := variant.newPool(blockSize, 2, 1, globalSem)
		if err != nil {
			b.Fatalf("newPool failed: %v", err)
		}
		wBlk, err := bp.Get()
		if err != nil {
			b.Fatalf("warmup Get failed: %v", err)
		}
		_ = touchAndVerifyBlock(b, wBlk, src, readBuf)
		bp.Release(wBlk)

		before := takeTelemetrySnapshot()
		b.ResetTimer()

		var localSum uint64
		for i := 0; i < b.N; i++ {
			blk, getErr := bp.Get()
			if getErr != nil {
				b.Fatalf("Get failed: %v", getErr)
			}
			localSum += touchAndVerifyBlock(b, blk, src, readBuf)
			bp.Release(blk)
		}

		b.StopTimer()
		after := takeTelemetrySnapshot()
		benchSink.Add(localSum)
		_ = bp.ClearFreeBlockChannel(true)
		reportTelemetry(b, before, after)
		return
	}

	var readyWg sync.WaitGroup
	var doneWg sync.WaitGroup
	var cleanupWg sync.WaitGroup
	startCh := make(chan struct{})
	cleanupCh := make(chan struct{})
	var opIdx atomic.Int64
	totalOps := int64(b.N)

	readyWg.Add(concurrency)
	doneWg.Add(concurrency)
	cleanupWg.Add(concurrency)

	for range concurrency {
		go func() {
			defer cleanupWg.Done()
			readBuf := make([]byte, blockSize)
			bp, err := variant.newPool(blockSize, 2, 1, globalSem)
			if err != nil {
				b.Errorf("newPool failed: %v", err)
				readyWg.Done()
				doneWg.Done()
				return
			}
			wBlk, err := bp.Get()
			if err != nil {
				b.Errorf("warmup Get failed: %v", err)
				readyWg.Done()
				doneWg.Done()
				return
			}
			_ = touchAndVerifyBlock(b, wBlk, src, readBuf)
			bp.Release(wBlk)
			readyWg.Done()

			<-startCh
			var workerSum uint64
			for opIdx.Add(1) <= totalOps {
				blk, getErr := bp.Get()
				if getErr != nil {
					b.Errorf("Get failed: %v", getErr)
					break
				}
				workerSum += touchAndVerifyBlock(b, blk, src, readBuf)
				bp.Release(blk)
			}
			benchSink.Add(workerSum)
			doneWg.Done()

			// Wait until after telemetry snapshot is taken before deallocating pool
			<-cleanupCh
			_ = bp.ClearFreeBlockChannel(true)
		}()
	}

	readyWg.Wait()
	before := takeTelemetrySnapshot()
	b.ResetTimer()
	close(startCh)
	doneWg.Wait()
	b.StopTimer()
	after := takeTelemetrySnapshot()
	close(cleanupCh)
	cleanupWg.Wait()
	reportTelemetry(b, before, after)
}
