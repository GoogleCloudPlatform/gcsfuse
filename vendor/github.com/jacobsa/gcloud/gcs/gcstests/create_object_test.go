package gcstests

import (
	"crypto/md5"
	"encoding/hex"
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
	"sort"
	"strings"
	"testing"
	"testing/iotest"
	"time"
)

func TestCreate(t *testing.T) { RunTests(t) }

type CreateTest struct {
	ctx    context.Context
	bucket gcs.Bucket
	clock  timeutil.Clock
}

var _ SetUpInterface = &CreateTest{}

func init() { RegisterTestSuite(&CreateTest{}) }

////////////////////////////////////////////////////////////////////////
// Initialization
////////////////////////////////////////////////////////////////////////

func (t *CreateTest) SetUp(ti *TestInfo) {
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

func (t *CreateTest) createObject(name string, contents string) error {
	_, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		name,
		[]byte(contents))

	return err
}

func (t *CreateTest) readObject(objectName string) (contents string, err error) {
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
func (t *CreateTest) advanceTime() {
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
func (t *CreateTest) matchesStartTime(start time.Time) Matcher {
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
// Create Tests
////////////////////////////////////////////////////////////////////////

func (t *CreateTest) EmptyObject() {
	// Create the object.
	AssertEq(nil, t.createObject("foo", ""))

	// Ensure it shows up in a listing.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	AssertEq(1, len(listing.Objects))
	o := listing.Objects[0]

	AssertEq("foo", o.Name)
	ExpectEq(0, o.Size)
}

func (t *CreateTest) NonEmptyObject() {
	// Create the object.
	AssertEq(nil, t.createObject("foo", "taco"))

	// Ensure it shows up in a listing.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	AssertEq(1, len(listing.Objects))
	o := listing.Objects[0]

	AssertEq("foo", o.Name)
	ExpectEq(len("taco"), o.Size)
}

func (t *CreateTest) Overwrite() {
	var err error

	// Create a first version of an object, with some custom metadata.
	_, err = t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name: "foo",
			Metadata: map[string]string{
				"foo": "bar",
			},
			Contents: strings.NewReader("taco"),
		})

	AssertEq(nil, err)

	// Overwrite it with another version.
	_, err = t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:     "foo",
			Contents: strings.NewReader("burrito"),
		})

	AssertEq(nil, err)

	// The second version should show up in a listing.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	AssertEq(1, len(listing.Objects))
	o := listing.Objects[0]

	AssertEq("foo", o.Name)
	ExpectEq(len("burrito"), o.Size)
	ExpectEq(0, len(o.Metadata))

	// The second version should be what we get when we read the object.
	contents, err := t.readObject("foo")
	AssertEq(nil, err)
	ExpectEq("burrito", contents)
}

