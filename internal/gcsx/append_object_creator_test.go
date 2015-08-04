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

package gcsx

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/mock_gcs"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
)

func TestAppendObjectCreator(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func deleteReqName(expected string) (m Matcher) {
	m = NewMatcher(
		func(c interface{}) (err error) {
			req, ok := c.(*gcs.DeleteObjectRequest)
			if !ok {
				err = fmt.Errorf("which has type %T", c)
				return
			}

			if req.Name != expected {
				err = fmt.Errorf("which is for name %q", req.Name)
				return
			}

			return
		},
		fmt.Sprintf("Delete request for name %q", expected))

	return
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const prefix = ".gcsfuse_tmp/"

type AppendObjectCreatorTest struct {
	ctx     context.Context
	bucket  mock_gcs.MockBucket
	creator objectCreator

	srcObject   gcs.Object
	srcContents string
}

var _ SetUpInterface = &AppendObjectCreatorTest{}

func init() { RegisterTestSuite(&AppendObjectCreatorTest{}) }

func (t *AppendObjectCreatorTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx

	// Create the bucket.
	t.bucket = mock_gcs.NewMockBucket(ti.MockController, "bucket")

	// Create the creator.
	t.creator = newAppendObjectCreator(prefix, t.bucket)
}

func (t *AppendObjectCreatorTest) call() (o *gcs.Object, err error) {
	o, err = t.creator.Create(
		t.ctx,
		&t.srcObject,
		strings.NewReader(t.srcContents))

	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *AppendObjectCreatorTest) CallsCreateObject() {
	t.srcContents = "taco"

	// CreateObject
	var req *gcs.CreateObjectRequest
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(DoAll(SaveArg(1, &req), Return(nil, errors.New(""))))

	// Call
	t.call()

	AssertNe(nil, req)
	ExpectTrue(strings.HasPrefix(req.Name, prefix), "Name: %s", req.Name)
	ExpectThat(req.GenerationPrecondition, Pointee(Equals(0)))

	b, err := ioutil.ReadAll(req.Contents)
	AssertEq(nil, err)
	ExpectEq(t.srcContents, string(b))
}

func (t *AppendObjectCreatorTest) CreateObjectFails() {
	var err error

	// CreateObject
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(Return(nil, errors.New("taco")))

	// Call
	_, err = t.call()

	ExpectThat(err, Error(HasSubstr("CreateObject")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *AppendObjectCreatorTest) CreateObjectReturnsPreconditionError() {
	var err error

	// CreateObject
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(Return(nil, &gcs.PreconditionError{Err: errors.New("taco")}))

	// Call
	_, err = t.call()

	ExpectThat(err, HasSameTypeAs(&gcs.PreconditionError{}))
	ExpectThat(err, Error(HasSubstr("CreateObject")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *AppendObjectCreatorTest) CallsComposeObjects() {
	t.srcObject.Name = "foo"
	t.srcObject.Generation = 17

	// CreateObject
	tmpObject := &gcs.Object{
		Name:       "bar",
		Generation: 19,
	}

	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(Return(tmpObject, nil))

	// ComposeObjects
	var req *gcs.ComposeObjectsRequest
	ExpectCall(t.bucket, "ComposeObjects")(Any(), Any()).
		WillOnce(DoAll(SaveArg(1, &req), Return(nil, errors.New(""))))

	// DeleteObject
	ExpectCall(t.bucket, "DeleteObject")(Any(), deleteReqName(tmpObject.Name)).
		WillOnce(Return(nil))

	// Call
	t.call()

	AssertNe(nil, req)
	ExpectEq(t.srcObject.Name, req.DstName)
	ExpectThat(
		req.DstGenerationPrecondition,
		Pointee(Equals(t.srcObject.Generation)))

	AssertEq(2, len(req.Sources))
	var src gcs.ComposeSource

	src = req.Sources[0]
	ExpectEq(t.srcObject.Name, src.Name)
	ExpectEq(t.srcObject.Generation, src.Generation)

	src = req.Sources[1]
	ExpectEq(tmpObject.Name, src.Name)
	ExpectEq(tmpObject.Generation, src.Generation)
}

func (t *AppendObjectCreatorTest) ComposeObjectsFails() {
	// CreateObject
	tmpObject := &gcs.Object{
		Name: "bar",
	}

	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(Return(tmpObject, nil))

	// ComposeObjects
	ExpectCall(t.bucket, "ComposeObjects")(Any(), Any()).
		WillOnce(Return(nil, errors.New("taco")))

	// DeleteObject
	ExpectCall(t.bucket, "DeleteObject")(Any(), deleteReqName(tmpObject.Name)).
		WillOnce(Return(errors.New("")))

	// Call
	_, err := t.call()

	ExpectThat(err, Error(HasSubstr("ComposeObjects")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *AppendObjectCreatorTest) ComposeObjectsReturnsPreconditionError() {
	// CreateObject
	tmpObject := &gcs.Object{
		Name: "bar",
	}

	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(Return(tmpObject, nil))

	// ComposeObjects
	ExpectCall(t.bucket, "ComposeObjects")(Any(), Any()).
		WillOnce(Return(nil, &gcs.PreconditionError{Err: errors.New("taco")}))

	// DeleteObject
	ExpectCall(t.bucket, "DeleteObject")(Any(), deleteReqName(tmpObject.Name)).
		WillOnce(Return(errors.New("")))

	// Call
	_, err := t.call()

	ExpectThat(err, HasSameTypeAs(&gcs.PreconditionError{}))
	ExpectThat(err, Error(HasSubstr("ComposeObjects")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *AppendObjectCreatorTest) ComposeObjectsReturnsNotFoundError() {
	// CreateObject
	tmpObject := &gcs.Object{
		Name: "bar",
	}

	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(Return(tmpObject, nil))

	// ComposeObjects
	ExpectCall(t.bucket, "ComposeObjects")(Any(), Any()).
		WillOnce(Return(nil, &gcs.NotFoundError{Err: errors.New("taco")}))

	// DeleteObject
	ExpectCall(t.bucket, "DeleteObject")(Any(), deleteReqName(tmpObject.Name)).
		WillOnce(Return(errors.New("")))

	// Call
	_, err := t.call()

	ExpectThat(err, HasSameTypeAs(&gcs.PreconditionError{}))
	ExpectThat(err, Error(HasSubstr("Synthesized")))
	ExpectThat(err, Error(HasSubstr("ComposeObjects")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *AppendObjectCreatorTest) CallsDeleteObject() {
	// CreateObject
	tmpObject := &gcs.Object{
		Name: "bar",
	}

	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(Return(tmpObject, nil))

	// ComposeObjects
	composed := &gcs.Object{}
	ExpectCall(t.bucket, "ComposeObjects")(Any(), Any()).
		WillOnce(Return(composed, nil))

	// DeleteObject
	ExpectCall(t.bucket, "DeleteObject")(Any(), deleteReqName(tmpObject.Name)).
		WillOnce(Return(errors.New("")))

	// Call
	t.call()
}

func (t *AppendObjectCreatorTest) DeleteObjectFails() {
	// CreateObject
	tmpObject := &gcs.Object{
		Name: "bar",
	}

	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(Return(tmpObject, nil))

	// ComposeObjects
	composed := &gcs.Object{}
	ExpectCall(t.bucket, "ComposeObjects")(Any(), Any()).
		WillOnce(Return(composed, nil))

	// DeleteObject
	ExpectCall(t.bucket, "DeleteObject")(Any(), Any()).
		WillOnce(Return(errors.New("taco")))

	// Call
	_, err := t.call()

	ExpectThat(err, Error(HasSubstr("DeleteObject")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *AppendObjectCreatorTest) DeleteObjectSucceeds() {
	// CreateObject
	tmpObject := &gcs.Object{
		Name: "bar",
	}

	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(Return(tmpObject, nil))

	// ComposeObjects
	composed := &gcs.Object{}
	ExpectCall(t.bucket, "ComposeObjects")(Any(), Any()).
		WillOnce(Return(composed, nil))

	// DeleteObject
	ExpectCall(t.bucket, "DeleteObject")(Any(), Any()).
		WillOnce(Return(nil))

	// Call
	o, err := t.call()

	AssertEq(nil, err)
	ExpectEq(composed, o)
}
