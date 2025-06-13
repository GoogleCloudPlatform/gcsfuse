// Copyright 2024 Google LLC
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

package buffered_reader

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type workerPoolTestSuite struct {
	suite.Suite
	assert *assert.Assertions
}

func (suite *workerPoolTestSuite) SetupTest() {
}

func (suite *workerPoolTestSuite) cleanupTest() {
}

func (suite *workerPoolTestSuite) TestCreate() {
	suite.assert = assert.New(suite.T())

	tp := NewWorkerPool(0, nil)
	suite.assert.Nil(tp)

	tp = NewWorkerPool(1, nil)
	suite.assert.Nil(tp)

	tp = NewWorkerPool(1, func(*PrefetchTask) {})
	suite.assert.NotNil(tp)
	suite.assert.Equal(tp.worker, uint32(1))
}

func (suite *workerPoolTestSuite) TestStartStop() {
	suite.assert = assert.New(suite.T())

	r := func(i *PrefetchTask) {
		suite.assert.Equal(i.failCnt, int32(1))
	}

	tp := NewWorkerPool(2, r)
	suite.assert.NotNil(tp)
	suite.assert.Equal(tp.worker, uint32(2))

	tp.Start()
	suite.assert.NotNil(tp.priorityCh)
	suite.assert.NotNil(tp.normalCh)

	tp.Stop()
}

func (suite *workerPoolTestSuite) TestSchedule() {
	suite.assert = assert.New(suite.T())

	r := func(i *PrefetchTask) {
		suite.assert.Equal(i.failCnt, int32(1))
	}

	tp := NewWorkerPool(2, r)
	suite.assert.NotNil(tp)
	suite.assert.Equal(tp.worker, uint32(2))

	tp.Start()
	suite.assert.NotNil(tp.priorityCh)
	suite.assert.NotNil(tp.normalCh)

	tp.Schedule(false, &PrefetchTask{failCnt: 1})
	tp.Schedule(true, &PrefetchTask{failCnt: 1})

	time.Sleep(1 * time.Second)
	tp.Stop()
}

func (suite *workerPoolTestSuite) TestPrioritySchedule() {
	suite.assert = assert.New(suite.T())

	callbackCnt := int32(0)
	r := func(i *PrefetchTask) {
		suite.assert.Equal(i.failCnt, int32(5))
		atomic.AddInt32(&callbackCnt, 1)
	}

	tp := NewWorkerPool(10, r)
	suite.assert.NotNil(tp)
	suite.assert.Equal(tp.worker, uint32(10))

	tp.Start()
	suite.assert.NotNil(tp.priorityCh)
	suite.assert.NotNil(tp.normalCh)

	for i := 0; i < 100; i++ {
		tp.Schedule(i < 20, &PrefetchTask{failCnt: 5})
	}

	time.Sleep(1 * time.Second)
	suite.assert.Equal(callbackCnt, int32(100))
	tp.Stop()
}

func TestWorkerPoolSuite(t *testing.T) {
	suite.Run(t, new(workerPoolTestSuite))
}
