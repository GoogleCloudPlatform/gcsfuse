# Write Data Generation Pipeline — ChanPool Design

**Status:** Current · March 30, 2026  
**Scope:** `internal/benchmark/datapool.go`, `internal/benchmark/engine.go`,
`internal/benchmark/datagen.go`, `internal/benchmark/constants.go`

---

## 1. Problem Statement

GCS write benchmarks have a tension between two requirements:

1. **Data must be compressible-opaque.** GCS storage (and intermediate network
   components) may behave differently on zero-filled or repetitively-patterned
   data. Random data is required for realistic benchmarks.

2. **Data generation must not contaminate latency measurements.** On a
   multi-core machine, generating 6–7 MiB of Xoshiro256++ random data takes
   ~0.5 ms per goroutine. At P99 write latencies of 200–500 ms this is
   negligible, but at high concurrency with many ops, the cumulative effect
   skews throughput and latency distributions.

### Why not generate data inline?

```go
// REJECTED — generation time is inside the latency window
data := make([]byte, size)    // + fillRandom() contends the CPU with GCS I/O
start := time.Now()
client.CreateObject(ctx, req) // latency includes gen time!
```

### Why not a simple pre-generated static buffer?

A single statically-generated buffer shared across goroutines works for reads
(drain buffer) but creates a problem for writes: GCS object names must be
unique (or rotated), and **concurrent goroutines need non-overlapping data
ranges** to avoid uploading the same bits for every object. More importantly,
any implementation that blocks all goroutines to rotate the buffer creates a
**group barrier** — see §3.

---

## 2. ChanPool Architecture

`ChanPool` is the production write pool. It pre-fills a set of independent slot
buffers and circulates them through a pair of channels.  A dynamic cohort of
fill goroutines (`fillLoop`) keeps the `ready` channel topped up; a per-fill
headroom controller (`maybeAdjustProducers`) scales the cohort up or down
without a background ticker or mutex in the steady-state path.

### 2.1 Key Data Types

```go
// datapool.go
type chanPoolSlot struct {
    idx  int    // index into ChanPool.slots — used by ReleaseSlot()
    data []byte // pre-filled with random bytes; len == p.slotSz
}

type ChanPool struct {
    slotSz     int64
    entropy    uint64
    seqCounter *atomic.Uint64  // shared with engine.writeBlockSeq

    slots []chanPoolSlot        // backing array; one entry per depth
    free  chan *chanPoolSlot    // slots waiting to be refilled
    ready chan *chanPoolSlot    // slots fully filled, ready for consumers

    // Set once by RunProducer before any fill goroutine starts; read-only
    // thereafter so spawnProducer needs no synchronisation to read it.
    parentCtx    context.Context
    maxProducers atomic.Int32  // ceil(NumCPU / poolMaxProducerDivisor)
    minProducers atomic.Int32  // adaptive floor — raised when headroom < target proves floor too low; never scale below

    // Dynamic producer tracking — mutex only on spawn/retire (rare path).
    numProducers    atomic.Int32
    producerMu      sync.Mutex
    producerCancels []context.CancelFunc // one entry per active fill goroutine

    // Per-fill controller state — all atomic, zero mutex in steady state.
    fillCount       atomic.Uint64 // total fills across all goroutines
    lastCheckNs     atomic.Int64  // UnixNano at last headroom evaluation
    lastCheckFillNs atomic.Int64  // totalFillNs snapshot at last evaluation
    surplusStreak   atomic.Int32  // consecutive above-surplus evaluations

    // Telemetry (nanoseconds and bytes, atomic)
    bytesProduced   atomic.Int64
    bytesConsumed   atomic.Int64
    producerStallNs atomic.Int64
    consumerStallNs atomic.Int64
    totalFillNs     atomic.Int64  // cumulative ns inside fillRandom across all producers
}
```

### 2.2 Slot Lifecycle

