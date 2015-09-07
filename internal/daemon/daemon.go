// Copyright 2015 Google Inc. All Rights Reserved.
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

// Helper code for daemonizing gcsfuse, synchronizing on successful mount.
//
// The details of this package are subject to change.
package daemon

import (
	"errors"
	"io"
	"os"
)

// For use by gcsfuse: signal that mounting was successful (allowing the caller
// of the process to return in success) or that there was a failure to mount
// the file system (allowing the caller of the process to display an
// appropriate error message).
//
// Do nothing if the process wasn't invoked with Mount.
func SignalOutcome(outcome error) (err error) {
	err = errors.New("TODO")
	return
}

// For use by gcsfuse: return a writer that should be used for logging status
// messages while in the process of mounting.
//
// The returned writer must not be written to after calling SignalOutcome.
func StatusWriter() (w io.Writer) {
	panic("TODO")
}

// Invoke gcsfuse with the supplied arguments, waiting until it successfully
// mounts or reports that is has failed. Write status updates while mounting
// into the supplied file (which may be nil for silence). Return nil only if it
// mounts successfully.
func Mount(
	gcsfusePath string,
	args []string,
	status *os.File) (err error) {
	err = errors.New("TODO")
	return
}
