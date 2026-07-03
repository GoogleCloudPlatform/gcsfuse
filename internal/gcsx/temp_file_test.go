// Copyright 2015 Google LLC
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

package gcsx_test

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func readAll(rs io.ReadSeeker) (content []byte, err error) {
	_, err = rs.Seek(0, 0)
	if err != nil {
		err = fmt.Errorf("Seek: %w", err)
		return
	}

	content, err = io.ReadAll(rs)
	if err != nil {
		err = fmt.Errorf("ReadFull: %w", err)
		return
	}

	return
}

type dummyReadCloser struct {
	io.Reader
}

func (rc dummyReadCloser) Close() error {
	return nil
}

////////////////////////////////////////////////////////////////////////
// Invariant-checking temp file
////////////////////////////////////////////////////////////////////////

// A wrapper around a TempFile that calls CheckInvariants whenever invariants
// should hold. For catching logic errors early in the test.
type checkingTempFile struct {
	wrapped gcsx.TempFile
}

func (tf *checkingTempFile) Stat() (gcsx.StatResult, error) {
	tf.wrapped.CheckInvariants()
	defer tf.wrapped.CheckInvariants()
	return tf.wrapped.Stat()
}

func (tf *checkingTempFile) Read(b []byte) (int, error) {
	tf.wrapped.CheckInvariants()
	defer tf.wrapped.CheckInvariants()
	return tf.wrapped.Read(b)
}

func (tf *checkingTempFile) Seek(offset int64, whence int) (int64, error) {
	tf.wrapped.CheckInvariants()
	defer tf.wrapped.CheckInvariants()
	return tf.wrapped.Seek(offset, whence)
}

func (tf *checkingTempFile) ReadAt(b []byte, o int64) (int, error) {
	tf.wrapped.CheckInvariants()
	defer tf.wrapped.CheckInvariants()
	return tf.wrapped.ReadAt(b, o)
}

func (tf *checkingTempFile) WriteAt(b []byte, o int64) (int, error) {
	tf.wrapped.CheckInvariants()
	defer tf.wrapped.CheckInvariants()
	return tf.wrapped.WriteAt(b, o)
}

func (tf *checkingTempFile) Truncate(n int64) error {
	tf.wrapped.CheckInvariants()
	defer tf.wrapped.CheckInvariants()
	return tf.wrapped.Truncate(n)
}

func (tf *checkingTempFile) SetMtime(mtime time.Time) {
	tf.wrapped.CheckInvariants()
	defer tf.wrapped.CheckInvariants()
	tf.wrapped.SetMtime(mtime)
}

func (tf *checkingTempFile) Destroy() {
	tf.wrapped.CheckInvariants()
	tf.wrapped.Destroy()
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const initialContent = "tacoburrito"

const initialContentSize = len(initialContent)

func setupTest(t *testing.T) (*timeutil.SimulatedClock, checkingTempFile) {
	// Set up the clock.
	clock := &timeutil.SimulatedClock{}
	clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))

	// And the temp file.
	var tf checkingTempFile
	var err error
	tf.wrapped, err = gcsx.NewTempFile(
		dummyReadCloser{strings.NewReader(initialContent)},
		"",
		clock)

	require.NoError(t, err)

	t.Cleanup(func() {
		tf.Destroy()
	})

	return clock, tf
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func TestStat_InitialState(t *testing.T) {
	_, tf := setupTest(t)
	sr, err := tf.Stat()

	require.NoError(t, err)
	assert.Equal(t, int64(initialContentSize), sr.Size)
	assert.Equal(t, int64(initialContentSize), sr.DirtyThreshold)
	assert.Nil(t, sr.Mtime)
}

func TestReadAt(t *testing.T) {
	_, tf := setupTest(t)
	// Call
	var buf [2]byte
	n, err := tf.ReadAt(buf[:], 1)

	assert.Equal(t, 2, n)
	assert.NoError(t, err)
	assert.Equal(t, initialContent[1:3], string(buf[:]))

	n, err = tf.ReadAt(buf[:], int64(initialContentSize)-1)
	assert.Equal(t, 1, n)
	assert.Equal(t, io.EOF, err)
	assert.Equal(t,
		initialContent[initialContentSize-1:initialContentSize],
		string(buf[0:n]),
	)

	// Check Stat.
	sr, err := tf.Stat()

	require.NoError(t, err)
	assert.Equal(t, int64(initialContentSize), sr.Size)
	assert.Equal(t, int64(initialContentSize), sr.DirtyThreshold)
	assert.Nil(t, sr.Mtime)
}

func TestWriteAt(t *testing.T) {
	clock, tf := setupTest(t)
	// Call
	p := []byte("fo")
	n, err := tf.WriteAt(p, 1)

	assert.Equal(t, 2, n)
	assert.NoError(t, err)

	// Check Stat.
	sr, err := tf.Stat()

	require.NoError(t, err)
	assert.Equal(t, int64(initialContentSize), sr.Size)
	assert.Equal(t, int64(1), sr.DirtyThreshold)
	require.NotNil(t, sr.Mtime)
	assert.Equal(t, clock.Now(), *sr.Mtime)

	// Read back.
	expected := []byte(initialContent)
	expected[1] = 'f'
	expected[2] = 'o'

	actual, err := readAll(&tf)
	require.NoError(t, err)
	assert.Equal(t, string(expected), string(actual))
}

func TestTruncate(t *testing.T) {
	clock, tf := setupTest(t)
	// Call
	err := tf.Truncate(2)
	assert.NoError(t, err)

	// Check Stat.
	sr, err := tf.Stat()

	require.NoError(t, err)
	assert.Equal(t, int64(2), sr.Size)
	assert.Equal(t, int64(2), sr.DirtyThreshold)
	require.NotNil(t, sr.Mtime)
	assert.Equal(t, clock.Now(), *sr.Mtime)

	// Read back.
	expected := initialContent[0:2]

	actual, err := readAll(&tf)
	require.NoError(t, err)
	assert.Equal(t, expected, string(actual))
}

func TestSetMtime(t *testing.T) {
	clock, tf := setupTest(t)
	mtime := time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local)
	assert.NotEqual(t, clock.Now(), mtime)

	// Set.
	tf.SetMtime(mtime)

	// Check.
	sr, err := tf.Stat()

	require.NoError(t, err)
	require.NotNil(t, sr.Mtime)
	assert.Equal(t, mtime, *sr.Mtime)
}
