package gcstests

import (
	"fmt"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	"github.com/jacobsa/gcloud/gcs/gcstests"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/syncutil"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
	"io/ioutil"
	"strings"
	"testing"
	"time"
)

func TestCompose(t *testing.T) { RunTests(t) }

type ComposeTest struct {
	ctx    context.Context
	bucket gcs.Bucket
	clock  timeutil.Clock
}

var _ SetUpInterface = &ComposeTest{}

func init() { RegisterTestSuite(&ComposeTest{}) }

////////////////////////////////////////////////////////////////////////
// Initialization
////////////////////////////////////////////////////////////////////////

func (t *ComposeTest) SetUp(ti *TestInfo) {
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

func (t *ComposeTest) createObject(name string, contents string) error {
	_, err := gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		name,
		[]byte(contents))

	return err
}

func (t *ComposeTest) readObject(objectName string) (contents string, err error) {
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
func (t *ComposeTest) advanceTime() {
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
func (t *ComposeTest) matchesStartTime(start time.Time) Matcher {
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
// Compose Tests
////////////////////////////////////////////////////////////////////////

func (t *ComposeTest) createSources(
	contents []string) (objs []*gcs.Object, err error) {
	objs = make([]*gcs.Object, len(contents))

	// Write indices into a channel.
	indices := make(chan int, len(contents))
	for i := range contents {
		indices <- i
	}
	close(indices)

	// Run a bunch of workers.
	b := syncutil.NewBundle(t.ctx)

	const parallelism = 128
	for i := 0; i < parallelism; i++ {
		b.Add(func(ctx context.Context) (err error) {
			for i := range indices {
				// Create an object. Include some metadata; it should be ignored by
				// ComposeObjects.
				objs[i], err = t.bucket.CreateObject(
					ctx,
					&gcs.CreateObjectRequest{
						Name:            fmt.Sprint(i),
						Contents:        strings.NewReader(contents[i]),
						ContentType:     "application/json",
						ContentLanguage: "de",
						Metadata: map[string]string{
							"foo": "bar",
						},
					},
				)

				if err != nil {
					err = fmt.Errorf("CreateObject: %v", err)
					return
				}
			}

			return
		})
	}

	err = b.Join()
	return
}

func (t *ComposeTest) OneSimpleSource() {
	// Create source objects.
	sources, err := t.createSources([]string{
		"taco",
	})

	AssertEq(nil, err)

	// Compose them.
	t.advanceTime()
	composeTime := t.clock.Now()

	o, err := t.bucket.ComposeObjects(
		t.ctx,
		&gcs.ComposeObjectsRequest{
			DstName: "foo",
			Sources: []gcs.ComposeSource{
				gcs.ComposeSource{
					Name: sources[0].Name,
				},
			},
		})

	t.advanceTime()
	AssertEq(nil, err)

	// Check the result.
	ExpectEq("foo", o.Name)
	ExpectEq("", o.ContentType)
	ExpectEq("", o.ContentLanguage)
	ExpectEq("", o.CacheControl)
	// Disabled due to Google-internal bug 31476941.
	// ExpectThat(o.Owner, MatchesRegexp("^user-.*"))
	ExpectEq(len("taco"), o.Size)
	ExpectEq("", o.ContentEncoding)
	ExpectEq(1, o.ComponentCount)
	ExpectEq(nil, o.MD5)
	ExpectEq(gcstests.ComputeCrc32C("taco"), *o.CRC32C)
	ExpectThat(o.MediaLink, MatchesRegexp("download/storage.*foo"))
	ExpectEq(nil, o.Metadata)
	ExpectLt(sources[0].Generation, o.Generation)
	ExpectEq(1, o.MetaGeneration)
	ExpectEq("STANDARD", o.StorageClass)
	ExpectThat(o.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(o.Updated, t.matchesStartTime(composeTime))

	// Check contents.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, "foo")

	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
}

func (t *ComposeTest) TwoSimpleSources() {
	// Create source objects.
	sources, err := t.createSources([]string{
		"taco",
		"burrito",
	})

	AssertEq(nil, err)

	// Compose them.
	t.advanceTime()
	composeTime := t.clock.Now()

	o, err := t.bucket.ComposeObjects(
		t.ctx,
		&gcs.ComposeObjectsRequest{
			DstName: "foo",
			Sources: []gcs.ComposeSource{
				gcs.ComposeSource{
					Name: sources[0].Name,
				},

				gcs.ComposeSource{
					Name: sources[1].Name,
				},
			},
		})

	t.advanceTime()
	AssertEq(nil, err)

	// Check the result.
	ExpectEq("foo", o.Name)
	ExpectEq("", o.ContentType)
	ExpectEq("", o.ContentLanguage)
	ExpectEq("", o.CacheControl)
	// Disabled due to Google-internal bug 31476941.
	// ExpectThat(o.Owner, MatchesRegexp("^user-.*"))
	ExpectEq(len("tacoburrito"), o.Size)
	ExpectEq("", o.ContentEncoding)
	ExpectEq(2, o.ComponentCount)
	ExpectEq(nil, o.MD5)
	ExpectEq(gcstests.ComputeCrc32C("tacoburrito"), *o.CRC32C)
	ExpectThat(o.MediaLink, MatchesRegexp("download/storage.*foo"))
	ExpectEq(nil, o.Metadata)
	ExpectLt(sources[0].Generation, o.Generation)
	ExpectLt(sources[1].Generation, o.Generation)
	ExpectEq(1, o.MetaGeneration)
	ExpectEq("STANDARD", o.StorageClass)
	ExpectThat(o.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(o.Updated, t.matchesStartTime(composeTime))

	// Check contents.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, "foo")

	AssertEq(nil, err)
	ExpectEq("tacoburrito", string(contents))
}

func (t *ComposeTest) ManySimpleSources() {
	// Create source objects.
	sources, err := t.createSources([]string{
		"taco",
		"",
		"burrito",
		"enchilada",
		"queso",
		"",
	})

	AssertEq(nil, err)

	// Compose them.
	req := &gcs.ComposeObjectsRequest{
		DstName: "foo",
	}

	for _, src := range sources {
		req.Sources = append(req.Sources, gcs.ComposeSource{Name: src.Name})
	}

	t.advanceTime()
	composeTime := t.clock.Now()

	o, err := t.bucket.ComposeObjects(t.ctx, req)

	t.advanceTime()
	AssertEq(nil, err)

	// Check the result.
	ExpectEq("foo", o.Name)
	ExpectEq("", o.ContentType)
	ExpectEq("", o.ContentLanguage)
	ExpectEq("", o.CacheControl)
	// Disabled due to Google-internal bug 31476941.
	// ExpectThat(o.Owner, MatchesRegexp("^user-.*"))
	ExpectEq(len("tacoburritoenchiladaqueso"), o.Size)
	ExpectEq("", o.ContentEncoding)
	ExpectEq(6, o.ComponentCount)
	ExpectEq(nil, o.MD5)
	ExpectEq(gcstests.ComputeCrc32C("tacoburritoenchiladaqueso"), *o.CRC32C)
	ExpectThat(o.MediaLink, MatchesRegexp("download/storage.*foo"))
	ExpectEq(nil, o.Metadata)
	ExpectEq(1, o.MetaGeneration)
	ExpectEq("STANDARD", o.StorageClass)
	ExpectThat(o.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(o.Updated, t.matchesStartTime(composeTime))

	for _, src := range sources {
		ExpectLt(src.Generation, o.Generation)
	}

	// Check contents.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, "foo")

	AssertEq(nil, err)
	ExpectEq("tacoburritoenchiladaqueso", string(contents))
}

func (t *ComposeTest) RepeatedSources() {
	// Create source objects.
	sources, err := t.createSources([]string{
		"taco",
		"burrito",
	})

	AssertEq(nil, err)

	// Compose them, using each multiple times.
	t.advanceTime()
	composeTime := t.clock.Now()

	o, err := t.bucket.ComposeObjects(
		t.ctx,
		&gcs.ComposeObjectsRequest{
			DstName: "foo",
			Sources: []gcs.ComposeSource{
				gcs.ComposeSource{
					Name: sources[0].Name,
				},

				gcs.ComposeSource{
					Name: sources[1].Name,
				},

				gcs.ComposeSource{
					Name: sources[0].Name,
				},

				gcs.ComposeSource{
					Name: sources[1].Name,
				},
			},
		})

	t.advanceTime()
	AssertEq(nil, err)

	// Check the result.
	ExpectEq("foo", o.Name)
	ExpectEq("", o.ContentType)
	ExpectEq("", o.ContentLanguage)
	ExpectEq("", o.CacheControl)
	// Disabled due to Google-internal bug 31476941.
	// ExpectThat(o.Owner, MatchesRegexp("^user-.*"))
	ExpectEq(len("tacoburritotacoburrito"), o.Size)
	ExpectEq("", o.ContentEncoding)
	ExpectEq(4, o.ComponentCount)
	ExpectEq(nil, o.MD5)
	ExpectEq(gcstests.ComputeCrc32C("tacoburritotacoburrito"), *o.CRC32C)
	ExpectThat(o.MediaLink, MatchesRegexp("download/storage.*foo"))
	ExpectEq(nil, o.Metadata)
	ExpectLt(sources[0].Generation, o.Generation)
	ExpectLt(sources[1].Generation, o.Generation)
	ExpectEq(1, o.MetaGeneration)
	ExpectEq("STANDARD", o.StorageClass)
	ExpectThat(o.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(o.Updated, t.matchesStartTime(composeTime))

	// Check contents.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, "foo")

	AssertEq(nil, err)
	ExpectEq("tacoburritotacoburrito", string(contents))
}

func (t *ComposeTest) CompositeSources() {
	// Create two source objects.
	sources, err := t.createSources([]string{
		"taco",
		"burrito",
	})

	AssertEq(nil, err)

	// Compose them to form another source object.
	sources = append(sources, nil)
	sources[2], err = t.bucket.ComposeObjects(
		t.ctx,
		&gcs.ComposeObjectsRequest{
			DstName: "2",
			Sources: []gcs.ComposeSource{
				gcs.ComposeSource{
					Name: sources[0].Name,
				},

				gcs.ComposeSource{
					Name: sources[1].Name,
				},
			},
		})

	// Now compose that a couple of times along with one of the originals.
	t.advanceTime()
	composeTime := t.clock.Now()

	o, err := t.bucket.ComposeObjects(
		t.ctx,
		&gcs.ComposeObjectsRequest{
			DstName: "foo",
			Sources: []gcs.ComposeSource{
				gcs.ComposeSource{
					Name: sources[2].Name,
				},

				gcs.ComposeSource{
					Name: sources[0].Name,
				},

				gcs.ComposeSource{
					Name: sources[2].Name,
				},
			},
		})

	t.advanceTime()
	AssertEq(nil, err)

	// Check the result.
	ExpectEq("foo", o.Name)
	ExpectEq("", o.ContentType)
	ExpectEq("", o.ContentLanguage)
	ExpectEq("", o.CacheControl)
	// Disabled due to Google-internal bug 31476941.
	// ExpectThat(o.Owner, MatchesRegexp("^user-.*"))
	ExpectEq(len("tacoburritotacotacoburrito"), o.Size)
	ExpectEq("", o.ContentEncoding)
	ExpectEq(5, o.ComponentCount)
	ExpectEq(nil, o.MD5)
	ExpectEq(gcstests.ComputeCrc32C("tacoburritotacotacoburrito"), *o.CRC32C)
	ExpectThat(o.MediaLink, MatchesRegexp("download/storage.*foo"))
	ExpectEq(nil, o.Metadata)
	ExpectLt(sources[0].Generation, o.Generation)
	ExpectLt(sources[2].Generation, o.Generation)
	ExpectEq(1, o.MetaGeneration)
	ExpectEq("STANDARD", o.StorageClass)
	ExpectThat(o.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(o.Updated, t.matchesStartTime(composeTime))

	// Check contents.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, "foo")

	AssertEq(nil, err)
	ExpectEq("tacoburritotacotacoburrito", string(contents))
}

func (t *ComposeTest) Metadata() {
	// Create source objects.
	sources, err := t.createSources([]string{
		"taco",
		"burrito",
	})

	AssertEq(nil, err)

	// Compose them, including metadata.
	o, err := t.bucket.ComposeObjects(
		t.ctx,
		&gcs.ComposeObjectsRequest{
			DstName: "foo",
			Sources: []gcs.ComposeSource{
				gcs.ComposeSource{
					Name: sources[0].Name,
				},

				gcs.ComposeSource{
					Name: sources[1].Name,
				},
			},
			ContentType: "image/jpeg",
			Metadata: map[string]string{
				"key0": "val0",
				"key1": "val1",
			},
		})

	t.advanceTime()
	AssertEq(nil, err)

	// Check the result.
	AssertEq("foo", o.Name)
	ExpectEq(1, o.MetaGeneration)

	ExpectEq("image/jpeg", o.ContentType)
	ExpectEq(2, len(o.Metadata))
	ExpectEq("val0", o.Metadata["key0"])
	ExpectEq("val1", o.Metadata["key1"])
}

func (t *ComposeTest) DestinationNameMatchesSource() {
	// Create source objects.
	sources, err := t.createSources([]string{
		"taco",
		"burrito",
	})

	AssertEq(nil, err)

	// Compose on top of the first's name.
	o, err := t.bucket.ComposeObjects(
		t.ctx,
		&gcs.ComposeObjectsRequest{
			DstName: sources[0].Name,
			Sources: []gcs.ComposeSource{
				gcs.ComposeSource{
					Name: sources[0].Name,
				},

				gcs.ComposeSource{
					Name: sources[1].Name,
				},
			},
		})

	AssertEq(nil, err)

	// Check the result.
	ExpectEq(sources[0].Name, o.Name)
	ExpectEq(len("tacoburrito"), o.Size)
	ExpectEq(2, o.ComponentCount)
	ExpectLt(sources[0].Generation, o.Generation)
	ExpectLt(sources[1].Generation, o.Generation)

	// Check contents.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, sources[0].Name)

	AssertEq(nil, err)
	ExpectEq("tacoburrito", string(contents))
}

func (t *ComposeTest) OneSourceDoesntExist() {
	// Create source objects.
	sources, err := t.createSources([]string{
		"taco",
		"burrito",
	})

	AssertEq(nil, err)

	// Attempt to compose them with a name that doesn't exist.
	_, err = t.bucket.ComposeObjects(
		t.ctx,
		&gcs.ComposeObjectsRequest{
			DstName: "foo",
			Sources: []gcs.ComposeSource{
				gcs.ComposeSource{
					Name: sources[0].Name,
				},

				gcs.ComposeSource{
					Name: "blah",
				},

				gcs.ComposeSource{
					Name: sources[1].Name,
				},
			},
		})

	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))

	// Make sure the destination object doesn't exist.
	_, err = t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: "foo"})

	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *ComposeTest) ExplicitGenerations_Exist() {
	// Create source objects.
	sources, err := t.createSources([]string{
		"taco",
		"burrito",
	})

	AssertEq(nil, err)

	// Compose them.
	o, err := t.bucket.ComposeObjects(
		t.ctx,
		&gcs.ComposeObjectsRequest{
			DstName: "foo",
			Sources: []gcs.ComposeSource{
				gcs.ComposeSource{
					Name:       sources[0].Name,
					Generation: sources[0].Generation,
				},

				gcs.ComposeSource{
					Name:       sources[1].Name,
					Generation: sources[1].Generation,
				},
			},
		})

	AssertEq(nil, err)

	// Check the result.
	ExpectEq("foo", o.Name)
	ExpectEq(len("tacoburrito"), o.Size)
}

func (t *ComposeTest) ExplicitGenerations_OneDoesntExist() {
	// Create source objects.
	sources, err := t.createSources([]string{
		"taco",
		"burrito",
		"enchilada",
	})

	AssertEq(nil, err)

	// Attempt to compose them, with the wrong generation for one of them.
	_, err = t.bucket.ComposeObjects(
		t.ctx,
		&gcs.ComposeObjectsRequest{
			DstName: "foo",
			Sources: []gcs.ComposeSource{
				gcs.ComposeSource{
					Name:       sources[0].Name,
					Generation: sources[0].Generation,
				},

				gcs.ComposeSource{
					Name:       sources[1].Name,
					Generation: sources[1].Generation + 1,
				},

				gcs.ComposeSource{
					Name:       sources[2].Name,
					Generation: sources[2].Generation,
				},
			},
		})

	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))

	// Make sure the destination object doesn't exist.
	_, err = t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: "foo"})

	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *ComposeTest) DestinationExists_NoPreconditions() {
	// Create source objects.
	sources, err := t.createSources([]string{
		"taco",
		"burrito",
	})

	AssertEq(nil, err)

	// Attempt to compose them on top of the first.
	o, err := t.bucket.ComposeObjects(
		t.ctx,
		&gcs.ComposeObjectsRequest{
			DstName: sources[0].Name,
			Sources: []gcs.ComposeSource{
				gcs.ComposeSource{
					Name: sources[0].Name,
				},

				gcs.ComposeSource{
					Name: sources[1].Name,
				},
			},
		})

	AssertEq(nil, err)

	// Check the result.
	ExpectEq(sources[0].Name, o.Name)
	ExpectEq(len("tacoburrito"), o.Size)
	ExpectEq(2, o.ComponentCount)
	ExpectLt(sources[0].Generation, o.Generation)
	ExpectLt(sources[1].Generation, o.Generation)

	// Check contents.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, sources[0].Name)

	AssertEq(nil, err)
	ExpectEq("tacoburrito", string(contents))
}

