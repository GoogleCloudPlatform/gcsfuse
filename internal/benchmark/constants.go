// Package benchmark — named constants for all tuning, sizing, and behavioral
// parameters.  Every numeric literal that governs pool geometry, progress
// intervals, error thresholds, or unit conversions must be referenced here so
// that a single edit changes the behaviour everywhere the value is used.
//
// Cryptographic / algorithmic constants (Xoshiro256++ magic numbers, bit
// rotation counts, etc.) live in datagen.go alongside the algorithm they
// parametrise and are intentionally excluded from this file.

package benchmark

import "time"

// ── Write-pool sizing ────────────────────────────────────────────────────────

// poolSlotSizeCap is the maximum object size that travels through the
// WritePool fast path.  Objects larger than this value are generated inline
// (heap allocation on demand) instead of being served from a pre-filled slot.
const poolSlotSizeCap int64 = 512 << 20 // 512 MiB

// poolBytesPerSide is the total RAM budget for the ChanPool slot pool.
// depth = poolBytesPerSide / slotSize, so total RSS ≈ poolBytesPerSide
// (down from 2× with the old DoubleDataPool).  Increasing this value adds
// more pre-filled slots (deeper pipeline buffer); decreasing it saves RAM.
const poolBytesPerSide int64 = 8 << 30 // 8 GiB total slot pool

// poolAbsoluteMinDepth is the hard safety floor applied inside newDataPool and
// newDoubleDataPool after the caller-supplied depth is received.  A depth of 4
// guarantees that the producer can always keep at least a few slots ahead of
// the consumers, preventing trivial pipelines from deadlocking.
const poolAbsoluteMinDepth = 4

// poolBuildMinDepth is the minimum ring depth enforced by buildWritePool when
// it derives depth from the poolBytesPerSide budget.  This floor applies when
// the object size is so large that bytesPerSide/slotSize would otherwise fall
// below this value, ensuring a minimum level of pipeline concurrency regardless
// of object size.
const poolBuildMinDepth = 32

// poolMaxDepth is the hard ceiling on ring depth enforced by buildWritePool.
// Without this cap, very small object sizes produce an astronomical slot count
// (depth = bytesPerSide / slotSize) that consumes the full poolBytesPerSide
// budget in RAM even though the ring never needs thousands of pre-filled slots
// to keep the producer ahead of the consumers.
//
// The cap must be high enough that production workloads are not artificially
// limited.  UNet3D-like objects at 6.85 MiB with 8 GiB/side gives a natural
// depth of 1,193.  At 2048 the full budget is used for all production object
// sizes down to ~4 MiB; for smaller objects the cap still provides ample
// pipeline depth per-goroutine while bounding per-side RAM to 2048 × slotSize.
// Unit tests override poolBytesPerSide via overridePoolBytesPerSide (128 MiB),
// so with 8 KiB test objects their capped depth is 2048 × 8 KiB × 2 = 32 MiB.
const poolMaxDepth = 2048

// poolFirstTouchPageStride is the stride (bytes) used when first-touching newly
// allocated slot memory.  Walking every 4 KiB OS page before the benchmark
// starts forces the kernel to map physical RAM to those virtual addresses,
// eliminating soft-fault stalls during the measurement phase.
const poolFirstTouchPageStride int64 = 4096 // one OS page

// cacheLinePadBytes is the number of padding bytes added to dpSlotState to
// fill a 64-byte CPU cache line.
//
//	sizeof(atomic.Uint32) = 4 bytes
//	64 (cache line) − 4 (field) = 60 bytes of padding
//
// This prevents false sharing: the producer and consumers each hammer the
// state field of a different slot, and those slots' state fields never occupy
// the same cache line.
const cacheLinePadBytes = 60

// ── Pool / pipeline telemetry ────────────────────────────────────────────────

// poolRatioWarnThreshold is the slot-based headroom ratio below which a
// WARNING is emitted during progress reporting.  A ratio below 1.5 means the
// pool controller has not yet settled or the system is at capacity.
// Slot-based headroom = numProducers × intervalNs / totalFillNs_delta.
const poolRatioWarnThreshold = 1.5

