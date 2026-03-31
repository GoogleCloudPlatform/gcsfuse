# Memory and Buffer Analysis — `gcs-bench bench`

**Status:** Current · March 30, 2026  
**Scope:** All memory allocations made during a `gcs-bench bench` run, from startup through export  
**Note:** Supersedes the original investigation document (preserved as `memory-analysis.md.bak`);
all proposed fixes described in the original are now implemented.

---

## 1. Executive Summary

`gcs-bench bench` uses two separate buffer strategies to keep data-generation cost outside the
latency measurement window:

- **Read path**: a `sync.Pool` of **256 KiB drain buffers** shared across all goroutines. The pool
  eliminates per-read heap allocation and GC pressure; in steady state, zero allocations occur on
  the read hot path.

- **Write path**: a `ChanPool` — a pre-filled, channel-pair slot pool. All slots are allocated and
  filled with Xoshiro256++ random data **before `Run()` is called**, so consumers never wait for
  data generation. Total RAM ≈ 8 GiB for typical production workloads (UNet3D). Data-generation
  cost is excluded from all write latency measurements.

- **Objects larger than 512 MiB** fall back to an inline allocation path where generation time is
  included in the latency window. This is a known limitation, documented below.

- **No OS page cache involvement.** All I/O is issued directly via GCS gRPC/HTTP2; local filesystem
  caching has zero influence on reported numbers.

---

## 2. Code Map

| File | Role |
|------|------|
| `cmd/benchmark.go` | Entry point; creates `logBuf`, wraps stdout/stderr |
| `internal/benchmark/engine.go` | `Engine`, `doRead`, `doWrite`, `buildWritePool`, `readBufPool` |
| `internal/benchmark/datapool.go` | `ChanPool` (production), `DoubleDataPool`, `DataPool` (legacy) |
| `internal/benchmark/datagen.go` | `fillRandom` — parallel Xoshiro256++ data generation |
| `internal/benchmark/constants.go` | `poolBytesPerSide` (8 GiB), `poolSlotSizeCap` (512 MiB), etc. |
| `internal/benchmark/histograms.go` | `TrackHistograms` — HDR histogram allocation |
| `internal/benchmark/preflight.go` | Pre-flight checks; one small sentinel object PUT/GET |

---

## 3. Allocation Inventory

### 3.1 Read Path — `doRead()` — `sync.Pool` of 256 KiB Buffers

```go
// Engine struct (engine.go)
readBufPool sync.Pool   // pools *[]byte of readDrainBufSize (256 KiB)

// doRead() hot path — zero allocation per read in steady state
bufPtr := e.readBufPool.Get().(*[]byte)
buf := *bufPtr
defer e.readBufPool.Put(bufPtr)

var bytesRead int64
for {
    n, readErr := reader.Read(buf)
    // only 'n' (byte count) is used — buffer contents are discarded
}
```

**Behaviour:**
- `sync.Pool` is initialized in `NewEngine()` with a `New` func that allocates one `*[]byte` of
  `readDrainBufSize` (256 KiB, defined in `constants.go`).
- The pool's victim-cache mechanism (Go 1.13+) keeps buffers alive across GC cycles. In steady
  state all N concurrency goroutines are holding a buffer and immediately returning it — **zero
  allocations per read**.
- At most `N × 256 KiB` can be live simultaneously (e.g., 64 goroutines = 16 MiB).
- Buffer contents are never inspected — only the byte count `n` from `reader.Read(buf)` matters.
  The buffer is safe to return to the pool immediately after `doRead()` returns.

### 3.2 Write Path — `ChanPool` (Production Default)

`ChanPool` is the production write pool, created by `buildWritePool()` inside `NewEngine()` when
any track has `op-type: write` and the largest object size ≤ `poolSlotSizeCap` (512 MiB).

**Architecture — channel-pair circulation:**

```
constructor fills all slots → ready channel
                                    │
             ┌──────────────────────┘
             ▼
    AcquireSlot() ◀── consumer goroutine
             │
         GCS upload (in flight, holding one slot)
             │
    ReleaseSlot() ──▶ free channel
             │
             └──────▶ RunProducer(): fillRandom(slot.data) ──▶ ready channel
```

A single slow GCS upload (e.g. 2000 ms tail latency) holds exactly **one** slot.
The remaining `depth − 1` slots cycle freely. No group barrier exists — consumers never wait for
an entire pool's worth of uploads to clear.

**Sizing:**

```
depth = clamp(poolBytesPerSide / slotSize, poolBuildMinDepth, poolMaxDepth)
      = clamp(8 GiB / slotSize, 32, 2048)
```

