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

type MockFileLeaser interface {
	lease.FileLeaser
	oglemock.MockObject
}

type mockFileLeaser struct {
	controller  oglemock.Controller
	description string
}

func NewMockFileLeaser(
	c oglemock.Controller,
	desc string) MockFileLeaser {
	return &mockFileLeaser{
		controller:  c,
		description: desc,
	}
}

func (m *mockFileLeaser) Oglemock_Id() uintptr {
	return uintptr(unsafe.Pointer(m))
}

func (m *mockFileLeaser) Oglemock_Description() string {
	return m.description
}

func (m *mockFileLeaser) NewFile() (o0 lease.ReadWriteLease, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"NewFile",
		file,
		line,
		[]interface{}{})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockFileLeaser.NewFile: invalid return values: %v", retVals))
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
