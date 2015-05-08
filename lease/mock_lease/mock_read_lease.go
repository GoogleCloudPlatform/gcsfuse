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
	runtime "runtime"
	unsafe "unsafe"
)

type MockReadLease interface {
	lease.ReadLease
	oglemock.MockObject
}

type mockReadLease struct {
	controller  oglemock.Controller
	description string
}

func NewMockReadLease(
	c oglemock.Controller,
	desc string) MockReadLease {
	return &mockReadLease{
		controller:  c,
		description: desc,
	}
}

func (m *mockReadLease) Oglemock_Id() uintptr {
	return uintptr(unsafe.Pointer(m))
}

func (m *mockReadLease) Oglemock_Description() string {
	return m.description
}

func (m *mockReadLease) Read(p0 []uint8) (o0 int, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Read",
		file,
		line,
		[]interface{}{p0})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockReadLease.Read: invalid return values: %v", retVals))
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

func (m *mockReadLease) ReadAt(p0 []uint8, p1 int64) (o0 int, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"ReadAt",
		file,
		line,
		[]interface{}{p0, p1})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockReadLease.ReadAt: invalid return values: %v", retVals))
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

func (m *mockReadLease) Revoke() {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Revoke",
		file,
		line,
		[]interface{}{})

	if len(retVals) != 0 {
		panic(fmt.Sprintf("mockReadLease.Revoke: invalid return values: %v", retVals))
	}

	return
}

func (m *mockReadLease) Revoked() (o0 bool) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Revoked",
		file,
		line,
		[]interface{}{})

	if len(retVals) != 1 {
		panic(fmt.Sprintf("mockReadLease.Revoked: invalid return values: %v", retVals))
	}

	// o0 bool
	if retVals[0] != nil {
		o0 = retVals[0].(bool)
	}

	return
}

func (m *mockReadLease) Seek(p0 int64, p1 int) (o0 int64, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Seek",
		file,
		line,
		[]interface{}{p0, p1})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockReadLease.Seek: invalid return values: %v", retVals))
	}

	// o0 int64
	if retVals[0] != nil {
		o0 = retVals[0].(int64)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

func (m *mockReadLease) Size() (o0 int64) {
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
		panic(fmt.Sprintf("mockReadLease.Size: invalid return values: %v", retVals))
	}

	// o0 int64
	if retVals[0] != nil {
		o0 = retVals[0].(int64)
	}

	return
}

func (m *mockReadLease) Upgrade() (o0 lease.ReadWriteLease, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Upgrade",
		file,
		line,
		[]interface{}{})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockReadLease.Upgrade: invalid return values: %v", retVals))
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
