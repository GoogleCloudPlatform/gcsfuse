// This file was auto-generated using createmock. See the following page for
// more information:
//
//     https://github.com/jacobsa/oglemock
//

package mock_gcscaching

import (
	fmt "fmt"
	gcs "github.com/jacobsa/gcloud/gcs"
	gcscaching "github.com/jacobsa/gcloud/gcs/gcscaching"
	oglemock "github.com/jacobsa/oglemock"
	runtime "runtime"
	time "time"
	unsafe "unsafe"
)

type MockStatCache interface {
	gcscaching.StatCache
	oglemock.MockObject
}

type mockStatCache struct {
	controller  oglemock.Controller
	description string
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

	return
}

func (m *mockStatCache) CheckInvariants() {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"CheckInvariants",
		file,
		line,
		[]interface{}{})

	if len(retVals) != 0 {
		panic(fmt.Sprintf("mockStatCache.CheckInvariants: invalid return values: %v", retVals))
	}

	return
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

	return
}

func (m *mockStatCache) Insert(p0 *gcs.Object, p1 time.Time) {
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

	return
}

func (m *mockStatCache) LookUp(p0 string, p1 time.Time) (o0 bool, o1 *gcs.Object) {
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

	// o1 *gcs.Object
	if retVals[1] != nil {
		o1 = retVals[1].(*gcs.Object)
	}

	return
}
