package gcstests

import (
	"crypto/md5"
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
	"strings"
	"testing"
	"time"
)

func TestCopy(t *testing.T) { RunTests(t) }

type CopyTest struct {
	ctx    context.Context
	bucket gcs.Bucket
	clock  timeutil.Clock
}

var _ SetUpInterface = &CopyTest{}

func init() { RegisterTestSuite(&CopyTest{}) }

////////////////////////////////////////////////////////////////////////
// Initialization
////////////////////////////////////////////////////////////////////////

func (t *CopyTest) SetUp(ti *TestInfo) {
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

func (t *CopyTest) createObject(name string, contents string) error {
	_, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		name,
		[]byte(contents))

	return err
}

func (t *CopyTest) readObject(objectName string) (contents string, err error) {
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
func (t *CopyTest) advanceTime() {
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
func (t *CopyTest) matchesStartTime(start time.Time) Matcher {
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
// Copy Tests
////////////////////////////////////////////////////////////////////////

func (t *CopyTest) SourceDoesntExist() {
	var err error

	// Copy
	req := &gcs.CopyObjectRequest{
		SrcName: "foo",
		DstName: "bar",
	}

	_, err = t.bucket.CopyObject(t.ctx, req)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))

	// List
	objects, runs, err := gcsutil.ListAll(
		t.ctx,
		t.bucket,
		&gcs.ListObjectsRequest{})

	AssertEq(nil, err)
	ExpectThat(objects, ElementsAre())
	ExpectThat(runs, ElementsAre())
}

func (t *CopyTest) DestinationDoesntExist() {
	var err error

	// Create a source object with explicit attributes set.
	createTime := t.clock.Now()
	src, err := t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:            "foo",
			ContentType:     "text/plain",
			ContentLanguage: "fr",
			CacheControl:    "public",
			Metadata: map[string]string{
				"foo": "bar",
				"baz": "qux",
			},

			Contents: strings.NewReader("taco"),
		})

	AssertEq(nil, err)
	AssertThat(src.Updated, t.matchesStartTime(createTime))

	// Ensure the time below doesn't match exactly.
	t.advanceTime()

	// Copy to a destination object.
	req := &gcs.CopyObjectRequest{
		SrcName: "foo",
		DstName: "bar",
	}

	dst, err := t.bucket.CopyObject(t.ctx, req)

	AssertEq(nil, err)
	ExpectEq("bar", dst.Name)
	ExpectEq("text/plain", dst.ContentType)
	ExpectEq("fr", dst.ContentLanguage)
	ExpectEq("public", dst.CacheControl)
	ExpectThat(dst.Owner, MatchesRegexp("^user-.*"))
	ExpectEq(len("taco"), dst.Size)
	ExpectEq(1, dst.ComponentCount)
	ExpectThat(dst.MD5, Pointee(DeepEquals(md5.Sum([]byte("taco")))))
	ExpectEq(gcstests.ComputeCrc32C("taco"), *dst.CRC32C)
	ExpectThat(dst.MediaLink, MatchesRegexp("download/storage.*bar"))
	ExpectThat(dst.Metadata, DeepEquals(src.Metadata))
	ExpectLt(0, dst.Generation)
	ExpectEq(1, dst.MetaGeneration)
	ExpectEq("STANDARD", dst.StorageClass)
	ExpectThat(dst.Deleted, DeepEquals(time.Time{}))
	ExpectThat(dst.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(dst.Updated, t.matchesStartTime(createTime))

	// The object should be readable.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, "bar")

	AssertEq(nil, err)
	ExpectEq("taco", string(contents))

	// And stattable.
	statO, err := t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: "bar"})

	AssertEq(nil, err)
	ExpectThat(statO, Pointee(DeepEquals(*dst)))
}

