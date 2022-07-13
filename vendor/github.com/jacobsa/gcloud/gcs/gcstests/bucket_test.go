package gcstests

import (
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	"github.com/jacobsa/gcloud/gcs/gcstests"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
	"io/ioutil"
	"testing"
	"time"
)

func TestBucket(t *testing.T) { RunTests(t) }

type BucketTest struct {
	ctx    context.Context
	bucket gcs.Bucket
	clock  timeutil.Clock
}

var _ SetUpInterface = &BucketTest{}

func init() { RegisterTestSuite(&BucketTest{}) }

////////////////////////////////////////////////////////////////////////
// Initialization
////////////////////////////////////////////////////////////////////////

func (t *BucketTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	clock := &timeutil.SimulatedClock{}
	clock.SetTime(time.Date(2022, 7, 12, 18, 56, 0, 0, time.Local))
	t.clock = clock

	// Set up the fake bucket.
	t.bucket = gcsfake.NewFakeBucket(t.ctx, clock, "some_bucket")

}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (t *BucketTest) createObject(name string, contents string) error {
	_, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		name,
		[]byte(contents))

	return err
}

func (t *BucketTest) readObject(objectName string) (contents string, err error) {
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
func (t *BucketTest) advanceTime() {
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
func (t *BucketTest) matchesStartTime(start time.Time) Matcher {
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
// List Tests
////////////////////////////////////////////////////////////////////////

func (t *BucketTest) EmptyBucket() {
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertNe(nil, listing)
	ExpectThat(listing.Objects, ElementsAre())
	ExpectThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)
}

func (t *BucketTest) NewlyCreatedObject() {
	// Create an object.
	AssertEq(nil, t.createObject("a", "taco"))

	// List all objects in the bucket.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertNe(nil, listing)
	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	var o *gcs.Object
	AssertEq(1, len(listing.Objects))

	// a
	o = listing.Objects[0]
	AssertEq("a", o.Name)
	ExpectEq(len("taco"), o.Size)
}

func (t *BucketTest) TrivialQuery() {
	// Create few objects.
	AssertEq(nil, t.createObject("a", "taco"))
	AssertEq(nil, t.createObject("b", "burrito"))
	AssertEq(nil, t.createObject("c", "enchilada"))

	// List all objects in the bucket.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertNe(nil, listing)
	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	var o *gcs.Object
	AssertEq(3, len(listing.Objects))

	// a
	o = listing.Objects[0]
	AssertEq("a", o.Name)
	ExpectEq(len("taco"), o.Size)

	// b
	o = listing.Objects[1]
	AssertEq("b", o.Name)
	ExpectEq(len("burrito"), o.Size)

	// c
	o = listing.Objects[2]
	AssertEq("c", o.Name)
	ExpectEq(len("enchilada"), o.Size)
}

func (t *BucketTest) Delimiter_SingleRune() {
	// Create several objects.
	AssertEq(
		nil,
		gcstests.CreateEmpty(
			t.ctx,
			t.bucket,
			[]string{
				"!",
				"a",
				"b",
				"b!foo",
				"b!bar",
				"b!baz!qux",
				"c!",
				"d!taco",
				"d!burrito",
				"e",
			}))

	// List with the delimiter "!".
	req := &gcs.ListObjectsRequest{
		Delimiter: "!",
	}

	listing, err := t.bucket.ListObjects(t.ctx, req)
	AssertEq(nil, err)
	AssertNe(nil, listing)
	AssertEq("", listing.ContinuationToken)

	// Collapsed runs
	ExpectThat(listing.CollapsedRuns, ElementsAre("!", "b!", "c!", "d!"))

	// Objects
	AssertEq(3, len(listing.Objects))

	ExpectEq("a", listing.Objects[0].Name)
	ExpectEq("b", listing.Objects[1].Name)
	ExpectEq("e", listing.Objects[2].Name)
}

func (t *BucketTest) Delimiter_MultiRune() {
	// Create several objects.
	AssertEq(
		nil,
		gcstests.CreateEmpty(
			t.ctx,
			t.bucket,
			[]string{
				"!",
				"!!",
				"!!!",
				"!!!!",
				"!!!!!!!!!",
				"a",
				"b",
				"b!",
				"b!foo",
				"b!!",
				"b!!!",
				"b!!foo",
				"b!!!foo",
				"b!!bar",
				"b!!baz!!qux",
				"c!!",
				"d!!taco",
				"d!!burrito",
				"e",
			}))

	// List with the delimiter "!!".
	req := &gcs.ListObjectsRequest{
		Delimiter: "!!",
	}

	listing, err := t.bucket.ListObjects(t.ctx, req)
	AssertEq(nil, err)
	AssertNe(nil, listing)
	AssertEq("", listing.ContinuationToken)

	// Collapsed runs
	ExpectThat(listing.CollapsedRuns, ElementsAre("!!", "b!!", "c!!", "d!!"))

	// Objects
	AssertEq(6, len(listing.Objects))

	ExpectEq("!", listing.Objects[0].Name)
	ExpectEq("a", listing.Objects[1].Name)
	ExpectEq("b", listing.Objects[2].Name)
	ExpectEq("b!", listing.Objects[3].Name)
	ExpectEq("b!foo", listing.Objects[4].Name)
	ExpectEq("e", listing.Objects[5].Name)
}

func (t *BucketTest) Prefix() {
	// Create several objects.
	AssertEq(
		nil,
		gcstests.CreateEmpty(
			t.ctx,
			t.bucket,
			[]string{
				"a",
				"a\x7f",
				"b",
				"b\x00",
				"b\x01",
				"b타코",
				"c",
			}))

	// List with the prefix "b".
	req := &gcs.ListObjectsRequest{
		Prefix: "b",
	}

	listing, err := t.bucket.ListObjects(t.ctx, req)
	AssertEq(nil, err)
	AssertNe(nil, listing)
	AssertEq("", listing.ContinuationToken)
	AssertThat(listing.CollapsedRuns, ElementsAre())

	// Objects
	AssertEq(4, len(listing.Objects))

	ExpectEq("b", listing.Objects[0].Name)
	ExpectEq("b\x00", listing.Objects[1].Name)
	ExpectEq("b\x01", listing.Objects[2].Name)
	ExpectEq("b타코", listing.Objects[3].Name)
}

func (t *BucketTest) PrefixAndDelimiter_SingleRune() {
	// Create several objects.
	AssertEq(
		nil,
		gcstests.CreateEmpty(
			t.ctx,
			t.bucket,
			[]string{
				"blag",
				"blag!",
				"blah",
				"blah!a",
				"blah!a\x7f",
				"blah!b",
				"blah!b!",
				"blah!b!asd",
				"blah!b\x00",
				"blah!b\x00!",
				"blah!b\x00!asd",
				"blah!b\x00!asd!sdf",
				"blah!b\x01",
				"blah!b\x01!",
				"blah!b\x01!asd",
				"blah!b\x01!asd!sdf",
				"blah!b타코",
				"blah!b타코!",
				"blah!b타코!asd",
				"blah!b타코!asd!sdf",
				"blah!c",
			}))

	// List with the prefix "blah!b" and the delimiter "!".
	req := &gcs.ListObjectsRequest{
		Prefix:    "blah!b",
		Delimiter: "!",
	}

	listing, err := t.bucket.ListObjects(t.ctx, req)
	AssertEq(nil, err)
	AssertNe(nil, listing)
	AssertEq("", listing.ContinuationToken)

	// Collapsed runs
	ExpectThat(
		listing.CollapsedRuns,
		ElementsAre(
			"blah!b\x00!",
			"blah!b\x01!",
			"blah!b!",
			"blah!b타코!",
		))

	// Objects
	AssertEq(4, len(listing.Objects))

	ExpectEq("blah!b", listing.Objects[0].Name)
	ExpectEq("blah!b\x00", listing.Objects[1].Name)
	ExpectEq("blah!b\x01", listing.Objects[2].Name)
	ExpectEq("blah!b타코", listing.Objects[3].Name)
}

func (t *BucketTest) PrefixAndDelimiter_MultiRune() {
	// Create several objects.
	AssertEq(
		nil,
		gcstests.CreateEmpty(
			t.ctx,
			t.bucket,
			[]string{
				"blag",
				"blag!!",
				"blah",
				"blah!!a",
				"blah!!a\x7f",
				"blah!!b",
				"blah!!b!",
				"blah!!b!!",
				"blah!!b!!asd",
				"blah!!b\x00",
				"blah!!b\x00!",
				"blah!!b\x00!!",
				"blah!!b\x00!!asd",
				"blah!!b\x00!!asd!sdf",
				"blah!!b\x01",
				"blah!!b\x01!",
				"blah!!b\x01!!",
				"blah!!b\x01!!asd",
				"blah!!b\x01!!asd!sdf",
				"blah!!b타코",
				"blah!!b타코!",
				"blah!!b타코!!",
				"blah!!b타코!!asd",
				"blah!!b타코!!asd!sdf",
				"blah!!c",
			}))

	// List with the prefix "blah!b" and the delimiter "!".
	req := &gcs.ListObjectsRequest{
		Prefix:    "blah!!b",
		Delimiter: "!!",
	}

	listing, err := t.bucket.ListObjects(t.ctx, req)
	AssertEq(nil, err)
	AssertNe(nil, listing)
	AssertEq("", listing.ContinuationToken)

	// Collapsed runs
	ExpectThat(
		listing.CollapsedRuns,
		ElementsAre(
			"blah!!b\x00!!",
			"blah!!b\x01!!",
			"blah!!b!!",
			"blah!!b타코!!",
		))

	// Objects
	AssertEq(8, len(listing.Objects))

	ExpectEq("blah!!b", listing.Objects[0].Name)
	ExpectEq("blah!!b\x00", listing.Objects[1].Name)
	ExpectEq("blah!!b\x00!", listing.Objects[2].Name)
	ExpectEq("blah!!b\x01", listing.Objects[3].Name)
	ExpectEq("blah!!b\x01!", listing.Objects[4].Name)
	ExpectEq("blah!!b!", listing.Objects[5].Name)
	ExpectEq("blah!!b타코", listing.Objects[6].Name)
	ExpectEq("blah!!b타코!", listing.Objects[7].Name)
}

func (t *BucketTest) Cursor_BucketEndsWithRunOfIndividualObjects() {
	// Create a good number of objects, containing a run of objects sharing a
	// prefix under the delimiter "!".
	AssertEq(
		nil,
		gcstests.CreateEmpty(
			t.ctx,
			t.bucket,
			[]string{
				"a",
				"b",
				"c",
				"c!0",
				"c!1",
				"c!2",
				"c!3",
				"c!4",
				"d!",
				"e",
				"e!",
				"f!",
				"g!",
				"h",
			}))

	// List repeatedly with a small value for MaxResults. Keep track of all of
	// the objects and runs we find.
	req := &gcs.ListObjectsRequest{
		Delimiter:  "!",
		MaxResults: 2,
	}

	var objects []string
	var runs []string

	for {
		listing, err := t.bucket.ListObjects(t.ctx, req)
		AssertEq(nil, err)

		for _, o := range listing.Objects {
			objects = append(objects, o.Name)
		}

		for _, p := range listing.CollapsedRuns {
			runs = append(runs, p)
		}

		if listing.ContinuationToken == "" {
			break
		}

		req.ContinuationToken = listing.ContinuationToken
	}

	// Check the results.
	ExpectThat(
		objects,
		ElementsAre(
			"a",
			"b",
			"c",
			"e",
			"h",
		))

	ExpectThat(
		runs,
		ElementsAre(
			"c!",
			"d!",
			"e!",
			"f!",
			"g!",
		))
}

func (t *BucketTest) Cursor_BucketEndsWithRunOfObjectsGroupedByDelimiter() {
	// Create a good number of objects, containing runs of objects sharing a
	// prefix under the delimiter "!" at the end of the bucket.
	AssertEq(
		nil,
		gcstests.CreateEmpty(
			t.ctx,
			t.bucket,
			[]string{
				"a",
				"b",
				"c",
				"c!",
				"c!0",
				"c!1",
				"c!2",
				"d!",
				"d!0",
				"d!1",
				"d!2",
			}))

	// List repeatedly with a small value for MaxResults. Keep track of all of
	// the objects and runs we find.
	req := &gcs.ListObjectsRequest{
		Delimiter:  "!",
		MaxResults: 2,
	}

	var objects []string
	var runs []string

	for {
		listing, err := t.bucket.ListObjects(t.ctx, req)
		AssertEq(nil, err)

		for _, o := range listing.Objects {
			objects = append(objects, o.Name)
		}

		for _, p := range listing.CollapsedRuns {
			runs = append(runs, p)
		}

		if listing.ContinuationToken == "" {
			break
		}

		req.ContinuationToken = listing.ContinuationToken
	}

	// Check the results.
	ExpectThat(
		objects,
		ElementsAre(
			"a",
			"b",
			"c",
		))

	ExpectThat(
		runs,
		ElementsAre(
			"c!",
			"d!",
		))
}

////////////////////////////////////////////////////////////////////////
// Stat Tests
////////////////////////////////////////////////////////////////////////

func (t *BucketTest) NonExistentObject() {
	req := &gcs.StatObjectRequest{
		Name: "foo",
	}

	_, err := t.bucket.StatObject(t.ctx, req)

	AssertThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
	ExpectThat(err, Error(MatchesRegexp("not found|404")))
}

func (t *BucketTest) StatAfterCreating() {
	// Create an object.
	createTime := t.clock.Now()
	orig, err := gcsutil.CreateObject(t.ctx, t.bucket, "foo", []byte("taco"))
	AssertEq(nil, err)
	AssertThat(orig.Updated, t.matchesStartTime(createTime))

	// Ensure the time below doesn't match exactly.
	t.advanceTime()

	// Stat it.
	req := &gcs.StatObjectRequest{
		Name: "foo",
	}

	o, err := t.bucket.StatObject(t.ctx, req)
	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq("foo", o.Name)
	ExpectEq(orig.Generation, o.Generation)
	ExpectEq(len("taco"), o.Size)
	ExpectThat(o.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(o.Updated, timeutil.TimeEq(orig.Updated))
}

func (t *BucketTest) StatAfterOverwriting() {
	// Create an object.
	_, err := gcsutil.CreateObject(t.ctx, t.bucket, "foo", []byte("taco"))
	AssertEq(nil, err)

	// Ensure the time below doesn't match exactly.
	t.advanceTime()

	// Overwrite it.
	overwriteTime := t.clock.Now()
	o2, err := gcsutil.CreateObject(t.ctx, t.bucket, "foo", []byte("burrito"))
	AssertEq(nil, err)
	AssertThat(o2.Updated, t.matchesStartTime(overwriteTime))

	// Ensure the time below doesn't match exactly.
	t.advanceTime()

	// Stat it.
	req := &gcs.StatObjectRequest{
		Name: "foo",
	}

	o, err := t.bucket.StatObject(t.ctx, req)
	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq("foo", o.Name)
	ExpectEq(o2.Generation, o.Generation)
	ExpectEq(len("burrito"), o.Size)
	ExpectThat(o.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(o.Updated, timeutil.TimeEq(o2.Updated))
}

func (t *BucketTest) StatAfterUpdating() {
	// Create an object.
	createTime := t.clock.Now()
	orig, err := gcsutil.CreateObject(t.ctx, t.bucket, "foo", []byte("taco"))
	AssertEq(nil, err)
	AssertThat(orig.Updated, t.matchesStartTime(createTime))

	// Ensure the time below doesn't match exactly.
	t.advanceTime()

	// Update the object.
	ureq := &gcs.UpdateObjectRequest{
		Name:        "foo",
		ContentType: gcstests.MakeStringPtr("image/png"),
	}

	updateTime := t.clock.Now()
	o2, err := t.bucket.UpdateObject(t.ctx, ureq)
	AssertEq(nil, err)
	AssertNe(o2.MetaGeneration, orig.MetaGeneration)

	// 'Updated' should be the update time, not the creation time.
	ExpectThat(o2.Updated, t.matchesStartTime(updateTime))
	ExpectTrue(
		orig.Updated.Before(o2.Updated),
		"orig.Updated: %v\n"+
			"o2.Updated:   %v",
		orig.Updated,
		o2.Updated)

	// Ensure the time below doesn't match exactly.
	t.advanceTime()

	// Stat the object.
	req := &gcs.StatObjectRequest{
		Name: "foo",
	}

	o, err := t.bucket.StatObject(t.ctx, req)
	AssertEq(nil, err)
	AssertNe(nil, o)

	ExpectEq("foo", o.Name)
	ExpectEq(o2.Generation, o.Generation)
	ExpectEq(o2.MetaGeneration, o.MetaGeneration)
	ExpectEq(len("taco"), o.Size)
	ExpectThat(o.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(o.Updated, timeutil.TimeEq(o2.Updated))
}

////////////////////////////////////////////////////////////////////////
// Delete Tests
////////////////////////////////////////////////////////////////////////

func (t *BucketTest) NoParticularGeneration_NameDoesntExist() {
	// No error should be returned.
	err := t.bucket.DeleteObject(
		t.ctx,
		&gcs.DeleteObjectRequest{
			Name: "foobar",
		})

	ExpectEq(nil, err)
}

func (t *BucketTest) NoParticularGeneration_Successful() {
	// Create an object.
	AssertEq(nil, t.createObject("a", "taco"))

	// Delete it.
	AssertEq(
		nil,
		t.bucket.DeleteObject(
			t.ctx,
			&gcs.DeleteObjectRequest{
				Name: "a",
			}))

	// It shouldn't show up in a listing.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertNe(nil, listing)
	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)
	ExpectThat(listing.Objects, ElementsAre())

	// It shouldn't be readable.
	req := &gcs.ReadObjectRequest{
		Name: "a",
	}

	rc, err := t.bucket.NewReader(t.ctx, req)
	if err == nil {
		defer rc.Close()
		_, err = rc.Read(make([]byte, 1))
	}

	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *BucketTest) ParticularGeneration_NameDoesntExist() {
	// No error should be returned.
	err := t.bucket.DeleteObject(
		t.ctx,
		&gcs.DeleteObjectRequest{
			Name:       "foobar",
			Generation: 17,
		})

	ExpectEq(nil, err)
}

func (t *BucketTest) ParticularGeneration_GenerationDoesntExist() {
	const name = "foo"
	var err error

	// Create an object.
	o, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		name,
		[]byte("taco"))

	AssertEq(nil, err)

	// Attempt to delete a different generation. Though it doesn't exist, no
	// error should be returned.
	err = t.bucket.DeleteObject(
		t.ctx,
		&gcs.DeleteObjectRequest{
			Name:       name,
			Generation: o.Generation + 1,
		})

	AssertEq(nil, err)

	// The original generation should still exist.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, name)

	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
}

func (t *BucketTest) ParticularGeneration_Successful() {
	const name = "foo"
	var err error

	// Create an object.
	o, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		name,
		[]byte("taco"))

	AssertEq(nil, err)

	// Delete that particular generation.
	err = t.bucket.DeleteObject(
		t.ctx,
		&gcs.DeleteObjectRequest{
			Name:       name,
			Generation: o.Generation,
		})

	AssertEq(nil, err)

	// The object should no longer exist.
	_, err = gcsutil.ReadObject(t.ctx, t.bucket, name)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *BucketTest) MetaGenerationPrecondition_Unsatisfied_ObjectExists() {
	const name = "foo"
	var err error

	// Create an object.
	o, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		name,
		[]byte("taco"))

	AssertEq(nil, err)

	// Attempt to delete, with a precondition for the wrong meta-generation.
	precond := o.MetaGeneration + 1
	err = t.bucket.DeleteObject(
		t.ctx,
		&gcs.DeleteObjectRequest{
			Name:                       name,
			MetaGenerationPrecondition: &precond,
		})

	ExpectThat(err, HasSameTypeAs(&gcs.PreconditionError{}))

	// The object should still exist.
	_, err = gcsutil.ReadObject(t.ctx, t.bucket, name)
	ExpectEq(nil, err)
}

func (t *BucketTest) MetaGenerationPrecondition_Unsatisfied_ObjectDoesntExist() {
	const name = "foo"
	var err error

	// Attempt to delete a non-existent name with a meta-generation precondition.
	var precond int64 = 1
	err = t.bucket.DeleteObject(
		t.ctx,
		&gcs.DeleteObjectRequest{
			Name:                       name,
			MetaGenerationPrecondition: &precond,
		})

	ExpectEq(nil, err)
}

func (t *BucketTest) MetaGenerationPrecondition_Unsatisfied_WrongGeneration() {
	const name = "foo"
	var err error

	// Create an object.
	o, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		name,
		[]byte("taco"))

	AssertEq(nil, err)

	// Attempt to delete, with a precondition for the wrong meta-generation,
	// addressing the wrong object generation.
	precond := o.MetaGeneration + 1
	err = t.bucket.DeleteObject(
		t.ctx,
		&gcs.DeleteObjectRequest{
			Name:                       name,
			Generation:                 o.Generation + 1,
			MetaGenerationPrecondition: &precond,
		})

	ExpectEq(nil, err)

	// The object should still exist.
	_, err = gcsutil.ReadObject(t.ctx, t.bucket, name)
	ExpectEq(nil, err)
}

func (t *BucketTest) MetaGenerationPrecondition_Satisfied() {
	const name = "foo"
	var err error

	// Create an object.
	o, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		name,
		[]byte("taco"))

	AssertEq(nil, err)

	// Delete with a precondition.
	precond := o.MetaGeneration
	err = t.bucket.DeleteObject(
		t.ctx,
		&gcs.DeleteObjectRequest{
			Name:                       name,
			MetaGenerationPrecondition: &precond,
		})

	AssertEq(nil, err)

	// The object should no longer exist.
	_, err = gcsutil.ReadObject(t.ctx, t.bucket, name)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}
