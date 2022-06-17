package gcs

import (
	"cloud.google.com/go/storage"
	"github.com/jacobsa/gcloud/gcs"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
	"io"
	"strings"
	"testing"
)

func TestRead(t *testing.T) { RunTests(t) }

type ReadTest struct {
	ctx        context.Context
	bucketName string
	client     *storage.Client
}

var _ SetUpInterface = &ReadTest{}

func init() { RegisterTestSuite(&ReadTest{}) }

func (t *ReadTest) SetUp(ti *TestInfo) {
	var err error
	t.ctx = ti.Ctx

	t.bucketName = "testing-gcsfuse" // Testing on my personal bucket.
	t.client, err = storage.NewClient(t.ctx)
	AssertEq(nil, err)
}

// Offset equals 0 and Length equals 0. In this case printed string should be empty.
func (t *ReadTest) OffsetZeroLengthZero() {
	var err error
	const objectName = "file.txt"
	const start = 0
	const length = 0
	const end = start + length
	req := &gcs.ReadObjectRequest{
		Name:       objectName,
		Generation: 0,
		Range: &gcs.ByteRange{
			Start: uint64(start),
			Limit: uint64(end),
		},
	}
	rc, err := gcs.NewReaderSCL(t.ctx, req, t.bucketName, t.client) // NewRangeReader of Go Storage Client Library is used to create reader.
	AssertEq(nil, err)

	buf := new(strings.Builder)
	_, err = io.Copy(buf, rc)
	res := buf.String()
	AssertEq("", res)
}

// Offset equals 0 and Length is non-zero. In this case printed string should be "TThis is".
func (t *ReadTest) OffsetZeroLengthNonZero() {
	var err error
	const objectName = "file.txt"
	const start = 0
	const length = 8
	const end = start + length
	req := &gcs.ReadObjectRequest{
		Name:       objectName,
		Generation: 0,
		Range: &gcs.ByteRange{
			Start: uint64(start),
			Limit: uint64(end),
		},
	}
	rc, err := gcs.NewReaderSCL(t.ctx, req, t.bucketName, t.client)
	AssertEq(nil, err)

	buf := new(strings.Builder)
	_, err = io.Copy(buf, rc)
	res := buf.String()
	AssertEq("TThis is", res)
}

// Offset is non-zero and Length equals 0. In this case printed string should be empty.
func (t *ReadTest) OffsetNonZeroLengthZero() {
	var err error
	const objectName = "file.txt"
	const start = 3
	const length = 0
	const end = start + length
	req := &gcs.ReadObjectRequest{
		Name:       objectName,
		Generation: 0,
		Range: &gcs.ByteRange{
			Start: uint64(start),
			Limit: uint64(end),
		},
	}
	rc, err := gcs.NewReaderSCL(t.ctx, req, t.bucketName, t.client)
	AssertEq(nil, err)

	buf := new(strings.Builder)
	_, err = io.Copy(buf, rc)
	res := buf.String()
	AssertEq("", res)
}

// Offset is non-zero and Length is non-zero. In this case printed string should be "is is file".
func (t *ReadTest) OffsetNonZeroLengthNonZero() {
	var err error
	const objectName = "file.txt"
	const start = 3
	const length = 10
	const end = start + length
	req := &gcs.ReadObjectRequest{
		Name:       objectName,
		Generation: 0,
		Range: &gcs.ByteRange{
			Start: uint64(start),
			Limit: uint64(end),
		},
	}
	rc, err := gcs.NewReaderSCL(t.ctx, req, t.bucketName, t.client)
	AssertEq(nil, err)

	buf := new(strings.Builder)
	_, err = io.Copy(buf, rc)
	res := buf.String()
	AssertEq("is is file", res)
}

// Length greater than the total length of the file. In this case whole file should be printed.
func (t *ReadTest) LengthGreaterThanFileLength() {
	var err error
	const objectName = "file.txt"
	const start = 3
	const length = 100 // File length is 20. So 100 is greater than the total length of "file.txt".
	const end = start + length
	req := &gcs.ReadObjectRequest{
		Name:       objectName,
		Generation: 0,
		Range: &gcs.ByteRange{
			Start: uint64(start),
			Limit: uint64(end),
		},
	}
	rc, err := gcs.NewReaderSCL(t.ctx, req, t.bucketName, t.client)
	AssertEq(nil, err)

	buf := new(strings.Builder)
	_, err = io.Copy(buf, rc)
	res := buf.String()
	AssertEq("is is file.txt\ny\n", res)
}
