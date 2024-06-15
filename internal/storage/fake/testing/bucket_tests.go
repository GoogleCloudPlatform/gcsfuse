// Copyright 2023 Google Inc. All Rights Reserved.
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

// Tests registered by RegisterBucketTests.

package testing

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"math"
	"sort"
	"strings"
	"testing/iotest"
	"time"
	"unicode"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/syncutil"
	"github.com/jacobsa/timeutil"
	"golang.org/x/sys/unix"
)

////////////////////////////////////////////////////////////////////////
// Initialization
////////////////////////////////////////////////////////////////////////

// Make sure we can use a decent degree of parallelism when talking to GCS
// without getting "too many open files" errors, especially on OS X where the
// default rlimit is very low (256 as of 10.10.3).
func init() {
	var rlim unix.Rlimit
	var err error

	err = unix.Getrlimit(unix.RLIMIT_NOFILE, &rlim)
	if err != nil {
		panic(err)
	}

	before := rlim.Cur
	rlim.Cur = rlim.Max

	err = unix.Setrlimit(unix.RLIMIT_NOFILE, &rlim)
	if err != nil {
		panic(err)
	}

	log.Printf("Raised RLIMIT_NOFILE from %d to %d.", before, rlim.Cur)
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func createEmpty(
	ctx context.Context,
	bucket gcs.Bucket,
	objectNames []string) error {
	err := storageutil.CreateEmptyObjects(ctx, bucket, objectNames)
	return err
}

func computeCrc32C(s string) uint32 {
	return crc32.Checksum([]byte(s), crc32.MakeTable(crc32.Castagnoli))
}

func makeStringPtr(s string) *string {
	return &s
}

// Return a list of object names that might be problematic for GCS or the Go
// client but are nevertheless documented to be legal.
//
// Useful links:
//
//	https://cloud.google.com/storage/docs/bucket-naming
//	http://www.unicode.org/Public/7.0.0/ucd/UnicodeData.txt
//	http://www.unicode.org/versions/Unicode7.0.0/ch02.pdf (Table 2-3)
func interestingNames() (names []string) {
	const maxLegalLength = 1024

	names = []string{
		// Characters specifically mentioned by RFC 3986, i.e. that might be
		// important in URL encoding/decoding.
		"foo : bar",
		"foo / bar",
		"foo ? bar",
		"foo # bar",
		"foo [ bar",
		"foo ] bar",
		"foo @ bar",
		"foo ! bar",
		"foo $ bar",
		"foo & bar",
		"foo ' bar",
		"foo ( bar",
		"foo ) bar",
		"foo * bar",
		"foo + bar",
		"foo , bar",
		"foo ; bar",
		"foo = bar",
		"foo - bar",
		"foo . bar",
		"foo _ bar",
		"foo ~ bar",

		// Other tricky URL cases.
		"foo () bar",
		"foo [] bar",
		"foo // bar",
		"foo %?/ bar",
		"foo http://google.com/search?q=foo&bar=baz#qux bar",

		"foo ?bar",
		"foo? bar",
		"foo/ bar",
		"foo /bar",

		// Non-Roman scripts
		"타코",
		"世界",

		// Longest legal name
		strings.Repeat("a", maxLegalLength),

		// Null byte.
		"foo \u0000 bar",

		// Non-control characters that are discouraged, but not forbidden,
		// according to the documentation.
		"foo # bar",
		"foo []*? bar",

		// Angstrom symbol singleton and normalized forms.
		// Cf. http://unicode.org/reports/tr15/
		"foo \u212b bar",
		"foo \u0041\u030a bar",
		"foo \u00c5 bar",

		// Hangul separating jamo
		// Cf. http://www.unicode.org/versions/Unicode7.0.0/ch18.pdf (Table 18-10)
		"foo \u3131\u314f bar",
		"foo \u1100\u1161 bar",
		"foo \uac00 bar",

		// Unicode specials
		// Cf. http://en.wikipedia.org/wiki/Specials_%28Unicode_block%29
		"foo \ufff9 bar",
		"foo \ufffa bar",
		"foo \ufffb bar",
		"foo \ufffc bar",
		"foo \ufffd bar",
	}

	// All codepoints in Unicode general categories C* (control and special) and
	// Z* (space), except for:
	//
	//  *  Cn (non-character and reserved), which is not included in unicode.C.
	//  *  Co (private usage), which is large.
	//  *  Cs (surrages), which is large.
	//  *  U+000A and U+000D, which are forbidden by the docs.
	//
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if !unicode.In(r, unicode.C) && !unicode.In(r, unicode.Z) {
			continue
		}

		if unicode.In(r, unicode.Co) {
			continue
		}

		if unicode.In(r, unicode.Cs) {
			continue
		}

		if r == 0x0a || r == 0x0d {
			continue
		}

		names = append(names, fmt.Sprintf("foo %s bar", string(r)))
	}

	return
}

// Return a list of object names that are illegal in GCS.
// Cf. https://cloud.google.com/storage/docs/bucket-naming
func illegalNames() (names []string) {
	const maxLegalLength = 1024
	names = []string{
		// Empty and too long
		"",
		strings.Repeat("a", maxLegalLength+1),

		// Not valid UTF-8
		"foo\xff",

		// Carriage return and line feed
		"foo\u000abar",
		"foo\u000dbar",
	}

	return
}

// Given lists of strings A and B, return those values that are in A but not in
// B. If A contains duplicates of a value V not in B, the only guarantee is
// that V is returned at least once.
func listDifference(a []string, b []string) (res []string) {
	// This is slow, but more obviously correct than the fast algorithm.
	m := make(map[string]struct{})
	for _, s := range b {
		m[s] = struct{}{}
	}

	for _, s := range a {
		if _, ok := m[s]; !ok {
			res = append(res, s)
		}
	}

	return
}

