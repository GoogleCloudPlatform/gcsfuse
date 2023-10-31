package storage

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
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
// Helper functions
////////////////////////////////////////////////////////////////////////

// testNewChunkUploader tests a newly created ChunkUploader and asserts
// if it hasn't been created properly.
// If properly created, it casts it to chunkUploader pointer and returns.
func (t *ChunkUploaderTest) testNewChunkUploader(ctx context.Context,
	obj *storage.ObjectHandle,
	req *gcs.CreateObjectRequest,
	chunkSize int,
	progressFunc func(int64)) *chunkUploader {
	uploader, err := NewChunkUploader(ctx, obj, req, chunkSize, progressFunc)
	AssertNe(nil, uploader)
	AssertEq(nil, err)

	chunkUploader := uploader.(*chunkUploader)

	AssertNe(nil, chunkUploader)
	AssertEq(Initialized, chunkUploader.state)
	AssertEq(0, chunkUploader.totalUploadInitiatedSoFar)
	AssertEq(0, chunkUploader.BytesUploadedSoFar())
	AssertNe(nil, chunkUploader.writer)
	AssertEq(req.Name, chunkUploader.objectName)
	AssertEq(chunkSize, chunkUploader.writer.ChunkSize)

	return chunkUploader
}

func (t *ChunkUploaderTest) testUpload(uploader *chunkUploader, contents io.Reader) {
	err := uploader.Upload(context.Background(), contents)
	AssertEq(nil, err)
	AssertEq(Uploading, uploader.state)
}

func (t *ChunkUploaderTest) testUploadEmptyContent(uploader *chunkUploader, contents io.Reader) {
	err := uploader.Upload(context.Background(), contents)
	AssertEq(nil, err)
}

func (t *ChunkUploaderTest) testFailedUpload(uploader *chunkUploader, contents io.Reader) {
	err := uploader.Upload(context.Background(), contents)
	AssertNe(nil, err)
	AssertEq(UploadError, uploader.state)
}

func (t *ChunkUploaderTest) testClose(uploader *chunkUploader, objName string, len int) {
	o, err := uploader.Close(context.Background())
	AssertEq(nil, err)
	AssertNe(nil, o)
	AssertEq(Closed, uploader.state)
	AssertEq(objName, o.Name)
	AssertEq(len, o.Size)
}

func (t *ChunkUploaderTest) testFailedClose(uploader *chunkUploader, expectedErrStr string) {
	_, err := uploader.Close(context.Background())
	AssertNe(nil, err)
	AssertTrue(uploader.state == Closed)
	if !strings.Contains(err.Error(), expectedErrStr) {
		AddFailure("chunkUploader.Close() failed with wrong error. "+
			"Expected error-substring: \"%s\", Actual error-string: \"%s\"",
			expectedErrStr, err.Error())
	}
}

type failingContentReader struct{}

func (ilcr *failingContentReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("read failed as intended")
}

func genRandomContent(size int) string {
	b := make([]byte, size)

	_, err := rand.Read(b)
	AssertEq(nil, err)

	return string(b)
}

func numWholeChunksInContentSize(contentSize, chunkSize int) int {
	return int(math.Floor(float64(contentSize) / float64(chunkSize)))
}

func contentSizeFromChunkSizeMultiplier(chunkSize int, chunkSizeMultiplier float32) int {
	return int(float32(chunkSize) * chunkSizeMultiplier)
}

func createVarsForMultipleUploads(chunkSize int,
	chunkSizeMultipliersForUploads []float32) (contentsForNthUploads []string,
	numWholeChunksUptoNthUploads []int) {
	var totalContentSizesUptoNthUploads []int
	for i := range chunkSizeMultipliersForUploads {
		contentSizeForUpload := contentSizeFromChunkSizeMultiplier(chunkSize,
			chunkSizeMultipliersForUploads[i])
		content := genRandomContent(contentSizeForUpload)
		contentsForNthUploads = append(contentsForNthUploads, content)

		if i == 0 {
			totalContentSizesUptoNthUploads = append(totalContentSizesUptoNthUploads,
				contentSizeForUpload)
		} else {
			totalContentSizesUptoNthUploads = append(totalContentSizesUptoNthUploads,
				totalContentSizesUptoNthUploads[i-1]+contentSizeForUpload)
		}

		numWholeChunksUptoNthUploads =
			append(numWholeChunksUptoNthUploads,
				numWholeChunksInContentSize(totalContentSizesUptoNthUploads[i], chunkSize))
	}

	return
}

