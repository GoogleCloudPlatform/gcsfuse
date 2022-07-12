package gcstests

import (
	"fmt"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	"github.com/jacobsa/gcloud/gcs/gcstests"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
	"io/ioutil"
	"math"
	"testing"
	"time"
)

func TestRead(t *testing.T) { RunTests(t) }

type ReadTest struct {
	ctx    context.Context
	bucket gcs.Bucket
	clock  timeutil.Clock
}

var _ SetUpInterface = &ReadTest{}

func init() { RegisterTestSuite(&ReadTest{}) }


////////////////////////////////////////////////////////////////////////
// Initialization
////////////////////////////////////////////////////////////////////////

func (t *ReadTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	clock := &timeutil.SimulatedClock{}
	clock.SetTime(time.Date(2022, 7, 12, 18, 56, 0, 0, time.Local))
	t.clock = clock

	// Set up the fake bucket.
	t.bucket = gcsfake.NewFakeBucket(clock, "some_bucket")

}


////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (t *ReadTest) createObject(name string, contents string) error {
	_, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		name,
		[]byte(contents))

	return err
}

func (t *ReadTest) readObject(objectName string) (contents string, err error) {
	// Open a reader.
	req := &gcs.ReadObjectRequest{
		Name: objectName,
	}

	reader, err := t.bucket.NewReader(t.ctx, req)
	if err != nil {
		return
	}

	defer func() {
		AssertEq(nil, reader.Close())
	}()

	// Read the contents of the object.
	slice, err := ioutil.ReadAll(reader)
	if err != nil {
		return
	}

	// Transform to a string.
	contents = string(slice)

	return
}

// Ensure that the clock will report a different time after returning.
func (t *ReadTest) advanceTime() {
	// For simulated clocks, we can just advance the time.
	if c, ok := t.clock.(*timeutil.SimulatedClock); ok {
		c.AdvanceTime(time.Second)
		return
	}

	// Otherwise, sleep a moment.
	time.Sleep(time.Millisecond)
}

// Return a matcher that matches event times as reported by the bucket
// corresponding to the supplied start time as measured by the test.
func (t *ReadTest) matchesStartTime(start time.Time) Matcher {
	// For simulated clocks we can use exact equality.
	if _, ok := t.clock.(*timeutil.SimulatedClock); ok {
		return timeutil.TimeEq(start)
	}

	// Otherwise, we need to take into account latency between the start of our
	// call and the time the server actually executed the operation.
	const slop = 60 * time.Second
	return timeutil.TimeNear(start, slop)
}




////////////////////////////////////////////////////////////////////////
// Read Tests
////////////////////////////////////////////////////////////////////////

func (t *ReadTest) ObjectNameDoesntExist() {
	req := &gcs.ReadObjectRequest{
		Name: "foobar",
	}

	rc, err := t.bucket.NewReader(t.ctx, req)
	if err == nil {
		defer rc.Close()
		_, err = rc.Read(make([]byte, 1))
	}

	AssertThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
	ExpectThat(err, Error(MatchesRegexp("(?i)not found|404")))
}

func (t *ReadTest) EmptyObject() {
	// Create
	AssertEq(nil, t.createObject("foo", ""))

	// Read
	req := &gcs.ReadObjectRequest{
		Name: "foo",
	}

	r, err := t.bucket.NewReader(t.ctx, req)
	AssertEq(nil, err)

	contents, err := ioutil.ReadAll(r)
	AssertEq(nil, err)
	ExpectEq("", string(contents))

	// Close
	AssertEq(nil, r.Close())
}

func (t *ReadTest) NonEmptyObject() {
	// Create
	AssertEq(nil, t.createObject("foo", "taco"))

	// Read
	req := &gcs.ReadObjectRequest{
		Name: "foo",
	}

	r, err := t.bucket.NewReader(t.ctx, req)
	AssertEq(nil, err)

	contents, err := ioutil.ReadAll(r)
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))

	// Close
	AssertEq(nil, r.Close())
}

func (t *ReadTest) ParticularGeneration_NeverExisted() {
	// Create an object.
	o, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		"foo",
		[]byte{})

	AssertEq(nil, err)
	AssertGt(o.Generation, 0)

	// Attempt to read a different generation.
	req := &gcs.ReadObjectRequest{
		Name:       "foo",
		Generation: o.Generation + 1,
	}

	rc, err := t.bucket.NewReader(t.ctx, req)
	if err == nil {
		defer rc.Close()
		_, err = rc.Read(make([]byte, 1))
	}

	AssertThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
	ExpectThat(err, Error(MatchesRegexp("(?i)not found|404")))
}