func (t *ComposeTest) DestinationExists_GenerationPreconditionNotSatisfied() {
	// Create source objects.
	sources, err := t.createSources([]string{
		"taco",
		"burrito",
	})

	AssertEq(nil, err)

	// Attempt to compose them on top of the first.
	precond := sources[0].Generation + 1
	_, err = t.bucket.ComposeObjects(
		t.ctx,
		&gcs.ComposeObjectsRequest{
			DstName:                   sources[0].Name,
			DstGenerationPrecondition: &precond,

			Sources: []gcs.ComposeSource{
				gcs.ComposeSource{
					Name: sources[0].Name,
				},

				gcs.ComposeSource{
					Name: sources[1].Name,
				},
			},
		})

	ExpectThat(err, HasSameTypeAs(&gcs.PreconditionError{}))

	// Make sure the object wasn't overwritten.
	o, err := t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: sources[0].Name})

	AssertEq(nil, err)
	ExpectEq(sources[0].Generation, o.Generation)
	ExpectEq(len("taco"), o.Size)
}

func (t *ComposeTest) DestinationExists_MetaGenerationPreconditionNotSatisfied() {
	// Create source objects.
	sources, err := t.createSources([]string{
		"taco",
		"burrito",
	})

	AssertEq(nil, err)

	// Attempt to compose them on top of the first.
	precond := sources[0].MetaGeneration + 1
	_, err = t.bucket.ComposeObjects(
		t.ctx,
		&gcs.ComposeObjectsRequest{
			DstName:                       sources[0].Name,
			DstMetaGenerationPrecondition: &precond,

			Sources: []gcs.ComposeSource{
				gcs.ComposeSource{
					Name: sources[0].Name,
				},

				gcs.ComposeSource{
					Name: sources[1].Name,
				},
			},
		})

	ExpectThat(err, HasSameTypeAs(&gcs.PreconditionError{}))

	// Make sure the object wasn't overwritten.
	o, err := t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: sources[0].Name})

	AssertEq(nil, err)
	ExpectEq(sources[0].Generation, o.Generation)
	ExpectEq(sources[0].MetaGeneration, o.MetaGeneration)
	ExpectEq(len("taco"), o.Size)
}

