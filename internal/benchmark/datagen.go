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
	"encoding/binary"
	"math/bits"
	"math/rand/v2"
	"sync"
)

// genBlockSize is the unit of parallel data generation (1 MiB).
// Matches dgen-rs BLOCK_SIZE: large enough to amortise Rayon/goroutine
// overhead, small enough to distribute evenly across cores.
const genBlockSize = 1 << 20 // 1 MiB

// GenBlockSize is the exported form of genBlockSize for use by other packages.
const GenBlockSize = genBlockSize

// parallelGenThreshold is the minimum number of blocks before parallel
// generation is used. Below this, goroutine overhead outweighs the benefit.
const parallelGenThreshold = 2

// newWriteEntropy returns a cryptographically-seeded random uint64.
// math/rand/v2 global functions are automatically seeded from OS entropy
// (unlike math/rand v1), so this is equivalent to reading from /dev/urandom.
func newWriteEntropy() uint64 {
	return rand.Uint64()
}

// ── Xoshiro256++ ──────────────────────────────────────────────────────────────
//
// Xoshiro256++ is the same PRNG used by dgen-rs (rand_xoshiro::Xoshiro256PlusPlus).
// It is extremely fast on modern CPUs (compilers can auto-vectorise the
// inner loop with SIMD), passes all statistical tests in BigCrush, and
// produces genuinely high-entropy output: every 1 MiB block filled here is
// statistically incompressible.
//
// Reference: https://prng.di.unimi.it/

// xoshiro256pp holds the 256-bit state of a Xoshiro256++ generator.
type xoshiro256pp struct {
	s [4]uint64
}

// newXoshiro256pp seeds a Xoshiro256++ from a single uint64 using SplitMix64
// to expand it to 256 bits — identical to rand_xoshiro::Xoshiro256PlusPlus::seed_from_u64.
func newXoshiro256pp(seed uint64) xoshiro256pp {
	var r xoshiro256pp
	r.s[0] = splitMix64Step(&seed)
	r.s[1] = splitMix64Step(&seed)
	r.s[2] = splitMix64Step(&seed)
	r.s[3] = splitMix64Step(&seed)
	return r
}

// splitMix64Step advances the SplitMix64 state z by one step and returns the output.
// Used only for seeding; not used during bulk generation.
func splitMix64Step(z *uint64) uint64 {
	*z += 0x9e3779b97f4a7c15
	x := *z
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	return x ^ (x >> 31)
}

// next advances the Xoshiro256++ state and returns the next pseudo-random uint64.
//
//go:nosplit
func (r *xoshiro256pp) next() uint64 {
	result := bits.RotateLeft64(r.s[0]+r.s[3], 23) + r.s[0]
	t := r.s[1] << 17
	r.s[2] ^= r.s[0]
	r.s[3] ^= r.s[1]
	r.s[1] ^= r.s[2]
	r.s[0] ^= r.s[3]
	r.s[2] ^= t
	r.s[3] = bits.RotateLeft64(r.s[3], 45)
	return result
}

// fillBytes fills dst with Xoshiro256++ output.
// Writes 8 bytes per generator step; handles non-multiple-of-8 lengths.
func (r *xoshiro256pp) fillBytes(dst []byte) {
	i := 0
	for i+8 <= len(dst) {
		binary.LittleEndian.PutUint64(dst[i:], r.next())
		i += 8
	}
	// Tail: fewer than 8 bytes remain.
	if i < len(dst) {
		var tail [8]byte
		binary.LittleEndian.PutUint64(tail[:], r.next())
		copy(dst[i:], tail[:len(dst)-i])
	}
}

// ── fillRandom ────────────────────────────────────────────────────────────────

// FillRandom is the exported form of fillRandom for use by benchmarking
// commands and tests outside the package.  See fillRandom for full docs.
func FillRandom(dst []byte, entropy, startSeq uint64) {
	fillRandom(dst, entropy, startSeq)
}

// fillRandom fills dst with incompressible pseudo-random bytes.
//
// The buffer is divided into genBlockSize (1 MiB) blocks. Each block is seeded
// independently:
//
//	block_seed = entropy + startSeq + block_index
//
// This ensures that every block produced across all concurrent writers in a
// process has a unique RNG state, so the written objects are never identical
// and the data is genuinely incompressible.
//
// When dst spans ≥ parallelGenThreshold blocks, generation is parallelised
// with goroutines (one per block) using a shared WaitGroup — the same
// structure as dgen-rs's rayon::par_chunks_mut path.
func fillRandom(dst []byte, entropy uint64, startSeq uint64) {
	n := len(dst)
	if n == 0 {
		return
	}

	nblocks := (n + genBlockSize - 1) / genBlockSize

	if nblocks < parallelGenThreshold {
		// Sequential path: one RNG, one goroutine (small objects < 2 MiB).
		rng := newXoshiro256pp(entropy + startSeq)
		rng.fillBytes(dst)
		return
	}

	// Parallel path: each block is seeded and filled independently.
	// Goroutines write directly into non-overlapping sub-slices of dst — zero-copy,
	// equivalent to dgen-rs's par_chunks_mut with fill_block per chunk.
	var wg sync.WaitGroup
	wg.Add(nblocks)
	for i := 0; i < nblocks; i++ {
		start := i * genBlockSize
		end := start + genBlockSize
		if end > n {
			end = n
		}
		blockSeed := entropy + startSeq + uint64(i)
		go func(block []byte, seed uint64) {
			defer wg.Done()
			rng := newXoshiro256pp(seed)
			rng.fillBytes(block)
		}(dst[start:end], blockSeed)
	}
	wg.Wait()
}