func (t *CreateTest) ObjectAttributes_Default() {
	// Create an object with default attributes aside from the name.
	createTime := t.clock.Now()
	o, err := gcsutil.CreateObject(t.ctx, t.bucket, "foo", []byte("taco"))
	AssertEq(nil, err)

	// Ensure the time below doesn't match exactly.
	t.advanceTime()

	// Check the Object struct.
	ExpectEq("foo", o.Name)
	ExpectEq("", o.ContentType)
	ExpectEq("", o.ContentLanguage)
	ExpectEq("", o.CacheControl)
	ExpectThat(o.Owner, MatchesRegexp("^user-.*"))
	ExpectEq(len("taco"), o.Size)
	ExpectEq("", o.ContentEncoding)
	ExpectEq(1, o.ComponentCount)
	ExpectThat(o.MD5, Pointee(DeepEquals(md5.Sum([]byte("taco")))))
	ExpectEq(gcstests.ComputeCrc32C("taco"), *o.CRC32C)
	ExpectThat(o.MediaLink, MatchesRegexp("download/storage.*foo"))
	ExpectEq(nil, o.Metadata)
	ExpectLt(0, o.Generation)
	ExpectEq(1, o.MetaGeneration)
	ExpectEq("STANDARD", o.StorageClass)
	ExpectThat(o.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(o.Updated, t.matchesStartTime(createTime))

	// Make sure it matches what is in a listing.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	AssertEq(1, len(listing.Objects))
	ExpectThat(listing.Objects[0], DeepEquals(o))
}

func (t *CreateTest) ObjectAttributes_Explicit() {
	// Create an object with explicit attributes set.
	createTime := t.clock.Now()
	req := &gcs.CreateObjectRequest{
		Name:            "foo",
		ContentType:     "image/png",
		ContentLanguage: "fr",
		ContentEncoding: "gzip",
		CacheControl:    "public",
		Metadata: map[string]string{
			"foo": "bar",
			"baz": "qux",
		},

		Contents: strings.NewReader("taco"),
	}

	o, err := t.bucket.CreateObject(t.ctx, req)
	AssertEq(nil, err)

	// Ensure the time below doesn't match exactly.
	t.advanceTime()

	// Check the Object struct.
	ExpectEq("foo", o.Name)
	ExpectEq("image/png", o.ContentType)
	ExpectEq("fr", o.ContentLanguage)
	ExpectEq("public", o.CacheControl)
	ExpectThat(o.Owner, MatchesRegexp("^user-.*"))
	ExpectEq(len("taco"), o.Size)
	ExpectEq("gzip", o.ContentEncoding)
	ExpectEq(1, o.ComponentCount)
	ExpectThat(o.MD5, Pointee(DeepEquals(md5.Sum([]byte("taco")))))
	ExpectEq(gcstests.ComputeCrc32C("taco"), *o.CRC32C)
	ExpectThat(o.MediaLink, MatchesRegexp("download/storage.*foo"))
	ExpectThat(o.Metadata, DeepEquals(req.Metadata))
	ExpectLt(0, o.Generation)
	ExpectEq(1, o.MetaGeneration)
	ExpectEq("STANDARD", o.StorageClass)
	ExpectThat(o.Deleted, DeepEquals(time.Time{}))
	ExpectThat(o.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(o.Updated, t.matchesStartTime(createTime))

	// Make sure it matches what is in a listing.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	AssertEq(1, len(listing.Objects))
	ExpectThat(listing.Objects[0], DeepEquals(o))
}

func (t *CreateTest) ErrorAfterPartialContents() {
	const contents = "tacoburritoenchilada"

	// Set up a reader that will return some successful data, then an error.
	req := &gcs.CreateObjectRequest{
		Name: "foo",
		Contents: iotest.TimeoutReader(
			iotest.OneByteReader(
				strings.NewReader(contents))),
	}

	// An attempt to create the object should fail.
	_, err := t.bucket.CreateObject(t.ctx, req)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("timeout")))

	// The object should not show up in a listing.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	ExpectThat(listing.Objects, ElementsAre())
}

func (t *CreateTest) InterestingNames() {
	var err error

	// Grab a list of interesting legal names.
	names := gcstests.InterestingNames()

	// Make sure we can create each name.
	err = gcstests.ForEachString(
		t.ctx,
		names,
		func(ctx context.Context, name string) (err error) {
			err = t.createObject(name, name)
			if err != nil {
				err = fmt.Errorf("Failed to create %q: %v", name, err)
				return
			}

			return
		})

	AssertEq(nil, err)

	// Make sure we can read each, and that we get back the content we created
	// above.
	err = gcstests.ForEachString(
		t.ctx,
		names,
		func(ctx context.Context, name string) (err error) {
			contents, err := t.readObject(name)

			if err != nil {
				err = fmt.Errorf("Failed to read %q: %v", name, err)
				return
			}

			if contents != name {
				err = fmt.Errorf(
					"Incorrect contents for %q: %q",
					name,
					contents)

				return
			}

			return
		})

	AssertEq(nil, err)

	// Grab a listing and extract the names.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	var listingNames []string
	for _, o := range listing.Objects {
		listingNames = append(listingNames, o.Name)
	}

	// The names should have come back sorted by their UTF-8 encodings.
	AssertTrue(sort.IsSorted(sort.StringSlice(listingNames)))

	// Make sure all and only the expected names exist.
	if diff := gcstests.ListDifference(listingNames, names); len(diff) != 0 {
		var dumps []string
		for _, n := range diff {
			dumps = append(dumps, hex.Dump([]byte(n)))
		}

		AddFailure(
			"Unexpected names in listing:\n%s",
			strings.Join(dumps, "\n"))
	}

	if diff := gcstests.ListDifference(names, listingNames); len(diff) != 0 {
		var dumps []string
		for _, n := range diff {
			dumps = append(dumps, hex.Dump([]byte(n)))
		}

		AddFailure(
			"Names missing from listing:\n%s",
			strings.Join(dumps, "\n"))
	}
}

