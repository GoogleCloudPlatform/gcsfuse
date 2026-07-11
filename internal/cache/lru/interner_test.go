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

package lru_test

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
	"unsafe"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/stretchr/testify/assert"
)

func TestPathSegmentInterner_Deduplication(t *testing.T) {
	interner := lru.NewPathSegmentInterner()

	s1 := string(bytes.Clone([]byte("foo/bar")))
	s2 := fmt.Sprintf("%s/%s", "foo", "bar")

	i1 := interner.Intern(s1)
	i2 := interner.Intern(s2)

	assert.Equal(t, s1, i1)
	assert.Equal(t, s2, i2)
	assert.Same(t, unsafe.StringData(i1), unsafe.StringData(i2))
}

func TestPathSegmentInterner_EmptyString(t *testing.T) {
	interner := lru.NewPathSegmentInterner()
	assert.Equal(t, "", interner.Intern(""))
}

func TestPathSegmentInterner_ConcurrencyStress(t *testing.T) {
	interner := lru.NewPathSegmentInterner()
	var wg sync.WaitGroup

	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				key := fmt.Sprintf("dir-%d/file.txt", j%20)
				interned := interner.Intern(key)
				assert.Equal(t, key, interned)
			}
		}(i)
	}

	wg.Wait()
}
