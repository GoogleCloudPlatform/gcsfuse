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

package wrappers

import (
	"context"
	"errors"
	"syscall"
	"testing"

	"cloud.google.com/go/storage"
	. "github.com/jacobsa/ogletest"
)

func TestErrorMapping(t *testing.T) { RunTests(t) }

type errorMappingTest struct {
}

func init() { RegisterTestSuite(&errorMappingTest{}) }

func (t errorMappingTest) ErrNoShouldReturnEINTRWhenContextIsCancelled() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ExpectEq(syscall.EINTR, errno(ctx, errors.New("Error")))
}

func (t errorMappingTest) ErrNoShouldReturnActualErrorWhenContextIsNotCancelled() {
	ExpectEq(syscall.ENOENT, errno(context.Background(), storage.ErrObjectNotExist))
}
