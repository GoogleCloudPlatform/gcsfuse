package storage

import (
	"context"
	"testing"

	. "github.com/jacobsa/ogletest"
)

func TestFile(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type StreamedObjectWriterTest struct {
}

var _ SetUpInterface = &StreamedObjectWriterTest{}
var _ TearDownInterface = &StreamedObjectWriterTest{}

func init() { RegisterTestSuite(&StreamedObjectWriterTest{}) }

func (t *StreamedObjectWriterTest) SetUp(ti *TestInfo) {
}

func (t *StreamedObjectWriterTest) TearDown() {
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *StreamedObjectWriterTest) ObjectWithNoHandleOrRequest() {
	sow, err := NewStreamedObjectWriter(context.Background(), nil, nil)
	ExpectEq(nil, sow)
	ExpectNe(nil, err)
}
