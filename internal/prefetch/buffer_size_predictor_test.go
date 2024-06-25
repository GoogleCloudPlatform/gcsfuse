package prefetch_test

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/prefetch"
	. "github.com/jacobsa/ogletest"
)

func TestBufferSizePredictor(t *testing.T) { RunTests(t) }

type BufferSizePredictorTest struct {
	bufferSizePredictor *prefetch.BufferSizePredictor
}

func init() { RegisterTestSuite(&BufferSizePredictorTest{}) }

func (t *BufferSizePredictorTest) SetUp(*TestInfo) {
	t.bufferSizePredictor = prefetch.NewBufferSizePredictor(prefetch.GetDefaultPrefetchConfiguration())
}

////////////////////////////////////////////////////////////////////////
// Test functions
////////////////////////////////////////////////////////////////////////

func (t *BufferSizePredictorTest) Test_BSP() {
	defaultConf := prefetch.GetDefaultPrefetchConfiguration()

	ExpectEq(defaultConf.FirstBufferSize, t.bufferSizePredictor.GetCorrectBufferSize(0))
	ExpectEq(defaultConf.FirstBufferSize*defaultConf.SequentialPrefetchMultiplier, t.bufferSizePredictor.GetCorrectBufferSize(defaultConf.FirstBufferSize))
}