```
┌──────────────────────────────────────────────────────────────┐
│                    Slot circulation loop                     │
│                                                              │
│  ┌─────────────┐                                             │
│  │  Constructor │  fillRandom(slot.data)                     │
│  │  (pre-fill)  │──────────────────────────────────────────► │
│  └─────────────┘                                            ▼│
│                                              ┌────────────────┤
│                                              │  ready channel │
│  ┌──────────────────┐                        │  (buffered,    │
│  │ fill goroutines  │◄───free channel────────┤  cap=depth)    │
│  │ (N, dynamic)     │    (buffered,           └──────┬─────────┤
│  │                  │    cap=depth)                  │        │
│  │ fillRandom(slot) │                                ▼        │
│  │ ready ← slot     │        AcquireSlot() returns slot       │
│  └──────────────────┘               │                        │
│                                     ▼                        │
│                          Consumer goroutine                  │
│                          GCS CreateObject(slot.data)         │
│                          ReleaseSlot() → free ← slot         │
└──────────────────────────────────────────────────────────────┘
```

**Channel membership IS the slot state.** There are no locks, no atomics, no
CAS loops on the critical path. A slot is in exactly one of three places at any
given time: `ready`, `free`, or held by a consumer goroutine (in flight).


### 2.3 Pre-fill in the Constructor

```go
func newChanPool(slotSize int64, depth int, entropy uint64, seqCounter *atomic.Uint64) *ChanPool {
    p := &ChanPool{
        slotSz:     slotSize,
        entropy:    entropy,
        seqCounter: seqCounter,
        slots:      make([]chanPoolSlot, depth),
        free:       make(chan *chanPoolSlot, depth),
        ready:      make(chan *chanPoolSlot, depth),
    }
    // Allocate, first-touch, fill, and enqueue every slot into ready so the
    // first AcquireSlot call returns instantly — no ramp-up stall.
    // RunProducer's fill goroutines block on <-p.free until ReleaseSlot
    // feeds slots back from consumers.
    nBlocks := uint64((slotSize + genBlockSize - 1) / genBlockSize)
    for i := range p.slots {
        p.slots[i].idx = i
        p.slots[i].data = make([]byte, slotSize)
        // First-touch: walk every OS page so the kernel maps physical RAM
        // before the benchmark starts (eliminates soft-fault stalls).
        for off := int64(0); off < slotSize; off += poolFirstTouchPageStride {
            p.slots[i].data[off] = 0
        }
        startSeq := seqCounter.Add(nBlocks) - nBlocks  // reserve a unique block range
        fillRandom(p.slots[i].data, entropy, startSeq)
        p.bytesProduced.Add(slotSize)
        p.ready <- &p.slots[i]
    }
    return p
}
```

After `newChanPool` returns, all `depth` slots are in `ready`. The first
`AcquireSlot()` call returns in nanoseconds — no generation work is needed.

### 2.4 Producer Goroutines

Unlike earlier single-goroutine designs, `ChanPool` uses a **dynamic cohort**
of fill goroutines whose size is controlled at runtime by `maybeAdjustProducers`.

#### RunProducer — initialiser and sentinel

`RunProducer` is called once by the engine (as `go pool.RunProducer(poolCtx)`).
It stores the root context, sets producer limits, spawns the initial cohort,
and then simply waits for cancellation:

```go
func (p *ChanPool) RunProducer(ctx context.Context) {
    p.parentCtx = ctx              // stored before any goroutine reads it
    p.lastCheckNs.Store(time.Now().UnixNano())

    maxProd := int32(runtime.NumCPU() / poolMaxProducerDivisor)   // NumCPU/2
    if maxProd < 1 { maxProd = 1 }
    p.maxProducers.Store(maxProd)

    initial := int32(runtime.NumCPU() / poolInitialProducerDivisor) // NumCPU/4
    if initial < 1 { initial = 1 }
    if initial > maxProd { initial = maxProd }

    // Floor starts at the initial cohort size — never scale below our proven
    // safe baseline.  Raised on scale-up when we discover the floor was too low.
    p.minProducers.Store(initial)

    for i := int32(0); i < initial; i++ {
        p.spawnProducer()
    }

    <-ctx.Done()  // no ticker or controller loop — scaling lives in fillLoop
}
```

On a 96-core machine this starts 24 fill goroutines and caps at 48.

#### fillLoop — the fill body for each producer goroutine

Each fill goroutine runs `fillLoop` under its own cancellable context:

