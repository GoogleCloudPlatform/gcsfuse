package prefetch_test

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/prefetch"
	. "github.com/jacobsa/ogletest"
)

func TestQueue(t *testing.T) { RunTests(t) }

type CacheTest struct {
	q *prefetch.Queue
}

func init() { RegisterTestSuite(&CacheTest{}) }

func (t *CacheTest) SetUp(*TestInfo) {
	locker.EnableInvariantsCheck()
	t.q = prefetch.NewQueue()
}

////////////////////////////////////////////////////////////////////////
// Test functions
////////////////////////////////////////////////////////////////////////

func (t *CacheTest) Test_Push() {
	t.q.Push(4)
	t.q.Push(5)

	ExpectEq(4, t.q.Peek())
	ExpectEq(false, t.q.IsEmpty())
}

func (t *CacheTest) Test_Pop() {
	t.q.Push(4)
	t.q.Push(5)
	AssertEq(4, t.q.Peek())
	AssertEq(false, t.q.IsEmpty())

	val := t.q.Pop()

	ExpectEq(val, 4)
	ExpectEq(5, t.q.Peek())

	val = t.q.Pop()
	ExpectEq(val, 5)
	ExpectEq(nil, t.q.Peek())
	ExpectTrue(t.q.IsEmpty())

	val = t.q.Pop()
	ExpectEq(nil, val)
}

func (t *CacheTest) Test_Peek() {
	val := t.q.Peek()

	ExpectEq(nil, val)

	t.q.Push(4)

	val = t.q.Peek()
	ExpectEq(4, val)
}

func (t *CacheTest) Test_IsEmpty() {
	ExpectTrue(t.q.IsEmpty())

	t.q.Push(4)
	ExpectFalse(t.q.IsEmpty())

	_ = t.q.Pop()
	ExpectTrue(t.q.IsEmpty())
}