func (t *ComposeTest) DestinationExists_PreconditionsSatisfied() {
	// Create source objects.
	sources, err := t.createSources([]string{
		"taco",
		"burrito",
	})

	AssertEq(nil, err)

	// Attempt to compose them on top of the first.
	o, err := t.bucket.ComposeObjects(
		t.ctx,
		&gcs.ComposeObjectsRequest{
			DstName:                       sources[0].Name,
			DstGenerationPrecondition:     &sources[0].Generation,
			DstMetaGenerationPrecondition: &sources[0].MetaGeneration,

			Sources: []gcs.ComposeSource{
				gcs.ComposeSource{
					Name: sources[0].Name,
				},

				gcs.ComposeSource{
					Name: sources[1].Name,
				},
			},
		})

	AssertEq(nil, err)

	// Check the result.
	ExpectEq(sources[0].Name, o.Name)
	ExpectEq(len("tacoburrito"), o.Size)
	ExpectEq(2, o.ComponentCount)
	ExpectLt(sources[0].Generation, o.Generation)
	ExpectLt(sources[1].Generation, o.Generation)

	// Check contents.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, sources[0].Name)

	AssertEq(nil, err)
	ExpectEq("tacoburrito", string(contents))
}

func (t *ComposeTest) DestinationDoesntExist_PreconditionNotSatisfied() {
	// Create source objects.
	sources, err := t.createSources([]string{
		"taco",
		"burrito",
	})

	AssertEq(nil, err)

	// Attempt to compose them.
	var precond int64 = 1
	_, err = t.bucket.ComposeObjects(
		t.ctx,
		&gcs.ComposeObjectsRequest{
			DstName:                   "foo",
			DstGenerationPrecondition: &precond,

			Sources: []gcs.ComposeSource{
				gcs.ComposeSource{
					Name: sources[0].Name,
				},

				gcs.ComposeSource{
					Name: sources[1].Name,
				},
			},
		})

	ExpectThat(err, HasSameTypeAs(&gcs.PreconditionError{}))

	// Make sure the destination object doesn't exist.
	_, err = t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: "foo"})

	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *ComposeTest) DestinationDoesntExist_PreconditionSatisfied() {
	// Create source objects.
	sources, err := t.createSources([]string{
		"taco",
		"burrito",
	})

	AssertEq(nil, err)

	// Attempt to compose them.
	var precond int64 = 0
	o, err := t.bucket.ComposeObjects(
		t.ctx,
		&gcs.ComposeObjectsRequest{
			DstName:                   "foo",
			DstGenerationPrecondition: &precond,

			Sources: []gcs.ComposeSource{
				gcs.ComposeSource{
					Name: sources[0].Name,
				},

				gcs.ComposeSource{
					Name: sources[1].Name,
				},
			},
		})

	AssertEq(nil, err)

	// Check the result.
	ExpectEq("foo", o.Name)
	ExpectEq(len("tacoburrito"), o.Size)
	ExpectEq(2, o.ComponentCount)
	ExpectLt(sources[0].Generation, o.Generation)
	ExpectLt(sources[1].Generation, o.Generation)

	// Check contents.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, "foo")

	AssertEq(nil, err)
	ExpectEq("tacoburrito", string(contents))
}