func (t *CreateTest) IllegalNames() {
	var err error

	// Make sure we cannot create any of the names above.
	err = gcstests.ForEachString(
		t.ctx,
		gcstests.IllegalNames(),
		func(ctx context.Context, name string) (err error) {
			err = t.createObject(name, "")
			if err == nil {
				err = fmt.Errorf("Expected to not be able to create %q", name)
				return
			}

			if name == "" {
				if !strings.Contains(err.Error(), "Invalid") &&
					!strings.Contains(err.Error(), "Required") {
					err = fmt.Errorf("Unexpected error for %q: %v", name, err)
					return
				}
			} else {
				if !strings.Contains(err.Error(), "Invalid") {
					err = fmt.Errorf("Unexpected error for %q: %v", name, err)
					return
				}
			}

			err = nil
			return
		})

	AssertEq(nil, err)

	// No objects should have been created.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)
	ExpectThat(listing.Objects, ElementsAre())
}

func (t *CreateTest) IncorrectCRC32C() {
	const name = "foo"
	const contents = "taco"
	var err error

	// Attempt to create with the wrong checksum.
	crc32c := gcsutil.CRC32C([]byte(contents))
	*crc32c++

	req := &gcs.CreateObjectRequest{
		Name:     name,
		Contents: strings.NewReader(contents),
		CRC32C:   crc32c,
	}

	_, err = t.bucket.CreateObject(t.ctx, req)
	AssertThat(err, Error(HasSubstr("CRC32C")))
	AssertThat(err, Error(HasSubstr("match")))

	// It should not have been created.
	statReq := &gcs.StatObjectRequest{
		Name: name,
	}

	_, err = t.bucket.StatObject(t.ctx, statReq)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *CreateTest) CorrectCRC32C() {
	const name = "foo"
	const contents = "taco"
	var err error

	// Create
	req := &gcs.CreateObjectRequest{
		Name:     name,
		Contents: strings.NewReader(contents),
		CRC32C:   gcsutil.CRC32C([]byte(contents)),
	}

	o, err := t.bucket.CreateObject(t.ctx, req)
	AssertEq(nil, err)
	ExpectEq(len(contents), o.Size)
}

func (t *CreateTest) IncorrectMD5() {
	const name = "foo"
	const contents = "taco"
	var err error

	// Attempt to create with the wrong checksum.
	md5 := gcsutil.MD5([]byte(contents))
	(*md5)[13]++

	req := &gcs.CreateObjectRequest{
		Name:     name,
		Contents: strings.NewReader(contents),
		MD5:      md5,
	}

	_, err = t.bucket.CreateObject(t.ctx, req)
	AssertThat(err, Error(HasSubstr("MD5")))
	AssertThat(err, Error(HasSubstr("match")))

	// It should not have been created.
	statReq := &gcs.StatObjectRequest{
		Name: name,
	}

	_, err = t.bucket.StatObject(t.ctx, statReq)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *CreateTest) CorrectMD5() {
	const name = "foo"
	const contents = "taco"
	var err error

	// Create
	req := &gcs.CreateObjectRequest{
		Name:     name,
		Contents: strings.NewReader(contents),
		MD5:      gcsutil.MD5([]byte(contents)),
	}

	o, err := t.bucket.CreateObject(t.ctx, req)
	AssertEq(nil, err)
	ExpectEq(len(contents), o.Size)
}

func (t *CreateTest) CorrectCRC32CAndMD5() {
	const name = "foo"
	const contents = "taco"
	var err error

	// Create
	req := &gcs.CreateObjectRequest{
		Name:     name,
		Contents: strings.NewReader(contents),
		CRC32C:   gcsutil.CRC32C([]byte(contents)),
		MD5:      gcsutil.MD5([]byte(contents)),
	}

	o, err := t.bucket.CreateObject(t.ctx, req)
	AssertEq(nil, err)
	ExpectEq(len(contents), o.Size)
}

