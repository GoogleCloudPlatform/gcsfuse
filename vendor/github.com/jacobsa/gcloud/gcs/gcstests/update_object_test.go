package gcstests

import (
	"fmt"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	"github.com/jacobsa/gcloud/gcs/gcstests"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	"github.com/jacobsa/reqtrace"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
	"io/ioutil"
	"strings"
	"testing"
	"time"
)

func TestUpdate(t *testing.T) { RunTests(t) }

type UpdateTest struct {
	ctx    context.Context
	bucket gcs.Bucket
	clock  timeutil.Clock
}

var _ SetUpInterface = &UpdateTest{}

func init() { RegisterTestSuite(&UpdateTest{}) }

////////////////////////////////////////////////////////////////////////
// Initialization
////////////////////////////////////////////////////////////////////////

func (t *UpdateTest) SetUp(ti *TestInfo) {
	t.ctx, _ = reqtrace.Trace(context.Background(), "Overall test")
	clock := &timeutil.SimulatedClock{}
	clock.SetTime(time.Date(2022, 7, 12, 18, 56, 0, 0, time.Local))
	t.clock = clock

	// Set up the fake bucket.
	t.bucket = gcsfake.NewFakeBucket(t.ctx, clock, "some_bucket")

}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (t *UpdateTest) createObject(name string, contents string) error {
	_, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		name,
		[]byte(contents))

	return err
}

