package storage

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	. "github.com/jacobsa/ogletest"
	"google.golang.org/api/googleapi"
)

func TestChunkUploader(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ChunkUploaderTest struct {
	fakeStorage FakeStorage
	content     string
	req         *gcs.CreateObjectRequest
	obj         *storage.ObjectHandle
}

var _ SetUpInterface = &ChunkUploaderTest{}
var _ TearDownInterface = &ChunkUploaderTest{}

func init() { RegisterTestSuite(&ChunkUploaderTest{}) }

func (t *ChunkUploaderTest) SetUp(ti *TestInfo) {
	t.fakeStorage = NewFakeStorage()
	storageHandle := t.fakeStorage.CreateStorageHandle()
	bucketHandle := storageHandle.BucketHandle(TestBucketName, "")

	AssertNe(nil, bucketHandle)

	t.content = "Creating a new object"
	t.req = &gcs.CreateObjectRequest{
		Name:     "test_object",
		Contents: strings.NewReader(t.content),
	}

	t.obj = bucketHandle.bucket.Object(t.req.Name)

	AssertNe(nil, t.obj)
	AssertEq("test_object", t.obj.ObjectName())
}

func (t *ChunkUploaderTest) TearDown() {
	t.fakeStorage.ShutDown()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ChunkUploaderTest) CreateWithImproperInputs() {
	properReq := t.req
	properObj := t.obj
	diffCreateReq := &gcs.CreateObjectRequest{}
	*diffCreateReq = *properReq
	diffCreateReq.Name = "FakeName"
	inputs := []struct {
		ctx          context.Context
		obj          *storage.ObjectHandle
		req          *gcs.CreateObjectRequest
		chunkSize    int
		progressFunc func(int64)
		errStr       string
	}{{
		errStr: "ctx is nil",
	}, {
		ctx:    context.Background(),
		errStr: "nil obj or req",
	}, {
		ctx:    context.Background(),
		obj:    properObj,
		req:    properReq,
		errStr: "chunkSize = 0",
	}, {
		ctx:       context.Background(),
		obj:       properObj,
		req:       diffCreateReq,
		errStr:    "names of passed object-handle and createObjectRequest.Name don't match",
		chunkSize: 100,
	},
	}

	for _, input := range inputs {
		uploader, err := NewChunkUploader(input.ctx, input.obj, input.req, input.chunkSize, input.progressFunc)

		AssertEq(nil, uploader)
		AssertNe(nil, err)
		AssertTrue(strings.Contains(err.Error(), input.errStr))
	}
}

func (t *ChunkUploaderTest) CreateWithProperInputs() {
	inputs := []struct {
		ctx          context.Context
		obj          *storage.ObjectHandle
		req          *gcs.CreateObjectRequest
		chunkSize    int
		progressFunc func(int64)
	}{{
		ctx:       context.Background(),
		obj:       t.obj,
		req:       t.req,
		chunkSize: 1,
	}, {
		ctx:       context.Background(),
		obj:       t.obj,
		req:       t.req,
		chunkSize: 2 << 24,
	},
		{
			ctx:          context.Background(),
			obj:          t.obj,
			req:          t.req,
			chunkSize:    2 << 24,
			progressFunc: func(int64) {},
		},
	}

	for _, input := range inputs {
		uploader, err := NewChunkUploader(input.ctx, input.obj, input.req, input.chunkSize, input.progressFunc)

		AssertNe(nil, uploader)
		AssertEq(nil, err)
		AssertEq(Initialized, uploader.(*chunkUploader).state)
		AssertEq(0, uploader.(*chunkUploader).totalWriteInitiatedSoFar)
		AssertEq(0, uploader.BytesUploadedSoFar())
		AssertEq(t.req.Name, uploader.(*chunkUploader).objectName)

		chunkSize, err := uploader.(*chunkUploader).chunkSize()

		AssertEq(nil, err)
		AssertEq(input.chunkSize, chunkSize)
	}
}

func (t *ChunkUploaderTest) TestUploadEmptyContent() {
	// setup
	obj := t.obj
	chunkSize := googleapi.MinUploadChunkSize
	content := ""

	// test payload
	uploader, err := NewChunkUploader(context.Background(), obj, t.req, chunkSize, nil)

	// test
	AssertNe(nil, uploader)
	AssertEq(nil, err)

	// test payload
	err = uploader.UploadChunkAsync(strings.NewReader(content))

	// Test the return value of UploadChunkAsync and the state of uploader.
	AssertEq(nil, err)
	AssertEq(Waiting, uploader.(*chunkUploader).state)
	AssertEq(0, uploader.(*chunkUploader).totalWriteInitiatedSoFar)
}

type failingContentReader struct{}

func (ilcr *failingContentReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("read failed as intended")
}

func (t *ChunkUploaderTest) TestUploadFailingContentReader() {
	// setup
	fcr := &failingContentReader{}
	obj := t.obj
	chunkSize := googleapi.MinUploadChunkSize

	// test payload
	uploader, err := NewChunkUploader(context.Background(), obj, t.req, chunkSize, nil)

	// test
	AssertNe(nil, uploader)
	AssertEq(nil, err)

	// test payload
	err = uploader.UploadChunkAsync(fcr)

	// Test the return value of UploadChunkAsync and the state of uploader.
	AssertNe(nil, err)
	AssertEq(WriteError, uploader.(*chunkUploader).state)
	AssertEq(0, uploader.(*chunkUploader).totalWriteInitiatedSoFar)
}

func (t *ChunkUploaderTest) TestUploadSingleSubChunkUpload() {
	// A sub-chunk means content is smaller than chunkSize in size.

	// setup
	obj := t.obj
	chunkSize := googleapi.MinUploadChunkSize
	content := t.content
	contentSize := len(content)

	AssertLe(contentSize, chunkSize)

	numBytesSuccessfullyUploadedSoFar := make(chan int64, 1)
	defer close(numBytesSuccessfullyUploadedSoFar)

	// test payload
	uploader, err := NewChunkUploader(context.Background(), obj, t.req, chunkSize,
		func(n int64) {
			numBytesSuccessfullyUploadedSoFar <- n
		})

	// test
	AssertNe(nil, uploader)
	AssertEq(nil, err)

	// setup
	progressFuncCallbackTimeout := time.Millisecond

	// test payload
	err = uploader.UploadChunkAsync(strings.NewReader(content))

	// Test the return value of UploadChunkAsync and the state of uploader.
	AssertEq(nil, err)
	AssertEq(Writing, uploader.(*chunkUploader).state)
	AssertEq(contentSize, uploader.(*chunkUploader).totalWriteInitiatedSoFar)

	// Ensure that the progress callback is not called in this case (i.e. when upload contentSize < chunkSize).
	// For this, we wait for sometime to see if the unexpected callback comes. If it doesn't, we call
	// the test passing.
	// Alternatively we could put the AddFailure directly in the callback,
	// but then the test function would not wait for the unexpected callback to be called at all,
	// and that would cause a false pass of this test.
	select {
	case <-numBytesSuccessfullyUploadedSoFar:
		AddFailure("Received unexpected progressFunc callback")
		break
	case <-time.After(progressFuncCallbackTimeout):
		break
	}
}

func (t *ChunkUploaderTest) TestUploadSingleFullChunkUpload() {
	// setup
	obj := t.obj
	chunkSize := googleapi.MinUploadChunkSize

	b := make([]byte, chunkSize)
	_, err := rand.Read(b)

	AssertEq(nil, err)

	content := string(b)
	progressFuncCallbackTimeout := 10 * time.Millisecond

	numBytesSuccessfullyUploadedSoFar := make(chan int64, 1)
	defer close(numBytesSuccessfullyUploadedSoFar)

	// test payload
	uploader, err := NewChunkUploader(context.Background(), obj, t.req, chunkSize,
		func(n int64) {
			numBytesSuccessfullyUploadedSoFar <- n
		})

	// test
	AssertNe(nil, uploader)
	AssertEq(nil, err)

	// test payload
	err = uploader.UploadChunkAsync(strings.NewReader(content))

	// Test the return value of UploadChunkAsync and the state of uploader.
	AssertEq(nil, err)
	AssertEq(Writing, uploader.(*chunkUploader).state)

	// Ensure that the progress callback is called exactly once in this case (i.e. when upload contentSize == chunkSize).
	// For this, we wait for sometime to see if the callback comes. If it doesn't, we call
	// the test failing.
	select {
	case n := <-numBytesSuccessfullyUploadedSoFar:
		AssertEq(chunkSize, n)
		AssertEq(chunkSize, uploader.(*chunkUploader).totalWriteInitiatedSoFar)
		AssertEq(chunkSize, uploader.BytesUploadedSoFar())
	case <-time.After(progressFuncCallbackTimeout):
		AddFailure("Did not not receive write progressFunc callback in %v", progressFuncCallbackTimeout)
	}
}

func (t *ChunkUploaderTest) TestUploadSingleSuperChunkUpload() {
	// A super-chunk means content is larger than chunkSize in size.

	// setup
	obj := t.obj
	chunkSize := googleapi.MinUploadChunkSize
	numChunks := 2.5
	numFullChunks := int(math.Floor(numChunks))

	b := make([]byte, int(numChunks*float64(chunkSize)))
	_, err := rand.Read(b)

	AssertEq(nil, err)

	content := string(b)
	contentSize := len(content)
	progressFuncCallbackTimeout := time.Millisecond
	var numCallbacks int
	numBytesSuccessfullyUploadedSoFar := make(chan int64, numFullChunks)
	ctx := context.Background()

	// cleanup
	defer close(numBytesSuccessfullyUploadedSoFar)

	// test payload
	uploader, err := NewChunkUploader(ctx, obj, t.req, chunkSize,
		func(n int64) {
			numCallbacks++
			numBytesSuccessfullyUploadedSoFar <- n
		})

	// test
	AssertNe(nil, uploader)
	AssertEq(nil, err)

	// test payload
	err = uploader.UploadChunkAsync(strings.NewReader(content))

	// tests
	AssertEq(nil, err)
	AssertEq(Writing, uploader.(*chunkUploader).state)
	AssertEq(contentSize, uploader.(*chunkUploader).totalWriteInitiatedSoFar)

	// As all progress callbacks were called within the one UploadChunkAsync call,
	// numCallbacks would be incremented numFullChunks times by this point, and
	// uploader.BytesUploadedSoFar() would reflect numFullChunks chunks.
	AssertEq(numFullChunks, numCallbacks)
	AssertEq(numFullChunks*chunkSize, uploader.BytesUploadedSoFar())

	// verifying the incremental values of numBytes within
	// consecutive callbacks.
	for chunkCallbackIndex := 1; chunkCallbackIndex <= numFullChunks; chunkCallbackIndex++ {
		select {
		case n := <-numBytesSuccessfullyUploadedSoFar:
			AssertEq(chunkCallbackIndex*chunkSize, n)
		case <-time.After(progressFuncCallbackTimeout):
			AddFailure("Did not not receive %d-th write progressFunc callback in %v", chunkCallbackIndex, progressFuncCallbackTimeout)
		}
	}
}

func (t *ChunkUploaderTest) TestUploadMultipleUploads() {
	// setup
	obj := t.obj
	chunkSize := googleapi.MinUploadChunkSize
	numChunksForUploads := []float32{0.25, .5, 1.5, .625, 2, 0.5} // Keep these multiples of 1/chunkSize for simplicity of
	// floating-point calculations and comparisons.
	numUploads := len(numChunksForUploads)
	var contentsForUploads []string
	var contentSizesForUploadsCumulative []int
	var numFullChunksByUploadsCumulative []int
	for i := 0; i < numUploads; i++ {
		contentSize := int(numChunksForUploads[i] * float32(chunkSize))
		b := make([]byte, contentSize)

		_, err := rand.Read(b)
		AssertEq(nil, err)

		content := string(b)
		contentsForUploads = append(contentsForUploads, content)

		if i == 0 {
			contentSizesForUploadsCumulative = append(contentSizesForUploadsCumulative, contentSize)
		} else {
			contentSizesForUploadsCumulative = append(contentSizesForUploadsCumulative, contentSizesForUploadsCumulative[i-1]+contentSize)
		}

		numFullChunksByUploadsCumulative =
			append(numFullChunksByUploadsCumulative, int(math.Floor(float64(contentSizesForUploadsCumulative[i])/float64(chunkSize))))
	}
	progressFuncCallbackTimeout := time.Millisecond
	var numCallbacks int
	numBytesSuccessfullyUploadedSoFar := make(chan int64, numFullChunksByUploadsCumulative[numUploads-1])
	ctx := context.Background()

	// cleanup
	defer close(numBytesSuccessfullyUploadedSoFar)

	// test payload
	uploader, err := NewChunkUploader(ctx, obj, t.req, chunkSize,
		func(n int64) {
			numCallbacks++
			numBytesSuccessfullyUploadedSoFar <- n
		})

	// test
	AssertNe(nil, uploader)
	AssertEq(nil, err)

	chunkCallbackIndex := 1
	for i := 0; i < numUploads; i++ {
		// test payload
		err = uploader.UploadChunkAsync(strings.NewReader(contentsForUploads[i]))

		// tests
		AssertEq(nil, err)
		AssertEq(Writing, uploader.(*chunkUploader).state)
		AssertEq(contentSizesForUploadsCumulative[i], uploader.(*chunkUploader).totalWriteInitiatedSoFar)
		AssertEq(numFullChunksByUploadsCumulative[i], numCallbacks)
		AssertEq(numFullChunksByUploadsCumulative[i]*chunkSize, uploader.BytesUploadedSoFar())

		// Verify the values of numByte returned in each consecutive callback.
		for ; chunkCallbackIndex <= numFullChunksByUploadsCumulative[i]; chunkCallbackIndex++ {
			select {
			case n := <-numBytesSuccessfullyUploadedSoFar:
				AssertEq(chunkCallbackIndex*chunkSize, n)
			case <-time.After(progressFuncCallbackTimeout):
				AddFailure("Did not not receive %d-th write progressFunc callback in %v", chunkCallbackIndex, progressFuncCallbackTimeout)
			}
		}
	}
}

func (t *ChunkUploaderTest) TestCloseWithoutWrite() {
	obj := t.obj
	uploader, err := NewChunkUploader(context.Background(), obj, t.req, 1000, nil)

	AssertNe(nil, uploader)
	AssertEq(nil, err)

	o, err := uploader.Close()

	AssertNe(nil, o)
	AssertEq(nil, err)
	AssertEq(o.Name, t.req.Name)
	AssertEq(0, o.Size)
	AssertEq(Destroyed, uploader.(*chunkUploader).state)
}

func (t *ChunkUploaderTest) TestDoubleClosure() {
	obj := t.obj
	uploader, err := NewChunkUploader(context.Background(), obj, t.req, 1000, nil)

	AssertNe(nil, uploader)
	AssertEq(nil, err)

	o, err := uploader.Close()

	AssertNe(nil, o)
	AssertEq(nil, err)
	AssertEq(o.Name, t.req.Name)
	AssertEq(0, o.Size)
	AssertEq(Destroyed, uploader.(*chunkUploader).state)

	o, err = uploader.Close()

	AssertEq(nil, o)
	AssertNe(nil, err)
}

func (t *ChunkUploaderTest) TestCloseWithSingleSubChunkUpload() {
	obj := t.obj
	chunkSize := googleapi.MinUploadChunkSize
	contentSize := len(t.content)

	uploader, err := NewChunkUploader(context.Background(), obj, t.req, chunkSize, nil)

	AssertNe(nil, uploader)
	AssertEq(nil, err)

	err = uploader.UploadChunkAsync(t.req.Contents)

	AssertEq(nil, err)

	o, err := uploader.Close()

	AssertNe(nil, o)
	AssertEq(nil, err)
	AssertEq(o.Name, t.req.Name)
	AssertEq(contentSize, o.Size)
	AssertEq(Destroyed, uploader.(*chunkUploader).state)
}

func (t *ChunkUploaderTest) TestCloseWithMultipleSingleChunkUploads() {
	obj := t.obj
	chunkSize := 1000
	contents := [6]string{"hfue", "yrf8934h9", "iru328ry", "iy3rh34r489y8", "ie32hr83hr43rt9y8", "i9r38ry2u9j9"}
	var contentSize int

	uploader, err := NewChunkUploader(context.Background(), obj, t.req, chunkSize, nil)

	AssertNe(nil, uploader)
	AssertEq(nil, err)

	for _, content := range contents {
		err = uploader.UploadChunkAsync(strings.NewReader(content))

		AssertEq(nil, err)
		contentSize += len(content)
	}

	o, err := uploader.Close()

	AssertNe(nil, o)
	AssertEq(nil, err)
	AssertEq(o.Name, t.req.Name)
	AssertEq(contentSize, o.Size)
	AssertEq(Destroyed, uploader.(*chunkUploader).state)
}

func (t *ChunkUploaderTest) TestCloseWithMultipleMultichunkUploads() {
	obj := t.obj
	chunkSize := 4
	contents := [6]string{"hfuerj3ifj3920",
		"yrf8934h9or329",
		"iru328ryo9dj320j3",
		"iy3rh34r489y89j309",
		"ie32hr83hr43rt9y8do93j9",
		"i9r38ry2u9j9orj3"}
	var contentSize int

	uploader, err := NewChunkUploader(context.Background(), obj, t.req, chunkSize, nil)

	AssertNe(nil, uploader)
	AssertEq(nil, err)

	for _, content := range contents {
		err = uploader.UploadChunkAsync(strings.NewReader(content))
		AssertEq(nil, err)
		contentSize += len(content)
	}

	o, err := uploader.Close()

	AssertNe(nil, o)
	AssertEq(nil, err)
	AssertEq(o.Name, t.req.Name)
	AssertEq(contentSize, o.Size)
	AssertEq(Destroyed, uploader.(*chunkUploader).state)
}

func (t *ChunkUploaderTest) TestCloseFailedUploader() {
	// setup
	fcr := &failingContentReader{}
	obj := t.obj
	chunkSize := googleapi.MinUploadChunkSize

	// test payload
	uploader, err := NewChunkUploader(context.Background(), obj, t.req, chunkSize, nil)

	// test
	AssertNe(nil, uploader)
	AssertEq(nil, err)

	// A successful upload.
	uploadedContent := t.content
	err = uploader.UploadChunkAsync(strings.NewReader(uploadedContent))

	AssertEq(nil, err)
	AssertEq(Writing, uploader.(*chunkUploader).state)

	// test payload - forces failure of upload.
	err = uploader.UploadChunkAsync(fcr)

	// Test the return value of UploadChunkAsync and the state of uploader.
	AssertNe(nil, err)
	AssertEq(WriteError, uploader.(*chunkUploader).state)

	// An upload that should have passed
	// if not for the WriteError in status.
	failedContent := "jifhe4guyfbhufg78gufhewh7fgeyuig"
	err = uploader.UploadChunkAsync(strings.NewReader(failedContent))

	AssertNe(nil, err)
	AssertEq(WriteError, uploader.(*chunkUploader).state)

	// test payload
	o, err := uploader.Close()

	AssertNe(nil, o)
	AssertEq(nil, err)
	AssertEq(o.Name, t.req.Name)
	AssertEq(len(uploadedContent), o.Size)
	AssertEq(Destroyed, uploader.(*chunkUploader).state)
}
