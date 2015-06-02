// This file was auto-generated using createmock. See the following page for
// more information:
//
//     https://github.com/jacobsa/oglemock
//

package mock_lease

import (
	fmt "fmt"
	lease "github.com/googlecloudplatform/gcsfuse/lease"
	oglemock "github.com/jacobsa/oglemock"
	context "golang.org/x/net/context"
	runtime "runtime"
	unsafe "unsafe"
)

type MockReadProxy interface {
	lease.ReadProxy
	oglemock.MockObject
}

type mockReadProxy struct {
	controller  oglemock.Controller
	description string
}

func NewMockReadProxy(
	c oglemock.Controller,
	desc string) MockReadProxy {
	return &mockReadProxy{
		controller:  c,
		description: desc,
	}
}

func (m *mockReadProxy) Oglemock_Id() uintptr {
	return uintptr(unsafe.Pointer(m))
}

func (m *mockReadProxy) Oglemock_Description() string {
	return m.description
}

func (m *mockReadProxy) CheckInvariants() {
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
		panic(fmt.Sprintf("mockReadProxy.CheckInvariants: invalid return values: %v", retVals))
	}

	return
}

func (m *mockReadProxy) Destroy() {
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
		panic(fmt.Sprintf("mockReadProxy.Destroy: invalid return values: %v", retVals))
	}

	return
}

func (m *mockReadProxy) ReadAt(p0 context.Context, p1 []uint8, p2 int64) (o0 int, o1 error) {
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
		panic(fmt.Sprintf("mockReadProxy.ReadAt: invalid return values: %v", retVals))
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

func (m *mockReadProxy) Size() (o0 int64) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Size",
		file,
		line,
		[]interface{}{})

	if len(retVals) != 1 {
		panic(fmt.Sprintf("mockReadProxy.Size: invalid return values: %v", retVals))
	}

	// o0 int64
	if retVals[0] != nil {
		o0 = retVals[0].(int64)
	}

	return
}

func (m *mockReadProxy) Upgrade(p0 context.Context) (o0 lease.ReadWriteLease, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Upgrade",
		file,
		line,
		[]interface{}{p0})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockReadProxy.Upgrade: invalid return values: %v", retVals))
	}

	// o0 lease.ReadWriteLease
	if retVals[0] != nil {
		o0 = retVals[0].(lease.ReadWriteLease)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}