| Object Size | depth | Total RSS |
|-------------|-------|-----------|
| 1 MiB | 2048 (capped) | 2 GiB |
| 6.85 MiB (UNet3D avg) | 1,193 | 8 GiB |
| 32 MiB | 256 | 8 GiB |
| 128 MiB | 64 | 8 GiB |
| 512 MiB | 32 (floor) | 16 GiB |

**Pre-fill in constructor:** All `depth` slots are allocated, first-touched (one write per 4 KiB
page), filled with Xoshiro256++ random data, and placed into the `ready` channel before
`NewEngine()` returns. `RunProducer()` starts blocked on `<-p.free`; it enters the normal fill
cycle only as consumers call `ReleaseSlot()` and return slots to `free`.

**Result:** The first `AcquireSlot()` call returns instantly. Consumer stall at run start = 0.

**Physical RAM** is committed at construction time. For 1,193 slots × 6.85 MiB the first-touch
phase takes approximately 0.5 s at 14 GiB/s fill rate.

**Hot path — no allocation:**

```go
// engine.go:doWrite() — pool path
_, slotIdx, data, _ := e.writePool.AcquireSlot(ctx, size)
req := &gcs.CreateObjectRequest{
    Contents: io.NopCloser(bytes.NewReader(data)),  // zero-copy reference to slot
}
// ... CreateObject (network time only — data already pre-filled) ...
e.writePool.ReleaseSlot(0, slotIdx)
```

**Telemetry** — four atomic counters reported via `WritePool.Stats()`:

| Counter | Meaning |
|---------|---------|
| `bytesProduced` | Cumulative bytes filled and placed into `ready` |
| `bytesConsumed` | Cumulative bytes claimed by consumers (actual object sizes) |
| `producerStallNs` | Goroutine-ns the producer spent blocked on `<-p.free` |
| `consumerStallNs` | Goroutine-ns consumers spent blocked on `<-p.ready` |

Observed production values (UNet3D, 64 goroutines, GCS RAPID):
- `consumer-stall = 0.000 goroutine·s` (zero)
- `producer-stall ≈ 28%` (healthy — pool is full; producer waiting for consumers)
- `headroom ratio ≈ 4.7×`

### 3.3 Write Path — Inline Fallback (Objects > 512 MiB)

When `buildWritePool` returns `nil`, `doWrite` falls back to on-demand allocation:

```go
data := make([]byte, size)       // heap allocation, one per write op
fillRandom(data, entropy, seq)   // parallel Xoshiro256++ fill (~14 GiB/s)
// — latency window starts here (CreateObject) —
```

**Performance implication:** Allocation + fill time is **inside** the measured write latency.
For a 1 GiB object at 14 GiB/s fill rate, this adds ~73 ms of generation overhead per sample.

**When this occurs:**
- Object size > `poolSlotSizeCap` (512 MiB)
- Lognormal distribution where the configured `max > 512 MiB`

### 3.4 Object Path Lists

Each track pre-computes its full object path list at engine creation:

| Config | Entries | Approx RAM |
|--------|---------|-----------|
| `object-count: 1,000` | 1,000 | ~40 KB |
| UNet3D 28×28×64 | 50,176 | ~2 MB |
| `object-count: 1,000,000` | 1,000,000 | ~40 MB |

Read-only after creation; shared across goroutines with no locks.

### 3.5 Console Log Buffer

All console output is tee'd to `logBuf` and written to `console.log` at exit. Size:
~10–100 KB at `-v`, up to hundreds of KB at `-vvv`. Negligible performance impact.

### 3.6 HDR Histograms

Two `hdrhistogram.Histogram` instances per track (TTFB + total latency). Each pre-allocates
~100–400 KB. Total ~200–800 KB per track; no run-time allocations during recording.

### 3.7 GCS SDK Internal Buffers

gRPC-Go maintains per-stream receive/send buffers (~4 MB per stream). At 64 goroutines this adds
~256 MB of SDK-internal buffer, visible only via OS-level `/proc/<pid>/status` monitoring.

---

## 4. Summary Table

