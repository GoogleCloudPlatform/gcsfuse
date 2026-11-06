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

package wrappers

import (
	"context"
	"fmt"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFsErrStrAndCategory(t *testing.T) {
	t.Parallel()
	tests := []struct {
		fsErr            error
		expectedCategory metrics.FsErrorCategory
	}{
		{
			fsErr:            fmt.Errorf("some random error"),
			expectedCategory: errIO,
		},
		{
			fsErr:            syscall.ENOTEMPTY,
			expectedCategory: errDirNotEmpty,
		},
		{
			fsErr:            syscall.EEXIST,
			expectedCategory: errFileExists,
		},
		{
			fsErr:            syscall.EINVAL,
			expectedCategory: errInvalidArg,
		},
		{
			fsErr:            syscall.EINTR,
			expectedCategory: errInterrupt,
		},
		{
			fsErr:            syscall.ENOSYS,
			expectedCategory: errNotImplemented,
		},
		{
			fsErr:            syscall.ENOSPC,
			expectedCategory: errProcessMgmt,
		},
		{
			fsErr:            syscall.E2BIG,
			expectedCategory: errInvalidOp,
		},
		{
			fsErr:            syscall.EHOSTDOWN,
			expectedCategory: errNetwork,
		},
		{
			fsErr:            syscall.ENODATA,
			expectedCategory: errMisc,
		},
		{
			fsErr:            syscall.ENODEV,
			expectedCategory: errDevice,
		},
		{
			fsErr:            syscall.EISDIR,
			expectedCategory: errFileDir,
		},
		{
			fsErr:            syscall.ENOSYS,
			expectedCategory: errNotImplemented,
		},
		{
			fsErr:            syscall.ENFILE,
			expectedCategory: errTooManyFiles,
		},
		{
			fsErr:            syscall.EPERM,
			expectedCategory: errPerm,
		},
	}

	for idx, tc := range tests {
		t.Run(fmt.Sprintf("fsErrStrAndCategor_case_%d", idx), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expectedCategory, categorize(tc.fsErr))
		})
	}
}

// readCountEvent is one recorded metadata_cache/read_count event.
type readCountEvent struct {
	hit    bool
	status metrics.EntryStatus
	detail metrics.LookupDetail
}

// readCountRecorder records the metadata_cache/read_count events it receives
// and ignores every other metric.
type readCountRecorder struct {
	metrics.MetricHandle
	events []readCountEvent
}

func (r *readCountRecorder) MetadataCacheReadCount(inc int64, cacheHit bool, entryStatus metrics.EntryStatus, lookupDetail metrics.LookupDetail) {
	r.events = append(r.events, readCountEvent{cacheHit, entryStatus, lookupDetail})
}

// cacheReadingFS is a file system whose lookups and attribute reads read the
// metadata cache, and whose other ops read nothing.
type cacheReadingFS struct {
	fuseutil.NotImplementedFileSystem
}

func (*cacheReadingFS) LookUpInode(ctx context.Context, _ *fuseops.LookUpInodeOp) error {
	// A lookup probing two keys, a negative hit and a miss, then fetching from
	// GCS.
	metadata.RecordCacheRead(ctx, true, metrics.EntryStatusNegativeAttr, metrics.LookupDetailFoundAttr)
	metadata.RecordCacheRead(ctx, false, metrics.EntryStatusAttr, metrics.LookupDetailNotFoundAttr)
	metadata.RecordGCSFetch(ctx)
	return nil
}

func (*cacheReadingFS) GetInodeAttributes(ctx context.Context, _ *fuseops.GetInodeAttributesOp) error {
	metadata.RecordCacheRead(ctx, true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr)
	return nil
}