func (t *ReadTest) ParticularGeneration_HasBeenDeleted() {
	// Create an object.
	o, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		"foo",
		[]byte{})

	AssertEq(nil, err)
	AssertGt(o.Generation, 0)

	// Delete it.
	err = t.bucket.DeleteObject(
		t.ctx,
		&gcs.DeleteObjectRequest{
			Name: "foo",
		})

	AssertEq(nil, err)

	// Attempt to read by that generation.
	req := &gcs.ReadObjectRequest{
		Name:       "foo",
		Generation: o.Generation,
	}

	rc, err := t.bucket.NewReader(t.ctx, req)
	if err == nil {
		defer rc.Close()
		_, err = rc.Read(make([]byte, 1))
	}

	AssertThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
	ExpectThat(err, Error(MatchesRegexp("(?i)not found|404")))
}

func (t *ReadTest) ParticularGeneration_Exists() {
	// Create an object.
	o, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		"foo",
		[]byte("taco"))

	AssertEq(nil, err)
	AssertGt(o.Generation, 0)

	// Attempt to read the correct generation.
	req := &gcs.ReadObjectRequest{
		Name:       "foo",
		Generation: o.Generation,
	}

	r, err := t.bucket.NewReader(t.ctx, req)
	AssertEq(nil, err)

	contents, err := ioutil.ReadAll(r)
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))

	// Close
	AssertEq(nil, r.Close())
}

func (t *ReadTest) ParticularGeneration_ObjectHasBeenOverwritten() {
	// Create an object.
	o, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		"foo",
		[]byte("taco"))

	AssertEq(nil, err)
	AssertGt(o.Generation, 0)

	// Overwrite with a new generation.
	o2, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		"foo",
		[]byte("burrito"))

	AssertEq(nil, err)
	AssertGt(o2.Generation, 0)
	AssertNe(o.Generation, o2.Generation)

	// Reading by the old generation should fail.
	req := &gcs.ReadObjectRequest{
		Name:       "foo",
		Generation: o.Generation,
	}

	rc, err := t.bucket.NewReader(t.ctx, req)
	if err == nil {
		defer rc.Close()
		_, err = rc.Read(make([]byte, 1))
	}

	AssertThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
	ExpectThat(err, Error(MatchesRegexp("(?i)not found|404")))

	// Reading by the new generation should work.
	req.Generation = o2.Generation

	rc, err = t.bucket.NewReader(t.ctx, req)
	AssertEq(nil, err)

	contents, err := ioutil.ReadAll(rc)
	AssertEq(nil, err)
	ExpectEq("burrito", string(contents))

	// Close
	AssertEq(nil, rc.Close())
}

func (t *ReadTest) Ranges_EmptyObject() {
	// Create an empty object.
	AssertEq(nil, t.createObject("foo", ""))

	// Test cases.
	testCases := []struct {
		br gcs.ByteRange
	}{
		// Empty without knowing object length
		{gcs.ByteRange{0, 0}},

		{gcs.ByteRange{1, 1}},
		{gcs.ByteRange{1, 0}},

		{gcs.ByteRange{math.MaxInt64, math.MaxInt64}},
		{gcs.ByteRange{math.MaxInt64, 17}},
		{gcs.ByteRange{math.MaxInt64, 0}},

		{gcs.ByteRange{math.MaxUint64, math.MaxUint64}},
		{gcs.ByteRange{math.MaxUint64, 17}},
		{gcs.ByteRange{math.MaxUint64, 0}},

		// Not empty without knowing object length
		{gcs.ByteRange{0, 1}},
		{gcs.ByteRange{0, 17}},
		{gcs.ByteRange{0, math.MaxInt64}},
		{gcs.ByteRange{0, math.MaxUint64}},

		{gcs.ByteRange{1, 2}},
		{gcs.ByteRange{1, 17}},
		{gcs.ByteRange{1, math.MaxInt64}},
		{gcs.ByteRange{1, math.MaxUint64}},

		{gcs.ByteRange{math.MaxInt64, math.MaxInt64 + 1}},
		{gcs.ByteRange{math.MaxInt64, math.MaxUint64}},
	}

	// Turn test cases into read requests.
	var requests []*gcs.ReadObjectRequest
	for _, tc := range testCases {
		br := tc.br
		req := &gcs.ReadObjectRequest{
			Name:  "foo",
			Range: &br,
		}

		requests = append(requests, req)
	}

	// Make each request.
	contents, errs := gcstests.ReadMultiple(
		t.ctx,
		t.bucket,
		requests)

	AssertEq(len(testCases), len(contents))
	AssertEq(len(testCases), len(errs))
	for i, tc := range testCases {
		desc := fmt.Sprintf("Test case %d, range %v", i, tc.br)
		ExpectEq(nil, errs[i], "%s", desc)
		ExpectEq("", string(contents[i]), "%s", desc)
	}
}