// This function invokes multiple uploads of different sizes,
// and different random contents using a chunkUploader.
//
// It verifies that all the upload calls succeeded, and
// for all of them, upload callbacks were called on every
// instance of content-size reaching multiples of chunkSize
// and that the BytesUploadedSoFar was updated correctly
// for the uploader.
//
// For example, let's take the following sequence of uploads.
//
// 1. In the first upload, we upload .25*chunkSize data,
// then no callback will be received during/after that upload.
//
// 2. In the next (second) upload,  we upload 0.5*chunkSize data,
// then again no callback will be received during/after that upload,
// as only 0.75*chunkSize data has been received so far by the uploader
// and a complete chunk has not been received yet.
//
// 3. In the next (third) callback, 1.5*chunkSize data is sent,
// so total 2.25 chunks were uploaded upto that upload.
// So, after this upload, we
// expect 2 callbacks to be received, if we wait long enough.
//
// 4. In the next (fourth) callback, .775*chunkSize data is sent,
// so total 3.025 chunks were uploaded upto that upload.
// So, after this upload, we
// expect 1 more callbacks to be received, if we wait long enough.
//
// This above test generalizes this scenario by taking any general
// set of uploads through chunk-size-multipliers for uploaders.
//
// Keep the multiplier values as multiples of 1/chunkSize for simplicity of floating-point
// calculations and comparisons.
func (t *ChunkUploaderTest) testUploadMultipleUploads(chunkSize int, chunkSizeMultipliersForUploads []float32) {
	// Set up inputs for uploads.
	numUploads := len(chunkSizeMultipliersForUploads)
	contentsForNthUploads, numWholeChunksUptoNthUploads :=
		createVarsForMultipleUploads(chunkSize, chunkSizeMultipliersForUploads)
	totalNumberOfWholeChunks := numWholeChunksUptoNthUploads[numUploads-1]
	numBytesSuccessfullyUploadedSoFar := make(chan int64, totalNumberOfWholeChunks)
	defer close(numBytesSuccessfullyUploadedSoFar)
	ctx := context.Background()
	var numCallbacks int
	uploader := t.testNewChunkUploader(ctx, t.obj, t.req, chunkSize,
		func(n int64) {
			numCallbacks++
			numBytesSuccessfullyUploadedSoFar <- n
		})

	chunkCallbackIndex := 0
	progressFuncCallbackTimeout := 10 * time.Millisecond
	for i := 0; i < numUploads; i++ {
		t.testUpload(uploader, strings.NewReader(contentsForNthUploads[i]))

		// For all the chunks completed during/after ith upload, confirm that its
		// callbacks were received and with right values of n.
		for ; chunkCallbackIndex < numWholeChunksUptoNthUploads[i]; chunkCallbackIndex++ {
			select {
			case n := <-numBytesSuccessfullyUploadedSoFar:
				AssertEq((chunkCallbackIndex+1)*chunkSize, n)
			case <-time.After(progressFuncCallbackTimeout):
				AddFailure("Did not not receive write progressFunc callback for "+
					"chunkSize=%v,chunkSizeMultipliersForUploads=%v,i=%v,chunkCallbackIndex=%v in %v",
					chunkSize, chunkSizeMultipliersForUploads, i, chunkCallbackIndex, progressFuncCallbackTimeout)
			}
		}

		AssertEq(chunkCallbackIndex, numCallbacks)
		AssertEq(chunkCallbackIndex*chunkSize, uploader.BytesUploadedSoFar())
	}
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ChunkUploaderTest) TestCreateWithImproperInputs() {
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
		errStr: "nil ObjectHandle or CreateObjectRequest",
	}, {
		ctx:    context.Background(),
		obj:    properObj,
		req:    properReq,
		errStr: "chunkSize <= 0",
	}, {
		ctx:       context.Background(),
		obj:       properObj,
		req:       diffCreateReq,
		errStr:    "names of passed ObjectHandle and CreateObjectRequest don't match",
		chunkSize: 100,
	},
	}

	for _, input := range inputs {
		uploader, err := NewChunkUploader(input.ctx, input.obj, input.req,
			input.chunkSize, input.progressFunc)

		AssertEq(nil, uploader)
		AssertNe(nil, err)
		AssertTrue(strings.Contains(err.Error(), input.errStr))
	}
}

