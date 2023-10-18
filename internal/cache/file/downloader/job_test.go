// Copyright 2023 Google Inc. All Rights Reserved.
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

package downloader

import (
	"container/list"
	"fmt"
	"os"
	"reflect"
	"testing"

	. "github.com/jacobsa/ogletest"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

func TestJob(t *testing.T) { RunTests(t) }

type jobTest struct {
	job *Job
}

func init() { RegisterTestSuite(&jobTest{}) }

func (jt *jobTest) SetUp(*TestInfo) {
	jt.job = NewJob(nil, nil, "", nil,
		0, os.FileMode(0777), 0, 0)
}

func (jt *jobTest) TestInit() {
	// Explicitly changing initialized values first.
	jt.job.status.Name = DOWNLOADING
	jt.job.status.Err = fmt.Errorf("some error")
	jt.job.status.Offset = -1
	jt.job.subscribers.PushBack(struct{}{})
	jt.job.cancelCtx = nil
	jt.job.cancelFunc = nil

	jt.job.init()

	ExpectEq(NOT_STARTED, jt.job.status.Name)
	ExpectEq(nil, jt.job.status.Err)
	ExpectEq(0, jt.job.status.Offset)
	ExpectTrue(reflect.DeepEqual(list.List{}, jt.job.subscribers))
	ExpectNe(nil, jt.job.cancelCtx)
	ExpectNe(nil, jt.job.cancelFunc)
}