// Issue all of the supplied read requests with some degree of parallelism.
func readMultiple(
	ctx context.Context,
	bucket gcs.Bucket,
	reqs []*gcs.ReadObjectRequest) (contents [][]byte, errs []error) {
	b := syncutil.NewBundle(ctx)

	// Feed indices into a channel.
	indices := make(chan int, len(reqs))
	for i := range reqs {
		indices <- i
	}
	close(indices)

	// Set up a function that deals with one request.
	contents = make([][]byte, len(reqs))
	errs = make([]error, len(reqs))

	handleRequest := func(ctx context.Context, i int) {
		var b []byte
		var err error
		defer func() {
			contents[i] = b
			errs[i] = err
		}()

		// Open a reader.
		rc, err := bucket.NewReader(ctx, reqs[i])
		if err != nil {
			err = fmt.Errorf("NewReader: %v", err)
			return
		}

		// Read from it.
		b, err = io.ReadAll(rc)
		if err != nil {
			err = fmt.Errorf("ReadAll: %v", err)
			return
		}

		// Close it.
		err = rc.Close()
		if err != nil {
			err = fmt.Errorf("Close: %v", err)
			return
		}
	}

	// Run several workers.
	const parallelsim = 32
	for i := 0; i < parallelsim; i++ {
		b.Add(func(ctx context.Context) (err error) {
			for i := range indices {
				handleRequest(ctx, i)
			}

			return
		})
	}

	AssertEq(nil, b.Join())
	return
}

// Invoke the supplied function for each string, with some degree of
// parallelism.
func forEachString(
	ctx context.Context,
	strings []string,
	f func(context.Context, string) error) (err error) {
	b := syncutil.NewBundle(ctx)

	// Feed strings into a channel.
	c := make(chan string, len(strings))
	for _, s := range strings {
		c <- s
	}
	close(c)

	// Consume the strings.
	const parallelism = 128
	for i := 0; i < parallelism; i++ {
		b.Add(func(ctx context.Context) (err error) {
			for s := range c {
				err = f(ctx, s)
				if err != nil {
					return
				}
			}
			return
		})
	}

	err = b.Join()
	return
}

////////////////////////////////////////////////////////////////////////
// Common
////////////////////////////////////////////////////////////////////////

type bucketTest struct {
	ctx                            context.Context
	bucket                         gcs.Bucket
	clock                          timeutil.Clock
	supportsCancellation           bool
	buffersEntireContentsForCreate bool
}

var _ bucketTestSetUpInterface = &bucketTest{}

func (t *bucketTest) setUpBucketTest(deps BucketTestDeps) {
	t.ctx = deps.ctx
	t.bucket = deps.Bucket
	t.clock = deps.Clock
	t.supportsCancellation = deps.SupportsCancellation
	t.buffersEntireContentsForCreate = deps.BuffersEntireContentsForCreate
}

func (t *bucketTest) createObject(name string, contents string) error {
	_, err := storageutil.CreateObject(
		t.ctx,
		t.bucket,
		name,
		[]byte(contents))

	return err
}

func (t *bucketTest) readObject(objectName string) (contents string, err error) {
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
	slice, err := io.ReadAll(reader)
	if err != nil {
		return
	}

	// Transform to a string.
	contents = string(slice)

	return
}