func (t *ChunkUploaderTest) TestCreateWithProperInputs() {
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
		t.testNewChunkUploader(input.ctx, input.obj, input.req, input.chunkSize, input.progressFunc)
	}
}

func (t *ChunkUploaderTest) TestUploadWithTimeout() {
	obj := t.obj
	chunkSize := googleapi.MinUploadChunkSize
	contentSize := 10 * chunkSize
	content := genRandomContent(contentSize)

	uploader := t.testNewChunkUploader(context.Background(), obj, t.req, chunkSize, nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	err := uploader.Upload(ctx, strings.NewReader(content))
	AssertNe(nil, err)
}

func (t *ChunkUploaderTest) TestUploadEmptyContent() {
	// setup
	obj := t.obj
	chunkSize := googleapi.MinUploadChunkSize
	content := ""

	uploader := t.testNewChunkUploader(context.Background(), obj, t.req, chunkSize, nil)
	t.testUploadEmptyContent(uploader, strings.NewReader(content))

	AssertEq(0, uploader.totalUploadInitiatedSoFar)
}

func (t *ChunkUploaderTest) TestUploadFailingContentReader() {
	// setup
	fcr := &failingContentReader{}
	obj := t.obj
	chunkSize := googleapi.MinUploadChunkSize

	// test payload
	uploader := t.testNewChunkUploader(context.Background(), obj, t.req, chunkSize, nil)
	t.testFailedUpload(uploader, fcr)

	AssertEq(0, uploader.totalUploadInitiatedSoFar)
}

func (t *ChunkUploaderTest) TestUploadSingleSubChunkUpload() {
	// A sub-chunk means content is smaller than chunkSize in size.

	obj := t.obj
	chunkSize := googleapi.MinUploadChunkSize
	content := t.content
	contentSize := len(content)

	AssertLe(contentSize, chunkSize)

	numBytesSuccessfullyUploadedSoFar := make(chan int64, 1)
	defer close(numBytesSuccessfullyUploadedSoFar)
	uploader := t.testNewChunkUploader(context.Background(), obj, t.req, chunkSize,
		func(n int64) {
			numBytesSuccessfullyUploadedSoFar <- n
		})
	t.testUpload(uploader, strings.NewReader(content))

	AssertEq(contentSize, uploader.totalUploadInitiatedSoFar)

	progressFuncCallbackTimeout := 100 * time.Millisecond
	// Ensure that the progress callback is not called in this case
	// (i.e. when upload contentSize < chunkSize).
	// For this, we wait for sometime to see if the unexpected callback
	// comes. If it doesn't, we call
	// the test passing.
	// Alternatively we could put the AddFailure directly in the callback,
	// but then the test function would not wait for the unexpected
	// callback to be called at all,
	// and that would cause a false pass of this test.
	select {
	case <-numBytesSuccessfullyUploadedSoFar:
		AddFailure("Received unexpected progressFunc callback")
		break
	case <-time.After(progressFuncCallbackTimeout):
		break
	}
}

func (t *ChunkUploaderTest) TestUploadSingleChunkUpload() {
	chunkSize := googleapi.MinUploadChunkSize
	t.testUploadMultipleUploads(chunkSize, []float32{1})
}

func (t *ChunkUploaderTest) TestUploadSingleSuperChunkUpload() {
	chunkSize := googleapi.MinUploadChunkSize
	t.testUploadMultipleUploads(chunkSize, []float32{2.5})
}

func (t *ChunkUploaderTest) TestUploadMultipleHeterogenousUploads() {
	chunkSize := googleapi.MinUploadChunkSize
	t.testUploadMultipleUploads(chunkSize, []float32{0.25, .75, 1.5, .775, 2, 0.5})
}

func (t *ChunkUploaderTest) TestUploadMultipleHomogeneousUploads() {
	chunkSize := googleapi.MinUploadChunkSize
	t.testUploadMultipleUploads(chunkSize, []float32{1, 1, 1, 1, 1, 1})
}

func (t *ChunkUploaderTest) TestCloseWithoutWrite() {
	obj := t.obj
	uploader := t.testNewChunkUploader(context.Background(), obj, t.req, 1000, nil)
	t.testClose(uploader, t.req.Name, 0)
}

func (t *ChunkUploaderTest) TestCloseWithTimeout() {
	obj := t.obj
	chunkSize := googleapi.MinUploadChunkSize
	contentSize := 10 * chunkSize
	content := genRandomContent(contentSize)
	uploader := t.testNewChunkUploader(context.Background(), obj, t.req, chunkSize, nil)
	t.testUpload(uploader, strings.NewReader(content))
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	_, err := uploader.Close(ctx)
	AssertNe(nil, err)
}

func (t *ChunkUploaderTest) TestDoubleClosure() {
	obj := t.obj
	uploader := t.testNewChunkUploader(context.Background(), obj, t.req, 1000, nil)

	t.testClose(uploader, t.req.Name, 0)

	// 2nd closure, expected to fail.
	t.testFailedClose(uploader,
		fmt.Sprintf("improper state (%v) for finalizing object", Closed))
}

func (t *ChunkUploaderTest) TestCloseWithSingleSubChunkUpload() {
	obj := t.obj
	chunkSize := googleapi.MinUploadChunkSize
	contentSize := len(t.content)

	uploader := t.testNewChunkUploader(context.Background(), obj, t.req, chunkSize, nil)
	t.testUpload(uploader, t.req.Contents)

	t.testClose(uploader, t.req.Name, contentSize)
}

func (t *ChunkUploaderTest) TestCloseWithMultipleSingleChunkUploads() {
	obj := t.obj
	chunkSize := 1000
	contents := [6]string{"hfue", "yrf8934h9", "iru328ry",
		"iy3rh34r489y8", "ie32hr83hr43rt9y8", "i9r38ry2u9j9"}
	var contentSize int

	uploader := t.testNewChunkUploader(context.Background(), obj, t.req, chunkSize, nil)

	for _, content := range contents {
		t.testUpload(uploader, strings.NewReader(content))
		contentSize += len(content)
	}

	t.testClose(uploader, t.req.Name, contentSize)
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

	uploader := t.testNewChunkUploader(context.Background(), obj, t.req, chunkSize, nil)

	for _, content := range contents {
		t.testUpload(uploader, strings.NewReader(content))
		contentSize += len(content)
	}

	t.testClose(uploader, t.req.Name, contentSize)
}

func (t *ChunkUploaderTest) TestCloseFailedUploader() {
	// setup
	fcr := &failingContentReader{}
	obj := t.obj
	chunkSize := googleapi.MinUploadChunkSize

	uploader := t.testNewChunkUploader(context.Background(), obj, t.req, chunkSize, nil)

	// A successful upload.
	uploadedContent := t.content
	t.testUpload(uploader, strings.NewReader(uploadedContent))

	// test payload - forces failure of upload.
	t.testFailedUpload(uploader, fcr)

	// The next upload will fail simply because of the WriteError in status.
	failedContent := "jifhe4guyfbhufg78gufhewh7fgeyuig"
	t.testFailedUpload(uploader, strings.NewReader(failedContent))

	// Close will pass despite WriteError in status.
	t.testClose(uploader, t.req.Name, len(uploadedContent))
}
