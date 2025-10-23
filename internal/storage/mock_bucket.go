// This file was auto-generated using createmock. See the following page for
// more information:
//
//     https://github.com/jacobsa/oglemock
//

package storage

import (
	fmt "fmt"
	runtime "runtime"
	unsafe "unsafe"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	oglemock "github.com/jacobsa/oglemock"
	context "golang.org/x/net/context"
)

type MockBucket interface {
	gcs.Bucket
	oglemock.MockObject
}

type mockBucket struct {
	controller  oglemock.Controller
	description string
}

// Deprecated: Please use testify_mock_bucket.go instead.
func NewMockBucket(
	c oglemock.Controller,
	desc string) MockBucket {
	return &mockBucket{
		controller:  c,
		description: desc,
	}
}

func (m *mockBucket) Oglemock_Id() uintptr {
	return uintptr(unsafe.Pointer(m))
}

func (m *mockBucket) Oglemock_Description() string {
	return m.description
}

func (m *mockBucket) ComposeObjects(p0 context.Context, p1 *gcs.ComposeObjectsRequest) (o0 *gcs.Object, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"ComposeObjects",
		file,
		line,
		[]any{p0, p1})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockBucket.ComposeObjects: invalid return values: %v", retVals))
	}

	// o0 *Object
	if retVals[0] != nil {
		o0 = retVals[0].(*gcs.Object)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

func (m *mockBucket) CopyObject(p0 context.Context, p1 *gcs.CopyObjectRequest) (o0 *gcs.Object, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"CopyObject",
		file,
		line,
		[]any{p0, p1})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockBucket.CopyObject: invalid return values: %v", retVals))
	}

	// o0 *Object
	if retVals[0] != nil {
		o0 = retVals[0].(*gcs.Object)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

func (m *mockBucket) CreateObject(p0 context.Context, p1 *gcs.CreateObjectRequest) (o0 *gcs.Object, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"CreateObject",
		file,
		line,
		[]any{p0, p1})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockBucket.CreateObject: invalid return values: %v", retVals))
	}

	// o0 *Object
	if retVals[0] != nil {
		o0 = retVals[0].(*gcs.Object)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

func (m *mockBucket) CreateObjectChunkWriter(p0 context.Context, p1 *gcs.CreateObjectRequest, p2 int, p3 func(bytesUploadedSoFar int64)) (o0 gcs.Writer, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"CreateObjectChunkWriter",
		file,
		line,
		[]any{p0, p1, p2, p3})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockBucket.CreateObjectChunkWriter: invalid return values: %v", retVals))
	}

	// o0 *storageWriter
	if retVals[0] != nil {
		o0 = retVals[0].(gcs.Writer)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

func (m *mockBucket) CreateAppendableObjectWriter(p0 context.Context, p1 *gcs.CreateObjectChunkWriterRequest) (o0 gcs.Writer, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"CreateAppendableObjectWriter",
		file,
		line,
		[]any{p0, p1})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockBucket.CreateAppendableObjectWriter: invalid return values: %v", retVals))
	}

	// o0 storageWriter
	if retVals[0] != nil {
		o0 = retVals[0].(gcs.Writer)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

func (m *mockBucket) FinalizeUpload(p0 context.Context, p1 gcs.Writer) (o0 *gcs.MinObject, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"FinalizeUpload",
		file,
		line,
		[]any{p0, p1})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockBucket.FinalizeUpload: invalid return values: %v", retVals))
	}

	// o0 *gcs.Object
	if retVals[0] != nil {
		o0 = retVals[0].(*gcs.MinObject)
	}
	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

func (m *mockBucket) FlushPendingWrites(p0 context.Context, p1 gcs.Writer) (o0 *gcs.MinObject, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"FlushPendingWrites",
		file,
		line,
		[]any{p0, p1})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockBucket.FlushPendingWrites: invalid return values: %v", retVals))
	}

	// o0 *gcs.MinObject
	if retVals[0] != nil {
		o0 = retVals[0].(*gcs.MinObject)
	}
	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

func (m *mockBucket) DeleteObject(p0 context.Context, p1 *gcs.DeleteObjectRequest) (o0 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"DeleteObject",
		file,
		line,
		[]any{p0, p1})

	if len(retVals) != 1 {
		panic(fmt.Sprintf("mockBucket.DeleteObject: invalid return values: %v", retVals))
	}

	// o0 error
	if retVals[0] != nil {
		o0 = retVals[0].(error)
	}

	return
}

func (m *mockBucket) MoveObject(p0 context.Context, p1 *gcs.MoveObjectRequest) (*gcs.Object, error) {
	var o0 *gcs.Object
	var o1 error
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"MoveObject",
		file,
		line,
		[]any{p0, p1})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockBucket.MoveObject: invalid return values: %v", retVals))
	}

	// o0 *Object
	if retVals[0] != nil {
		o0 = retVals[0].(*gcs.Object)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return o0, o1
}

