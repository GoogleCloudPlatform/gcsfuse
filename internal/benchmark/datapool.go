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

// Pre-fill pipeline implementations for write benchmarks.
//
// Two implementations are provided:
//
//   DataPool — single-ring lock-free pipeline (original design).
//     One producer fills a ring of slots sequentially; any number of consumers
//     acquire READY slots, upload to GCS, then release back to EMPTY.
//     Slot headroom ≈ 1.003× at typical UNet3D rates → ~8% consumer stall.
//
//   DoubleDataPool — double-buffered pipeline (preferred for production).
//     Two identically-sized rings alternate between ACTIVE (consumer) and
//     STANDBY (producer).  The producer fills the standby ring entirely before
//     consumers can exhaust the active ring (4.68× byte headroom → 0.54 s
//     fill time vs 2.54 s drain time).  Consumer stall ≈ 0 at steady state.
//
// Both types implement the WritePool interface so the engine is agnostic.
//
// Slot state machine (shared by both implementations)
// ────────────────────────────────────────────────────
//
//   EMPTY(0) ──producer CAS──▶ FILLING(1) ──fillRandom──▶ READY(2)
//     ▲                                                        │
//     └──────── consumer ReleaseSlot ── CONSUMING(3) ◀─CAS────┘
//
// Cache-line isolation
// ────────────────────
//   Each slot's atomic state field occupies its own 64-byte cache line (via
//   explicit padding).  This prevents "false sharing" where concurrent atomic
//   operations on adjacent slots in the ring contend for the same hardware
//   cache line — a common hidden bottleneck in lock-free ring implementations.
//
// Sizing guidelines
// ─────────────────
//   slotSize  ≥ max write object size (so every write uses exactly one slot).
//   depth     ≥ 2 × max_concurrent_writers (keeps producer ahead of consumers).
//   newDataPool enforces depth ≥ 4.