```go
func (p *ChanPool) fillLoop(ctx context.Context) {
    defer p.numProducers.Add(-1)  // decrement count when this goroutine exits
    nBlocks := uint64((p.slotSz + genBlockSize - 1) / genBlockSize)
    for {
        // Non-blocking receive first; fall to blocking with stall accounting.
        var s *chanPoolSlot
        select {
        case s = <-p.free:
        default:
            if ctx.Err() != nil { return }
            // Reporter-stall is healthy (pool full) — log at TRACE, not DEBUG.
            logger.Tracef("[pool] producer wait: free=0 ready=%d producers=%d\n", ...)
            stallStart := time.Now()
            select {
            case s = <-p.free:
            case <-ctx.Done(): return
            }
            p.producerStallNs.Add(time.Since(stallStart).Nanoseconds())
        }

        // Reserve a unique block-index range and fill the slot.
        startSeq := p.seqCounter.Add(nBlocks) - nBlocks
        fillStart := time.Now()
        fillRandom(s.data, p.entropy, startSeq)
        p.totalFillNs.Add(time.Since(fillStart).Nanoseconds())
        p.bytesProduced.Add(p.slotSz)

        select {
        case p.ready <- s:
        case <-ctx.Done():
            select { case p.ready <- s: default: }  // salvage the filled slot
            return
        }

        // Headroom check — at most once per fill round (every numProducers
        // fills collectively).  No mutex in the common case.
        n := p.fillCount.Add(1)
        numProd := uint64(p.numProducers.Load())
        if numProd > 0 && n%numProd == 0 {
            p.maybeAdjustProducers()
        }
    }
}
```

**A producer waiting on `<-p.free` is healthy** — it means the pool is already
full and consumers haven't returned slots yet. It is logged at `TRACE` so it
doesn't appear in normal `-vv` output.

#### maybeAdjustProducers — the headroom controller

The controller uses a **slot-based headroom ratio** — the only metric that
accurately reflects whether the fill pipeline can keep pace with consumers:

```
headroom = numProducers × dtNs / totalFillNs_delta
         = (producer slots per second) / (actual fill rate demanded)
```

A headroom of 2.0 means the fill goroutines could serve twice the current
upload demand; 1.0 means they are exactly at capacity.

To prevent oscillation — where aggressive scale-down drops producers below the
level needed to sustain 2× headroom, causing a crash-and-recover loop — the
controller maintains an **adaptive minimum floor** (`minProducers`). The floor
starts at the initial cohort size and is **raised each time scale-up fires**:
if we needed N producers to recover headroom, the floor rises to N so future
scale-downs cannot erode below that proven safe minimum. The floor never
decreases; it only rises as the workload demands more fill capacity.

Scale-up and scale-down rules:

| Condition | Action |
|-----------|--------|
| `headroom < poolHeadroomTarget` (2.0×) | Spawn to `needed` goroutines in one step; **raise `minProducers` floor to `needed`** if `needed` exceeds current floor |
| `headroom > poolHeadroomSurplus` (4.0×) for `poolDownscaleStreak` (10) consecutive rounds **and `numProd > minProducers`** | Retire one goroutine (hysteresis); **never drops below the adaptive floor** |
| Otherwise | Reset `surplusStreak` counter; no change |

Only one goroutine executes `maybeAdjustProducers` per fill round (gated on
`fillCount % numProducers == 0`).  The mutex is acquired only when actually
spawning or retiring a goroutine, so the steady-state path is lock-free.