func (t *CreateTest) GenerationPrecondition_Zero_Unsatisfied() {
	// Create an existing object.
	o, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		"foo",
		[]byte("taco"))

	// Request to create another version of the object, with a precondition
	// saying it shouldn't exist. The request should fail.
	var gen int64 = 0
	req := &gcs.CreateObjectRequest{
		Name:                   "foo",
		Contents:               strings.NewReader("burrito"),
		GenerationPrecondition: &gen,
	}

	_, err = t.bucket.CreateObject(t.ctx, req)

	AssertThat(err, HasSameTypeAs(&gcs.PreconditionError{}))
	ExpectThat(err, Error(MatchesRegexp("object exists|googleapi.*412")))

	// The old version should show up in a listing.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	AssertEq(1, len(listing.Objects))
	AssertEq("foo", listing.Objects[0].Name)
	ExpectEq(o.Generation, listing.Objects[0].Generation)
	ExpectEq(len("taco"), listing.Objects[0].Size)

	// We should see the old contents when we read.
	contents, err := t.readObject("foo")
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
}

func (t *CreateTest) GenerationPrecondition_Zero_Satisfied() {
	// Request to create an object with a precondition saying it shouldn't exist.
	// The request should succeed.
	var gen int64 = 0
	req := &gcs.CreateObjectRequest{
		Name:                   "foo",
		Contents:               strings.NewReader("burrito"),
		GenerationPrecondition: &gen,
	}

	o, err := t.bucket.CreateObject(t.ctx, req)
	AssertEq(nil, err)

	ExpectEq(len("burrito"), o.Size)
	ExpectNe(0, o.Generation)

	// The object should show up in a listing.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	AssertEq(1, len(listing.Objects))
	AssertEq("foo", listing.Objects[0].Name)
	ExpectEq(o.Generation, listing.Objects[0].Generation)
	ExpectEq(len("burrito"), listing.Objects[0].Size)

	// We should see the new contents when we read.
	contents, err := t.readObject("foo")
	AssertEq(nil, err)
	ExpectEq("burrito", string(contents))
}

func (t *CreateTest) GenerationPrecondition_NonZero_Unsatisfied_Missing() {
	// Request to create a non-existent object with a precondition saying it
	// should already exist with some generation number. The request should fail.
	var gen int64 = 17
	req := &gcs.CreateObjectRequest{
		Name:                   "foo",
		Contents:               strings.NewReader("burrito"),
		GenerationPrecondition: &gen,
	}

	_, err := t.bucket.CreateObject(t.ctx, req)

	AssertThat(err, HasSameTypeAs(&gcs.PreconditionError{}))
	ExpectThat(err, Error(MatchesRegexp("object doesn't exist|googleapi.*412")))

	// Nothing should show up in a listing.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)
	ExpectEq(0, len(listing.Objects))
}

func (t *CreateTest) GenerationPrecondition_NonZero_Unsatisfied_Present() {
	// Create an existing object.
	o, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		"foo",
		[]byte("taco"))

	// Request to create another version of the object, with a precondition for
	// the wrong generation. The request should fail.
	var gen int64 = o.Generation + 1
	req := &gcs.CreateObjectRequest{
		Name:                   "foo",
		Contents:               strings.NewReader("burrito"),
		GenerationPrecondition: &gen,
	}

	_, err = t.bucket.CreateObject(t.ctx, req)

	AssertThat(err, HasSameTypeAs(&gcs.PreconditionError{}))
	ExpectThat(err, Error(MatchesRegexp("generation|googleapi.*412")))

	// The old version should show up in a listing.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	AssertEq(1, len(listing.Objects))
	AssertEq("foo", listing.Objects[0].Name)
	ExpectEq(o.Generation, listing.Objects[0].Generation)
	ExpectEq(len("taco"), listing.Objects[0].Size)

	// We should see the old contents when we read.
	contents, err := t.readObject("foo")
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
}

