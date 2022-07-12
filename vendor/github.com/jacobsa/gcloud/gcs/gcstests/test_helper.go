package gcstests

import (
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"log"
	"strings"
	"unicode"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/syncutil"
	"golang.org/x/net/context"
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

func CreateEmpty(
	ctx context.Context,
	bucket gcs.Bucket,
	objectNames []string) error {
	err := gcsutil.CreateEmptyObjects(ctx, bucket, objectNames)
	return err
}

func ComputeCrc32C(s string) uint32 {
	return crc32.Checksum([]byte(s), crc32.MakeTable(crc32.Castagnoli))
}

func MakeStringPtr(s string) *string {
	return &s
}

// Return a list of object names that might be problematic for GCS or the Go
// client but are nevertheless documented to be legal.
//
// Useful links:
//
//     https://cloud.google.com/storage/docs/bucket-naming
//     http://www.unicode.org/Public/7.0.0/ucd/UnicodeData.txt
//     http://www.unicode.org/versions/Unicode7.0.0/ch02.pdf (Table 2-3)
//
func InterestingNames() (names []string) {
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
func IllegalNames() (names []string) {
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
func ListDifference(a []string, b []string) (res []string) {
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
func ReadMultiple(
	ctx context.Context,
	bucket gcs.Bucket,
	reqs []*gcs.ReadObjectRequest) (contents [][]byte, errs []error) {
	b := syncutil.NewBundle(ctx)

	// Feed indices into a channel.
	indices := make(chan int, len(reqs))
	for i, _ := range reqs {
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
		b, err = ioutil.ReadAll(rc)
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

		return
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
func ForEachString(
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
