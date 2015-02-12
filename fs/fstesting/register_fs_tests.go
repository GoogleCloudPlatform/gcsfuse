// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package fstesting

import (
	"reflect"
	"strings"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/ogletest"
	"github.com/jacobsa/ogletest/srcutil"
)

// An interface that all FS tests must implement.
type fsTestSetUpInterface interface {
	setUpFsTest(b gcs.Bucket)
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
	makeBucket func() gcs.Bucket,
	prototype fsTestSetUpInterface) {
	suitePointerType := reflect.TypeOf(prototype)
	suiteType := suitePointerType.Elem()

	// We don't need anything fancy at the suite level.
	var ts ogletest.TestSuite
	ts.Name = getSuiteName(suiteType)

	// For each method, we create a test function.
	for _, method := range getTestMethods(suitePointerType) {
		var tf ogletest.TestFunction
		tf.Name = method.Name

		// Create an instance to be shared among SetUp and the test function itself.
		var instance reflect.Value = reflect.New(suiteType)

		// SetUp should create a bucket and then initialize the suite object,
		// remembering that the suite implements fsTestSetUpInterface.
		tf.SetUp = func(*ogletest.TestInfo) {
			bucket := makeBucket()
			instance.Interface().(fsTestSetUpInterface).setUpFsTest(bucket)
		}

		// The test function itself should simply invoke the method.
		methodCopy := method
		tf.Run = func() {
			methodCopy.Func.Call([]reflect.Value{instance})
		}

		// Save the test function.
		ts.TestFunctions = append(ts.TestFunctions, tf)
	}

	// Register the suite.
	ogletest.Register(ts)
}

// Given a function that returns an initialized, empty bucket, register test
// suites that exercise a file system wrapping that bucket.
func RegisterFSTests(makeBucket func() gcs.Bucket) {
	// A list of empty instances of the test suites we want to register.
	suitePrototypes := []fsTestSetUpInterface{
		&readOnlyTest{},
	}

	// Register each.
	for _, suitePrototype := range suitePrototypes {
		registerTestSuite(makeBucket, suitePrototype)
	}
}