```go
func (p *ChanPool) maybeAdjustProducers() {
    now := time.Now().UnixNano()
    dtNs := now - p.lastCheckNs.Load()
    if dtNs < int64(time.Millisecond) { return }  // too short to measure

    dFillNs := p.totalFillNs.Load() - p.lastCheckFillNs.Load()
    p.lastCheckNs.Store(now)
    p.lastCheckFillNs.Store(p.totalFillNs.Load())

    if dFillNs <= 0 { return }  // pool idle — nothing to act on

    numProd := p.numProducers.Load()
    maxProd  := p.maxProducers.Load()
    headroom := float64(numProd) * float64(dtNs) / float64(dFillNs)

    switch {
    case headroom < poolHeadroomTarget && numProd < maxProd:
        needed := int32(math.Ceil(poolHeadroomTarget / headroom * float64(numProd)))
        if needed > maxProd { needed = maxProd }
        for p.numProducers.Load() < needed { p.spawnProducer() }
        // Raise the adaptive floor: if we needed more than the current floor,
        // the floor was too conservative.  Future scale-downs stop here.
        if needed > p.minProducers.Load() { p.minProducers.Store(needed) }
        p.surplusStreak.Store(0)

    case headroom > poolHeadroomSurplus && numProd > p.minProducers.Load():
        // Only enter this case when above the floor — the switch guard itself
        // prevents scale-down below minProducers.
        streak := p.surplusStreak.Add(1)
        if streak >= poolDownscaleStreak {
            if p.surplusStreak.CompareAndSwap(streak, 0) {
                // Double-check floor under the CAS — numProducers may have changed.
                if p.numProducers.Load() > p.minProducers.Load() {
                    p.retireProducer()  // cancel the most recently spawned goroutine (LIFO)
                }
            }
        }

    default:
        p.surplusStreak.Store(0)  // healthy zone — reset hysteresis counter
    }
}
```

Scale-up emits `[pool] scaling up: headroom=X.XXx → N producers (floor raised to N)` at **DEBUG**.
Scale-down emits `[pool] scaling down: headroom=X.XXx for 10 consecutive checks → N producers (floor=N)` at **DEBUG**.
These are verbose internal tuning events; they appear only at `-vv` and above.

### 2.5 Consumer Path

```go
// engine.go: doWrite()
func (e *Engine) doWrite(ctx context.Context, ts *trackState, rng *rand.Rand, objectName string) error {
    size := sampleObjectSize(rng, ts)

    // Pool fast path: zero heap allocation, no inline fill in the latency window.
    if e.writePool != nil && size <= e.writePool.SlotSize() {
        poolIdx, slotIdx, data, err := e.writePool.AcquireSlot(ctx, size)
        if err != nil { return err }
        req := &gcs.CreateObjectRequest{
            Name:     objectName,
            Contents: io.NopCloser(bytes.NewReader(data)),
        }
        start := time.Now()
        obj, err := e.bucket.CreateObject(ctx, req)  // ← latency window
        elapsed := time.Since(start)
        // Release immediately — GCS has already read the data from the reader.
        e.writePool.ReleaseSlot(poolIdx, slotIdx)
        // ... record histogram, return
    }
    // inline fallback (large objects) — see §7
}
```

`AcquireSlot` blocks on `<-p.ready` only when the pool is exhausted. In that
case it logs at DEBUG with a diagnostic so the operator knows why:

```go
func (p *ChanPool) AcquireSlot(ctx context.Context, size int64) (poolIdx, slotIdx int, data []byte, err error) {
    var s *chanPoolSlot
    select {
    case s = <-p.ready:   // non-blocking fast path (normal case)
    default:
        if ctx.Err() != nil { return -1, -1, nil, ctx.Err() }
        // Log at DEBUG — this should never appear in a healthy run.
        logger.Debugf("[pool] consumer stall: free=%d ready=0/%d producers=%d — " +
            "all filled slots consumed; producers cannot keep pace\n", ...)
        stallStart := time.Now()
        select {
        case s = <-p.ready:
        case <-ctx.Done(): return -1, -1, nil, ctx.Err()
        }
        p.consumerStallNs.Add(time.Since(stallStart).Nanoseconds())
    }
    p.bytesConsumed.Add(size)
    return 0, s.idx, s.data[:size], nil
}
```

`ReleaseSlot` puts the slot directly onto `free`:

```go
func (p *ChanPool) ReleaseSlot(poolIdx, slotIdx int) {
    p.free <- &p.slots[slotIdx]  // poolIdx is ignored for ChanPool (always 0)
}
```

---

## 3. Why Channels (Not Ring Buffers or Double Buffers)

### 3.1 Root Cause of the Group Barrier Problem

The `DoubleDataPool` predecessor uses two fixed-size slot rings that alternate
between ACTIVE (consumers drain) and STANDBY (producer fills).  When all ACTIVE
slots are exhausted, consumers wait for the producer to signal `standbyReady`.
Before the producer can begin filling the standby ring it must wait for all
in-flight uploads on that ring to drain (`inFlight[stbyIdx] == 0`):