func (t *UpdateTest) readObject(objectName string) (contents string, err error) {
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
func (t *UpdateTest) advanceTime() {
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
func (t *UpdateTest) matchesStartTime(start time.Time) Matcher {
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
// Update Tests
////////////////////////////////////////////////////////////////////////

func (t *UpdateTest) NonExistentObject() {
	req := &gcs.UpdateObjectRequest{
		Name:        "foo",
		ContentType: gcstests.MakeStringPtr("image/png"),
	}

	_, err := t.bucket.UpdateObject(t.ctx, req)
	fmt.Println(err)
	AssertThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *UpdateTest) RemoveAllFields() {
	// Create an object with explicit attributes set.
	createReq := &gcs.CreateObjectRequest{
		Name:            "foo",
		ContentType:     "image/png",
		ContentEncoding: "gzip",
		ContentLanguage: "fr",
		CacheControl:    "public",
		Metadata: map[string]string{
			"foo": "bar",
		},

		Contents: strings.NewReader("taco"),
	}

	_, err := t.bucket.CreateObject(t.ctx, createReq)
	AssertEq(nil, err)

	// Remove all of the fields that were set, aside from user metadata.
	req := &gcs.UpdateObjectRequest{
		Name:            "foo",
		ContentEncoding: gcstests.MakeStringPtr(""),
		ContentLanguage: gcstests.MakeStringPtr(""),
		ContentType:     gcstests.MakeStringPtr(""),
		CacheControl:    gcstests.MakeStringPtr(""),
	}

	o, err := t.bucket.UpdateObject(t.ctx, req)
	AssertEq(nil, err)

	// Check the returned object.
	AssertEq("foo", o.Name)
	AssertEq(len("taco"), o.Size)
	AssertEq(2, o.MetaGeneration)

	ExpectEq("", o.ContentType)
	ExpectEq("", o.ContentEncoding)
	ExpectEq("", o.ContentLanguage)
	ExpectEq("", o.CacheControl)

	ExpectThat(o.Metadata, DeepEquals(createReq.Metadata))

	// Check that a listing agrees.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	AssertEq(1, len(listing.Objects))
	ExpectThat(listing.Objects[0], DeepEquals(o))
}

func (t *UpdateTest) ModifyAllFields() {
	// Create an object with explicit attributes set.
	createReq := &gcs.CreateObjectRequest{
		Name:            "foo",
		ContentType:     "image/png",
		ContentEncoding: "gzip",
		ContentLanguage: "fr",
		CacheControl:    "public",
		Metadata: map[string]string{
			"foo": "bar",
		},

		Contents: strings.NewReader("taco"),
	}

	_, err := t.bucket.CreateObject(t.ctx, createReq)
	AssertEq(nil, err)

	// Modify all of the fields that were set, aside from user metadata.
	req := &gcs.UpdateObjectRequest{
		Name:            "foo",
		ContentType:     gcstests.MakeStringPtr("image/jpeg"),
		ContentEncoding: gcstests.MakeStringPtr("bzip2"),
		ContentLanguage: gcstests.MakeStringPtr("de"),
		CacheControl:    gcstests.MakeStringPtr("private"),
	}

	o, err := t.bucket.UpdateObject(t.ctx, req)
	AssertEq(nil, err)

	// Check the returned object.
	AssertEq("foo", o.Name)
	AssertEq(len("taco"), o.Size)
	AssertEq(2, o.MetaGeneration)

	ExpectEq("image/jpeg", o.ContentType)
	ExpectEq("bzip2", o.ContentEncoding)
	ExpectEq("de", o.ContentLanguage)
	ExpectEq("private", o.CacheControl)

	ExpectThat(o.Metadata, DeepEquals(createReq.Metadata))

	// Check that a listing agrees.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	AssertEq(1, len(listing.Objects))
	ExpectThat(listing.Objects[0], DeepEquals(o))
}

func (t *UpdateTest) MixedModificationsToFields() {
	// Create an object with some explicit attributes set.
	createReq := &gcs.CreateObjectRequest{
		Name:            "foo",
		ContentType:     "image/png",
		ContentEncoding: "gzip",
		ContentLanguage: "fr",
		Metadata: map[string]string{
			"foo": "bar",
		},

		Contents: strings.NewReader("taco"),
	}

	_, err := t.bucket.CreateObject(t.ctx, createReq)
	AssertEq(nil, err)

	// Leave one field unmodified, delete one field, modify an existing field,
	// and add a new field.
	req := &gcs.UpdateObjectRequest{
		Name:            "foo",
		ContentType:     nil,
		ContentEncoding: gcstests.MakeStringPtr(""),
		ContentLanguage: gcstests.MakeStringPtr("de"),
		CacheControl:    gcstests.MakeStringPtr("private"),
	}

	o, err := t.bucket.UpdateObject(t.ctx, req)
	AssertEq(nil, err)

	// Check the returned object.
	AssertEq("foo", o.Name)
	AssertEq(len("taco"), o.Size)
	AssertEq(2, o.MetaGeneration)

	ExpectEq("image/png", o.ContentType)
	ExpectEq("", o.ContentEncoding)
	ExpectEq("de", o.ContentLanguage)
	ExpectEq("private", o.CacheControl)

	ExpectThat(o.Metadata, DeepEquals(createReq.Metadata))

	// Check that a listing agrees.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	AssertEq(1, len(listing.Objects))
	ExpectThat(listing.Objects[0], DeepEquals(o))
}

func (t *UpdateTest) AddUserMetadata() {
	// Create an object with no user metadata.
	orig, err := gcsutil.CreateObject(t.ctx, t.bucket, "foo", []byte("taco"))
	AssertEq(nil, err)

	AssertEq(nil, orig.Metadata)

	// Add some metadata.
	req := &gcs.UpdateObjectRequest{
		Name: "foo",
		Metadata: map[string]*string{
			"0": gcstests.MakeStringPtr("taco"),
			"1": gcstests.MakeStringPtr("burrito"),
		},
	}

	o, err := t.bucket.UpdateObject(t.ctx, req)
	AssertEq(nil, err)

	// Check the returned object.
	AssertEq("foo", o.Name)
	AssertEq(len("taco"), o.Size)
	AssertEq(2, o.MetaGeneration)

	ExpectThat(
		o.Metadata,
		DeepEquals(
			map[string]string{
				"0": "taco",
				"1": "burrito",
			}))

	// Check that a listing agrees.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	AssertEq(1, len(listing.Objects))
	ExpectThat(listing.Objects[0], DeepEquals(o))
}

func (t *UpdateTest) MixedModificationsToUserMetadata() {
	// Create an object with some user metadata.
	createReq := &gcs.CreateObjectRequest{
		Name: "foo",
		Metadata: map[string]string{
			"0": "taco",
			"2": "enchilada",
			"3": "queso",
		},

		Contents: strings.NewReader("taco"),
	}

	orig, err := t.bucket.CreateObject(t.ctx, createReq)
	AssertEq(nil, err)

	AssertThat(orig.Metadata, DeepEquals(createReq.Metadata))

	// Leave an existing field untouched, add a new field, remove an existing
	// field, and modify an existing field.
	req := &gcs.UpdateObjectRequest{
		Name: "foo",
		Metadata: map[string]*string{
			"1": gcstests.MakeStringPtr("burrito"),
			"2": nil,
			"3": gcstests.MakeStringPtr("updated"),
		},
	}

	o, err := t.bucket.UpdateObject(t.ctx, req)
	AssertEq(nil, err)

	// Check the returned object.
	AssertEq("foo", o.Name)
	AssertEq(len("taco"), o.Size)
	AssertEq(2, o.MetaGeneration)

	ExpectThat(
		o.Metadata,
		DeepEquals(
			map[string]string{
				"0": "taco",
				"1": "burrito",
				"3": "updated",
			}))

	// Check that a listing agrees.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	AssertEq(1, len(listing.Objects))
	ExpectThat(listing.Objects[0], DeepEquals(o))
}

func (t *UpdateTest) UpdateTime() {
	// Create an object.
	createTime := t.clock.Now()
	o, err := gcsutil.CreateObject(t.ctx, t.bucket, "foo", []byte{})
	AssertEq(nil, err)
	AssertThat(o.Updated, t.matchesStartTime(createTime))

	// Ensure the time below doesn't match exactly.
	t.advanceTime()

	// Modify a field.
	req := &gcs.UpdateObjectRequest{
		Name:        "foo",
		ContentType: gcstests.MakeStringPtr("image/jpeg"),
	}

	updateTime := t.clock.Now()
	o2, err := t.bucket.UpdateObject(t.ctx, req)
	AssertEq(nil, err)

	// 'Updated' should be the update time, not the creation time.
	ExpectThat(o2.Updated, t.matchesStartTime(updateTime))
	ExpectTrue(
		o.Updated.Before(o2.Updated),
		"o.Updated:  %v\n"+
			"o2.Updated: %v",
		o.Updated,
		o2.Updated)
}

func (t *UpdateTest) ParticularGeneration_NameDoesntExist() {
	req := &gcs.UpdateObjectRequest{
		Name:        "foo",
		Generation:  17,
		ContentType: gcstests.MakeStringPtr("image/png"),
	}

	_, err := t.bucket.UpdateObject(t.ctx, req)

	AssertThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *UpdateTest) ParticularGeneration_GenerationDoesntExist() {
	// Create an object.
	createReq := &gcs.CreateObjectRequest{
		Name:     "foo",
		Contents: strings.NewReader(""),
	}

	o, err := t.bucket.CreateObject(t.ctx, createReq)
	AssertEq(nil, err)

	// Attempt to update the wrong generation by giving it a new content
	// language.
	req := &gcs.UpdateObjectRequest{
		Name:            o.Name,
		Generation:      o.Generation + 1,
		ContentLanguage: gcstests.MakeStringPtr("fr"),
	}

	_, err = t.bucket.UpdateObject(t.ctx, req)

	AssertThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
	ExpectThat(err, Error(MatchesRegexp("not found|404")))

	// The original object should be unaffected.
	o, err = t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: o.Name})

	AssertEq(nil, err)
	ExpectEq("", o.ContentLanguage)
}

func (t *UpdateTest) ParticularGeneration_Successful() {
	// Create an object.
	createReq := &gcs.CreateObjectRequest{
		Name:     "foo",
		Contents: strings.NewReader(""),
	}

	o, err := t.bucket.CreateObject(t.ctx, createReq)
	AssertEq(nil, err)

	// Update it with an explicit generation.
	req := &gcs.UpdateObjectRequest{
		Name:            o.Name,
		Generation:      o.Generation,
		ContentLanguage: gcstests.MakeStringPtr("fr"),
	}

	o, err = t.bucket.UpdateObject(t.ctx, req)

	AssertEq(nil, err)
	ExpectEq("fr", o.ContentLanguage)

	// Stat and make sure it took effect.
	o, err = t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: o.Name})

	AssertEq(nil, err)
	ExpectEq("fr", o.ContentLanguage)
}