func (t *CopyTest) DestinationExists() {
	var err error

	// Create a source object with explicit attributes set.
	createTime := t.clock.Now()
	src, err := t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:            "foo",
			ContentType:     "text/plain",
			ContentLanguage: "fr",
			CacheControl:    "public",
			Metadata: map[string]string{
				"foo": "bar",
				"baz": "qux",
			},

			Contents: strings.NewReader("taco"),
		})

	AssertEq(nil, err)
	AssertThat(src.Updated, t.matchesStartTime(createTime))

	// Ensure the time below doesn't match exactly.
	t.advanceTime()

	// Create an existing object with the destination name with other explicit
	// attributes set.
	orig, err := t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:            "bar",
			ContentType:     "application/octet-stream",
			ContentLanguage: "de",
			Metadata: map[string]string{
				"foo": "blah",
			},

			Contents: strings.NewReader("burrito"),
		})

	AssertEq(nil, err)

	// Copy over the existing object.
	req := &gcs.CopyObjectRequest{
		SrcName: "foo",
		DstName: "bar",
	}

	dst, err := t.bucket.CopyObject(t.ctx, req)

	AssertEq(nil, err)
	ExpectEq("bar", dst.Name)
	ExpectEq("text/plain", dst.ContentType)
	ExpectEq("fr", dst.ContentLanguage)
	ExpectEq("public", dst.CacheControl)
	ExpectThat(dst.Owner, MatchesRegexp("^user-.*"))
	ExpectEq(len("taco"), dst.Size)
	ExpectEq(1, dst.ComponentCount)
	ExpectThat(dst.MD5, Pointee(DeepEquals(md5.Sum([]byte("taco")))))
	ExpectEq(gcstests.ComputeCrc32C("taco"), *dst.CRC32C)
	ExpectThat(dst.MediaLink, MatchesRegexp("download/storage.*bar"))
	ExpectThat(dst.Metadata, DeepEquals(src.Metadata))
	ExpectLt(orig.Generation, dst.Generation)
	ExpectEq(1, dst.MetaGeneration)
	ExpectEq("STANDARD", dst.StorageClass)
	ExpectThat(dst.Deleted, DeepEquals(time.Time{}))
	ExpectThat(dst.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(dst.Updated, t.matchesStartTime(createTime))

	// The object should be readable.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, "bar")

	AssertEq(nil, err)
	ExpectEq("taco", string(contents))

	// And stattable.
	statO, err := t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: "bar"})

	AssertEq(nil, err)
	ExpectThat(statO, Pointee(DeepEquals(*dst)))
}

func (t *CopyTest) DestinationIsSameName() {
	var err error

	// Create a source object with explicit attributes set.
	createTime := t.clock.Now()
	src, err := t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:            "foo",
			ContentType:     "text/plain",
			ContentLanguage: "fr",
			CacheControl:    "public",
			Metadata: map[string]string{
				"foo": "bar",
				"baz": "qux",
			},

			Contents: strings.NewReader("taco"),
		})

	AssertEq(nil, err)
	AssertThat(src.Updated, t.matchesStartTime(createTime))

	// Ensure the time below doesn't match exactly.
	t.advanceTime()

	// Copy over itself.
	req := &gcs.CopyObjectRequest{
		SrcName: "foo",
		DstName: "foo",
	}

	dst, err := t.bucket.CopyObject(t.ctx, req)

	AssertEq(nil, err)
	ExpectEq("foo", dst.Name)
	ExpectEq("text/plain", dst.ContentType)
	ExpectEq("fr", dst.ContentLanguage)
	ExpectEq("public", dst.CacheControl)
	ExpectThat(dst.Owner, MatchesRegexp("^user-.*"))
	ExpectEq(len("taco"), dst.Size)
	ExpectEq(1, dst.ComponentCount)
	ExpectThat(dst.MD5, Pointee(DeepEquals(md5.Sum([]byte("taco")))))
	ExpectEq(gcstests.ComputeCrc32C("taco"), *dst.CRC32C)
	ExpectThat(dst.MediaLink, MatchesRegexp("download/storage.*foo"))
	ExpectThat(dst.Metadata, DeepEquals(src.Metadata))
	ExpectLt(src.Generation, dst.Generation)
	ExpectEq(1, dst.MetaGeneration)
	ExpectEq("STANDARD", dst.StorageClass)
	ExpectThat(dst.Deleted, DeepEquals(time.Time{}))
	ExpectThat(dst.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(dst.Updated, t.matchesStartTime(createTime))

	// The object should be readable.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, "foo")

	AssertEq(nil, err)
	ExpectEq("taco", string(contents))

	// And stattable.
	statO, err := t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: "foo"})

	AssertEq(nil, err)
	ExpectThat(statO, Pointee(DeepEquals(*dst)))
}

