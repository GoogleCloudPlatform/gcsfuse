package functional_tests_test

import (
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/test_suite"
	"testing"
)

type testStruct struct {
}

func (s *testStruct) Setup(t *testing.T) {
	t.Log("Per-test setup code")
}
func (s *testStruct) Teardown(t *testing.T) {
	t.Log("Per-test teardown code")
}

func (s *testStruct) TestSomething(t *testing.T) {
	t.Log("TestSomething")
}

func (s *testStruct) TestSomethingElse(t *testing.T) {
	t.Log("TestSomethingElse")
}

func Test(t *testing.T) {
	test_suite.RunSubTests(t, &testStruct{})
}