| Component | Allocation Site | Sizing | Example (64 goroutines, UNet3D 6.85 MiB) | GC-managed |
|-----------|----------------|--------|------------------------------------------|------------|
| Read drain buffers | `Engine.readBufPool` (sync.Pool) | N × 256 KiB | 64 × 256 KiB = 16 MiB | Pooled — near-zero GC |
| Write ChanPool | `newChanPool` at startup | `poolBytesPerSide / slotSize` | 1,193 × 6.85 MiB = **8 GiB** | No (fixed pre-alloc) |
| Write inline fallback | `doWrite` per op | 1 × objectSize | Disabled when pool active | Yes (> 512 MiB only) |
| Object path lists | `buildObjectPaths` at startup | objectCount × ~40 B | 50,176 × 40 B ≈ 2 MB | No (held for run) |
| Console log buffer | `cmd/benchmark.go` | grows with verbosity | ~10–100 KB | No (released at exit) |
| HDR histograms | `NewTrackHistograms` at startup | ~400 KB × 2 per track | ~800 KB/track | No (fixed pre-alloc) |
| GCS SDK buffers | internal (SDK) | ~4 MB per active stream | ~256 MB | No (SDK-managed) |

---

## 5. Key Findings

### 5.1 Reads: sync.Pool eliminates GC pressure

The `sync.Pool` keeps all drain buffers cached across GC cycles. GC pressure from read buffers is
now negligible in production. The 256 KiB size keeps the read loop tight while fitting all
simultaneous buffers in L3 cache.

### 5.2 Writes: ChanPool achieves zero consumer stall at 8 GiB RAM

The channel-pair design eliminates the group-barrier problem of the prior designs. Confirmed
production results (UNet3D, 64 goroutines, GCS RAPID zonal bucket):

```
consumer-stall:  0.000 goroutine·s total
producer-stall:  27.028 s (27.98%)   — healthy: pool full, producer waiting
headroom ratio:  4.70×
throughput:      3.472 GiB/s
P99 latency:     250 ms  (was 520 ms with DoubleDataPool)
P99.9 latency:   1,110 ms  (was 1,698 ms)
RSS:             ~8.25 GiB  (was 16 GiB with DoubleDataPool)
```

### 5.3 Large-object writes: inline path overstates latency

Objects > 512 MiB always use the inline fallback. At 14 GiB/s fill rate, a 1 GiB object adds
~73 ms of generation overhead per write sample. Users should be aware of this for very large
object benchmarks.

### 5.4 Object path list scalability

For UNet3D (50,176 objects) the list is ~2 MB — completely negligible. At 1,000,000+ objects
it approaches 40 MB; at 10,000,000+ it may become significant.

---

## 6. Observing Memory During a Run

### 6.1 OS-level peak RSS (no code changes needed)

```bash
/usr/bin/time -v gcs-bench bench --config bench.yaml
# → "Maximum resident set size: ..."
```

The benchmark also captures `VmHWM` internally via `readPeakRSSKiB()` and reports it in the
YAML result file under `runtime.peak_rss_kib`.

### 6.2 Pool pipeline telemetry (always logged)

**Per-interval during prepare (at `-v`):**
```
  [pool] produce=14.85 GiB/s  consume=3.16 GiB/s  ratio=4.699  producer-stall=27.98%  consumer-stall=0.00%
```

**Final summary at end of every write run:**
```
[pool] final: produced=336.12 GiB (4.66 GiB/s)  consumed=71.52 GiB (0.99 GiB/s)  ratio=4.700  producer-stall=27.028s (27.98%)  consumer-stall=0.000s
```

**Stall events at `-vv` (DEBUG):**
```
[pool] producer stalled: free=0 ready=1193 depth=1193
[pool] consumer stalled: free=1193 ready=0 depth=1193
```

### 6.3 Go runtime stats in result YAML

```yaml
runtime:
  go_heap_alloc_bytes: 134217728
  go_heap_sys_bytes:   9663676416   # includes ChanPool backing arrays
  gc_cycles:           3
  gc_pause_total_ns:   2100000
  peak_rss_kib:        8650240      # ~8.25 GiB — ChanPool dominates
```

---

## 7. Design Evolution

| Version | Write Pool | RAM (UNet3D) | Consumer Stall |
|---------|-----------|--------------|----------------|
| v1 `DataPool` | Lock-free single ring | variable | ~8% (producer lag) |
| v2 `DoubleDataPool` | Double-buffered rings + group barrier | 16 GiB | 873–1524% on pool swap |
| v3 `ChanPool` (current) | Channel-pair, per-slot circulation | 8 GiB | **0.000 goroutine·s** |

The `DoubleDataPool` group barrier was the root cause of catastrophic stalls: when a pool swap
was triggered, all 64 goroutines blocked until every in-flight upload completed. A single 2000 ms
GCS tail event caused 64 goroutines × 2000 ms = 128 goroutine·s of wasted time per swap, cycling
every ~2.5 s.

See `docs/write-pool-design.md` for the full `ChanPool` architectural design document.
