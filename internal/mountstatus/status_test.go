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

package mountstatus

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func setStatusFileForTesting(f *os.File) func() {
	old := statusFile
	statusFile = f

	return func() {
		statusFile = old
	}
}

func TestOpenStatusPipe(t *testing.T) {
	t.Run("ValidNamedPipe", func(t *testing.T) {
		r, w, err := os.Pipe()
		require.NoError(t, err)
		defer func() { _ = r.Close() }()
		defer func() { _ = w.Close() }()

		f, err := openStatusPipe(strconv.FormatUint(uint64(w.Fd()), 10))
		require.NoError(t, err)
		require.NotNil(t, f)
		assert.Equal(t, w.Fd(), f.Fd())
	})

	t.Run("InvalidFDString", func(t *testing.T) {
		f, err := openStatusPipe("not-an-int")
		assert.Error(t, err)
		assert.Nil(t, f)
	})

	t.Run("NegativeFD", func(t *testing.T) {
		f, err := openStatusPipe("-5")
		assert.Error(t, err)
		assert.Nil(t, f)
	})

	t.Run("NonPipeFile", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "regular_file")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()
		defer func() { _ = tmpFile.Close() }()

		f, err := openStatusPipe(strconv.FormatUint(uint64(tmpFile.Fd()), 10))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "descriptor is not a named pipe")
		assert.Nil(t, f)
	})

	t.Run("ReadOnlyPipe", func(t *testing.T) {
		r, w, err := os.Pipe()
		require.NoError(t, err)
		defer func() { _ = r.Close() }()
		defer func() { _ = w.Close() }()

		f, err := openStatusPipe(strconv.FormatUint(uint64(r.Fd()), 10))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "descriptor is not open for writing")
		assert.Nil(t, f)
	})
}

func TestInitStatusReporter(t *testing.T) {
	t.Run("NotForeground", func(t *testing.T) {
		r, w, err := os.Pipe()
		require.NoError(t, err)
		defer func() { _ = r.Close() }()
		defer func() { _ = w.Close() }()

		t.Setenv("GCSFUSE_STATUS_FD", strconv.FormatUint(uint64(w.Fd()), 10))

		teardown := setStatusFileForTesting(nil)
		defer teardown()

		InitStatusReporter(false)

		assert.Nil(t, statusFile)
	})

	t.Run("NoEnvVariable", func(t *testing.T) {
		t.Setenv("GCSFUSE_STATUS_FD", "")

		teardown := setStatusFileForTesting(nil)
		defer teardown()

		InitStatusReporter(true)

		assert.Nil(t, statusFile)
	})

	t.Run("InvalidFDInEnv", func(t *testing.T) {
		t.Setenv("GCSFUSE_STATUS_FD", "invalid_fd")

		teardown := setStatusFileForTesting(nil)
		defer teardown()

		InitStatusReporter(true)

		assert.Nil(t, statusFile)
	})

	t.Run("ValidPipeForeground", func(t *testing.T) {
		r, w, err := os.Pipe()
		require.NoError(t, err)
		defer func() { _ = r.Close() }()
		defer func() { _ = w.Close() }()

		t.Setenv("GCSFUSE_STATUS_FD", strconv.FormatUint(uint64(w.Fd()), 10))

		teardown := setStatusFileForTesting(nil)
		defer teardown()

		InitStatusReporter(true)

		require.NotNil(t, statusFile)
		assert.Equal(t, w.Fd(), statusFile.Fd())
	})

	t.Run("AlreadyInitialized", func(t *testing.T) {
		r, w, err := os.Pipe()
		require.NoError(t, err)
		defer func() { _ = r.Close() }()
		defer func() { _ = w.Close() }()

		teardown := setStatusFileForTesting(w)
		defer teardown()

		InitStatusReporter(true)

		assert.Equal(t, w, statusFile)
	})
}

func TestReportStatus(t *testing.T) {
	t.Run("StatusFileNil", func(t *testing.T) {
		teardown := setStatusFileForTesting(nil)
		defer teardown()

		// Should safely return without panic.
		ReportStatus(codes.OK, "no-op message")
	})

	t.Run("ReportError", func(t *testing.T) {
		r, w, err := os.Pipe()
		require.NoError(t, err)
		defer func() { _ = r.Close() }()
		defer func() { _ = w.Close() }()

		teardown := setStatusFileForTesting(w)
		defer teardown()

		ReportStatus(codes.PermissionDenied, "Waiting for IAM permissions")

		decoder := json.NewDecoder(r)
		var payload StatusPayload
		err = decoder.Decode(&payload)
		require.NoError(t, err)

		assert.Equal(t, codes.PermissionDenied, payload.Status)
		assert.Equal(t, "Waiting for IAM permissions", payload.Error)
	})

	t.Run("ReportOK", func(t *testing.T) {
		r, w, err := os.Pipe()
		require.NoError(t, err)
		defer func() { _ = r.Close() }()
		defer func() { _ = w.Close() }()

		teardown := setStatusFileForTesting(w)
		defer teardown()

		ReportStatus(codes.OK, "File system has been successfully mounted.")

		decoder := json.NewDecoder(r)
		var payload StatusPayload
		err = decoder.Decode(&payload)
		require.NoError(t, err)

		assert.Equal(t, codes.OK, payload.Status)
		assert.Equal(t, "File system has been successfully mounted.", payload.Error)
	})

	t.Run("WriteErrorOnClosedPipe", func(t *testing.T) {
		r, w, err := os.Pipe()
		require.NoError(t, err)
		_ = r.Close()
		_ = w.Close()

		teardown := setStatusFileForTesting(w)
		defer teardown()

		// Writing to closed pipe should log warning and not panic.
		ReportStatus(codes.OK, "msg to closed pipe")
	})
}

func TestReportStatus_ConcurrentSafety(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer func() { _ = r.Close() }()
	defer func() { _ = w.Close() }()

	teardown := setStatusFileForTesting(w)
	defer teardown()

	const numGoroutines = 20
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ReportStatus(codes.Code(idx%17), fmt.Sprintf("status message %d", idx))
		}(i)
	}

	wg.Wait()
	_ = w.Close()

	decoder := json.NewDecoder(r)
	count := 0
	for {
		var payload StatusPayload
		if err := decoder.Decode(&payload); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("JSON decode error on line %d: %v", count+1, err)
		}
		count++
	}

	assert.Equal(t, numGoroutines, count)
}
