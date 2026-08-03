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

package storageutil

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRenameFolderOpPoller struct {
	mock.Mock
}

func (m *mockRenameFolderOpPoller) Poll(ctx context.Context, opts ...gax.CallOption) (*controlpb.Folder, error) {
	args := m.Called(ctx)
	if f, ok := args.Get(0).(*controlpb.Folder); ok {
		return f, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRenameFolderOpPoller) Done() bool {
	args := m.Called()
	return args.Bool(0)
}

func TestPollRenameFolderOperation_ImmediateSuccess(t *testing.T) {
	ctx := context.Background()
	poller := new(mockRenameFolderOpPoller)
	expectedFolder := &controlpb.Folder{Name: "projects/_/buckets/b/folders/f"}
	poller.On("Poll", ctx).Return(expectedFolder, nil).Once()
	poller.On("Done").Return(true).Once()
	cfg := RenamePollConfig{
		Initial:    1 * time.Millisecond,
		Multiplier: 1.1,
		Max:        10 * time.Millisecond,
	}

	folder, err := PollRenameFolderOperation(ctx, poller, cfg)

	assert.NoError(t, err)
	assert.Equal(t, expectedFolder, folder)
	poller.AssertExpectations(t)
}

func TestPollRenameFolderOperation_ImmediateError(t *testing.T) {
	ctx := context.Background()
	poller := new(mockRenameFolderOpPoller)
	expectedErr := errors.New("folder not found")
	poller.On("Poll", ctx).Return(nil, expectedErr).Once()
	cfg := DefaultRenamePollConfig()

	folder, err := PollRenameFolderOperation(ctx, poller, cfg)

	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, folder)
	poller.AssertExpectations(t)
}

func TestPollRenameFolderOperation_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	poller := new(mockRenameFolderOpPoller)
	expectedFolder := &controlpb.Folder{Name: "projects/_/buckets/b/folders/renamed"}
	poller.On("Poll", ctx).Return(nil, nil).Once()
	poller.On("Done").Return(false).Once()
	poller.On("Poll", ctx).Return(nil, nil).Once()
	poller.On("Done").Return(false).Once()
	poller.On("Poll", ctx).Return(expectedFolder, nil).Once()
	poller.On("Done").Return(true).Once()
	cfg := RenamePollConfig{
		Initial:    1 * time.Millisecond,
		Multiplier: 1.1,
		Max:        10 * time.Millisecond,
	}

	folder, err := PollRenameFolderOperation(ctx, poller, cfg)

	assert.NoError(t, err)
	assert.Equal(t, expectedFolder, folder)
	poller.AssertExpectations(t)
}

func TestPollRenameFolderOperation_TransientRPCErrorPropagated(t *testing.T) {
	ctx := context.Background()
	poller := new(mockRenameFolderOpPoller)
	transientErr := errors.New("transient network error")
	poller.On("Poll", ctx).Return(nil, nil).Once()
	poller.On("Done").Return(false).Once()
	poller.On("Poll", ctx).Return(nil, transientErr).Once()
	cfg := RenamePollConfig{
		Initial:    1 * time.Millisecond,
		Multiplier: 1.1,
		Max:        10 * time.Millisecond,
	}

	folder, err := PollRenameFolderOperation(ctx, poller, cfg)

	assert.ErrorIs(t, err, transientErr)
	assert.Nil(t, folder)
	poller.AssertExpectations(t)
}

func TestPollRenameFolderOperation_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	poller := new(mockRenameFolderOpPoller)
	poller.On("Poll", ctx).Return(nil, nil).Once()
	poller.On("Done").Return(false).Once()
	cancel()
	cfg := RenamePollConfig{
		Initial:    50 * time.Millisecond,
		Multiplier: 1.1,
		Max:        100 * time.Millisecond,
	}

	folder, err := PollRenameFolderOperation(ctx, poller, cfg)

	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, folder)
	poller.AssertExpectations(t)
}

func TestPollRenameFolderOperation_FixedFastPhaseWindow(t *testing.T) {
	ctx := context.Background()
	poller := new(mockRenameFolderOpPoller)
	expectedFolder := &controlpb.Folder{Name: "projects/_/buckets/b/folders/renamed"}
	poller.On("Poll", ctx).Return(nil, nil).Once()
	poller.On("Done").Return(false).Once()
	poller.On("Poll", ctx).Return(nil, nil).Once()
	poller.On("Done").Return(false).Once()
	poller.On("Poll", ctx).Return(expectedFolder, nil).Once()
	poller.On("Done").Return(true).Once()
	cfg := RenamePollConfig{
		Initial:         1 * time.Millisecond,
		FastPhaseWindow: 10 * time.Millisecond,
		Multiplier:      2.0,
		Max:             100 * time.Millisecond,
	}

	folder, err := PollRenameFolderOperation(ctx, poller, cfg)

	assert.NoError(t, err)
	assert.Equal(t, expectedFolder, folder)
	poller.AssertExpectations(t)
}
