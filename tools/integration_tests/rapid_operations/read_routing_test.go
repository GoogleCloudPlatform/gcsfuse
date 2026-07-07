// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// limitations under the License.

package rapid_operations

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/suite"
)

const (
	fileSizeForReadRouting = 50 * operations.OneMiB
)

// ReadRoutingTestSuite groups all tests related to read routing across different topologies.
type ReadRoutingTestSuite struct {
	BaseSuite
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ReadRoutingTestSuite) TestSequentialReadAfterAppendableUpload() {
	filePath, expectedContent := t.createGCSFile(true, fileSizeForReadRouting)

	readSequentiallyAndVerify(t.T(), filePath, expectedContent)
}

func (t *ReadRoutingTestSuite) TestRandomReadAfterAppendableUpload() {
	filePath, expectedContent := t.createGCSFile(true, fileSizeForReadRouting)

	readRandomlyAndVerify(t.T(), filePath, expectedContent)
}

func (t *ReadRoutingTestSuite) TestSequentialReadAfterResumableUpload() {
	filePath, expectedContent := t.createGCSFile(false, fileSizeForReadRouting)

	readSequentiallyAndVerify(t.T(), filePath, expectedContent)
}

func (t *ReadRoutingTestSuite) TestRandomReadAfterResumableUpload() {
	filePath, expectedContent := t.createGCSFile(false, fileSizeForReadRouting)

	readRandomlyAndVerify(t.T(), filePath, expectedContent)
}

////////////////////////////////////////////////////////////////////////
// Test Runner
////////////////////////////////////////////////////////////////////////

func TestReadRoutingTestSuite(t *testing.T) {
	RunTests(t, "ReadRoutingTestSuite", func(primaryFlags, secondaryFlags []string) suite.TestingSuite {
		return &ReadRoutingTestSuite{BaseSuite{primaryFlags: primaryFlags, secondaryFlags: secondaryFlags}}
	})
}