func (t *CreateTest) GenerationPrecondition_NonZero_Satisfied() {
	// Create an existing object.
	orig, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		"foo",
		[]byte("taco"))

	// Request to create another version of the object, with a precondition
	// saying it should exist with the appropriate generation number. The request
	// should succeed.
	var gen int64 = orig.Generation
	req := &gcs.CreateObjectRequest{
		Name:                   "foo",
		Contents:               strings.NewReader("burrito"),
		GenerationPrecondition: &gen,
	}

	o, err := t.bucket.CreateObject(t.ctx, req)
	AssertEq(nil, err)

	ExpectEq(len("burrito"), o.Size)
	ExpectNe(orig.Generation, o.Generation)

	// The new version should show up in a listing.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	AssertEq(1, len(listing.Objects))
	AssertEq("foo", listing.Objects[0].Name)
	ExpectEq(o.Generation, listing.Objects[0].Generation)
	ExpectEq(len("burrito"), listing.Objects[0].Size)

	// We should see the new contents when we read.
	contents, err := t.readObject("foo")
	AssertEq(nil, err)
	ExpectEq("burrito", string(contents))
}

func (t *CreateTest) MetaGenerationPrecondition_Unsatisfied_ObjectDoesntExist() {
	var err error

	// Request to create a missing object, with a precondition for
	// meta-generation. The request should fail.
	var metagen int64 = 1
	req := &gcs.CreateObjectRequest{
		Name:                       "foo",
		Contents:                   strings.NewReader("burrito"),
		MetaGenerationPrecondition: &metagen,
	}

	_, err = t.bucket.CreateObject(t.ctx, req)

	AssertThat(err, HasSameTypeAs(&gcs.PreconditionError{}))
	ExpectThat(err, Error(MatchesRegexp("doesn't exist|googleapi.*412")))

	// Nothing should show up in a listing.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	ExpectEq(0, len(listing.Objects))
}

func (t *CreateTest) MetaGenerationPrecondition_Unsatisfied_ObjectExists() {
	// Create an existing object.
	o, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		"foo",
		[]byte("taco"))

	// Request to create another version of the object, with a precondition for
	// the wrong meta-generation. The request should fail.
	var metagen int64 = o.MetaGeneration + 1
	req := &gcs.CreateObjectRequest{
		Name:                       "foo",
		Contents:                   strings.NewReader("burrito"),
		MetaGenerationPrecondition: &metagen,
	}

	_, err = t.bucket.CreateObject(t.ctx, req)

	AssertThat(err, HasSameTypeAs(&gcs.PreconditionError{}))
	ExpectThat(err, Error(MatchesRegexp("meta-generation|googleapi.*412")))

	// The old version should show up in a listing.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	AssertEq(1, len(listing.Objects))
	AssertEq("foo", listing.Objects[0].Name)
	ExpectEq(o.Generation, listing.Objects[0].Generation)
	ExpectEq(o.MetaGeneration, listing.Objects[0].MetaGeneration)
	ExpectEq(len("taco"), listing.Objects[0].Size)

	// We should see the old contents when we read.
	contents, err := t.readObject("foo")
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
}

func (t *CreateTest) MetaGenerationPrecondition_Satisfied() {
	// Create an existing object.
	orig, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		"foo",
		[]byte("taco"))

	// Request to create another version of the object, with a satisfied
	// precondition.
	req := &gcs.CreateObjectRequest{
		Name:                       "foo",
		Contents:                   strings.NewReader("burrito"),
		MetaGenerationPrecondition: &orig.MetaGeneration,
	}

	o, err := t.bucket.CreateObject(t.ctx, req)
	AssertEq(nil, err)

	ExpectEq(len("burrito"), o.Size)
	ExpectNe(orig.Generation, o.Generation)
	ExpectEq(1, o.MetaGeneration)

	// The new version should show up in a listing.
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)

	AssertEq(1, len(listing.Objects))
	AssertEq("foo", listing.Objects[0].Name)
	ExpectEq(o.Generation, listing.Objects[0].Generation)
	ExpectEq(o.MetaGeneration, listing.Objects[0].MetaGeneration)
	ExpectEq(len("burrito"), listing.Objects[0].Size)

	// We should see the new contents when we read.
	contents, err := t.readObject("foo")
	AssertEq(nil, err)
	ExpectEq("burrito", string(contents))
}