func (m *mockBucket) DeleteFolder(ctx context.Context, folderName string) (o0 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"DeleteFolder",
		file,
		line,
		[]any{ctx, folderName})
	if len(retVals) != 1 {
		panic(fmt.Sprintf("mockBucket.DeleteFolder: invalid return values: %v", retVals))
	}
	// o0 string
	if retVals[0] != nil {
		o0 = retVals[0].(error)
	}
	return
}

func (m *mockBucket) ListObjects(p0 context.Context, p1 *gcs.ListObjectsRequest) (o0 *gcs.Listing, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"ListObjects",
		file,
		line,
		[]any{p0, p1})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockBucket.ListObjects: invalid return values: %v", retVals))
	}

	// o0 *Listing
	if retVals[0] != nil {
		o0 = retVals[0].(*gcs.Listing)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

func (m *mockBucket) Name() (o0 string) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Name",
		file,
		line,
		[]any{})

	if len(retVals) != 1 {
		panic(fmt.Sprintf("mockBucket.Name: invalid return values: %v", retVals))
	}

	// o0 string
	if retVals[0] != nil {
		o0 = retVals[0].(string)
	}

	return
}

func (m *mockBucket) BucketType() (o0 gcs.BucketType) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"BucketType",
		file,
		line,
		[]any{})

	if len(retVals) != 1 {
		panic(fmt.Sprintf("mockBucket.BucketType: invalid return values: %v", retVals))
	}

	// o0 string
	if retVals[0] != nil {
		o0 = retVals[0].(gcs.BucketType)
	}

	return
}

func (m *mockBucket) NewReaderWithReadHandle(p0 context.Context, p1 *gcs.ReadObjectRequest) (o0 gcs.StorageReader, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"NewReaderWithReadHandle",
		file,
		line,
		[]any{p0, p1})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockBucket.NewReaderWithReadHandle: invalid return values: %v", retVals))
	}

	// o0 gcs.StorageReader
	if retVals[0] != nil {
		o0 = retVals[0].(gcs.StorageReader)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

func (m *mockBucket) StatObject(p0 context.Context,
	p1 *gcs.StatObjectRequest) (o0 *gcs.MinObject, o1 *gcs.ExtendedObjectAttributes, o2 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"StatObject",
		file,
		line,
		[]any{p0, p1})

	if len(retVals) != 3 {
		panic(fmt.Sprintf("mockBucket.StatObject: invalid return values: %v", retVals))
	}

	// o0 *MinObject
	if retVals[0] != nil {
		o0 = retVals[0].(*gcs.MinObject)
	}

	// o1 *ExtendedObjectAttributes
	if retVals[1] != nil {
		o1 = retVals[1].(*gcs.ExtendedObjectAttributes)
	}

	// o2 error
	if retVals[2] != nil {
		o2 = retVals[2].(error)
	}

	return
}

func (m *mockBucket) UpdateObject(p0 context.Context, p1 *gcs.UpdateObjectRequest) (o0 *gcs.Object, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"UpdateObject",
		file,
		line,
		[]any{p0, p1})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockBucket.UpdateObject: invalid return values: %v", retVals))
	}

	// o0 *Object
	if retVals[0] != nil {
		o0 = retVals[0].(*gcs.Object)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

func (m *mockBucket) GetFolder(
	ctx context.Context,
	prefix string) (o0 *gcs.Folder, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"GetFolder",
		file,
		line,
		[]any{ctx, prefix})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockBucket.GetFolder: invalid return values: %v", retVals))
	}

	if retVals[0] != nil {
		o0 = retVals[0].(*gcs.Folder)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}
	return
}

func (m *mockBucket) CreateFolder(ctx context.Context, prefix string) (o0 *gcs.Folder, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"CreateFolder",
		file,
		line,
		[]any{ctx, prefix})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockBucket.GetFolder: invalid return values: %v", retVals))
	}

	if retVals[0] != nil {
		o0 = retVals[0].(*gcs.Folder)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}
	return
}

func (m *mockBucket) RenameFolder(ctx context.Context, folderName string, destinationFolderId string) (o0 *gcs.Folder, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"RenameFolder",
		file,
		line,
		[]any{ctx, folderName, destinationFolderId})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockBucket.RenameFolder: invalid return values: %v", retVals))
	}
	// o0 string
	if retVals[0] != nil {
		o0 = retVals[0].(*gcs.Folder)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}
	return
}

func (m *mockBucket) GCSName(obj *gcs.MinObject) string {
	return obj.Name
}

func (m *mockBucket) NewMultiRangeDownloader(
	p0 context.Context, p1 *gcs.MultiRangeDownloaderRequest) (o0 gcs.MultiRangeDownloader, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"NewMultiRangeDownloader",
		file,
		line,
		[]any{p0, p1})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockBucket.NewMultiRangeDownloader: invalid return values: %v", retVals))
	}

	// o0 io.ReadCloser
	if retVals[0] != nil {
		o0 = retVals[0].(gcs.MultiRangeDownloader)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}
