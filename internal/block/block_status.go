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

package block

// blockStatusInProgress
type blockStatusInProgress struct {
	state BlockState
}

func (b blockStatusInProgress) State() BlockState {
	return b.state
}

func (b blockStatusInProgress) Error() error {
	return nil
}

// blockStatusDownloadSuccess
type blockStatusDownloadSuccess struct {
	state BlockState
}

func (b blockStatusDownloadSuccess) State() BlockState {
	return b.state
}

func (b blockStatusDownloadSuccess) Error() error {
	return nil
}

// blockStatusDownloadError
type blockStatusDownloadError struct {
	state BlockState
	err   error
}

func (b blockStatusDownloadError) State() BlockState {
	return b.state
}

func (b blockStatusDownloadError) Error() error {
	return b.err
}

// blockStatusDownloadCancelled
type blockStatusDownloadCancelled struct {
	state BlockState
	err   error
}

func (b blockStatusDownloadCancelled) State() BlockState {
	return b.state
}

func (b blockStatusDownloadCancelled) Error() error {
	return b.err
}

// BlockStatus represents the status of a block download.
// It provides methods to retrieve the current state of the block and any error
// that may have occurred during the download process specifically for Failed state.
type BlockStatus interface {
	// State returns the current state of the block.
	State() BlockState

	// Error returns an error if the block is in a failed or cancelled state.
	Error() error
}

// NewBlockStatus creates a new BlockStatus based on the provided state and error.
func NewBlockStatus(state BlockState, err error) BlockStatus {
	switch state {
	case BlockStateDownloaded:
		return blockStatusDownloadSuccess{state: state}
	case BlockStateDownloadFailed:
		return blockStatusDownloadError{state: state, err: err}
	case BlockStateDownloadCancelled:
		return blockStatusDownloadCancelled{state: state, err: err}
	case BlockStateInProgress:
		return blockStatusInProgress{state: state}
	default:
		return nil // or handle unexpected state
	}
}

// BlockState represents the status of the block.
type BlockState int

const (
	BlockStateInProgress        BlockState = iota // Download of this block is in progress
	BlockStateDownloaded                          // Download of this block is complete
	BlockStateDownloadFailed                      // Download of this block has failed
	BlockStateDownloadCancelled                   // Download of this block has been cancelled
)