func (t *ComposeTest) ZeroSources() {
	// GCS doesn't like zero-source requests (and so neither should our fake).
	req := &gcs.ComposeObjectsRequest{
		DstName: "foo",
	}

	_, err := t.bucket.ComposeObjects(t.ctx, req)
	ExpectThat(err, Error(HasSubstr("at least one")))
}

func (t *ComposeTest) TooManySources() {
	// Create an original object.
	src, err := t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:     "src",
			Contents: strings.NewReader(""),
		})

	AssertEq(nil, err)

	// Attempt to compose too many copies of it.
	req := &gcs.ComposeObjectsRequest{
		DstName: "foo",
	}

	for i := 0; i < gcs.MaxSourcesPerComposeRequest+1; i++ {
		req.Sources = append(req.Sources, gcs.ComposeSource{Name: src.Name})
	}

	_, err = t.bucket.ComposeObjects(t.ctx, req)

	ExpectThat(err, Error(HasSubstr("source components")))
}

func (t *ComposeTest) ComponentCountLimits() {
	// The tests below assume that we can hit the max component count with two
	// rounds of composing.
	AssertEq(
		gcs.MaxComponentCount,
		gcs.MaxSourcesPerComposeRequest*gcs.MaxSourcesPerComposeRequest)

	// Create a single original object.
	small, err := t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:     "small",
			Contents: strings.NewReader("a"),
		})

	AssertEq(nil, err)

	// Compose as many copies of it as possible.
	req := &gcs.ComposeObjectsRequest{
		DstName: "medium",
	}

	for i := 0; i < gcs.MaxSourcesPerComposeRequest; i++ {
		req.Sources = append(req.Sources, gcs.ComposeSource{Name: small.Name})
	}

	medium, err := t.bucket.ComposeObjects(t.ctx, req)

	AssertEq(nil, err)
	AssertEq(gcs.MaxSourcesPerComposeRequest, medium.ComponentCount)
	AssertEq(gcs.MaxSourcesPerComposeRequest, medium.Size)

	// Compose that many copies over again to hit the maximum component count
	// limit.
	req = &gcs.ComposeObjectsRequest{
		DstName: "large",
	}

	for i := 0; i < gcs.MaxSourcesPerComposeRequest; i++ {
		req.Sources = append(req.Sources, gcs.ComposeSource{Name: medium.Name})
	}

	large, err := t.bucket.ComposeObjects(t.ctx, req)

	AssertEq(nil, err)
	AssertEq(gcs.MaxComponentCount, large.ComponentCount)
	AssertEq(gcs.MaxComponentCount, large.Size)

	// Attempting to add one more component should fail.
	req = &gcs.ComposeObjectsRequest{
		DstName: "foo",
		Sources: []gcs.ComposeSource{
			gcs.ComposeSource{Name: large.Name},
			gcs.ComposeSource{Name: small.Name},
		},
	}

	_, err = t.bucket.ComposeObjects(t.ctx, req)

	ExpectThat(err, Error(HasSubstr("too many components")))
}

