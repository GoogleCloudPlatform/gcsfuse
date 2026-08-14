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
	"os"
	"strconv"
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

func TestOpenStatusPipe_ValidNamedPipe(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer func() { _ = r.Close() }()
	defer func() { _ = w.Close() }()

	f, err := openStatusPipe(strconv.Itoa(int(w.Fd())))

	require.NoError(t, err)
	require.NotNil(t, f)
	defer func() { _ = f.Close() }()
	assert.Equal(t, w.Fd(), f.Fd())
}

func TestOpenStatusPipe_InvalidFDString(t *testing.T) {
	f, err := openStatusPipe("not-an-int")

	assert.Error(t, err)
	assert.Nil(t, f)
}

func TestInitStatusReporter_NotForeground(t *testing.T) {
	teardown := setStatusFileForTesting(nil)
	defer teardown()

	InitStatusReporter(false)

	assert.Nil(t, statusFile)
}

func TestInitStatusReporter_NoEnvVariable(t *testing.T) {
	t.Setenv("GCSFUSE_STATUS_FD", "")
	teardown := setStatusFileForTesting(nil)
	defer teardown()

	InitStatusReporter(true)

	assert.Nil(t, statusFile)
}

func TestInitStatusReporter_InvalidFDInEnv(t *testing.T) {
	t.Setenv("GCSFUSE_STATUS_FD", "invalid_fd")
	teardown := setStatusFileForTesting(nil)
	defer teardown()

	InitStatusReporter(true)

	assert.Nil(t, statusFile)
}

func TestInitStatusReporter_ValidPipeForeground(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer func() { _ = r.Close() }()
	defer func() { _ = w.Close() }()
	t.Setenv("GCSFUSE_STATUS_FD", strconv.Itoa(int(w.Fd())))
	teardown := setStatusFileForTesting(nil)
	defer func() {
		if statusFile != nil {
			_ = statusFile.Close()
		}
		teardown()
	}()

	InitStatusReporter(true)

	require.NotNil(t, statusFile)
	assert.Equal(t, w.Fd(), statusFile.Fd())
}

func TestInitStatusReporter_AlreadyInitialized(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer func() { _ = r.Close() }()
	defer func() { _ = w.Close() }()
	teardown := setStatusFileForTesting(w)
	defer teardown()

	InitStatusReporter(true)

	assert.Equal(t, w, statusFile)
}

func TestReportStatus_NilStatusFile(t *testing.T) {
	teardown := setStatusFileForTesting(nil)
	defer teardown()

	ReportStatus(codes.OK, "no-op message")
}

func TestReportStatus_ReportPayload(t *testing.T) {
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
}
