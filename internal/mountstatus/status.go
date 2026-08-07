// Copyright 2024 Google LLC
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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc/codes"
)

var statusFile *os.File

// InitStatusReporter configures the status reporting mechanism used to communicate
// mount progress and outcomes with the CSI sidecar.
// It only activates if running in foreground mode and GCSFUSE_STATUS_FD is provided.
// It performs safety checks to ensure the file descriptor represents a valid, writable pipe
// before caching it for subsequent calls to ReportStatus.
func InitStatusReporter(isForeground bool) {
	if !isForeground {
		return
	}

	statusFDStr := os.Getenv("GCSFUSE_STATUS_FD")
	if statusFDStr != "" {
		fd, err := strconv.ParseUint(statusFDStr, 10, 32)
		if err != nil {
			logger.Warnf("Failed to parse GCSFUSE_STATUS_FD: %v", err)
			return
		}

		f := os.NewFile(uintptr(fd), "status_pipe")
		if f == nil {
			logger.Warnf("Failed to create file from GCSFUSE_STATUS_FD")
			return
		}

		stat, err := f.Stat()
		if err != nil {
			logger.Warnf("Failed to stat GCSFUSE_STATUS_FD: %v", err)
			return
		}

		if stat.Mode()&os.ModeNamedPipe == 0 {
			logger.Warnf("GCSFUSE_STATUS_FD is not a pipe")
			return
		}

		flags, err := unix.FcntlInt(f.Fd(), unix.F_GETFL, 0)
		if err != nil {
			logger.Warnf("Failed to get flags for GCSFUSE_STATUS_FD: %v", err)
			return
		}

		accMode := flags & unix.O_ACCMODE
		if accMode != unix.O_WRONLY && accMode != unix.O_RDWR {
			logger.Warnf("GCSFUSE_STATUS_FD is not writable")
			return
		}

		statusFile = f
	}
}

// StatusPayload defines the JSON structure sent to the CSI sidecar over the UDS pipe.
// It maps the internal mount state to a gRPC code and descriptive error message.
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
