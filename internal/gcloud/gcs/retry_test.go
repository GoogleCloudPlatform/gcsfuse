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

package gcs

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"golang.org/x/net/context"

	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
)

func TestRetry(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func contentsAre(s string) Matcher {
	pred := func(c interface{}) (err error) {
		// Convert.
		req, ok := c.(*CreateObjectRequest)
		if !ok {
			err = fmt.Errorf("which has type %T", c)
			return
		}

		// Read.
		contents, err := ioutil.ReadAll(req.Contents)
		if err != nil {
			err = fmt.Errorf("whose contents cannot be read: %v", err)
			return
		}

		// Compare.
		if string(contents) != s {
			err = errors.New("whose contents don't match")
			return
		}

		return
	}

	return NewMatcher(pred, "has the specified contents")
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type retryBucketTest struct {
	ctx     context.Context
	wrapped MockBucket
	bucket  Bucket
}

func (t *retryBucketTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.wrapped = NewMockBucket(ti.MockController, "wrapped")
	t.bucket = newRetryBucket(time.Second, t.wrapped)
}

////////////////////////////////////////////////////////////////////////
// CreateObject
////////////////////////////////////////////////////////////////////////

type RetryBucket_CreateObjectTest struct {
	retryBucketTest

	req CreateObjectRequest
	obj *Object
}

func init() { RegisterTestSuite(&RetryBucket_CreateObjectTest{}) }

func (t *RetryBucket_CreateObjectTest) call() (err error) {
	t.obj, err = t.bucket.CreateObject(t.ctx, &t.req)
	return
}

func (t *RetryBucket_CreateObjectTest) ErrorReading() {
	var err error

	// Pass in a reader that will return an error.
	t.req.Contents = ioutil.NopCloser(
		iotest.OneByteReader(
			iotest.TimeoutReader(
				strings.NewReader("foobar"))))

	// Call
	err = t.call()

	ExpectThat(err, Error(HasSubstr("ReadAll")))
	ExpectThat(err, Error(HasSubstr("timeout")))
}

func (t *RetryBucket_CreateObjectTest) CallsWrapped() {
	const expected = "taco"

	// Request
	t.req.Contents = ioutil.NopCloser(strings.NewReader(expected))

	// Wrapped
	ExpectCall(t.wrapped, "CreateObject")(Any(), contentsAre(expected)).
		WillOnce(Return(nil, errors.New("")))

	// Call
	t.call()
}

func (t *RetryBucket_CreateObjectTest) Successful() {
	var err error

	// Request
	t.req.Contents = ioutil.NopCloser(strings.NewReader(""))

	// Wrapped
	expected := &Object{}
	ExpectCall(t.wrapped, "CreateObject")(Any(), Any()).
		WillOnce(Return(expected, nil))

	// Call
	err = t.call()

	AssertEq(nil, err)
	ExpectEq(expected, t.obj)
}

func (t *RetryBucket_CreateObjectTest) ShouldNotRetry() {
	var err error

	// Request
	t.req.Contents = ioutil.NopCloser(strings.NewReader(""))

	// Wrapped
	expected := errors.New("taco")
	ExpectCall(t.wrapped, "CreateObject")(Any(), Any()).
		WillOnce(Return(nil, expected))

	// Call
	err = t.call()

	ExpectTrue(errors.Is(err, expected))
}

func (t *RetryBucket_CreateObjectTest) CallsWrappedForRetry() {
	const expected = "taco"

	// Request
	t.req.Contents = ioutil.NopCloser(strings.NewReader(expected))

	// Wrapped
	retryable := io.ErrUnexpectedEOF

	ExpectCall(t.wrapped, "CreateObject")(Any(), contentsAre(expected)).
		WillOnce(Return(nil, retryable)).
		WillOnce(Return(nil, errors.New("")))

	// Call
	t.call()
}

func (t *RetryBucket_CreateObjectTest) RetrySuccessful() {
	var err error

	// Request
	t.req.Contents = ioutil.NopCloser(strings.NewReader(""))

	// Wrapped
	retryable := io.ErrUnexpectedEOF
	expected := &Object{}

	ExpectCall(t.wrapped, "CreateObject")(Any(), Any()).
		WillOnce(Return(nil, retryable)).
		WillOnce(Return(expected, nil))

	// Call
	err = t.call()

	AssertEq(nil, err)
	ExpectEq(expected, t.obj)
}