```
// DoubleDataPool coordination (still in codebase, not the production default)
// Producer side — the only synchronisation point per fill cycle:
for d.inFlight[stbyIdx].Load() > 0 {
    runtime.Gosched()  // ← all-uploads-complete barrier
}
// Now fill every slot unconditionally (exclusive ownership):
for i := range d.slots[stbyIdx] { fillRandom(...) }
d.standbyReady.Store(1)
```

With 64 goroutines and P99.9 GCS latency of 2000 ms, even a single slow upload
prevents the standby pool from being filled and re-promoted.  The barrier stall
can be as long as the slowest tail request — directly inflating measured
write latency for all other goroutines.

### 3.2 ChanPool Eliminates Group Barriers

Each slot is **independently** tracked via channel membership. When one upload
takes 2000 ms, **only that one goroutine** holds its slot for 2000 ms. The other
63 goroutines continue uploading from their own slots. There is no barrier.

A single slow request holding 6.85 MiB for 2000 ms instead of ~80 ms means
one slot is occupied longer, but with 1,193 slots total (8 GiB pool), there
are 1,192 other ready slots. Consumer stall = 0.

### 3.3 Why Not a Single Ring Buffer?

A single circular ring buffer with an atomic head pointer could also work, but:

1. **ABA problem**: A goroutine holding slot `i` that stalls long enough for the
   producer to lap the ring would read a slot that is being rewritten. Detecting
   this requires reference counting or comparison — adding complexity.

2. **Head contention**: With 64 goroutines, an atomic counter on a ring head is
   a hot cache line. Channel sends/receives spread the scheduling cost across the
   Go runtime's work-stealing scheduler.

3. **Blocking is natural**: Channel semantics give blocking for free, with full
   context cancellation support (`<-ctx.Done()`). A ring buffer requires a
   separate parking mechanism.

---


## 4. Sizing

```go
// constants.go
poolBytesPerSide  int64 = 8 << 30  // 8 GiB total pool budget
poolSlotSizeCap   int64 = 512 << 20 // objects > 512 MiB use inline fallback
poolBuildMinDepth       = 32        // always at least 32 slots
poolMaxDepth            = 2048      // never more than 2048 slots
```

```
depth = clamp(poolBytesPerSide / slotSize, poolBuildMinDepth, poolMaxDepth)
```

| Object Size | depth | Total Pool RSS |
|-------------|-------|---------------|
| 512 KiB | 2048 (capped) | 1 GiB |
| 1 MiB | 2048 (capped) | 2 GiB |
| 4 MiB | 2048 (capped) | 8 GiB |
| 6.85 MiB (UNet3D avg) | 1,193 | 8 GiB |
| 32 MiB | 256 | 8 GiB |
| 128 MiB | 64 | 8 GiB |
| 512 MiB | 32 (floor) | 16 GiB |
| > 512 MiB | — (inline fallback) | 0 (allocated per op) |

### Rationale for 8 GiB

8 GiB is the observed headroom needed to keep `consumer-stall = 0` at P99.9
tail latencies of ~2000 ms with 64 goroutines uploading 6.85 MiB objects:

```
Minimum pool coverage at P99.9:
  = goroutines × max_outstanding_hold_time × ops_rate
  = 64 goroutines × 2.0 s × 85 ops/s × 6.85 MiB/op
  ≈ 74 GiB theoretical max

In practice (UNet3D, GCS RAPID zonal endpoint, empty bucket, live run):
  - slot headroom: ~4.7× sustained (controller settles to 7-12 producers)
  - consumer-stall: 0.000 s  (never exhausted — confirmed across 50,176 objects)
```

The 8 GiB floor comfortably covers P99.9 tail events without approaching 16 GiB.

---

## 5. Data Generation — Xoshiro256++

Random data is generated by `fillRandom` in `datagen.go`:

```go
// datagen.go
func fillRandom(buf []byte, entropy uint64, seq uint64) {
    // Parallel fill using Rayon-style goroutine fan-out (chunk per CPU)
    // Each chunk has a unique Xoshiro256++ seed derived from entropy + seq + chunkIdx
    // → All chunks are independent; ordering doesn't matter
    // → Zero allocations: goroutines write directly into buf sub-slices
}
```

