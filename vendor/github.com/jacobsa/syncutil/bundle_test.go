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

package syncutil_test

import (
	"errors"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/syncutil"
	"golang.org/x/net/context"
)

func TestOgletest(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type BundleTest struct {
	bundle       *syncutil.Bundle
	cancelParent context.CancelFunc
}

var _ SetUpTestSuiteInterface = &BundleTest{}
var _ SetUpInterface = &BundleTest{}

func init() { RegisterTestSuite(&BundleTest{}) }

func (t *BundleTest) SetUpTestSuite() {
	// Make sure parallelism is allowed.
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func (t *BundleTest) SetUp(ti *TestInfo) {
	// Set up the parent context.
	parentCtx, cancelParent := context.WithCancel(context.Background())
	t.cancelParent = cancelParent

	// Set up the bundle.
	t.bundle = syncutil.NewBundle(parentCtx)
}

////////////////////////////////////////////////////////////////////////
// Test functions
////////////////////////////////////////////////////////////////////////

func (t *BundleTest) NoOperations() {
	ExpectEq(nil, t.bundle.Join())
}

func (t *BundleTest) SingleOp_Success() {
	t.bundle.Add(func(c context.Context) error {
		return nil
	})

	ExpectEq(nil, t.bundle.Join())
}

func (t *BundleTest) SingleOp_Error() {
	expected := errors.New("taco")
	t.bundle.Add(func(c context.Context) error {
		return expected
	})

	ExpectEq(expected, t.bundle.Join())
}

func (t *BundleTest) SingleOp_ParentCancelled() {
	// Start an op that waits for the context to be cancelled before returning an
	// expected value.
	expected := errors.New("taco")
	t.bundle.Add(func(c context.Context) error {
		<-c.Done()
		return expected
	})

	// Cancel the parent context, then join the bundle. The op should see the
	// cancellation, so we shouldn't deadlock and we should get the expected
	// value.
	t.cancelParent()
	ExpectEq(expected, t.bundle.Join())
}

func (t *BundleTest) MultipleOps_Success() {
	for i := 0; i < 4; i++ {
		t.bundle.Add(func(c context.Context) error {
			return nil
		})
	}

	ExpectEq(nil, t.bundle.Join())
}

func (t *BundleTest) MultipleOps_UnorderedErrors() {
	// Start multiple ops, each returning a different error.
	errs := []error{
		errors.New("taco"),
		errors.New("burrito"),
		errors.New("enchilada"),
	}

	for i := 0; i < len(errs); i++ {
		iCopy := i
		t.bundle.Add(func(c context.Context) error {
			return errs[iCopy]
		})
	}

	// Joining the bundle should result in some error from the list.
	ExpectThat(errs, Contains(t.bundle.Join()))
}

func (t *BundleTest) MultipleOps_OneError_OthersDontWait() {
	expected := errors.New("taco")

	// Add two operations that succeed and one that fails.
	t.bundle.Add(func(c context.Context) error { return nil })
	t.bundle.Add(func(c context.Context) error { return expected })
	t.bundle.Add(func(c context.Context) error { return nil })

	// We should see the failure.
	ExpectEq(expected, t.bundle.Join())
}

func (t *BundleTest) MultipleOps_OneError_OthersWaitForCancellation() {
	expected := errors.New("taco")

	// Add several ops that wait for cancellation then succeed, and one that
	// returns an error.
	t.bundle.Add(func(c context.Context) error { <-c.Done(); return nil })
	t.bundle.Add(func(c context.Context) error { <-c.Done(); return nil })
	t.bundle.Add(func(c context.Context) error { return expected })
	t.bundle.Add(func(c context.Context) error { <-c.Done(); return nil })
	t.bundle.Add(func(c context.Context) error { <-c.Done(); return nil })

	// We should see the failure.
	ExpectEq(expected, t.bundle.Join())
}

func (t *BundleTest) MultipleOps_ParentCancelled() {
	expected := errors.New("taco")

	// Start multiple ops that wait for the context to be cancelled before
	// returning an expected value.
	t.bundle.Add(func(c context.Context) error { <-c.Done(); return expected })
	t.bundle.Add(func(c context.Context) error { <-c.Done(); return expected })
	t.bundle.Add(func(c context.Context) error { <-c.Done(); return expected })
	t.bundle.Add(func(c context.Context) error { <-c.Done(); return expected })
	t.bundle.Add(func(c context.Context) error { <-c.Done(); return expected })

	// Cancel the parent context, then join the bundle. The ops should see the
	// cancellation, so we shouldn't deadlock and we should get the expected
	// value.
	t.cancelParent()
	ExpectEq(expected, t.bundle.Join())
}

func (t *BundleTest) MultipleOps_PreviousError_NewOpsObserve() {
	var wg sync.WaitGroup
	signalCancellation := func(c context.Context) error {
		<-c.Done()
		wg.Done()
		return nil
	}

	// Start an op that will let us know when it is cancelled.
	wg.Add(1)
	t.bundle.Add(signalCancellation)

	// Start an op that returns an error.
	expected := errors.New("taco")
	t.bundle.Add(func(c context.Context) error { return expected })

	// Wait for the error to be observed.
	wg.Wait()

	// Further ops should be immediately cancelled.
	wg = sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		t.bundle.Add(signalCancellation)
	}

	wg.Wait()

	// Join.
	ExpectEq(expected, t.bundle.Join())
}

func (t *BundleTest) MultipleOps_PreviousParentCancel_NewOpsObserve() {
	var wg sync.WaitGroup
	signalCancellation := func(c context.Context) error {
		<-c.Done()
		wg.Done()
		return nil
	}

	// Cancel the parent context.
	t.cancelParent()

	// Further ops should be immediately cancelled.
	wg = sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		t.bundle.Add(signalCancellation)
	}

	wg.Wait()

	// Join.
	t.bundle.Join()
}