func (t *CopyTest) InterestingNames() {
	var err error

	// Create a source object.
	const srcName = "foo"
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, srcName, []byte{})
	AssertEq(nil, err)

	// Make sure we can use each interesting name as a copy destination.
	err = gcstests.ForEachString(
		t.ctx,
		gcstests.InterestingNames(),
		func(ctx context.Context, name string) (err error) {
			_, err = t.bucket.CopyObject(
				ctx,
				&gcs.CopyObjectRequest{
					SrcName: srcName,
					DstName: name,
				})

			if err != nil {
				err = fmt.Errorf("Failed to copy %q: %v", name, err)
				return
			}

			return
		})

	AssertEq(nil, err)
}

func (t *CopyTest) IllegalNames() {
	var err error

	// Create a source object.
	const srcName = "foo"
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, srcName, []byte{})
	AssertEq(nil, err)

	// Make sure we can't use any illegal name as a copy destination.
	err = gcstests.ForEachString(
		t.ctx,
		gcstests.IllegalNames(),
		func(ctx context.Context, name string) (err error) {
			_, err = t.bucket.CopyObject(
				ctx,
				&gcs.CopyObjectRequest{
					SrcName: srcName,
					DstName: name,
				})

			if err == nil {
				err = fmt.Errorf("Expected to not be able to copy to %q", name)
				return
			}

			if name == "" {
				if !strings.Contains(err.Error(), "Invalid") &&
					!strings.Contains(err.Error(), "Not Found") {
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
}

func (t *CopyTest) ParticularSourceGeneration_NameDoesntExist() {
	var err error

	// Copy
	req := &gcs.CopyObjectRequest{
		SrcName:       "foo",
		SrcGeneration: 17,
		DstName:       "bar",
	}

	_, err = t.bucket.CopyObject(t.ctx, req)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *CopyTest) ParticularSourceGeneration_GenerationDoesntExist() {
	var err error

	// Create a source object.
	src, err := t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:     "foo",
			Contents: strings.NewReader("taco"),
		})

	AssertEq(nil, err)

	// Send a copy request for the wrong generation number.
	req := &gcs.CopyObjectRequest{
		SrcName:       src.Name,
		SrcGeneration: src.Generation + 1,
		DstName:       "bar",
	}

	_, err = t.bucket.CopyObject(t.ctx, req)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *CopyTest) ParticularSourceGeneration_Exists() {
	var err error

	// Create a source object.
	src, err := t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:     "foo",
			Contents: strings.NewReader("taco"),
		})

	AssertEq(nil, err)

	// Send a copy request for the right generation number.
	req := &gcs.CopyObjectRequest{
		SrcName:       src.Name,
		SrcGeneration: src.Generation,
		DstName:       "bar",
	}

	_, err = t.bucket.CopyObject(t.ctx, req)
	ExpectEq(nil, err)
}

func (t *CopyTest) SrcMetaGenerationPrecondition_Unsatisfied() {
	var err error

	// Create a source object.
	src, err := t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:     "foo",
			Contents: strings.NewReader(""),
		})

	AssertEq(nil, err)

	// Attempt to copy, with a precondition.
	precond := src.MetaGeneration + 1
	req := &gcs.CopyObjectRequest{
		SrcName:                       "foo",
		DstName:                       "bar",
		SrcMetaGenerationPrecondition: &precond,
	}

	_, err = t.bucket.CopyObject(t.ctx, req)
	AssertThat(err, HasSameTypeAs(&gcs.PreconditionError{}))

	// The object should not have been created.
	_, err = t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: "bar"})

	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *CopyTest) SrcMetaGenerationPrecondition_Satisfied() {
	var err error

	// Create a source object.
	src, err := t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:     "foo",
			Contents: strings.NewReader(""),
		})

	AssertEq(nil, err)

	// Copy, with a precondition.
	req := &gcs.CopyObjectRequest{
		SrcName:                       "foo",
		DstName:                       "bar",
		SrcMetaGenerationPrecondition: &src.MetaGeneration,
	}

	_, err = t.bucket.CopyObject(t.ctx, req)
	AssertEq(nil, err)

	// The object should have been created.
	_, err = t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: "bar"})

	ExpectEq(nil, err)
}
