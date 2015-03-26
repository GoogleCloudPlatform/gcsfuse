// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fstesting

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync"

	"github.com/googlecloudplatform/gcsfuse/fs"
	"github.com/jacobsa/ogletest"
	"github.com/jacobsa/ogletest/srcutil"
)

// Dependencies and configration needed by the tests registered by
// RegisterFSTests.
type FSTestConfig struct {
	// Configuration for the fuse server to be created.
	ServerConfig fs.ServerConfig
}

// An interface that all FS tests must implement.
type fsTestInterface interface {
	setUpFSTest(config FSTestConfig)
	tearDownFsTest()
}

func getSuiteName(suiteType reflect.Type) string {
	return strings.Title(suiteType.Name())
}

func isExported(name string) bool {
	return len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z'
}

func getTestMethods(suitePointerType reflect.Type) []reflect.Method {
	var exportedMethods []reflect.Method
	for _, m := range srcutil.GetMethodsInSourceOrder(suitePointerType) {
		if isExported(m.Name) {
			exportedMethods = append(exportedMethods, m)
		}
	}

	return exportedMethods
}

func registerTestSuite(
	conditionName string,
	makeConfig func() FSTestConfig,
	prototype fsTestInterface) {
	suitePointerType := reflect.TypeOf(prototype)
	suiteType := suitePointerType.Elem()

	// We don't need anything fancy at the suite level.
	var ts ogletest.TestSuite
	ts.Name = fmt.Sprintf("%s.%s", conditionName, getSuiteName(suiteType))

	// For each method, we create a test function.
	for _, method := range getTestMethods(suitePointerType) {
		var tf ogletest.TestFunction
		tf.Name = method.Name

		// Create an instance to be shared among SetUp and the test function itself.
		var instance reflect.Value = reflect.New(suiteType)

		// SetUp should create a bucket and then initialize the suite object,
		// remembering that the suite implements fsTestInterface.
		tf.SetUp = func(*ogletest.TestInfo) {
			instance.Interface().(fsTestInterface).setUpFSTest(makeConfig())
		}

		// The test function itself should simply invoke the method.
		methodCopy := method
		tf.Run = func() {
			methodCopy.Func.Call([]reflect.Value{instance})
		}

		// TearDown should work much like SetUp.
		tf.TearDown = func() {
			instance.Interface().(fsTestInterface).tearDownFsTest()
		}

		// Save the test function.
		ts.TestFunctions = append(ts.TestFunctions, tf)
	}

	// Register the suite.
	ogletest.Register(ts)
}

// Given a function that returns appropriate test config, register test suites
// that exercise a file system created from that config. The condition name
// should be something like "RealGCS" or "FakeGCS".
func RegisterFSTests(conditionName string, makeConfig func() FSTestConfig) {
	ensureSignalHandler()

	// A list of empty instances of the test suites we want to register.
	suitePrototypes := []fsTestInterface{
		&foreignModsTest{},
		&implicitDirsTest{},
		&openTest{},
		&modesTest{},
		&directoryTest{},
		&fileTest{},
	}

	// Register each.
	for _, suitePrototype := range suitePrototypes {
		registerTestSuite(conditionName, makeConfig, suitePrototype)
	}
}

var signalHandlerOnce sync.Once

// Make sure the user doesn't freeze the program by hitting Ctrl-C, which is
// frustrating.
//
// Why does the freeze happen? When the kernel goes to clean up after the
// process, it attempts to close its files. That involves telling the kernel
// fuse module to flush, which in turn attempts to contact the process. But
// the user mode end of fuse in the process is already dead, so we have a
// deadlock with a kernel stack that looks something like this:
//
//     [<ffffffff812aa77a>] wait_answer_interruptible+0x6a/0xa0
//     [<ffffffff812aab7b>] __fuse_request_send+0x1fb/0x280
//     [<ffffffff812aac12>] fuse_request_send+0x12/0x20
//     [<ffffffff812b3837>] fuse_flush+0xd7/0x120
//     [<ffffffff811ba79f>] filp_close+0x2f/0x70
//     [<ffffffff811daad8>] put_files_struct+0x88/0xe0
//     [<ffffffff811dabd7>] exit_files+0x47/0x50
//     [<ffffffff81069c86>] do_exit+0x296/0xa50
//     [<ffffffff8106a4bf>] do_group_exit+0x3f/0xa0
//     [<ffffffff8106a534>] SyS_exit_group+0x14/0x20
//     [<ffffffff8172f82d>] system_call_fastpath+0x1a/0x1f
//     [<ffffffffffffffff>] 0xffffffffffffffff
//
// SIGKILL probably does the same thing, but we can't catch that. If a
// process gets into this state, the only way to kill it for sure is to write
// to the 'abort' file in the appropriate sub-directory of
// /sys/fs/fuse/connections, as documented here:
//
//     https://www.kernel.org/doc/Documentation/filesystems/fuse.txt
//
func ensureSignalHandler() {
	signalHandlerOnce.Do(func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt)

		go func() {
			for {
				<-sigChan
				fmt.Println("SIGINT is a bad idea; it will cause freezes.")
				fmt.Println("See register_fs_tests.go for details.")
			}
		}()
	})
}