**Xoshiro256++** properties relevant to benchmarking:
- 256-bit state → passes all BigCrush randomness tests
- ~14 GiB/s throughput per core at vector width (AVX2)
- No global lock — each goroutine has its own RNG state
- `seq` uniquifies every slot refill: two rapid refills of the same slot will
  always produce different bytes, preventing GCS deduplication at intermediate
  caches (though GCS presently does not deduplicate object content)

---

## 6. Telemetry and Observability

### 6.1 Always-logged pool stats (at INFO / `-v`)

**Per-interval during prepare and bench phases:**
```
  [pool] producers=12  fill=19.49 GiB/s  upload=4.15 GiB/s  headroom=4.70x  consumer-stall=0.00%
```

**When headroom is low (< poolRatioWarnThreshold = 1.5×):**
```
  [pool] WARNING: low headroom — producers=2  fill=3.10 GiB/s  upload=4.20 GiB/s  headroom=1.36x  consumer-stall=0.00%
```

**When consumer stall is detected:**
```
[pool] WARNING: consumer stall detected — producers=2  fill=2.90 GiB/s  upload=4.20 GiB/s  headroom=1.28x  consumer-stall=3.47%
```

**Final summary (end of write phase):**
```
[pool] final: producers=10  fill=18.62 GiB/s  upload=4.00 GiB/s  headroom=4.66x  consumer-stall=0.000s
```

| Field | Meaning | Healthy value |
|-------|---------|--------------|
| `producers` | Current number of active fill goroutines | auto-tuned; typically NumCPU/8–NumCPU/4 |
| `fill` | GiB/s rate data is written into `ready` channel | 10–20 GiB/s (Xoshiro256++ limit) |
| `upload` | GiB/s rate data exits via `AcquireSlot` (= GCS write throughput) | matches GCS target |
| `headroom` | `numProducers × intervalNs / totalFillNs_delta` — how many times faster fill is than upload demand | **≥ 2.0×** (target); ≥ 4.0× triggers scale-down |
| `consumer-stall%` | Fraction of time consumers blocked on `<-p.ready` (pool empty) | **Must be 0.00%** |

A non-zero `consumer-stall%` means GCS uploads were delayed by data generation,
directly inflating measured write latency.

### 6.2 Dynamic scaling events (at DEBUG / `-vv`)

```
[pool] scaling up:   headroom=1.46x → 14 producers (floor raised to 14)
[pool] scaling down: headroom=9.30x for 10 consecutive checks → 13 producers (floor=14)
```

Scale-up fires immediately when headroom drops below `poolHeadroomTarget` (2.0×).
Each scale-up also raises the **adaptive floor** (`minProducers`) to the new
producer count when that count exceeds the current floor. Subsequent scale-downs
stop at the floor, preventing the oscillation pattern where the controller
drops below the level that was just proven necessary.

Scale-down fires after `poolDownscaleStreak` (10) consecutive evaluations above
`poolHeadroomSurplus` (4.0×), **and only when `numProducers > minProducers`**.
This hysteresis-plus-floor combination eliminates the crash-and-recover loop
observed in earlier versions (63 scale-up / 286 scale-down events in a 90-second
run; worst headroom crash to 0.56×).

Both events log at DEBUG so they do not appear in normal `-v` output. They are
intentionally verbose diagnostic aids for tuning the controller constants.

### 6.3 Debug/trace stall events

**Producer wait** (at `TRACE` / `-vvv`):
```
[pool] producer wait: free=0 ready=1193 producers=10
```
This is the healthy steady-state event — all slots are full and ready for
consumers.  The fill goroutine blocks briefly on `<-p.free` until a consumer
calls `ReleaseSlot`.

**Consumer stall** (at `DEBUG` / `-vv`):
```
[pool] consumer stall: free=856 ready=0/1193 producers=2 — all filled slots consumed; producers cannot keep pace
```
This indicates pool exhaustion and should be investigated.  The `free=N` field
shows how many slots are waiting to be refilled — a high count with `ready=0`
means the fill goroutines are overwhelmed.

### 6.4 PoolStats interface