import (
	"context"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

// WritePool is the interface implemented by both DataPool (single-ring) and
// DoubleDataPool (double-buffered).  The engine interacts exclusively through
// this interface so the two implementations are interchangeable.
type WritePool interface {
	// RunProducer is the body of the background fill goroutine.
	// It runs until ctx is cancelled.  Call as: go pool.RunProducer(ctx).
	RunProducer(ctx context.Context)

	// AcquireSlot blocks until pre-filled data is available, marks the slot
	// CONSUMING, and returns an opaque (poolIdx, slotIdx) pair that must be
	// passed verbatim to ReleaseSlot once the upload is complete.
	// Returns ctx.Err() if ctx is cancelled before a slot is available.
	AcquireSlot(ctx context.Context, size int64) (poolIdx, slotIdx int, data []byte, err error)

	// ReleaseSlot frees the slot identified by (poolIdx, slotIdx) so the
	// producer can refill it.  Must be called exactly once per AcquireSlot.
	ReleaseSlot(poolIdx, slotIdx int)

	// Stats returns a monotonically-increasing snapshot of telemetry counters.
	// Subtract two snapshots to compute per-interval rates.
	Stats() PoolStats

	// SlotSize returns the maximum object size (bytes) supported by this pool.
	SlotSize() int64
}

// dpState values — must match the state machine diagram above.
const (
	dpEmpty     uint32 = 0
	dpFilling   uint32 = 1
	dpReady     uint32 = 2
	dpConsuming uint32 = 3
)

// dpSlotState is an atomic state field padded to exactly one 64-byte cache
// line.  It is kept separate from the data slice so that the hot CAS path
// (producer and consumers both hammering state) never touches the same cache
// line as the data pointer.
type dpSlotState struct {
	v    atomic.Uint32
	_pad [cacheLinePadBytes]byte // sizeof(atomic.Uint32)=4 → cacheLinePadBytes filler → 64 B total
}

// dpSlot holds one ring entry: isolated state + pre-allocated data buffer.
type dpSlot struct {
	st   dpSlotState
	data []byte
}

// DataPool manages a fixed-size ring of pre-filled write buffers.
type DataPool struct {
	depth      int
	slotSize   int64
	slots      []dpSlot
	entropy    uint64
	seqCounter *atomic.Uint64 // shared with engine.writeBlockSeq

	// Pipeline telemetry — written atomically by producer and consumers.
	// Read via Stats(); delta between two snapshots gives interval rates.
	bytesProduced   atomic.Int64 // bytes filled and marked READY (full slotSize per slot)
	bytesConsumed   atomic.Int64 // bytes claimed by consumers (actual object sizes)
	producerStallNs atomic.Int64 // producer goroutine-ns spent waiting for EMPTY slots
	consumerStallNs atomic.Int64 // consumer goroutine-ns spent waiting for READY slots
}

// newDataPool allocates all slot memory up-front and returns a ready-to-use
// DataPool.  Start the producer via go pool.RunProducer(ctx).
//
//	slotSize   bytes per slot; must be ≥ max object size written by the engine
//	depth      ring depth; clamped to ≥ 4
//	entropy    Xoshiro256++ seed (from newWriteEntropy in the engine)
//	seqCounter shared block-sequence counter (engine.writeBlockSeq)
func newDataPool(slotSize int64, depth int, entropy uint64, seqCounter *atomic.Uint64) *DataPool {
	if depth < poolAbsoluteMinDepth {
		depth = poolAbsoluteMinDepth
	}
	if slotSize < genBlockSize {
		slotSize = genBlockSize
	}

	p := &DataPool{
		depth:      depth,
		slotSize:   slotSize,
		slots:      make([]dpSlot, depth),
		entropy:    entropy,
		seqCounter: seqCounter,
	}

	// Pre-allocate and first-touch every slot's backing array.
	// First-touch walks every OS page (poolFirstTouchPageStride bytes) so the kernel maps physical
	// RAM to these virtual addresses before benchmark start, avoiding
	// soft-fault stalls during the measurement phase.
	for i := range p.slots {
		p.slots[i].data = make([]byte, slotSize)
		for off := int64(0); off < slotSize; off += poolFirstTouchPageStride {
			p.slots[i].data[off] = 0
		}
		p.slots[i].st.v.Store(dpEmpty)
	}

	return p
}

// RunProducer is the body of the background fill goroutine.  It loops forever,
// finding EMPTY slots in ring order, filling them with unique Xoshiro256++
// data, and marking them READY.  Stops when ctx is cancelled.
//
// Call as:  go pool.RunProducer(ctx)
func (p *DataPool) RunProducer(ctx context.Context) {
	idx := 0
	for {
		if ctx.Err() != nil {
			return
		}

		slot := &p.slots[idx]

		// CAS EMPTY → FILLING.
		// If the slot is still CONSUMING (GCS upload in flight), yield and retry
		// the same index — the producer must NOT skip ahead or it would fill a
		// slot the consumer will later overwrite with stale EMPTY.
		if !slot.st.v.CompareAndSwap(dpEmpty, dpFilling) {
			stallStart := time.Now()
			runtime.Gosched()
			p.producerStallNs.Add(time.Since(stallStart).Nanoseconds())
			continue
		}

		// Claim a contiguous block-index range so fill seeds never overlap
		// with inline fills in doWrite (fallback path) or other slots.
		nBlocks := uint64((p.slotSize + genBlockSize - 1) / genBlockSize)
		startSeq := p.seqCounter.Add(nBlocks) - nBlocks

		// Parallel Xoshiro256++ fill: goroutines write into non-overlapping
		// 1 MiB sub-slices of slot.data — zero allocation, zero copy.
		fillRandom(slot.data, p.entropy, startSeq)

		slot.st.v.Store(dpReady)
		p.bytesProduced.Add(p.slotSize)

		idx = (idx + 1) % p.depth
	}
}

// AcquireSlot blocks until a READY slot is available, CASes it to CONSUMING,
// and returns its index and a view of the pre-filled bytes truncated to size.
//
// poolIdx is always 0 for DataPool; it is returned so DataPool satisfies the
// WritePool interface alongside DoubleDataPool (which uses poolIdx 0 or 1).
// Returns (-1, -1, nil, ctx.Err()) if ctx is cancelled before a slot is available.
// The caller MUST call ReleaseSlot(poolIdx, slotIdx) after the upload completes.
func (p *DataPool) AcquireSlot(ctx context.Context, size int64) (poolIdx, slotIdx int, data []byte, err error) {
	for {
		if ctx.Err() != nil {
			return -1, -1, nil, ctx.Err()
		}

		for i := range p.slots {
			if p.slots[i].st.v.CompareAndSwap(dpReady, dpConsuming) {
				p.bytesConsumed.Add(size)
				return 0, i, p.slots[i].data[:size], nil
			}
		}

		// No READY slot: producer hasn't caught up yet.
		// Yield so the producer goroutine and its fill goroutines can run.
		stallStart := time.Now()
		runtime.Gosched()
		p.consumerStallNs.Add(time.Since(stallStart).Nanoseconds())
	}
}

// ReleaseSlot marks slot slotIdx EMPTY so the producer can refill it.
// poolIdx is ignored for DataPool (always 0); it is present to satisfy WritePool.
// Must be called exactly once per successful AcquireSlot, after all bytes
// in the returned slice have been consumed.
func (p *DataPool) ReleaseSlot(poolIdx, slotIdx int) {
	p.slots[slotIdx].st.v.Store(dpEmpty)
}

// SlotSize returns the size in bytes of each pre-allocated slot.
func (p *DataPool) SlotSize() int64 { return p.slotSize }

// PoolStats is a point-in-time snapshot of pool telemetry counters.
// Take two snapshots and subtract to compute per-interval rates.
type PoolStats struct {
	BytesProduced   int64 // cumulative bytes filled and placed into ready (slotSize per slot)
	BytesConsumed   int64 // cumulative bytes claimed by consumers (actual object sizes)
	ProducerStallNs int64 // cumulative goroutine-ns producers spent waiting for a free slot
	ConsumerStallNs int64 // cumulative goroutine-ns consumers spent waiting for a filled slot
	// TotalFillNs is cumulative wall-ns spent inside fillRandom across all producer
	// goroutines.  Divide by the number of slots filled to get average fill time;
	// combine with NumProducers to compute slot-fill capacity independent of pool
	// saturation state.
	TotalFillNs int64
	// NumProducers is the number of active fill goroutines at the moment Stats()
	// was called.  Used by the engine to compute slot-based headroom.
	NumProducers int32
	// MinProducers is the current adaptive floor — the minimum producer count that
	// the controller will not scale below.  Raised dynamically when headroom drops
	// below poolHeadroomTarget.
	MinProducers int32
}

// Stats returns a consistent snapshot of all telemetry counters.
// The counters are monotonically increasing; subtract two snapshots to get
// interval values for rate and stall-fraction calculations.
func (p *DataPool) Stats() PoolStats {
	return PoolStats{
		BytesProduced:   p.bytesProduced.Load(),
		BytesConsumed:   p.bytesConsumed.Load(),
		ProducerStallNs: p.producerStallNs.Load(),
		ConsumerStallNs: p.consumerStallNs.Load(),
		TotalFillNs:     0, // DataPool does not instrument per-fill timing
		NumProducers:    1, // DataPool always has exactly one producer goroutine
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DoubleDataPool — double-buffered pre-fill pipeline
// ─────────────────────────────────────────────────────────────────────────────

// DoubleDataPool implements WritePool using two identically-sized slot rings
// that alternate between ACTIVE (consumers drain) and STANDBY (producer fills).
//
// Why this eliminates consumer stall
// ───────────────────────────────────
//
//	With UNet3D settings (slotSize=32 MiB, depth=256):
//	  Producer fill rate ≈ 14.75 GiB/s  →  fill time for 8 GiB ≈ 0.54 s
//	  Consumer write rate ≈  3.15 GiB/s  →  drain time for 8 GiB ≈ 2.54 s
//	The standby pool is fully loaded ≈ 2 s before consumers exhaust the active
//	pool, so consumers never wait for data — stall ≈ 0 at steady state.
//
// Coordination protocol
// ─────────────────────
//  1. At construction pool[0] is filled synchronously; pool[1] is empty.
//  2. RunProducer goroutine starts filling pool[1] immediately.
//  3. Consumers drain pool[0].  When pool[0] is fully drained (no READY slots
//     remain) and pool[1] is ready (standbyReady==1), the first consumer to
//     notice CAS active 0→1 and clears standbyReady.
//  4. The producer waits for inFlight[stbyIdx]==0 (all uploads from the old
//     active complete), then fills every slot unconditionally — no per-slot CAS.
//  5. Cycle repeats.
//
// Producer stall
// ──────────────
//
//	The producer waits only once per cycle: for inFlight[stbyIdx]==0.
//	Once that condition holds, it has exclusive ownership of the standby pool
//	and can fill all slots with no per-slot coordination.  Consumer stalls are
//	eliminated because the pool swap and fill are decoupled.
//
// RAM usage
// ─────────
//
//	2 × depth × slotSize = 2 × 256 × 32 MiB = 16 GiB (for UNet3D defaults).
type DoubleDataPool struct {
	slotSz int64
	depth  int

	// Two independently-addressable slot arrays.
	slots [2][]dpSlot

	// entropy and seqCounter are used during the initial synchronous fill of
	// pool[0] and then owned exclusively by RunProducer for subsequent fills.
	entropy    uint64
	seqCounter *atomic.Uint64

	// active is the pool index (0 or 1) that consumers are currently draining.
	// Written only by the consumer that wins the swap CAS; read by all.
	active atomic.Uint32

	// standbyReady is set to 1 by the producer when the standby pool is fully
	// loaded and can be swapped in.  Cleared to 0 by the consumer that wins
	// the swap CAS.
	standbyReady atomic.Uint32

	// inFlight tracks how many consumer goroutines currently hold a reference
	// into each pool (i.e. between AcquireSlot and ReleaseSlot).  The producer
	// waits for inFlight[stbyIdx]==0 before overwriting the standby pool.
	inFlight [2]atomic.Int32

	// Telemetry counters — semantics identical to DataPool counterparts.
	bytesProduced   atomic.Int64 // bytes filled and marked READY (full slotSz per slot)
	bytesConsumed   atomic.Int64 // bytes claimed by consumers (actual object sizes)
	producerStallNs atomic.Int64 // producer goroutine-ns spent waiting for EMPTY slots
	consumerStallNs atomic.Int64 // consumer goroutine-ns spent waiting for READY slots
}

// newDoubleDataPool allocates two slot rings and synchronously fills pool[0]
// so consumers can start immediately without any initial stall.
//
//	slotSize   bytes per slot; must be ≥ max object size written by the engine
//	depth      ring depth per pool; clamped to ≥ 4
//	entropy    Xoshiro256++ seed (from newWriteEntropy in the engine)
//	seqCounter shared block-sequence counter (engine.writeBlockSeq)
func newDoubleDataPool(slotSize int64, depth int, entropy uint64, seqCounter *atomic.Uint64) *DoubleDataPool {
	if depth < poolAbsoluteMinDepth {
		depth = poolAbsoluteMinDepth
	}
	if slotSize < genBlockSize {
		slotSize = genBlockSize
	}

	d := &DoubleDataPool{
		slotSz:     slotSize,
		depth:      depth,
		entropy:    entropy,
		seqCounter: seqCounter,
	}

	// Allocate and first-touch both slot arrays up front so the kernel maps
	// physical RAM before the benchmark starts.
	for p := range d.slots {
		d.slots[p] = make([]dpSlot, depth)
		for i := range d.slots[p] {
			d.slots[p][i].data = make([]byte, slotSize)
			for off := int64(0); off < slotSize; off += poolFirstTouchPageStride {
				d.slots[p][i].data[off] = 0
			}
			d.slots[p][i].st.v.Store(dpEmpty)
		}
	}

	// Synchronously fill pool[0] so consumers can start the moment Run() begins,
	// with zero initial stall.
	nBlocks := uint64((slotSize + genBlockSize - 1) / genBlockSize)
	for i := range d.slots[0] {
		startSeq := seqCounter.Add(nBlocks) - nBlocks
		fillRandom(d.slots[0][i].data, entropy, startSeq)
		d.slots[0][i].st.v.Store(dpReady)
		d.bytesProduced.Add(slotSize)
	}
	// pool[0] is active; pool[1] is standby (EMPTY, to be filled by RunProducer).
	d.active.Store(0)
	d.standbyReady.Store(0)

	return d
}

// RunProducer is the body of the background fill goroutine for DoubleDataPool.
// The producer owns the standby pool exclusively: no consumer ever reads from
// it.  The only synchronization point per cycle is waiting for all in-flight
// uploads from the previous active pool to complete (inFlight[stbyIdx]==0),
// after which every slot is overwritten in one uncontested pass.
func (d *DoubleDataPool) RunProducer(ctx context.Context) {
	// pool[0] was pre-filled in the constructor; begin by filling pool[1].
	stbyIdx := uint32(1)
	nBlocks := uint64((d.slotSz + genBlockSize - 1) / genBlockSize)

	for {
		if ctx.Err() != nil {
			return
		}

		// ── Wait for all consumers to finish with the standby pool. ──────────
		// When this pool was last active, some goroutines may still be uploading
		// (inFlight > 0).  We must not overwrite their data buffers until they
		// call ReleaseSlot and drop the reference.  This is the producer's ONLY
		// synchronization point — there are no per-slot CAS loops.
		for d.inFlight[stbyIdx].Load() > 0 {
			if ctx.Err() != nil {
				return
			}
			stallStart := time.Now()
			runtime.Gosched()
			d.producerStallNs.Add(time.Since(stallStart).Nanoseconds())
		}

		// ── Fill all slots unconditionally — no coordination needed. ─────────
		// inFlight[stbyIdx]==0 guarantees exclusive ownership: no consumer holds
		// a reference into this pool and none will until we signal standbyReady.
		for i := range d.slots[stbyIdx] {
			startSeq := d.seqCounter.Add(nBlocks) - nBlocks
			fillRandom(d.slots[stbyIdx][i].data, d.entropy, startSeq)
			d.slots[stbyIdx][i].st.v.Store(dpReady)
			d.bytesProduced.Add(d.slotSz)
		}

		// Standby pool fully loaded: signal consumers that they may swap.
		d.standbyReady.Store(1)

		// Wait for consumers to flip activeIdx to stbyIdx.
		for d.active.Load() != stbyIdx {
			if ctx.Err() != nil {
				return
			}
			runtime.Gosched()
		}

		// Swap complete. The old active is now our new standby — repeat.
		stbyIdx = 1 - stbyIdx
	}
}

// AcquireSlot blocks until a READY slot is available in the active pool.
// If the active pool is fully drained and the standby pool is loaded, it
// atomically promotes the standby pool to active and drains from there.
//
// Returns (-1, -1, nil, ctx.Err()) if ctx is cancelled.
// The caller MUST call ReleaseSlot(poolIdx, slotIdx) after the upload completes.
func (d *DoubleDataPool) AcquireSlot(ctx context.Context, size int64) (poolIdx, slotIdx int, data []byte, err error) {
	for {
		if ctx.Err() != nil {
			return -1, -1, nil, ctx.Err()
		}

		ai := d.active.Load()
		pool := d.slots[ai]

		// Scan active pool for any READY slot.
		for i := range pool {
			if pool[i].st.v.CompareAndSwap(dpReady, dpConsuming) {
				d.inFlight[ai].Add(1) // producer must not overwrite until we release
				d.bytesConsumed.Add(size)
				return int(ai), i, pool[i].data[:size], nil
			}
		}

		// Active pool fully drained. Promote standby if it is ready.
		if d.standbyReady.Load() == 1 {
			stby := uint32(1 - ai)
			if d.active.CompareAndSwap(ai, stby) {
				d.standbyReady.Store(0) // release producer to start filling old active
			}
			// Win or lose the CAS, READY slots are available in the (new) active — retry.
			continue
		}

		// Neither pool ready. Re-check in case another goroutine just swapped.
		if d.active.Load() != ai {
			continue
		}
		stallStart := time.Now()
		runtime.Gosched()
		d.consumerStallNs.Add(time.Since(stallStart).Nanoseconds())
	}
}

// ReleaseSlot signals that the caller has finished with the slot data.
// The inFlight counter for this pool is decremented, allowing the producer
// to overwrite the slot when it next fills this pool as standby.
func (d *DoubleDataPool) ReleaseSlot(poolIdx, slotIdx int) {
	d.slots[poolIdx][slotIdx].st.v.Store(dpEmpty)
	d.inFlight[poolIdx].Add(-1)
}

// Stats returns a consistent snapshot of all telemetry counters.
func (d *DoubleDataPool) Stats() PoolStats {
	return PoolStats{
		BytesProduced:   d.bytesProduced.Load(),
		BytesConsumed:   d.bytesConsumed.Load(),
		ProducerStallNs: d.producerStallNs.Load(),
		ConsumerStallNs: d.consumerStallNs.Load(),
		TotalFillNs:     0, // DoubleDataPool does not instrument per-fill timing
		NumProducers:    1, // DoubleDataPool always has exactly one producer goroutine
	}
}

// SlotSize returns the size in bytes of each pre-allocated slot.
func (d *DoubleDataPool) SlotSize() int64 { return d.slotSz }

// ─────────────────────────────────────────────────────────────────────────────
// ChanPool — channel-pair pre-fill pipeline (production default)
// ─────────────────────────────────────────────────────────────────────────────
//
// Architecture
// ────────────
//
//	Slots circulate independently between two buffered channels:
//
//	  free  ──producer fills──▶  ready  ──consumer uploads──▶  free
//
//	A single slow GCS upload (e.g. 2000 ms tail latency) holds exactly ONE slot.
//	The remaining depth-1 slots cycle at full speed through the producer and
//	all other consumers simultaneously.  No group barrier exists — unlike the
//	DoubleDataPool model, consumers NEVER wait for an entire pool's worth of
//	in-flight uploads to clear.
//
// Stall analysis (UNet3D settings: depth ≈ 1193, 64 goroutines)
// ──────────────────────────────────────────────────────────────
//
//	Producer stall: free is empty — all slots are in 'ready' or held by consumers.
//	  Max in-flight = 64 (one per goroutine).  'ready' capacity = depth.
//	  Producer blocks only if depth + 64 > depth, which is structurally impossible.
//	  In practice freeSlots never empties (produce rate ≈ 4.6× consume rate).
//
//	Consumer stall: ready is empty — producer hasn't filled a slot yet.
//	  Only possible during the initial ramp-up (first ~0.5 ms per slot at 14 GiB/s
//	  fill rate) or if produce rate falls below consume rate — neither applies here.
//
// Debug logging
// ─────────────
//
//	At -vv (DEBUG level), stall events emit free/ready queue depths so you can
//	diagnose unexpected behaviour without modifying the code.
//
// RAM usage
// ─────────
//
//	1 × depth × slotSize ≈ 8 GiB (half the previous DoubleDataPool cost).

// chanPoolSlot is one slot in the ChanPool.  Unlike dpSlot there is no state
// atomic — channel membership IS the state (free=unfilled, ready=filled).
type chanPoolSlot struct {
	idx  int    // index into ChanPool.slots; used as the slotIdx token
	data []byte // pre-allocated backing buffer (length == ChanPool.slotSz)
}

// ChanPool implements WritePool using a channel-pair architecture.
type ChanPool struct {
	slotSz     int64
	entropy    uint64
	seqCounter *atomic.Uint64

	slots []chanPoolSlot     // pre-allocated backing store; indexed by chanPoolSlot.idx
	free  chan *chanPoolSlot // unfilled slots waiting for the producer
	ready chan *chanPoolSlot // filled slots waiting for a consumer

	// Set once by RunProducer before any fill goroutine is spawned; read-only
	// thereafter — no synchronisation needed.
	parentCtx    context.Context // root context for spawning new fill goroutines
	maxProducers atomic.Int32    // fill goroutine cap; set by RunProducer
	minProducers atomic.Int32    // adaptive floor — raised when headroom < target proves floor too low; never scale below

	// Dynamic producer management — mutex only touched on actual spawn/retire (rare).
	numProducers    atomic.Int32         // current number of active fill goroutines
	producerMu      sync.Mutex           // guards producerCancels
	producerCancels []context.CancelFunc // one entry per active fill goroutine

	// Per-fill controller state — all atomic, no mutex in the fill path.
	fillCount       atomic.Uint64 // total fills completed across all producers
	lastCheckNs     atomic.Int64  // time.Now().UnixNano() at last headroom evaluation
	lastCheckFillNs atomic.Int64  // totalFillNs snapshot at last headroom evaluation
	surplusStreak   atomic.Int32  // consecutive evaluations with headroom > poolHeadroomSurplus

	// Telemetry — semantics identical to DataPool/DoubleDataPool.
	bytesProduced   atomic.Int64 // bytes written into slots by the producer(s)
	bytesConsumed   atomic.Int64 // bytes claimed by consumers (actual object sizes)
	producerStallNs atomic.Int64 // cumulative goroutine-ns producers waited for a free slot
	consumerStallNs atomic.Int64 // cumulative goroutine-ns consumers waited for a ready slot
	totalFillNs     atomic.Int64 // cumulative ns spent inside fillRandom across all producers
}

// newChanPool allocates depth slots, synchronously fills all of them with
// random data, and places them into the ready channel so that consumers can
// start immediately with zero stall.  RunProducer begins blocked on free and
// enters the normal fill cycle as ReleaseSlot drains slots back to it.
//
//	slotSize   bytes per slot; must be ≥ max object size written by the engine
//	depth      total slot count; clamped to ≥ poolAbsoluteMinDepth
//	entropy    Xoshiro256++ seed
//	seqCounter shared block-sequence counter (engine.writeBlockSeq)
func newChanPool(slotSize int64, depth int, entropy uint64, seqCounter *atomic.Uint64) *ChanPool {
	if depth < poolAbsoluteMinDepth {
		depth = poolAbsoluteMinDepth
	}
	if slotSize < genBlockSize {
		slotSize = genBlockSize
	}

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
	// RunProducer will block on <-p.free until ReleaseSlot feeds slots back.
	nBlocks := uint64((slotSize + genBlockSize - 1) / genBlockSize)
	for i := range p.slots {
		p.slots[i].idx = i
		p.slots[i].data = make([]byte, slotSize)
		for off := int64(0); off < slotSize; off += poolFirstTouchPageStride {
			p.slots[i].data[off] = 0
		}
		startSeq := seqCounter.Add(nBlocks) - nBlocks
		fillRandom(p.slots[i].data, entropy, startSeq)
		p.bytesProduced.Add(slotSize)
		p.ready <- &p.slots[i]
	}

	return p
}

// fillLoop is the body of a single fill goroutine.  It receives free slots,
// fills them with random data, and sends them to ready.  After each successful
// fill it participates in per-fill headroom evaluation (see maybeAdjustProducers)
// so that scaling decisions are made at most once per fill round across all
// goroutines — with no mutex touched in the steady-state path.
//
// fillLoop exits when its personal ctx is cancelled (by the parent shutting down
// or by retireProducer retiring this specific goroutine).
func (p *ChanPool) fillLoop(ctx context.Context) {
	defer p.numProducers.Add(-1)
	nBlocks := uint64((p.slotSz + genBlockSize - 1) / genBlockSize)
	for {
		// Non-blocking receive first; fall back to blocking with stall accounting.
		var s *chanPoolSlot
		select {
		case s = <-p.free:
		default:
			if ctx.Err() != nil {
				return
			}
			// Producer waiting on a free slot is a healthy steady-state event
			// (all slots are either ready-to-consume or held by upload goroutines).
			// Log at TRACE so it doesn't flood DEBUG output.
			logger.Tracef("[pool] producer wait: free=0 ready=%d producers=%d\n",
				len(p.ready), p.numProducers.Load())
			stallStart := time.Now()
			select {
			case s = <-p.free:
			case <-ctx.Done():
				return
			}
			p.producerStallNs.Add(time.Since(stallStart).Nanoseconds())
		}

		startSeq := p.seqCounter.Add(nBlocks) - nBlocks
		fillStart := time.Now()
		fillRandom(s.data, p.entropy, startSeq)
		p.totalFillNs.Add(time.Since(fillStart).Nanoseconds())
		p.bytesProduced.Add(p.slotSz)

		select {
		case p.ready <- s:
		case <-ctx.Done():
			// Pool is shutting down.  Salvage the filled slot so no slot is
			// permanently lost from the circulation budget.
			select {
			case p.ready <- s:
			default:
			}
			return
		}

		// Headroom check — at most once per fill round (every numProducers fills
		// across all goroutines).  The mutex is only touched if we actually scale.
		n := p.fillCount.Add(1)
		numProd := uint64(p.numProducers.Load())
		if numProd > 0 && n%numProd == 0 {
			p.maybeAdjustProducers()
		}
	}
}

// spawnProducer launches one additional fill goroutine under p.parentCtx.
// The goroutine's personal cancellation function is stored so retireProducer
// can terminate exactly one goroutine at a time.
// The mutex is held only for the slice append — not during any fill work.
func (p *ChanPool) spawnProducer() {
	ctx, cancel := context.WithCancel(p.parentCtx)
	p.producerMu.Lock()
	p.producerCancels = append(p.producerCancels, cancel)
	p.producerMu.Unlock()
	p.numProducers.Add(1)
	go p.fillLoop(ctx)
}

// retireProducer cancels the most recently spawned fill goroutine (LIFO).
// The goroutine decrements numProducers via its defer when it exits.
// Does nothing if no producers are active.
func (p *ChanPool) retireProducer() {
	p.producerMu.Lock()
	defer p.producerMu.Unlock()
	n := len(p.producerCancels)
	if n == 0 {
		return
	}
	p.producerCancels[n-1]()
	p.producerCancels = p.producerCancels[:n-1]
}

// maybeAdjustProducers evaluates slot-based headroom and scales the fill
// goroutine count up or down.  It is called at most once per fill round
// (every numProducers fills collectively) directly from fillLoop — completely
// lock-free in the steady state.  The mutex is only acquired when actually
// spawning or retiring.
//
// Scale-up is immediate: as soon as headroom < poolHeadroomTarget.
// Scale-down requires poolDownscaleStreak consecutive above-surplus evaluations
// (hysteresis) so a momentary upload speed improvement doesn't prematurely
// shed producers that would be needed moments later.
func (p *ChanPool) maybeAdjustProducers() {
	now := time.Now().UnixNano()
	dtNs := now - p.lastCheckNs.Load()
	if dtNs < int64(time.Millisecond) {
		// Two goroutines landed on the same round slot (numProducers changed
		// mid-flight).  Skip: the interval is too short to measure meaningfully.
		return
	}

	curFillNs := p.totalFillNs.Load()
	dFillNs := curFillNs - p.lastCheckFillNs.Load()

	// Advance checkpoints so the next evaluation measures a fresh window.
	p.lastCheckNs.Store(now)
	p.lastCheckFillNs.Store(curFillNs)

	if dFillNs <= 0 {
		return // pool idle (fully pre-filled, no consumer demand) — nothing to act on
	}

	numProd := p.numProducers.Load()
	maxProd := p.maxProducers.Load()
	// Slot-based headroom = producerSlotsPerSec / consumerSlotsPerSec
	//                     = numProducers × dtNs / totalFillNs_delta
	headroom := float64(numProd) * float64(dtNs) / float64(dFillNs)

	switch {
	case headroom < poolHeadroomTarget && numProd < maxProd:
		// Scale up immediately — compute how many producers reach target in one step.
		needed := int32(math.Ceil(poolHeadroomTarget / headroom * float64(numProd)))
		if needed > maxProd {
			needed = maxProd
		}
		for p.numProducers.Load() < needed {
			p.spawnProducer()
		}
		// Raise the adaptive floor: if we needed more than the current floor, the
		// floor was too conservative.  Future scale-downs will stop here.
		if needed > p.minProducers.Load() {
			p.minProducers.Store(needed)
		}
		p.surplusStreak.Store(0)
		logger.Debugf("[pool] scaling up: headroom=%.2fx → %d producers (floor raised to %d)\n",
			headroom, p.numProducers.Load(), p.minProducers.Load())

	case headroom > poolHeadroomSurplus && numProd > p.minProducers.Load():
		// Increment surplus streak.  Retire only after poolDownscaleStreak
		// consecutive above-surplus evaluations (hysteresis against transient spikes).
		// Never scale below the adaptive floor — the floor captures the minimum we
		// have proven necessary to sustain 2× headroom.
		streak := p.surplusStreak.Add(1)
		if streak >= poolDownscaleStreak {
			// CAS to 0 ensures exactly one goroutine retires at this threshold
			// even if two goroutines simultaneously reach the streak limit.
			if p.surplusStreak.CompareAndSwap(streak, 0) {
				// Double-check floor under the CAS — numProducers may have changed.
				if p.numProducers.Load() > p.minProducers.Load() {
					p.retireProducer()
					logger.Debugf("[pool] scaling down: headroom=%.2fx for %d consecutive checks → %d producers (floor=%d)\n",
						headroom, poolDownscaleStreak, p.numProducers.Load(), p.minProducers.Load())
				}
			}
		}

	default:
		// Headroom is in the healthy zone (≥ target, ≤ surplus) — reset streak
		// so a future surplus window must be sustained before any retirement.
		p.surplusStreak.Store(0)
	}
}

// RunProducer initialises the dynamic producer system and waits for ctx to be
// cancelled.  It stores parentCtx and spawns the initial cohort of fill
// goroutines; all subsequent scaling decisions are made inside fillLoop via
// maybeAdjustProducers — no ticker or controller goroutine is needed.
//
// Call as:  go pool.RunProducer(ctx)
func (p *ChanPool) RunProducer(ctx context.Context) {
	// Store parentCtx before spawning any goroutines so that spawnProducer()
	// calls from within fillLoop see a valid context.
	p.parentCtx = ctx
	p.lastCheckNs.Store(time.Now().UnixNano())

	maxProd := int32(runtime.NumCPU() / poolMaxProducerDivisor)
	if maxProd < 1 {
		maxProd = 1
	}
	p.maxProducers.Store(maxProd)

	initial := int32(runtime.NumCPU() / poolInitialProducerDivisor)
	if initial < 1 {
		initial = 1
	}
	if initial > maxProd {
		initial = maxProd
	}
	// Floor starts at the initial cohort size — never scale below our proven safe
	// baseline.  Raised on scale-up when we discover the floor was too low.
	p.minProducers.Store(initial)

	for i := int32(0); i < initial; i++ {
		p.spawnProducer()
	}

	<-ctx.Done()
}

// AcquireSlot blocks until a filled slot is available in the ready channel.
// Returns (0, slotIdx, data[:size], nil) on success.
// Returns (-1, -1, nil, ctx.Err()) if ctx is cancelled.
// The caller MUST call ReleaseSlot(0, slotIdx) after the upload completes.
func (p *ChanPool) AcquireSlot(ctx context.Context, size int64) (poolIdx, slotIdx int, data []byte, err error) {
	var s *chanPoolSlot
	select {
	case s = <-p.ready:
	default:
		if ctx.Err() != nil {
			return -1, -1, nil, ctx.Err()
		}
		// Pool exhausted — log at DEBUG with diagnostics so the operator can
		// understand why: free=N shows slots waiting to be refilled; a high
		// free count with ready=0 means producers are overwhelmed.
		logger.Debugf("[pool] consumer stall: free=%d ready=0/%d producers=%d — all filled slots consumed; producers cannot keep pace\n",
			len(p.free), cap(p.ready), p.numProducers.Load())
		stallStart := time.Now()
		select {
		case s = <-p.ready:
		case <-ctx.Done():
			return -1, -1, nil, ctx.Err()
		}
		p.consumerStallNs.Add(time.Since(stallStart).Nanoseconds())
	}
	p.bytesConsumed.Add(size)
	return 0, s.idx, s.data[:size], nil
}

// ReleaseSlot returns the slot identified by slotIdx back to the free channel
// so the producer can refill it.  Must be called exactly once per AcquireSlot.
// poolIdx is ignored (always 0 for ChanPool); it is present to satisfy WritePool.
func (p *ChanPool) ReleaseSlot(poolIdx, slotIdx int) {
	p.free <- &p.slots[slotIdx]
}

// Stats returns a consistent snapshot of all telemetry counters.
func (p *ChanPool) Stats() PoolStats {
	return PoolStats{
		BytesProduced:   p.bytesProduced.Load(),
		BytesConsumed:   p.bytesConsumed.Load(),
		ProducerStallNs: p.producerStallNs.Load(),
		ConsumerStallNs: p.consumerStallNs.Load(),
		TotalFillNs:     p.totalFillNs.Load(),
		NumProducers:    p.numProducers.Load(),
		MinProducers:    p.minProducers.Load(),
	}
}

// SlotSize returns the maximum object size (bytes) this pool supports.
func (p *ChanPool) SlotSize() int64 { return p.slotSz }
