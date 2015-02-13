// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package fstesting

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/ogletest"
	"github.com/jacobsa/ogletest/srcutil"
)

// An interface that all FS tests must implement.
type fsTestInterface interface {
	setUpFsTest(b gcs.Bucket)
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
	makeBucket func() gcs.Bucket,
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
			bucket := makeBucket()
			instance.Interface().(fsTestInterface).setUpFsTest(bucket)
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

// Given a function that returns an initialized, empty bucket, register test
// suites that exercise a file system wrapping that bucket. The condition name
// should be something like "RealGCS" or "FakeGCS".
func RegisterFSTests(conditionName string, makeBucket func() gcs.Bucket) {
	// A list of empty instances of the test suites we want to register.
	suitePrototypes := []fsTestInterface{
		&readOnlyTest{},
	}

	// Register each.
	for _, suitePrototype := range suitePrototypes {
		registerTestSuite(conditionName, makeBucket, suitePrototype)
	}
}