func TestMonitoring_RecordsOneMetadataCacheReadCountPerOp(t *testing.T) {
	ctx := context.Background()
	recorder := &readCountRecorder{MetricHandle: metrics.NewNoopMetrics()}
	fs := WithMonitoring(&cacheReadingFS{}, recorder)

	require.NoError(t, fs.LookUpInode(ctx, &fuseops.LookUpInodeOp{}))
	require.NoError(t, fs.GetInodeAttributes(ctx, &fuseops.GetInodeAttributesOp{}))
	// Reads nothing, so records nothing.
	_ = fs.OpenDir(ctx, &fuseops.OpenDirOp{})

	assert.Equal(t, []readCountEvent{
		{false, metrics.EntryStatusAttr, metrics.LookupDetailNotFoundAttr},
		{true, metrics.EntryStatusPositiveAttr, metrics.LookupDetailFoundAttr},
	}, recorder.events)
}

// readBytesRecorder sums the fs/read_bytes_count increments it receives and
// ignores every other metric.
type readBytesRecorder struct {
	metrics.MetricHandle
	bytes int64
}

func (r *readBytesRecorder) FsReadBytesCount(inc int64) {
	r.bytes += inc
}

// readFileFS is a file system whose reads return bytesRead bytes and err.
type readFileFS struct {
	fuseutil.NotImplementedFileSystem
	bytesRead int
	err       error
}

func (f *readFileFS) ReadFile(_ context.Context, op *fuseops.ReadFileOp) error {
	op.BytesRead = f.bytesRead
	return f.err
}

func TestMonitoring_ReadFileRecordsBytesReturnedToKernel(t *testing.T) {
	tests := []struct {
		name      string
		bytesRead int
		err       error
		wantBytes int64
	}{
		{
			// The kernel asks for 10 bytes, but only 4 remain before EOF.
			name:      "short read at EOF counts bytes returned, not requested",
			bytesRead: 4,
			wantBytes: 4,
		},
		{
			// FUSE replies to a failed read with just the error, so no data reaches
			// the kernel even though the file system set BytesRead.
			name:      "failed read is not counted",
			bytesRead: 10,
			err:       syscall.EIO,
			wantBytes: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &readBytesRecorder{MetricHandle: metrics.NewNoopMetrics()}
			fs := WithMonitoring(&readFileFS{bytesRead: tc.bytesRead, err: tc.err}, recorder)

			err := fs.ReadFile(context.Background(), &fuseops.ReadFileOp{Size: 10, Dst: make([]byte, 10)})

			assert.ErrorIs(t, err, tc.err)
			assert.Equal(t, tc.wantBytes, recorder.bytes)
		})
	}
}

// writeBytesRecorder sums the fs/write_bytes_count increments it receives and
// ignores every other metric.
type writeBytesRecorder struct {
	metrics.MetricHandle
	bytes int64
	calls int
}

func (r *writeBytesRecorder) FsWriteBytesCount(inc int64) {
	r.bytes += inc
	r.calls++
}

// writeFileFS is a file system whose writes return err.
type writeFileFS struct {
	fuseutil.NotImplementedFileSystem
	err error
}

func (f *writeFileFS) WriteFile(_ context.Context, _ *fuseops.WriteFileOp) error {
	return f.err
}

func TestMonitoring_WriteFileRecordsBytesAccepted(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		err       error
		wantBytes int64
		wantCalls int
	}{
		{
			// FUSE requires exactly len(Data) bytes to be written on success, so the
			// whole buffer is accounted for.
			name:      "successful write counts the whole buffer",
			data:      []byte("hello world"),
			wantBytes: 11,
			wantCalls: 1,
		},
		{
			// FUSE replies to a failed write with just the error; none of the data
			// is considered written.
			name:      "failed write is not counted",
			data:      []byte("hello world"),
			err:       syscall.EIO,
			wantBytes: 0,
			wantCalls: 0,
		},
		{
			name:      "empty write records nothing",
			data:      nil,
			wantBytes: 0,
			wantCalls: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &writeBytesRecorder{MetricHandle: metrics.NewNoopMetrics()}
			fs := WithMonitoring(&writeFileFS{err: tc.err}, recorder)

			err := fs.WriteFile(context.Background(), &fuseops.WriteFileOp{Data: tc.data})

			assert.ErrorIs(t, err, tc.err)
			assert.Equal(t, tc.wantBytes, recorder.bytes)
			assert.Equal(t, tc.wantCalls, recorder.calls)
		})
	}
}
