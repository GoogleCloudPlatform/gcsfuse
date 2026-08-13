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
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc/codes"
)

var statusFile *os.File

// openStatusPipe validates and opens the file descriptor provided via GCSFUSE_STATUS_FD.
// It ensures the descriptor represents a valid, writable named pipe and sets FD_CLOEXEC.
func openStatusPipe(statusFDStr string) (*os.File, error) {
	fd, err := strconv.Atoi(statusFDStr)
	if err != nil || fd < 0 {
		return nil, fmt.Errorf("invalid file descriptor %q", statusFDStr)
	}

	f := os.NewFile(uintptr(fd), "status_pipe")
	if f == nil {
		return nil, errors.New("failed to wrap descriptor in os.File")
	}

	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat failed: %w", err)
	}

	if stat.Mode()&os.ModeNamedPipe == 0 {
		return nil, errors.New("descriptor is not a named pipe")
	}

	flags, err := unix.FcntlInt(f.Fd(), unix.F_GETFL, 0)
	if err != nil {
		return nil, fmt.Errorf("fcntl F_GETFL failed: %w", err)
	}

	if accMode := flags & unix.O_ACCMODE; accMode != unix.O_WRONLY && accMode != unix.O_RDWR {
		return nil, errors.New("descriptor is not open for writing")
	}

	// Set close-on-exec so child processes do not inherit the status pipe.
	unix.CloseOnExec(fd)
	return f, nil
}

// InitStatusReporter configures the status reporting mechanism used to communicate
// mount progress and outcomes with the CSI sidecar.
// It only activates if running in foreground mode and GCSFUSE_STATUS_FD is provided.
func InitStatusReporter(isForeground bool) {
	if !isForeground || statusFile != nil {
		return
	}

	statusFDStr := os.Getenv("GCSFUSE_STATUS_FD")
	if statusFDStr == "" {
		return
	}

	f, err := openStatusPipe(statusFDStr)
	if err != nil {
		logger.Warnf("Failed to initialize status reporter from GCSFUSE_STATUS_FD: %v", err)
		return
	}

	statusFile = f
}

// StatusPayload defines the JSON structure sent to the CSI sidecar over the status pipe.
type StatusPayload struct {
	Status codes.Code `json:"status"`
	Error  string     `json:"error"`
}

// ReportStatus writes the provided gRPC code and error message as a JSON payload
// to the status file descriptor. It is safe to call even if the pipe is closed
// or unavailable, as any write failures will simply be logged as warnings.
func ReportStatus(code codes.Code, errMsg string) {
	if statusFile == nil {
		return
	}

	payload := StatusPayload{
		Status: code,
		Error:  errMsg,
	}

	if err := json.NewEncoder(statusFile).Encode(payload); err != nil {
		logger.Warnf("Failed to report status to GCSFUSE_STATUS_FD: %v", err)
	}
}