// Ensure that the clock will report a different time after returning.
func (t *bucketTest) advanceTime() {
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
func (t *bucketTest) matchesStartTime(start time.Time) Matcher {
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
// Create
////////////////////////////////////////////////////////////////////////

type createTest struct {
	bucketTest
}

func (t *createTest) EmptyObject() {
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

func (t *createTest) NonEmptyObject() {
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

func (t *createTest) Overwrite() {
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

func (t *createTest) ObjectAttributes_Default() {
	// Create an object with default attributes aside from the name.
	createTime := t.clock.Now()
	o, err := storageutil.CreateObject(t.ctx, t.bucket, "foo", []byte("taco"))
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
	ExpectEq(computeCrc32C("taco"), *o.CRC32C)
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

func (t *createTest) ObjectAttributes_Explicit() {
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
	ExpectEq(computeCrc32C("taco"), *o.CRC32C)
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

func (t *createTest) ErrorAfterPartialContents() {
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

func (t *createTest) InterestingNames() {
	var err error

	// Grab a list of interesting legal names.
	names := interestingNames()

	// Make sure we can create each name.
	err = forEachString(
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
	err = forEachString(
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
	if diff := listDifference(listingNames, names); len(diff) != 0 {
		var dumps []string
		for _, n := range diff {
			dumps = append(dumps, hex.Dump([]byte(n)))
		}

		AddFailure(
			"Unexpected names in listing:\n%s",
			strings.Join(dumps, "\n"))
	}

	if diff := listDifference(names, listingNames); len(diff) != 0 {
		var dumps []string
		for _, n := range diff {
			dumps = append(dumps, hex.Dump([]byte(n)))
		}

		AddFailure(
			"Names missing from listing:\n%s",
			strings.Join(dumps, "\n"))
	}
}

func (t *createTest) IllegalNames() {
	var err error

	// Make sure we cannot create any of the names above.
	err = forEachString(
		t.ctx,
		illegalNames(),
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

func (t *createTest) IncorrectCRC32C() {
	const name = "foo"
	const contents = "taco"
	var err error

	// Attempt to create with the wrong checksum.
	crc32c := storageutil.CRC32C([]byte(contents))
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

	_, _, err = t.bucket.StatObject(t.ctx, statReq)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *createTest) CorrectCRC32C() {
	const name = "foo"
	const contents = "taco"
	var err error

	// Create
	req := &gcs.CreateObjectRequest{
		Name:     name,
		Contents: strings.NewReader(contents),
		CRC32C:   storageutil.CRC32C([]byte(contents)),
	}

	o, err := t.bucket.CreateObject(t.ctx, req)
	AssertEq(nil, err)
	ExpectEq(len(contents), o.Size)
}

func (t *createTest) IncorrectMD5() {
	const name = "foo"
	const contents = "taco"
	var err error

	// Attempt to create with the wrong checksum.
	md5 := storageutil.MD5([]byte(contents))
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

	_, _, err = t.bucket.StatObject(t.ctx, statReq)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *createTest) CorrectMD5() {
	const name = "foo"
	const contents = "taco"
	var err error

	// Create
	req := &gcs.CreateObjectRequest{
		Name:     name,
		Contents: strings.NewReader(contents),
		MD5:      storageutil.MD5([]byte(contents)),
	}

	o, err := t.bucket.CreateObject(t.ctx, req)
	AssertEq(nil, err)
	ExpectEq(len(contents), o.Size)
}

func (t *createTest) CorrectCRC32CAndMD5() {
	const name = "foo"
	const contents = "taco"
	var err error

	// Create
	req := &gcs.CreateObjectRequest{
		Name:     name,
		Contents: strings.NewReader(contents),
		CRC32C:   storageutil.CRC32C([]byte(contents)),
		MD5:      storageutil.MD5([]byte(contents)),
	}

	o, err := t.bucket.CreateObject(t.ctx, req)
	AssertEq(nil, err)
	ExpectEq(len(contents), o.Size)
}

func (t *createTest) GenerationPrecondition_Zero_Unsatisfied() {
	// Create an existing object.
	o, err := storageutil.CreateObject(
		t.ctx,
		t.bucket,
		"foo",
		[]byte("taco"))

	AssertEq(nil, err)

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

func (t *createTest) GenerationPrecondition_Zero_Satisfied() {
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

func (t *createTest) GenerationPrecondition_NonZero_Unsatisfied_Missing() {
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

func (t *createTest) GenerationPrecondition_NonZero_Unsatisfied_Present() {
	// Create an existing object.
	o, err := storageutil.CreateObject(
		t.ctx,
		t.bucket,
		"foo",
		[]byte("taco"))

	AssertEq(nil, err)

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

func (t *createTest) GenerationPrecondition_NonZero_Satisfied() {
	// Create an existing object.
	orig, err := storageutil.CreateObject(
		t.ctx,
		t.bucket,
		"foo",
		[]byte("taco"))

	AssertEq(nil, err)

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

func (t *createTest) MetaGenerationPrecondition_Unsatisfied_ObjectDoesntExist() {
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

func (t *createTest) MetaGenerationPrecondition_Unsatisfied_ObjectExists() {
	// Create an existing object.
	o, err := storageutil.CreateObject(
		t.ctx,
		t.bucket,
		"foo",
		[]byte("taco"))

	AssertEq(nil, err)
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

func (t *createTest) MetaGenerationPrecondition_Satisfied() {
	// Create an existing object.
	orig, err := storageutil.CreateObject(
		t.ctx,
		t.bucket,
		"foo",
		[]byte("taco"))
	AssertEq(nil, err)

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

////////////////////////////////////////////////////////////////////////
// Copy
////////////////////////////////////////////////////////////////////////

type copyTest struct {
	bucketTest
}

func (t *copyTest) SourceDoesntExist() {
	var err error

	// Copy
	req := &gcs.CopyObjectRequest{
		SrcName: "foo",
		DstName: "bar",
	}

	_, err = t.bucket.CopyObject(t.ctx, req)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))

	// List
	objects, runs, err := storageutil.ListAll(
		t.ctx,
		t.bucket,
		&gcs.ListObjectsRequest{})

	AssertEq(nil, err)
	ExpectThat(objects, ElementsAre())
	ExpectThat(runs, ElementsAre())
}

func (t *copyTest) DestinationDoesntExist() {
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
	ExpectEq(computeCrc32C("taco"), *dst.CRC32C)
	ExpectThat(dst.MediaLink, MatchesRegexp("download/storage.*bar"))
	ExpectThat(dst.Metadata, DeepEquals(src.Metadata))
	ExpectLt(0, dst.Generation)
	ExpectEq(1, dst.MetaGeneration)
	ExpectEq("STANDARD", dst.StorageClass)
	ExpectThat(dst.Deleted, DeepEquals(time.Time{}))
	ExpectThat(dst.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(dst.Updated, t.matchesStartTime(createTime))

	// The object should be readable.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, "bar")

	AssertEq(nil, err)
	ExpectEq("taco", string(contents))

	// And stattable.
	statMinObj, statExtObjAttr, err := t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: "bar",
			ForceFetchFromGcs:              true,
			ReturnExtendedObjectAttributes: true})

	AssertEq(nil, err)
	AssertNe(nil, statMinObj)
	AssertNe(nil, statExtObjAttr)
	statObj := storageutil.ConvertMinObjectAndExtendedObjectAttributesToObject(statMinObj, statExtObjAttr)
	ExpectThat(statObj, Pointee(DeepEquals(*dst)))
}

func (t *copyTest) DestinationExists() {
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
	ExpectEq(computeCrc32C("taco"), *dst.CRC32C)
	ExpectThat(dst.MediaLink, MatchesRegexp("download/storage.*bar"))
	ExpectThat(dst.Metadata, DeepEquals(src.Metadata))
	ExpectLt(orig.Generation, dst.Generation)
	ExpectEq(1, dst.MetaGeneration)
	ExpectEq("STANDARD", dst.StorageClass)
	ExpectThat(dst.Deleted, DeepEquals(time.Time{}))
	ExpectThat(dst.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(dst.Updated, t.matchesStartTime(createTime))

	// The object should be readable.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, "bar")

	AssertEq(nil, err)
	ExpectEq("taco", string(contents))

	// And stattable.
	statMinObj, statExtObjAttr, err := t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: "bar",
			ForceFetchFromGcs:              true,
			ReturnExtendedObjectAttributes: true})

	AssertEq(nil, err)
	AssertNe(nil, statMinObj)
	AssertNe(nil, statExtObjAttr)
	statObj := storageutil.ConvertMinObjectAndExtendedObjectAttributesToObject(statMinObj, statExtObjAttr)
	ExpectThat(statObj, Pointee(DeepEquals(*dst)))
}

func (t *copyTest) DestinationIsSameName() {
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
	ExpectEq(computeCrc32C("taco"), *dst.CRC32C)
	ExpectThat(dst.MediaLink, MatchesRegexp("download/storage.*foo"))
	ExpectThat(dst.Metadata, DeepEquals(src.Metadata))
	ExpectLt(src.Generation, dst.Generation)
	ExpectEq(1, dst.MetaGeneration)
	ExpectEq("STANDARD", dst.StorageClass)
	ExpectThat(dst.Deleted, DeepEquals(time.Time{}))
	ExpectThat(dst.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(dst.Updated, t.matchesStartTime(createTime))

	// The object should be readable.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, "foo")

	AssertEq(nil, err)
	ExpectEq("taco", string(contents))

	// And stattable.
	statMinObj, statExtObjAttr, err := t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: "foo",
			ForceFetchFromGcs:              true,
			ReturnExtendedObjectAttributes: true})

	AssertEq(nil, err)
	AssertNe(nil, statMinObj)
	AssertNe(nil, statExtObjAttr)
	statObj := storageutil.ConvertMinObjectAndExtendedObjectAttributesToObject(statMinObj, statExtObjAttr)
	ExpectThat(statObj, Pointee(DeepEquals(*dst)))
}

func (t *copyTest) InterestingNames() {
	var err error

	// Create a source object.
	const srcName = "foo"
	_, err = storageutil.CreateObject(t.ctx, t.bucket, srcName, []byte{})
	AssertEq(nil, err)

	// Make sure we can use each interesting name as a copy destination.
	err = forEachString(
		t.ctx,
		interestingNames(),
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

func (t *copyTest) IllegalNames() {
	var err error

	// Create a source object.
	const srcName = "foo"
	_, err = storageutil.CreateObject(t.ctx, t.bucket, srcName, []byte{})
	AssertEq(nil, err)

	// Make sure we can't use any illegal name as a copy destination.
	err = forEachString(
		t.ctx,
		illegalNames(),
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

func (t *copyTest) ParticularSourceGeneration_NameDoesntExist() {
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

func (t *copyTest) ParticularSourceGeneration_GenerationDoesntExist() {
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

func (t *copyTest) ParticularSourceGeneration_Exists() {
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

func (t *copyTest) SrcMetaGenerationPrecondition_Unsatisfied() {
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
	_, _, err = t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: "bar"})

	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *copyTest) SrcMetaGenerationPrecondition_Satisfied() {
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
	_, _, err = t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: "bar"})

	ExpectEq(nil, err)
}

////////////////////////////////////////////////////////////////////////
// Compose
////////////////////////////////////////////////////////////////////////

type composeTest struct {
	bucketTest
}

func (t *composeTest) createSources(
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

func (t *composeTest) OneSimpleSource() {
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
	ExpectEq(computeCrc32C("taco"), *o.CRC32C)
	ExpectThat(o.MediaLink, MatchesRegexp("download/storage.*foo"))
	ExpectEq(nil, o.Metadata)
	ExpectLt(sources[0].Generation, o.Generation)
	ExpectEq(1, o.MetaGeneration)
	ExpectEq("STANDARD", o.StorageClass)
	ExpectThat(o.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(o.Updated, t.matchesStartTime(composeTime))

	// Check contents.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, "foo")

	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
}

func (t *composeTest) TwoSimpleSources() {
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
	ExpectEq(computeCrc32C("tacoburrito"), *o.CRC32C)
	ExpectThat(o.MediaLink, MatchesRegexp("download/storage.*foo"))
	ExpectEq(nil, o.Metadata)
	ExpectLt(sources[0].Generation, o.Generation)
	ExpectLt(sources[1].Generation, o.Generation)
	ExpectEq(1, o.MetaGeneration)
	ExpectEq("STANDARD", o.StorageClass)
	ExpectThat(o.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(o.Updated, t.matchesStartTime(composeTime))

	// Check contents.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, "foo")

	AssertEq(nil, err)
	ExpectEq("tacoburrito", string(contents))
}

func (t *composeTest) ManySimpleSources() {
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
	ExpectEq(computeCrc32C("tacoburritoenchiladaqueso"), *o.CRC32C)
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
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, "foo")

	AssertEq(nil, err)
	ExpectEq("tacoburritoenchiladaqueso", string(contents))
}

func (t *composeTest) RepeatedSources() {
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
	ExpectEq(computeCrc32C("tacoburritotacoburrito"), *o.CRC32C)
	ExpectThat(o.MediaLink, MatchesRegexp("download/storage.*foo"))
	ExpectEq(nil, o.Metadata)
	ExpectLt(sources[0].Generation, o.Generation)
	ExpectLt(sources[1].Generation, o.Generation)
	ExpectEq(1, o.MetaGeneration)
	ExpectEq("STANDARD", o.StorageClass)
	ExpectThat(o.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(o.Updated, t.matchesStartTime(composeTime))

	// Check contents.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, "foo")

	AssertEq(nil, err)
	ExpectEq("tacoburritotacoburrito", string(contents))
}

func (t *composeTest) CompositeSources() {
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

	AssertEq(nil, err)

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
	ExpectEq(computeCrc32C("tacoburritotacotacoburrito"), *o.CRC32C)
	ExpectThat(o.MediaLink, MatchesRegexp("download/storage.*foo"))
	ExpectEq(nil, o.Metadata)
	ExpectLt(sources[0].Generation, o.Generation)
	ExpectLt(sources[2].Generation, o.Generation)
	ExpectEq(1, o.MetaGeneration)
	ExpectEq("STANDARD", o.StorageClass)
	ExpectThat(o.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(o.Updated, t.matchesStartTime(composeTime))

	// Check contents.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, "foo")

	AssertEq(nil, err)
	ExpectEq("tacoburritotacotacoburrito", string(contents))
}

func (t *composeTest) Metadata() {
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

func (t *composeTest) DestinationNameMatchesSource() {
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
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, sources[0].Name)

	AssertEq(nil, err)
	ExpectEq("tacoburrito", string(contents))
}

func (t *composeTest) OneSourceDoesntExist() {
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
	_, _, err = t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: "foo"})

	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *composeTest) ExplicitGenerations_Exist() {
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

func (t *composeTest) ExplicitGenerations_OneDoesntExist() {
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
	_, _, err = t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: "foo"})

	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *composeTest) DestinationExists_NoPreconditions() {
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
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, sources[0].Name)

	AssertEq(nil, err)
	ExpectEq("tacoburrito", string(contents))
}

func (t *composeTest) DestinationExists_GenerationPreconditionNotSatisfied() {
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
	m, _, err := t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: sources[0].Name})

	AssertEq(nil, err)
	ExpectEq(sources[0].Generation, m.Generation)
	ExpectEq(len("taco"), m.Size)
}

func (t *composeTest) DestinationExists_MetaGenerationPreconditionNotSatisfied() {
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
	m, _, err := t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: sources[0].Name})

	AssertEq(nil, err)
	ExpectEq(sources[0].Generation, m.Generation)
	ExpectEq(sources[0].MetaGeneration, m.MetaGeneration)
	ExpectEq(len("taco"), m.Size)
}

func (t *composeTest) DestinationExists_PreconditionsSatisfied() {
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
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, sources[0].Name)

	AssertEq(nil, err)
	ExpectEq("tacoburrito", string(contents))
}

func (t *composeTest) DestinationDoesntExist_PreconditionNotSatisfied() {
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
	_, _, err = t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: "foo"})

	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *composeTest) DestinationDoesntExist_PreconditionSatisfied() {
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
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, "foo")

	AssertEq(nil, err)
	ExpectEq("tacoburrito", string(contents))
}

func (t *composeTest) ZeroSources() {
	// GCS doesn't like zero-source requests (and so neither should our fake).
	req := &gcs.ComposeObjectsRequest{
		DstName: "foo",
	}

	_, err := t.bucket.ComposeObjects(t.ctx, req)
	ExpectThat(err, Error(HasSubstr("at least one")))
}

func (t *composeTest) TooManySources() {
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

func (t *composeTest) ComponentCountLimits() {
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

func (t *composeTest) InterestingNames() {
	var err error

	// Create a source object.
	const srcName = "foo"
	_, err = storageutil.CreateObject(t.ctx, t.bucket, srcName, []byte{})
	AssertEq(nil, err)

	// Make sure we can use each interesting name as a compose destination.
	err = forEachString(
		t.ctx,
		interestingNames(),
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

func (t *composeTest) IllegalNames() {
	var err error

	// Create a source object.
	const srcName = "foo"
	_, err = storageutil.CreateObject(t.ctx, t.bucket, srcName, []byte{})
	AssertEq(nil, err)

	// Make sure we can't use any illegal name as a compose destination.
	err = forEachString(
		t.ctx,
		illegalNames(),
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

////////////////////////////////////////////////////////////////////////
// Read
////////////////////////////////////////////////////////////////////////

type readTest struct {
	bucketTest
}

func (t *readTest) ObjectNameDoesntExist() {
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

func (t *readTest) EmptyObject() {
	// Create
	AssertEq(nil, t.createObject("foo", ""))

	// Read
	req := &gcs.ReadObjectRequest{
		Name: "foo",
	}

	r, err := t.bucket.NewReader(t.ctx, req)
	AssertEq(nil, err)

	contents, err := io.ReadAll(r)
	AssertEq(nil, err)
	ExpectEq("", string(contents))

	// Close
	AssertEq(nil, r.Close())
}

func (t *readTest) NonEmptyObject() {
	// Create
	AssertEq(nil, t.createObject("foo", "taco"))

	// Read
	req := &gcs.ReadObjectRequest{
		Name: "foo",
	}

	r, err := t.bucket.NewReader(t.ctx, req)
	AssertEq(nil, err)

	contents, err := io.ReadAll(r)
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))

	// Close
	AssertEq(nil, r.Close())
}

func (t *readTest) ParticularGeneration_NeverExisted() {
	// Create an object.
	o, err := storageutil.CreateObject(
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

func (t *readTest) ParticularGeneration_HasBeenDeleted() {
	// Create an object.
	o, err := storageutil.CreateObject(
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

func (t *readTest) ParticularGeneration_Exists() {
	// Create an object.
	o, err := storageutil.CreateObject(
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

	contents, err := io.ReadAll(r)
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))

	// Close
	AssertEq(nil, r.Close())
}

func (t *readTest) ParticularGeneration_ObjectHasBeenOverwritten() {
	// Create an object.
	o, err := storageutil.CreateObject(
		t.ctx,
		t.bucket,
		"foo",
		[]byte("taco"))

	AssertEq(nil, err)
	AssertGt(o.Generation, 0)

	// Overwrite with a new generation.
	o2, err := storageutil.CreateObject(
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

	contents, err := io.ReadAll(rc)
	AssertEq(nil, err)
	ExpectEq("burrito", string(contents))

	// Close
	AssertEq(nil, rc.Close())
}

func (t *readTest) Ranges_EmptyObject() {
	// Create an empty object.
	AssertEq(nil, t.createObject("foo", ""))

	// Test cases.
	testCases := []struct {
		br gcs.ByteRange
	}{
		// Empty without knowing object length
		{gcs.ByteRange{Start: 0, Limit: 0}},

		{gcs.ByteRange{Start: 1, Limit: 1}},
		{gcs.ByteRange{Start: 1, Limit: 0}},

		{gcs.ByteRange{Start: math.MaxInt64, Limit: math.MaxInt64}},
		{gcs.ByteRange{Start: math.MaxInt64, Limit: 17}},
		{gcs.ByteRange{Start: math.MaxInt64, Limit: 0}},

		{gcs.ByteRange{Start: math.MaxUint64, Limit: math.MaxUint64}},
		{gcs.ByteRange{Start: math.MaxUint64, Limit: 17}},
		{gcs.ByteRange{Start: math.MaxUint64, Limit: 0}},

		// Not empty without knowing object length
		{gcs.ByteRange{Start: 0, Limit: 1}},
		{gcs.ByteRange{Start: 0, Limit: 17}},
		{gcs.ByteRange{Start: 0, Limit: math.MaxInt64}},
		{gcs.ByteRange{Start: 0, Limit: math.MaxUint64}},

		{gcs.ByteRange{Start: 1, Limit: 2}},
		{gcs.ByteRange{Start: 1, Limit: 17}},
		{gcs.ByteRange{Start: 1, Limit: math.MaxInt64}},
		{gcs.ByteRange{Start: 1, Limit: math.MaxUint64}},

		{gcs.ByteRange{Start: math.MaxInt64, Limit: math.MaxInt64 + 1}},
		{gcs.ByteRange{Start: math.MaxInt64, Limit: math.MaxUint64}},
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
	contents, errs := readMultiple(
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

func (t *readTest) Ranges_NonEmptyObject() {
	// Create an object of length four.
	AssertEq(nil, t.createObject("foo", "taco"))

	// Test cases.
	testCases := []struct {
		br               gcs.ByteRange
		expectedContents string
	}{
		// Left anchored
		{gcs.ByteRange{Start: 0, Limit: math.MaxUint64}, "taco"},
		{gcs.ByteRange{Start: 0, Limit: 5}, "taco"},
		{gcs.ByteRange{Start: 0, Limit: 4}, "taco"},
		{gcs.ByteRange{Start: 0, Limit: 3}, "tac"},
		{gcs.ByteRange{Start: 0, Limit: 1}, "t"},
		{gcs.ByteRange{Start: 0, Limit: 0}, ""},

		// Floating left edge
		{gcs.ByteRange{Start: 1, Limit: math.MaxUint64}, "aco"},
		{gcs.ByteRange{Start: 1, Limit: 5}, "aco"},
		{gcs.ByteRange{Start: 1, Limit: 4}, "aco"},
		{gcs.ByteRange{Start: 1, Limit: 2}, "a"},
		{gcs.ByteRange{Start: 1, Limit: 1}, ""},
		{gcs.ByteRange{Start: 1, Limit: 0}, ""},

		// Left edge at right edge of object
		{gcs.ByteRange{Start: 4, Limit: math.MaxUint64}, ""},
		{gcs.ByteRange{Start: 4, Limit: math.MaxInt64 + 1}, ""},
		{gcs.ByteRange{Start: 4, Limit: math.MaxInt64 + 0}, ""},
		{gcs.ByteRange{Start: 4, Limit: math.MaxInt64 - 1}, ""},
		{gcs.ByteRange{Start: 4, Limit: 17}, ""},
		{gcs.ByteRange{Start: 4, Limit: 5}, ""},
		{gcs.ByteRange{Start: 4, Limit: 4}, ""},
		{gcs.ByteRange{Start: 4, Limit: 1}, ""},
		{gcs.ByteRange{Start: 4, Limit: 0}, ""},

		// Left edge past right edge of object
		{gcs.ByteRange{Start: 5, Limit: math.MaxUint64}, ""},
		{gcs.ByteRange{Start: 5, Limit: 17}, ""},
		{gcs.ByteRange{Start: 5, Limit: 5}, ""},
		{gcs.ByteRange{Start: 5, Limit: 4}, ""},
		{gcs.ByteRange{Start: 5, Limit: 1}, ""},
		{gcs.ByteRange{Start: 5, Limit: 0}, ""},

		// Left edge is 2^63 - 1
		{gcs.ByteRange{Start: math.MaxInt64, Limit: math.MaxUint64}, ""},
		{gcs.ByteRange{Start: math.MaxInt64, Limit: math.MaxInt64 + 1}, ""},
		{gcs.ByteRange{Start: math.MaxInt64, Limit: math.MaxInt64 + 0}, ""},
		{gcs.ByteRange{Start: math.MaxInt64, Limit: math.MaxInt64 - 1}, ""},
		{gcs.ByteRange{Start: math.MaxInt64, Limit: 5}, ""},
		{gcs.ByteRange{Start: math.MaxInt64, Limit: 4}, ""},
		{gcs.ByteRange{Start: math.MaxInt64, Limit: 1}, ""},
		{gcs.ByteRange{Start: math.MaxInt64, Limit: 0}, ""},

		// Left edge is 2^64 - 1
		{gcs.ByteRange{Start: math.MaxUint64, Limit: math.MaxUint64}, ""},
		{gcs.ByteRange{Start: math.MaxUint64, Limit: math.MaxInt64 + 1}, ""},
		{gcs.ByteRange{Start: math.MaxUint64, Limit: math.MaxInt64}, ""},
		{gcs.ByteRange{Start: math.MaxUint64, Limit: math.MaxInt64 - 1}, ""},
		{gcs.ByteRange{Start: math.MaxUint64, Limit: 5}, ""},
		{gcs.ByteRange{Start: math.MaxInt64, Limit: 4}, ""},
		{gcs.ByteRange{Start: math.MaxInt64, Limit: 1}, ""},
		{gcs.ByteRange{Start: math.MaxInt64, Limit: 0}, ""},
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
	contents, errs := readMultiple(
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

////////////////////////////////////////////////////////////////////////
// Stat
////////////////////////////////////////////////////////////////////////

type statTest struct {
	bucketTest
}

func (t *statTest) NonExistentObject() {
	req := &gcs.StatObjectRequest{
		Name: "foo",
	}

	_, _, err := t.bucket.StatObject(t.ctx, req)

	AssertThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
	ExpectThat(err, Error(MatchesRegexp("not found|404")))
}

func (t *statTest) StatAfterCreating() {
	// Create an object.
	createTime := t.clock.Now()
	orig, err := storageutil.CreateObject(t.ctx, t.bucket, "foo", []byte("taco"))
	AssertEq(nil, err)
	AssertThat(orig.Updated, t.matchesStartTime(createTime))

	// Ensure the time below doesn't match exactly.
	t.advanceTime()

	// Stat it.
	req := &gcs.StatObjectRequest{
		Name:                           "foo",
		ForceFetchFromGcs:              true,
		ReturnExtendedObjectAttributes: true,
	}

	m, e, err := t.bucket.StatObject(t.ctx, req)
	AssertEq(nil, err)
	AssertNe(nil, m)
	AssertNe(nil, e)

	ExpectEq("foo", m.Name)
	ExpectEq(orig.Generation, m.Generation)
	ExpectEq(len("taco"), m.Size)
	ExpectThat(e.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(m.Updated, timeutil.TimeEq(orig.Updated))
}

func (t *statTest) StatAfterOverwriting() {
	// Create an object.
	_, err := storageutil.CreateObject(t.ctx, t.bucket, "foo", []byte("taco"))
	AssertEq(nil, err)

	// Ensure the time below doesn't match exactly.
	t.advanceTime()

	// Overwrite it.
	overwriteTime := t.clock.Now()
	o2, err := storageutil.CreateObject(t.ctx, t.bucket, "foo", []byte("burrito"))
	AssertEq(nil, err)
	AssertThat(o2.Updated, t.matchesStartTime(overwriteTime))

	// Ensure the time below doesn't match exactly.
	t.advanceTime()

	// Stat it.
	req := &gcs.StatObjectRequest{
		Name:                           "foo",
		ForceFetchFromGcs:              true,
		ReturnExtendedObjectAttributes: true,
	}

	m, e, err := t.bucket.StatObject(t.ctx, req)
	AssertEq(nil, err)
	AssertNe(nil, m)
	AssertNe(nil, e)

	ExpectEq("foo", m.Name)
	ExpectEq(o2.Generation, m.Generation)
	ExpectEq(len("burrito"), m.Size)
	ExpectThat(e.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(m.Updated, timeutil.TimeEq(o2.Updated))
}

func (t *statTest) StatAfterUpdating() {
	// Create an object.
	createTime := t.clock.Now()
	orig, err := storageutil.CreateObject(t.ctx, t.bucket, "foo", []byte("taco"))
	AssertEq(nil, err)
	AssertThat(orig.Updated, t.matchesStartTime(createTime))

	// Ensure the time below doesn't match exactly.
	t.advanceTime()

	// Update the object.
	ureq := &gcs.UpdateObjectRequest{
		Name:        "foo",
		ContentType: makeStringPtr("image/png"),
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
		Name:                           "foo",
		ForceFetchFromGcs:              true,
		ReturnExtendedObjectAttributes: true,
	}

	m, e, err := t.bucket.StatObject(t.ctx, req)
	AssertEq(nil, err)
	AssertNe(nil, m)
	AssertNe(nil, e)

	ExpectEq("foo", m.Name)
	ExpectEq(o2.Generation, m.Generation)
	ExpectEq(o2.MetaGeneration, m.MetaGeneration)
	ExpectEq(len("taco"), m.Size)
	ExpectThat(e.Deleted, timeutil.TimeEq(time.Time{}))
	ExpectThat(m.Updated, timeutil.TimeEq(o2.Updated))
}

////////////////////////////////////////////////////////////////////////
// Update
////////////////////////////////////////////////////////////////////////

type updateTest struct {
	bucketTest
}

func (t *updateTest) NonExistentObject() {
	req := &gcs.UpdateObjectRequest{
		Name:        "foo",
		ContentType: makeStringPtr("image/png"),
	}

	_, err := t.bucket.UpdateObject(t.ctx, req)

	AssertThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
	ExpectThat(err, Error(MatchesRegexp("not found|404")))
}

func (t *updateTest) RemoveAllFields() {
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
		ContentEncoding: makeStringPtr(""),
		ContentLanguage: makeStringPtr(""),
		ContentType:     makeStringPtr(""),
		CacheControl:    makeStringPtr(""),
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

func (t *updateTest) ModifyAllFields() {
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
		ContentType:     makeStringPtr("image/jpeg"),
		ContentEncoding: makeStringPtr("bzip2"),
		ContentLanguage: makeStringPtr("de"),
		CacheControl:    makeStringPtr("private"),
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

func (t *updateTest) MixedModificationsToFields() {
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
		ContentEncoding: makeStringPtr(""),
		ContentLanguage: makeStringPtr("de"),
		CacheControl:    makeStringPtr("private"),
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

func (t *updateTest) AddUserMetadata() {
	// Create an object with no user metadata.
	orig, err := storageutil.CreateObject(t.ctx, t.bucket, "foo", []byte("taco"))
	AssertEq(nil, err)

	AssertEq(nil, orig.Metadata)

	// Add some metadata.
	req := &gcs.UpdateObjectRequest{
		Name: "foo",
		Metadata: map[string]*string{
			"0": makeStringPtr("taco"),
			"1": makeStringPtr("burrito"),
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

func (t *updateTest) MixedModificationsToUserMetadata() {
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
			"1": makeStringPtr("burrito"),
			"2": nil,
			"3": makeStringPtr("updated"),
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

func (t *updateTest) UpdateTime() {
	// Create an object.
	createTime := t.clock.Now()
	o, err := storageutil.CreateObject(t.ctx, t.bucket, "foo", []byte{})
	AssertEq(nil, err)
	AssertThat(o.Updated, t.matchesStartTime(createTime))

	// Ensure the time below doesn't match exactly.
	t.advanceTime()

	// Modify a field.
	req := &gcs.UpdateObjectRequest{
		Name:        "foo",
		ContentType: makeStringPtr("image/jpeg"),
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

func (t *updateTest) ParticularGeneration_NameDoesntExist() {
	req := &gcs.UpdateObjectRequest{
		Name:        "foo",
		Generation:  17,
		ContentType: makeStringPtr("image/png"),
	}

	_, err := t.bucket.UpdateObject(t.ctx, req)

	AssertThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
	ExpectThat(err, Error(MatchesRegexp("not found|404")))
}

func (t *updateTest) ParticularGeneration_GenerationDoesntExist() {
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
		ContentLanguage: makeStringPtr("fr"),
	}

	_, err = t.bucket.UpdateObject(t.ctx, req)

	AssertThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
	ExpectThat(err, Error(MatchesRegexp("not found|404")))

	// The original object should be unaffected.
	_, e, err := t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: o.Name,
			ForceFetchFromGcs:              true,
			ReturnExtendedObjectAttributes: true})

	AssertEq(nil, err)
	AssertNe(nil, e)
	ExpectEq("", e.ContentLanguage)
}

func (t *updateTest) ParticularGeneration_Successful() {
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
		ContentLanguage: makeStringPtr("fr"),
	}

	o, err = t.bucket.UpdateObject(t.ctx, req)

	AssertEq(nil, err)
	ExpectEq("fr", o.ContentLanguage)

	// Stat and make sure it took effect.
	_, e, err := t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: o.Name,
			ForceFetchFromGcs:              true,
			ReturnExtendedObjectAttributes: true})

	AssertEq(nil, err)
	AssertNe(nil, e)
	ExpectEq("fr", e.ContentLanguage)
}

func (t *updateTest) MetaGenerationPrecondition_Unsatisfied() {
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
		ContentLanguage:            makeStringPtr("fr"),
	}

	_, err = t.bucket.UpdateObject(t.ctx, req)
	ExpectThat(err, HasSameTypeAs(&gcs.PreconditionError{}))

	// The original object should be unaffected.
	_, e, err := t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: o.Name,
			ForceFetchFromGcs:              true,
			ReturnExtendedObjectAttributes: true})

	AssertEq(nil, err)
	AssertNe(nil, e)
	ExpectEq("", e.ContentLanguage)
}

func (t *updateTest) MetaGenerationPrecondition_Satisfied() {
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
		ContentLanguage:            makeStringPtr("fr"),
	}

	_, err = t.bucket.UpdateObject(t.ctx, req)
	AssertEq(nil, err)

	// The object should have been updated.
	_, e, err := t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{Name: o.Name,
			ForceFetchFromGcs:              true,
			ReturnExtendedObjectAttributes: true})

	AssertEq(nil, err)
	AssertNe(nil, e)
	ExpectEq("fr", e.ContentLanguage)
}

////////////////////////////////////////////////////////////////////////
// Delete
////////////////////////////////////////////////////////////////////////

type deleteTest struct {
	bucketTest
}

func (t *deleteTest) NoParticularGeneration_NameDoesntExist() {
	// No error should be returned.
	err := t.bucket.DeleteObject(
		t.ctx,
		&gcs.DeleteObjectRequest{
			Name: "foobar",
		})

	ExpectEq(nil, err)
}

func (t *deleteTest) NoParticularGeneration_Successful() {
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

func (t *deleteTest) ParticularGeneration_NameDoesntExist() {
	// No error should be returned.
	err := t.bucket.DeleteObject(
		t.ctx,
		&gcs.DeleteObjectRequest{
			Name:       "foobar",
			Generation: 17,
		})

	ExpectEq(nil, err)
}

func (t *deleteTest) ParticularGeneration_GenerationDoesntExist() {
	const name = "foo"
	var err error

	// Create an object.
	o, err := storageutil.CreateObject(
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
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, name)

	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
}

func (t *deleteTest) ParticularGeneration_Successful() {
	const name = "foo"
	var err error

	// Create an object.
	o, err := storageutil.CreateObject(
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
	_, err = storageutil.ReadObject(t.ctx, t.bucket, name)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *deleteTest) MetaGenerationPrecondition_Unsatisfied_ObjectExists() {
	const name = "foo"
	var err error

	// Create an object.
	o, err := storageutil.CreateObject(
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
	_, err = storageutil.ReadObject(t.ctx, t.bucket, name)
	ExpectEq(nil, err)
}

func (t *deleteTest) MetaGenerationPrecondition_Unsatisfied_ObjectDoesntExist() {
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

func (t *deleteTest) MetaGenerationPrecondition_Unsatisfied_WrongGeneration() {
	const name = "foo"
	var err error

	// Create an object.
	o, err := storageutil.CreateObject(
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
	_, err = storageutil.ReadObject(t.ctx, t.bucket, name)
	ExpectEq(nil, err)
}

func (t *deleteTest) MetaGenerationPrecondition_Satisfied() {
	const name = "foo"
	var err error

	// Create an object.
	o, err := storageutil.CreateObject(
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
	_, err = storageutil.ReadObject(t.ctx, t.bucket, name)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

////////////////////////////////////////////////////////////////////////
// List
////////////////////////////////////////////////////////////////////////

type listTest struct {
	bucketTest
}

func (t *listTest) EmptyBucket() {
	listing, err := t.bucket.ListObjects(t.ctx, &gcs.ListObjectsRequest{})
	AssertEq(nil, err)

	AssertNe(nil, listing)
	ExpectThat(listing.Objects, ElementsAre())
	ExpectThat(listing.CollapsedRuns, ElementsAre())
	AssertEq("", listing.ContinuationToken)
}

func (t *listTest) NewlyCreatedObject() {
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

func (t *listTest) TrivialQuery() {
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

func (t *listTest) Delimiter_SingleRune() {
	// Create several objects.
	AssertEq(
		nil,
		createEmpty(
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

func (t *listTest) Delimiter_MultiRune() {
	// Create several objects.
	AssertEq(
		nil,
		createEmpty(
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

func (t *listTest) Prefix() {
	// Create several objects.
	AssertEq(
		nil,
		createEmpty(
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

func (t *listTest) PrefixAndDelimiter_SingleRune() {
	// Create several objects.
	AssertEq(
		nil,
		createEmpty(
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

func (t *listTest) PrefixAndDelimiter_MultiRune() {
	// Create several objects.
	AssertEq(
		nil,
		createEmpty(
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

func (t *listTest) Cursor_BucketEndsWithRunOfIndividualObjects() {
	// Create a good number of objects, containing a run of objects sharing a
	// prefix under the delimiter "!".
	AssertEq(
		nil,
		createEmpty(
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

		runs = append(runs, listing.CollapsedRuns...)

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

func (t *listTest) Cursor_BucketEndsWithRunOfObjectsGroupedByDelimiter() {
	// Create a good number of objects, containing runs of objects sharing a
	// prefix under the delimiter "!" at the end of the bucket.
	AssertEq(
		nil,
		createEmpty(
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

		runs = append(runs, listing.CollapsedRuns...)

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
// Cancellation
////////////////////////////////////////////////////////////////////////

type cancellationTest struct {
	bucketTest
}

// A Reader that slowly returns junk, forever. A channel is closed after 1 MiB
// has been read.
type bottomlessReader struct {
	OneMegRead chan struct{}
	n          int
}

func (rc *bottomlessReader) Read(p []byte) (n int, err error) {
	// Return zeroes.
	n = len(p)
	for i := 0; i < n; i++ {
		p[i] = 0
	}

	// But not too quickly.
	time.Sleep(time.Millisecond)

	// Notify once we hit a bunch of data read.
	rc.n += n
	if rc.n >= 1<<20 && rc.OneMegRead != nil {
		close(rc.OneMegRead)
		rc.OneMegRead = nil
	}

	return
}

func (t *cancellationTest) CreateObject() {
	const name = "foo"
	var err error

	if !t.supportsCancellation {
		log.Println("Cancellation not supported; skipping test.")
		return
	}

	if t.buffersEntireContentsForCreate {
		log.Println("Can't use a bottomless reader. Skipping test.")
		return
	}

	// Set up a cancellable context.
	ctx, cancel := context.WithCancel(t.ctx)

	// Begin a request to create an object using a bottomless reader for the
	// contents.
	oneMegRead := make(chan struct{})
	rc := &bottomlessReader{
		OneMegRead: oneMegRead,
	}

	errChan := make(chan error)
	go func() {
		req := &gcs.CreateObjectRequest{
			Name:     name,
			Contents: rc,
		}

		_, err := t.bucket.CreateObject(ctx, req)
		errChan <- err
	}()

	// Wait until some data has been read.
	<-oneMegRead

	// Wait a moment longer. The request should not yet be complete.
	select {
	case err = <-errChan:
		AddFailure("CreateObject returned early with error: %v", err)
		AbortTest()

	case <-time.After(10 * time.Millisecond):
	}

	// Cancel the request. Now it should return quickly with an appropriate
	// error.
	cancel()
	err = <-errChan

	ExpectThat(
		err,
		Error(
			AnyOf(
				HasSubstr("closed network connection"),
				HasSubstr("transport closed"),
				HasSubstr("request canceled"))))

	// The object should not have been created.
	statReq := &gcs.StatObjectRequest{
		Name: name,
	}

	_, _, err = t.bucket.StatObject(t.ctx, statReq)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *cancellationTest) ReadObject() {
	const name = "foo"
	var err error

	if !t.supportsCancellation {
		log.Println("Cancellation not supported; skipping test.")
		return
	}

	// Create an object that is larger than we are likely to buffer in total
	// throughout the HTTP library, etc.
	const size = 1 << 20
	_, err = t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:     name,
			Contents: io.LimitReader(rand.Reader, size),
		})

	AssertEq(nil, err)

	// Create a reader for the object using a cancellable context.
	ctx, cancel := context.WithCancel(t.ctx)
	rc, err := t.bucket.NewReader(
		ctx,
		&gcs.ReadObjectRequest{
			Name: name,
		})

	AssertEq(nil, err)

	defer rc.Close()

	// Read a few bytes; nothing should go wrong.
	const firstReadSize = 32
	_, err = io.ReadFull(rc, make([]byte, firstReadSize))
	AssertEq(nil, err)

	// Cancel the context.
	cancel()

	// The next read should return quickly in error.
	before := time.Now()
	_, err = io.ReadFull(rc, make([]byte, size-firstReadSize))

	ExpectThat(
		err,
		Error(
			AnyOf(
				HasSubstr("closed network connection"),
				HasSubstr("transport closed"),
				HasSubstr("request canceled"))))
	ExpectLt(time.Since(before), 50*time.Millisecond)
}
