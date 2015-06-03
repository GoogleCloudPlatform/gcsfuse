// This file was auto-generated using createmock. See the following page for
// more information:
//
//     https://github.com/jacobsa/oglemock
//

package mock_mutable

import (
	fmt "fmt"
	lease "github.com/googlecloudplatform/gcsfuse/lease"
	mutable "github.com/googlecloudplatform/gcsfuse/mutable"
	oglemock "github.com/jacobsa/oglemock"
	context "golang.org/x/net/context"
	runtime "runtime"
	unsafe "unsafe"
)

type MockContent interface {
	mutable.Content
	oglemock.MockObject
}

type mockContent struct {
	controller  oglemock.Controller
	description string
}

func NewMockContent(
	c oglemock.Controller,
	desc string) MockContent {
	return &mockContent{
		controller:  c,
		description: desc,
	}
}

func (m *mockContent) Oglemock_Id() uintptr {
	return uintptr(unsafe.Pointer(m))
}

func (m *mockContent) Oglemock_Description() string {
	return m.description
}

func (m *mockContent) CheckInvariants() {
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
		panic(fmt.Sprintf("mockContent.CheckInvariants: invalid return values: %v", retVals))
	}

	return
}

func (m *mockContent) Destroy() {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Destroy",
		file,
		line,
		[]interface{}{})

	if len(retVals) != 0 {
		panic(fmt.Sprintf("mockContent.Destroy: invalid return values: %v", retVals))
	}

	return
}

func (m *mockContent) ReadAt(p0 context.Context, p1 []uint8, p2 int64) (o0 int, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"ReadAt",
		file,
		line,
		[]interface{}{p0, p1, p2})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockContent.ReadAt: invalid return values: %v", retVals))
	}

	// o0 int
	if retVals[0] != nil {
		o0 = retVals[0].(int)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

func (m *mockContent) Release() (o0 lease.ReadWriteLease) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Release",
		file,
		line,
		[]interface{}{})

	if len(retVals) != 1 {
		panic(fmt.Sprintf("mockContent.Release: invalid return values: %v", retVals))
	}

	// o0 lease.ReadWriteLease
	if retVals[0] != nil {
		o0 = retVals[0].(lease.ReadWriteLease)
	}

	return
}

func (m *mockContent) Stat(p0 context.Context) (o0 mutable.StatResult, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Stat",
		file,
		line,
		[]interface{}{p0})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockContent.Stat: invalid return values: %v", retVals))
	}

	// o0 mutable.StatResult
	if retVals[0] != nil {
		o0 = retVals[0].(mutable.StatResult)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

func (m *mockContent) Truncate(p0 context.Context, p1 int64) (o0 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Truncate",
		file,
		line,
		[]interface{}{p0, p1})

	if len(retVals) != 1 {
		panic(fmt.Sprintf("mockContent.Truncate: invalid return values: %v", retVals))
	}

	// o0 error
	if retVals[0] != nil {
		o0 = retVals[0].(error)
	}

	return
}

func (m *mockContent) WriteAt(p0 context.Context, p1 []uint8, p2 int64) (o0 int, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"WriteAt",
		file,
		line,
		[]interface{}{p0, p1, p2})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockContent.WriteAt: invalid return values: %v", retVals))
	}

	// o0 int
	if retVals[0] != nil {
		o0 = retVals[0].(int)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}