func (t *ComposeTest) InterestingNames() {
	var err error

	// Create a source object.
	const srcName = "foo"
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, srcName, []byte{})
	AssertEq(nil, err)

	// Make sure we can use each interesting name as a compose destination.
	err = gcstests.ForEachString(
		t.ctx,
		gcstests.InterestingNames(),
		func(ctx context.Context, name string) (err error) {
			_, err = t.bucket.ComposeObjects(
				ctx,
				&gcs.ComposeObjectsRequest{
					DstName: name,
					Sources: []gcs.ComposeSource{
						gcs.ComposeSource{Name: srcName},
						gcs.ComposeSource{Name: srcName},
					},
				})

			if err != nil {
				err = fmt.Errorf("Failed to compose to %q: %v", name, err)
				return
			}

			return
		})

	AssertEq(nil, err)
}

func (t *ComposeTest) IllegalNames() {
	var err error

	// Create a source object.
	const srcName = "foo"
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, srcName, []byte{})
	AssertEq(nil, err)

	// Make sure we can't use any illegal name as a compose destination.
	err = gcstests.ForEachString(
		t.ctx,
		gcstests.IllegalNames(),
		func(ctx context.Context, name string) (err error) {
			_, err = t.bucket.ComposeObjects(
				ctx,
				&gcs.ComposeObjectsRequest{
					DstName: name,
					Sources: []gcs.ComposeSource{
						gcs.ComposeSource{Name: srcName},
						gcs.ComposeSource{Name: srcName},
					},
				})

			if err == nil {
				err = fmt.Errorf("Expected to not be able to compose to %q", name)
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