func (t *UpdateTest) MetaGenerationPrecondition_Unsatisfied() {
	// Create an object.
	createReq := &gcs.CreateObjectRequest{
		Name:     "foo",
		Contents: strings.NewReader(""),
	}

	o, err := t.bucket.CreateObject(t.ctx, createReq)
	AssertEq(nil, err)

	// Attempt to update with a bad precondition.
	precond := o.MetaGeneration + 1
	req := &gcs.UpdateObjectRequest{
		Name:                       o.Name,
		MetaGenerationPrecondition: &precond,
		ContentLanguage:            gcstests.MakeStringPtr("fr"),
	}

	_, err = t.bucket.UpdateObject(t.ctx, req)
	ExpectThat(err, HasSameTypeAs(&gcs.PreconditionError{}))

	// The original object should be unaffected.
	o, err = t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: o.Name})

	AssertEq(nil, err)
	ExpectEq("", o.ContentLanguage)
}

func (t *UpdateTest) MetaGenerationPrecondition_Satisfied() {
	// Create an object.
	createReq := &gcs.CreateObjectRequest{
		Name:     "foo",
		Contents: strings.NewReader(""),
	}

	o, err := t.bucket.CreateObject(t.ctx, createReq)
	AssertEq(nil, err)

	// Update with a good precondition.
	req := &gcs.UpdateObjectRequest{
		Name:                       o.Name,
		MetaGenerationPrecondition: &o.MetaGeneration,
		ContentLanguage:            gcstests.MakeStringPtr("fr"),
	}

	_, err = t.bucket.UpdateObject(t.ctx, req)
	AssertEq(nil, err)

	// The object should have been updated.
	o, err = t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: o.Name})

	AssertEq(nil, err)
	ExpectEq("fr", o.ContentLanguage)
}