func (t *ReadTest) Ranges_NonEmptyObject() {
	// Create an object of length four.
	AssertEq(nil, t.createObject("foo", "taco"))

	// Test cases.
	testCases := []struct {
		br               gcs.ByteRange
		expectedContents string
	}{
		// Left anchored
		{gcs.ByteRange{0, math.MaxUint64}, "taco"},
		{gcs.ByteRange{0, 5}, "taco"},
		{gcs.ByteRange{0, 4}, "taco"},
		{gcs.ByteRange{0, 3}, "tac"},
		{gcs.ByteRange{0, 1}, "t"},
		{gcs.ByteRange{0, 0}, ""},

		// Floating left edge
		{gcs.ByteRange{1, math.MaxUint64}, "aco"},
		{gcs.ByteRange{1, 5}, "aco"},
		{gcs.ByteRange{1, 4}, "aco"},
		{gcs.ByteRange{1, 2}, "a"},
		{gcs.ByteRange{1, 1}, ""},
		{gcs.ByteRange{1, 0}, ""},

		// Left edge at right edge of object
		{gcs.ByteRange{4, math.MaxUint64}, ""},
		{gcs.ByteRange{4, math.MaxInt64 + 1}, ""},
		{gcs.ByteRange{4, math.MaxInt64 + 0}, ""},
		{gcs.ByteRange{4, math.MaxInt64 - 1}, ""},
		{gcs.ByteRange{4, 17}, ""},
		{gcs.ByteRange{4, 5}, ""},
		{gcs.ByteRange{4, 4}, ""},
		{gcs.ByteRange{4, 1}, ""},
		{gcs.ByteRange{4, 0}, ""},

		// Left edge past right edge of object
		{gcs.ByteRange{5, math.MaxUint64}, ""},
		{gcs.ByteRange{5, 17}, ""},
		{gcs.ByteRange{5, 5}, ""},
		{gcs.ByteRange{5, 4}, ""},
		{gcs.ByteRange{5, 1}, ""},
		{gcs.ByteRange{5, 0}, ""},

		// Left edge is 2^63 - 1
		{gcs.ByteRange{math.MaxInt64, math.MaxUint64}, ""},
		{gcs.ByteRange{math.MaxInt64, math.MaxInt64 + 1}, ""},
		{gcs.ByteRange{math.MaxInt64, math.MaxInt64 + 0}, ""},
		{gcs.ByteRange{math.MaxInt64, math.MaxInt64 - 1}, ""},
		{gcs.ByteRange{math.MaxInt64, 5}, ""},
		{gcs.ByteRange{math.MaxInt64, 4}, ""},
		{gcs.ByteRange{math.MaxInt64, 1}, ""},
		{gcs.ByteRange{math.MaxInt64, 0}, ""},

		// Left edge is 2^64 - 1
		{gcs.ByteRange{math.MaxUint64, math.MaxUint64}, ""},
		{gcs.ByteRange{math.MaxUint64, math.MaxInt64 + 1}, ""},
		{gcs.ByteRange{math.MaxUint64, math.MaxInt64}, ""},
		{gcs.ByteRange{math.MaxUint64, math.MaxInt64 - 1}, ""},
		{gcs.ByteRange{math.MaxUint64, 5}, ""},
		{gcs.ByteRange{math.MaxInt64, 4}, ""},
		{gcs.ByteRange{math.MaxInt64, 1}, ""},
		{gcs.ByteRange{math.MaxInt64, 0}, ""},
	}

	// Turn test cases into read requests.
	var requests []*gcs.ReadObjectRequest
	for _, tc := range testCases {
		br := tc.br
		req := &gcs.ReadObjectRequest{
			Name:  "foo",
			Range: &br,
		}

		requests = append(requests, req)
	}

	// Make each request.
	contents, errs := gcstests.ReadMultiple(
		t.ctx,
		t.bucket,
		requests)

	AssertEq(len(testCases), len(contents))
	AssertEq(len(testCases), len(errs))
	for i, tc := range testCases {
		desc := fmt.Sprintf("Test case %d, range %v", i, tc.br)
		ExpectEq(nil, errs[i], "%s", desc)
		ExpectEq(tc.expectedContents, string(contents[i]), "%s", desc)
	}
}
