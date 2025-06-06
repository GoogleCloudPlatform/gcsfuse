// This file was auto-generated using createmock. See the following page for
// more information:
//
//     https://github.com/jacobsa/oglemock
//

package mock_gcscaching

import (
	fmt "fmt"
	runtime "runtime"
	time "time"
	unsafe "unsafe"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	oglemock "github.com/jacobsa/oglemock"
)

type MockStatCache interface {
	metadata.StatCache
	oglemock.MockObject
}

type mockStatCache struct {
	controller  oglemock.Controller
	description string
}

func (m *mockStatCache) AddNegativeEntryForFolder(p0 string, p1 time.Time) {
	// Get a folder name and line number for the caller.
	_, name, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"AddNegativeEntryForFolder",
		name,
		line,
		[]interface{}{p0, p1})

	if len(retVals) != 0 {
		panic(fmt.Sprintf("mockStatCache.AddNegativeEntryforFolder: invalid return values: %v", retVals))
	}
}

func NewMockStatCache(
	c oglemock.Controller,
	desc string) MockStatCache {
	return &mockStatCache{
		controller:  c,
		description: desc,
	}
}

func (m *mockStatCache) Oglemock_Id() uintptr {
	return uintptr(unsafe.Pointer(m))
}

func (m *mockStatCache) Oglemock_Description() string {
	return m.description
}

func (m *mockStatCache) AddNegativeEntry(p0 string, p1 time.Time) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"AddNegativeEntry",
		file,
		line,
		[]interface{}{p0, p1})

	if len(retVals) != 0 {
		panic(fmt.Sprintf("mockStatCache.AddNegativeEntry: invalid return values: %v", retVals))
	}
}

func (m *mockStatCache) Erase(p0 string) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Erase",
		file,
		line,
		[]interface{}{p0})

	if len(retVals) != 0 {
		panic(fmt.Sprintf("mockStatCache.Erase: invalid return values: %v", retVals))
	}
}

func (m *mockStatCache) Insert(p0 *gcs.MinObject, p1 time.Time) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Insert",
		file,
		line,
		[]interface{}{p0, p1})

	if len(retVals) != 0 {
		panic(fmt.Sprintf("mockStatCache.Insert: invalid return values: %v", retVals))
	}
}

func (m *mockStatCache) LookUp(p0 string, p1 time.Time) (o0 bool, o1 *gcs.MinObject) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"LookUp",
		file,
		line,
		[]interface{}{p0, p1})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockStatCache.LookUp: invalid return values: %v", retVals))
	}

	// o0 bool
	if retVals[0] != nil {
		o0 = retVals[0].(bool)
	}

	// o1 *gcs.MinObject
	if retVals[1] != nil {
		o1 = retVals[1].(*gcs.MinObject)
	}

	return
}

func (m *mockStatCache) InsertFolder(p0 *gcs.Folder, p1 time.Time) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"InsertFolder",
		file,
		line,
		[]interface{}{p0, p1})

	if len(retVals) != 0 {
		panic(fmt.Sprintf("mockStatCache.InsertFolder: invalid return values: %v", retVals))
	}
}

func (m *mockStatCache) LookUpFolder(p0 string, p1 time.Time) (o0 bool, o1 *gcs.Folder) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"LookUpFolder",
		file,
		line,
		[]interface{}{p0, p1})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockStatCache.LookUpFolder: invalid return values: %v", retVals))
	}

	// o0 bool
	if retVals[0] != nil {
		o0 = retVals[0].(bool)
	}

	// o1 *gcs.Folder
	if retVals[1] != nil {
		o1 = retVals[1].(*gcs.Folder)
	}

	return
}

func (m *mockStatCache) EraseEntriesWithGivenPrefix(p0 string) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"EraseEntriesWithGivenPrefix",
		file,
		line,
		[]interface{}{p0})

	if len(retVals) != 0 {
		panic(fmt.Sprintf("mockStatCache.LookUpFolder: invalid return values: %v", retVals))
	}
}