func (t *BundleTest) JoinWaitsForAllOps_Success() {
	// Set up a 64-bit counter that is guaranteed to be properly aligned.
	// Cf. "Bugs" section of http://godoc.org/sync/atomic.
	slice := make([]uint64, 1)
	var counter *uint64 = &slice[0]

	// Start several ops that sleep awhile, increment a counter, and return.
	const N = 100
	for i := 0; i < N; i++ {
		t.bundle.Add(func(c context.Context) error {
			numMs := rand.Float64() * 100
			time.Sleep(time.Duration(numMs) * time.Millisecond)
			atomic.AddUint64(counter, 1)
			return nil
		})
	}

	// Wait for all of the ops. Afterward, the counter should have the expected
	// value.
	AssertEq(nil, t.bundle.Join())
	ExpectEq(N, atomic.LoadUint64(counter))
}

func (t *BundleTest) JoinWaitsForAllOps_Error() {
	expected := errors.New("taco")

	// Set up a 64-bit counter that is guaranteed to be properly aligned.
	// Cf. "Bugs" section of http://godoc.org/sync/atomic.
	slice := make([]uint64, 1)
	var counter *uint64 = &slice[0]

	// Start several ops that sleep awhile, increment a counter, and return an
	// error.
	const N = 100
	for i := 0; i < N; i++ {
		t.bundle.Add(func(c context.Context) error {
			numMs := rand.Float64() * 100
			time.Sleep(time.Duration(numMs) * time.Millisecond)
			atomic.AddUint64(counter, 1)
			return expected
		})
	}

	// Wait for all of the ops. Afterward, the counter should have the expected
	// value.
	AssertEq(expected, t.bundle.Join())
	ExpectEq(N, atomic.LoadUint64(counter))
}

func (t *BundleTest) JoinWaitsForAllOps_ParentCancelled() {
	t.cancelParent()

	// Set up a 64-bit counter that is guaranteed to be properly aligned.
	// Cf. "Bugs" section of http://godoc.org/sync/atomic.
	slice := make([]uint64, 1)
	var counter *uint64 = &slice[0]

	// Start several ops that sleep awhile, increment a counter, and return an
	// error.
	const N = 100
	for i := 0; i < N; i++ {
		t.bundle.Add(func(c context.Context) error {
			numMs := rand.Float64() * 100
			time.Sleep(time.Duration(numMs) * time.Millisecond)
			atomic.AddUint64(counter, 1)
			return nil
		})
	}

	// Wait for all of the ops. Afterward, the counter should have the expected
	// value.
	AssertEq(nil, t.bundle.Join())
	ExpectEq(N, atomic.LoadUint64(counter))
}