```go
// datapool.go
type PoolStats struct {
    BytesProduced   int64 // cumulative bytes filled (slotSize per fill)
    BytesConsumed   int64 // cumulative bytes claimed (actual object sizes)
    ProducerStallNs int64 // cumulative goroutine-ns producers waited for a free slot
    ConsumerStallNs int64 // cumulative goroutine-ns consumers waited for a filled slot
    TotalFillNs     int64 // cumulative ns inside fillRandom across all producer goroutines
    NumProducers    int32 // number of active fill goroutines at time of Stats() call
    MinProducers    int32 // current adaptive floor — the minimum the controller will not scale below
}
```

Take two consecutive snapshots and subtract to compute per-interval rates.
`TotalFillNs` and `NumProducers` together give the slot-based headroom ratio
used by the progress reporter and by `maybeAdjustProducers`.

### 6.5 WritePool interface

```go
// datapool.go
type WritePool interface {
    // AcquireSlot blocks until a pre-filled slot of at least size bytes is
    // available.  Returns (poolIdx, slotIdx, data[:size], nil) on success.
    // Returns (-1, -1, nil, ctx.Err()) if ctx is cancelled.
    // Caller must call ReleaseSlot(poolIdx, slotIdx) when done.
    AcquireSlot(ctx context.Context, size int64) (poolIdx, slotIdx int, data []byte, err error)

    // ReleaseSlot returns the slot identified by (poolIdx, slotIdx) back to
    // the pool.  Must be called exactly once per AcquireSlot.
    ReleaseSlot(poolIdx, slotIdx int)

    // RunProducer initialises and runs the producer subsystem.  Must be
    // started in a separate goroutine before any AcquireSlot calls.
    RunProducer(ctx context.Context)

    // Stats returns a monotonically-increasing snapshot of telemetry counters.
    Stats() PoolStats

    // SlotSize returns the maximum object size (bytes) this pool supports.
    SlotSize() int64
}
```

---

## 7. Inline Fallback Path

When `buildWritePool` detects that all object sizes exceed `poolSlotSizeCap`
(512 MiB), it returns `nil`. `doWrite` falls back to:

```go
data := make([]byte, size)    // one heap allocation per op
fillRandom(data, entropy, seq)
// latency window includes generation time (~73 ms per GiB)
_, err = e.bucket.CreateObject(ctx, req)
```

This path is unavoidable for very large objects (> 512 MiB) because allocating
a 1× pool would require 16 GiB RAM minimum (`poolSlotSizeCap` floor = 32 slots).
Users benchmarking objects > 512 MiB should be aware that measured write latency
includes ~70–80 ms of CPU generation overhead per GB of object size.

---

## 8. Design Alternatives Considered and Rejected

| Alternative | Why Rejected |
|-------------|--------------|
| Static shared read-only buffer | Concurrent writers could alias; no uniqueness guarantee |
| Double-buffer with group barrier | Catastrophic stall at pool swap (see §3.1) |
| `sync.Pool` for write buffers | `sync.Pool` does not guarantee slot availability; under GC pressure buffers can be evicted, causing allocation on the write hot path |
| Per-goroutine fixed buffer | N goroutines × 6.85 MiB = only 439 MiB at 64 goroutines — insufficient headroom for P99.9 tail events |
| DataPool (single-ring CAS) | Still in codebase; suitable for workloads with small write pools. Uses per-slot atomic CAS (EMPTY→FILLING→READY→CONSUMING); no dynamic producer scaling (always 1 goroutine, 1 producer stall metric) |
| DoubleDataPool (double-buffer) | Still in codebase; the ChanPool predecessor. Single producer fills alternate rings; group barrier on `inFlight==0` causes stalls at P99.9 tail latencies (see §3.1) |
| OS huge pages / mmap | Adds deployment complexity; RSS savings minimal (8 GiB already resident) |

---


## 9. Related Documents

- [`docs/memory-analysis.md`](memory-analysis.md) — Full allocation inventory and RSS accounting
- [`docs/bench-user-guide.md`](bench-user-guide.md) — User-facing description of prepare progress output
- `internal/benchmark/datapool.go` — Full implementation
- `internal/benchmark/datagen.go` — Xoshiro256++ data generation
- `internal/benchmark/constants.go` — Pool sizing constants