// poolHeadroomTarget is the desired minimum slot-fill headroom ratio.  When the
// measured headroom falls below this value the controller spawns additional
// producer goroutines until the ratio is met.
const poolHeadroomTarget = 2.0

// poolHeadroomSurplus is the headroom level above which the controller
// considers retiring a fill goroutine to reclaim a CPU thread for I/O work.
// Retirement only happens after poolDownscaleStreak consecutive evaluations
// above this threshold (hysteresis).
const poolHeadroomSurplus = 4.0

// poolDownscaleStreak is the number of consecutive per-fill headroom
// evaluations that must all show headroom > poolHeadroomSurplus before a
// producer goroutine is retired.  This prevents premature scale-down caused
// by momentary upload speed spikes; the pool stays ready for throughput bursts.
const poolDownscaleStreak = int32(10)

// poolInitialProducerDivisor sets the initial fill-goroutine count to
// runtime.NumCPU() / poolInitialProducerDivisor.  On a 32-core machine this
// starts 8 producers — enough headroom without over-committing CPU at launch.
const poolInitialProducerDivisor = 4

// poolMaxProducerDivisor caps the fill-goroutine count at
// runtime.NumCPU() / poolMaxProducerDivisor.  On a 32-core machine the ceiling
// is 16 producers, leaving the other half of the cores for GCS I/O goroutines.
const poolMaxProducerDivisor = 2

// ── Data generation ──────────────────────────────────────────────────────────

// lognormalSigmaFallback is the σ multiplier used to estimate the upper bound
// of a lognormal SizeSpec when no explicit max is configured.
// mean + 3σ covers 99.87 % of draws (the classical three-sigma rule).
const lognormalSigmaFallback = 3.0

// ── Read infrastructure ──────────────────────────────────────────────────────

// readDrainBufSize is the capacity of each buffer kept in the engine's
// readBufPool.  Reusing fixed-size buffers across reads eliminates per-read
// heap allocations and the GC pressure they create.
const readDrainBufSize = 256 * 1024 // 256 KiB

// listMaxResults is the maximum number of objects requested in a single
// ListObjects call — used by preflight checks and the prepare-phase doList
// path.
const listMaxResults = 1000

// ── Progress reporting ───────────────────────────────────────────────────────

// prepareProgressInterval controls how often the prepare phase logs a progress
// line.  A 5-second interval balances observability against log verbosity
// during long-running pre-fill operations.
const prepareProgressInterval = 5 * time.Second

// benchProgressInterval controls how often the benchmark measurement phase
// logs a progress line.  A 10-second interval provides a steady reporting
// cadence without flooding the log on long benchmarks.
const benchProgressInterval = 10 * time.Second

// ── Error handling ───────────────────────────────────────────────────────────

// writeErrorLogLimit is the maximum number of write errors that are logged
// individually.  Errors beyond this count are silently tallied into the error
// counter, preventing log flooding during catastrophic failure scenarios while
// still giving the operator the first few diagnostic messages.
const writeErrorLogLimit int64 = 3

// ── Worker seeding ───────────────────────────────────────────────────────────

// workerRNGSeedStride is the per-worker offset applied when seeding
// goroutine-local random number generators.  Using a stride large enough to
// exceed the goroutine count per worker ensures that no two goroutines across
// different workers share an RNG seed, preserving statistical independence of
// the generated data.
const workerRNGSeedStride = 1000

// ── Unit conversion ──────────────────────────────────────────────────────────

// bytesPerGiB is the number of bytes in one gibibyte (2³⁰ = 1 073 741 824).
// Used for throughput reporting in GiB/s.
const bytesPerGiB float64 = 1 << 30

// nanosPerSecond is the number of nanoseconds in one second (10⁹).
// Used to convert between nanosecond-precision atomic counters and
// human-readable seconds in progress and summary output.
const nanosPerSecond float64 = 1e9
