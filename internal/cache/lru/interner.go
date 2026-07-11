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

package lru

import (
	"strings"
	"sync"
)

const maxCapacityPerShard = 8192

type internerShard struct {
	mu sync.RWMutex
	m  map[string]string
}

// PathSegmentInterner is a thread-safe 64-shard string interning pool.
type PathSegmentInterner struct {
	shards [64]internerShard
}

// NewPathSegmentInterner constructs a thread-safe 64-shard string interner.
func NewPathSegmentInterner() *PathSegmentInterner {
	p := &PathSegmentInterner{}
	for i := 0; i < 64; i++ {
		p.shards[i].m = make(map[string]string)
	}
	return p
}

// Intern returns the canonical interned instance of the string s.
func (p *PathSegmentInterner) Intern(s string) string {
	if len(s) == 0 {
		return ""
	}
	hash := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		hash ^= uint32(s[i])
		hash *= 16777619
	}
	shardIdx := hash & 63
	shard := &p.shards[shardIdx]

	shard.mu.RLock()
	if existing, ok := shard.m[s]; ok {
		shard.mu.RUnlock()
		return existing
	}
	shard.mu.RUnlock()

	shard.mu.Lock()
	if existing, ok := shard.m[s]; ok {
		shard.mu.Unlock()
		return existing
	}
	if len(shard.m) >= maxCapacityPerShard {
		shard.m = make(map[string]string, maxCapacityPerShard)
	}
	canonical := strings.Clone(s)
	shard.m[canonical] = canonical
	shard.mu.Unlock()
	return canonical
}

var globalInterner = NewPathSegmentInterner()

// Intern returns the canonical interned instance of string s via the global interner pool.
func Intern(s string) string {
	return globalInterner.Intern(s)
}

