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

package streamingwrites

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newUnbufferedWriteHandlerForTest() *unbufferedWriteHandlerImpl {
	return newUnbufferedWriteHandler(&CreateWriteHandlerRequest{ObjectName: "foo"})
}

func TestNewUnbufferedWriteHandler(t *testing.T) {
	before := time.Now()

	wh := newUnbufferedWriteHandlerForTest()

	assert.Equal(t, int64(0), wh.totalSize)
	assert.Equal(t, int64(-1), wh.truncatedSize)
	assert.False(t, wh.mtime.Before(before))
	assert.False(t, wh.mtime.After(time.Now()))
}

func TestUnbufferedWriteHandler_WriteFileInfo(t *testing.T) {
	mtime := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	testCases := []struct {
		name              string
		totalSize         int64
		truncatedSize     int64
		expectedTotalSize int64
	}{
		{
			name:              "NoPendingTruncate",
			totalSize:         10,
			truncatedSize:     -1,
			expectedTotalSize: 10,
		},
		{
			name:              "PendingForwardTruncate",
			totalSize:         10,
			truncatedSize:     25,
			expectedTotalSize: 25,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wh := newUnbufferedWriteHandlerForTest()
			wh.totalSize = tc.totalSize
			wh.truncatedSize = tc.truncatedSize
			wh.SetMtime(mtime)

			info := wh.WriteFileInfo()

			assert.Equal(t, tc.expectedTotalSize, info.TotalSize)
			assert.Equal(t, mtime, info.Mtime)
		})
	}
}

func TestUnbufferedWriteHandler_UnimplementedMethodsReturnError(t *testing.T) {
	ctx := context.Background()
	testCases := []struct {
		name string
		call func(wh *unbufferedWriteHandlerImpl) error
	}{
		{
			name: "Write",
			call: func(wh *unbufferedWriteHandlerImpl) error { return wh.Write(ctx, []byte("data"), 0) },
		},
		{
			name: "Sync",
			call: func(wh *unbufferedWriteHandlerImpl) error {
				_, err := wh.Sync(ctx)
				return err
			},
		},
		{
			name: "Flush",
			call: func(wh *unbufferedWriteHandlerImpl) error {
				_, err := wh.Flush(ctx)
				return err
			},
		},
		{
			name: "Truncate",
			call: func(wh *unbufferedWriteHandlerImpl) error { return wh.Truncate(10) },
		},
		{
			name: "Destroy",
			call: func(wh *unbufferedWriteHandlerImpl) error { return wh.Destroy() },
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wh := newUnbufferedWriteHandlerForTest()

			err := tc.call(wh)

			assert.ErrorIs(t, err, errNotImplemented)
		})
	}
}
