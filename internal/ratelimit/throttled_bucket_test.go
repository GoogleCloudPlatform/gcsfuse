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

package ratelimit

import (
	"io"
	"strings"
	"testing"

	"context"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// A throttle that records the tokens requested from it and never blocks.
type recordingThrottle struct {
	capacity uint64
	tokens   uint64
	waits    int
}

func (rt *recordingThrottle) Capacity() uint64 {
	return rt.capacity
}

func (rt *recordingThrottle) Wait(ctx context.Context, tokens uint64) error {
	rt.waits++
	rt.tokens += tokens
	return nil
}

// A gcs.Bucket that records the CreateObject request it receives. Only
// CreateObject may be called.
type capturingBucket struct {
	gcs.Bucket
	createReq *gcs.CreateObjectRequest
}

func (cb *capturingBucket) CreateObject(ctx context.Context, req *gcs.CreateObjectRequest) (*gcs.Object, error) {
	cb.createReq = req
	return &gcs.Object{Name: req.Name}, nil
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ThrottledBucketTest struct {
	suite.Suite
	ctx context.Context

	opThrottle      recordingThrottle
	egressThrottle  recordingThrottle
	ingressThrottle recordingThrottle

	bucket gcs.Bucket
}

func TestThrottledBucketSuite(t *testing.T) {
	suite.Run(t, new(ThrottledBucketTest))
}

func (t *ThrottledBucketTest) SetupTest() {
	t.ctx = context.Background()
	t.opThrottle = recordingThrottle{capacity: 1}
	t.egressThrottle = recordingThrottle{capacity: 1 << 20}
	t.ingressThrottle = recordingThrottle{capacity: 1 << 20}
	t.bucket = NewThrottledBucket(
		&t.opThrottle,
		&t.egressThrottle,
		&t.ingressThrottle,
		fake.NewFakeBucket(timeutil.RealClock(), "some_bucket", gcs.BucketType{}))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ThrottledBucketTest) TestCreateObjectThrottlesContents() {
	const contents = "taco burrito enchilada"
	req := &gcs.CreateObjectRequest{Name: "foo", Contents: strings.NewReader(contents)}

	o, err := t.bucket.CreateObject(t.ctx, req)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), uint64(len(contents)), o.Size)
	assert.Equal(t.T(), uint64(1), t.opThrottle.tokens)
	assert.Equal(t.T(), uint64(len(contents)), t.ingressThrottle.tokens)
	assert.Equal(t.T(), uint64(0), t.egressThrottle.tokens)
	// The caller's request must not be mutated.
	assert.IsType(t.T(), &strings.Reader{}, req.Contents)
}

func (t *ThrottledBucketTest) TestCreateObjectEmptyContentsChargesNothing() {
	o, err := t.bucket.CreateObject(t.ctx, &gcs.CreateObjectRequest{Name: "empty", Contents: strings.NewReader("")})

	require.NoError(t.T(), err)
	assert.Equal(t.T(), uint64(0), o.Size)
	assert.Equal(t.T(), uint64(1), t.opThrottle.tokens)
	assert.Equal(t.T(), uint64(0), t.ingressThrottle.tokens)
	assert.Equal(t.T(), 0, t.ingressThrottle.waits)
}

func (t *ThrottledBucketTest) TestCreateObjectNilContents() {
	var stub capturingBucket
	bucket := NewThrottledBucket(&t.opThrottle, &t.egressThrottle, &t.ingressThrottle, &stub)
	req := &gcs.CreateObjectRequest{Name: "foo"}

	_, err := bucket.CreateObject(t.ctx, req)

	require.NoError(t.T(), err)
	assert.Same(t.T(), req, stub.createReq)
	assert.Equal(t.T(), uint64(1), t.opThrottle.tokens)
	assert.Equal(t.T(), uint64(0), t.ingressThrottle.tokens)
}

func (t *ThrottledBucketTest) TestChunkWriterThrottlesWritesAndFinalizes() {
	const contents = "quesadilla"
	w, err := t.bucket.CreateObjectChunkWriter(t.ctx, &gcs.CreateObjectRequest{Name: "foo"}, 1024, nil)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), uint64(1), t.opThrottle.tokens)

	n, err := w.Write([]byte(contents))

	require.NoError(t.T(), err)
	assert.Equal(t.T(), len(contents), n)
	assert.Equal(t.T(), uint64(len(contents)), t.ingressThrottle.tokens)
	assert.Equal(t.T(), 1, t.ingressThrottle.waits)
	assert.Equal(t.T(), "foo", w.ObjectName())

	// Regression: the fake bucket type-asserts the writer it returned.
	minObj, err := t.bucket.FinalizeUpload(t.ctx, w)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), minObj)
	assert.Equal(t.T(), "foo", minObj.Name)
	assert.Equal(t.T(), uint64(len(contents)), minObj.Size)
}

func (t *ThrottledBucketTest) TestAppendableWriterThrottlesWritesAndFlushes() {
	const contents = "churro"
	req := &gcs.CreateObjectChunkWriterRequest{
		CreateObjectRequest: gcs.CreateObjectRequest{Name: "bar"},
	}
	w, err := t.bucket.CreateAppendableObjectWriter(t.ctx, req)
	require.NoError(t.T(), err)

	n, err := w.Write([]byte(contents))

	require.NoError(t.T(), err)
	assert.Equal(t.T(), len(contents), n)
	assert.Equal(t.T(), uint64(len(contents)), t.ingressThrottle.tokens)

	minObj, err := t.bucket.FlushPendingWrites(t.ctx, w)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), minObj)
	assert.Equal(t.T(), "bar", minObj.Name)
}

func (t *ThrottledBucketTest) TestReadConsumesEgressNotIngress() {
	const contents = "tamale"
	_, err := t.bucket.CreateObject(t.ctx, &gcs.CreateObjectRequest{Name: "foo", Contents: strings.NewReader(contents)})
	require.NoError(t.T(), err)
	ingressBefore := t.ingressThrottle.tokens

	rd, err := t.bucket.NewReaderWithReadHandle(t.ctx, &gcs.ReadObjectRequest{Name: "foo"})
	require.NoError(t.T(), err)
	got, err := io.ReadAll(rd)
	require.NoError(t.T(), err)
	require.NoError(t.T(), rd.Close())

	assert.Equal(t.T(), contents, string(got))
	assert.GreaterOrEqual(t.T(), t.egressThrottle.tokens, uint64(len(contents)))
	assert.Equal(t.T(), ingressBefore, t.ingressThrottle.tokens)
}

func (t *ThrottledBucketTest) TestCopyAndComposeConsumeNoIngress() {
	_, err := t.bucket.CreateObject(t.ctx, &gcs.CreateObjectRequest{Name: "src", Contents: strings.NewReader("flan")})
	require.NoError(t.T(), err)
	ingressBefore := t.ingressThrottle.tokens
	opsBefore := t.opThrottle.tokens

	_, err = t.bucket.CopyObject(t.ctx, &gcs.CopyObjectRequest{SrcName: "src", DstName: "copy"})
	require.NoError(t.T(), err)
	_, err = t.bucket.ComposeObjects(t.ctx, &gcs.ComposeObjectsRequest{
		DstName: "composed",
		Sources: []gcs.ComposeSource{{Name: "src"}, {Name: "copy"}},
	})
	require.NoError(t.T(), err)

	assert.Equal(t.T(), ingressBefore, t.ingressThrottle.tokens)
	assert.Equal(t.T(), opsBefore+2, t.opThrottle.tokens)
}
