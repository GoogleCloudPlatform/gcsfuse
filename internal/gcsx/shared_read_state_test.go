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

package gcsx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSharedReadState(t *testing.T) {
	state := NewSharedReadState()

	assert.NotNil(t, state)
	assert.Equal(t, ReadTypeUnknown, state.currentReadType)
	assert.Equal(t, "", state.activeReaderType)
}

func TestRecordRead(t *testing.T) {
	state := NewSharedReadState()

	// Record first read (sequential by default)
	state.RecordRead(0, 100)

	stats := state.GetReadStats()
	assert.Equal(t, uint64(100), stats.TotalBytesRead)
	assert.Equal(t, uint64(0), stats.RandomSeekCount) // First read is sequential
	assert.Equal(t, 0.0, stats.AverageBytesPerSeek)   // No random seeks yet
	assert.Equal(t, ReadTypeSequential, stats.CurrentReadType)

	// Record sequential read
	state.RecordRead(100, 50)

	stats = state.GetReadStats()
	assert.Equal(t, uint64(150), stats.TotalBytesRead)
	assert.Equal(t, uint64(0), stats.RandomSeekCount) // Still no random seeks
	assert.Equal(t, 0.0, stats.AverageBytesPerSeek)
	assert.Equal(t, ReadTypeSequential, stats.CurrentReadType)

	// Record random read
	state.RecordRead(500, 200)

	stats = state.GetReadStats()
	assert.Equal(t, uint64(350), stats.TotalBytesRead)
	assert.Equal(t, uint64(1), stats.RandomSeekCount) // One random seek
	assert.Equal(t, 350.0, stats.AverageBytesPerSeek) // 350 total bytes / 1 random seek
	assert.Equal(t, ReadTypeRandom, stats.CurrentReadType)
}

func TestSetActiveReaderType(t *testing.T) {
	state := NewSharedReadState()

	assert.Equal(t, "", state.GetActiveReaderType())

	state.SetActiveReaderType("BufferedReader")

	stats := state.GetReadStats()
	assert.Equal(t, "BufferedReader", stats.ActiveReaderType)
	assert.Equal(t, "BufferedReader", state.GetActiveReaderType())
}

func TestGetCurrentReadType(t *testing.T) {
	state := NewSharedReadState()

	// Initially unknown
	assert.Equal(t, ReadTypeUnknown, state.GetCurrentReadType())

	// After sequential read
	state.RecordRead(0, 100)
	assert.Equal(t, ReadTypeSequential, state.GetCurrentReadType())

	// After random read
	state.RecordRead(500, 50)
	assert.Equal(t, ReadTypeRandom, state.GetCurrentReadType())
}

func TestGetAverageBytesPerSeek(t *testing.T) {
	state := NewSharedReadState()

	// No seeks yet
	assert.Equal(t, 0.0, state.GetAverageBytesPerSeek())

	// Sequential reads (no seeks)
	state.RecordRead(0, 100)
	state.RecordRead(100, 100)
	assert.Equal(t, 0.0, state.GetAverageBytesPerSeek())

	// Random read (1 seek)
	state.RecordRead(500, 100)
	assert.Equal(t, 300.0, state.GetAverageBytesPerSeek()) // 300 total bytes / 1 seek

	// Another random read (2 seeks)
	state.RecordRead(1000, 200)
	assert.Equal(t, 250.0, state.GetAverageBytesPerSeek()) // 500 total bytes / 2 seeks
}

func TestReset(t *testing.T) {
	state := NewSharedReadState()

	// Add some data
	state.RecordRead(0, 100)
	state.RecordRead(500, 200) // random read
	state.SetActiveReaderType("BufferedReader")

	// Verify data is there
	stats := state.GetReadStats()
	assert.Equal(t, uint64(300), stats.TotalBytesRead)
	assert.Equal(t, uint64(1), stats.RandomSeekCount)
	assert.Equal(t, "BufferedReader", stats.ActiveReaderType)
	assert.Equal(t, ReadTypeRandom, stats.CurrentReadType)

	// Reset
	state.Reset()

	// Verify everything is cleared
	stats = state.GetReadStats()
	assert.Equal(t, uint64(0), stats.TotalBytesRead)
	assert.Equal(t, uint64(0), stats.RandomSeekCount)
	assert.Equal(t, 0.0, stats.AverageBytesPerSeek)
	assert.Equal(t, ReadTypeUnknown, stats.CurrentReadType)
	assert.Equal(t, "", stats.ActiveReaderType)
}
