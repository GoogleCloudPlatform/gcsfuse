// Copyright 2025 Google LLC
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

package block

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type prefetchBlockTest struct {
	MemoryBlockTest
}

func TestPrefetchBlockTestSuite(t *testing.T) {
	suite.Run(t, new(prefetchBlockTest))
}

func (testSuite *prefetchBlockTest) TestCreatePrefetchBlock() {
	pb, err := createPrefetchBlock(12)
	require.Nil(testSuite.T(), err)

	// No write operation, so size should be 0
	assert.Equal(testSuite.T(), int64(0), pb.Size())
	output, err := io.ReadAll(pb.Reader())
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), []byte{}, output)
	assert.Equal(testSuite.T(), int64(0), pb.GetId())
	assert.NotNil(testSuite.T(), pb.NotificationChannel())
}

func (testSuite *prefetchBlockTest) TestPrefetchBlockWrite() {
	pb, err := createPrefetchBlock(12)
	require.Nil(testSuite.T(), err)
	content := []byte("hi")

	n, err := pb.Write(content)

	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), len(content), n)
	output, err := io.ReadAll(pb.Reader())
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), content, output)
	assert.Equal(testSuite.T(), int64(2), pb.Size())
}

func (testSuite *prefetchBlockTest) TestPrefetchBlockWriteWithDataGreaterThanCapacity() {
	pb, err := createPrefetchBlock(1)
	require.Nil(testSuite.T(), err)
	content := []byte("hi")

	n, err := pb.Write(content)

	assert.NotNil(testSuite.T(), err)
	assert.Equal(testSuite.T(), 0, n)
	assert.EqualError(testSuite.T(), err, outOfCapacityError)
}

func (testSuite *prefetchBlockTest) TestPrefetchBlockReuse() {
	pb, err := createPrefetchBlock(12)
	require.Nil(testSuite.T(), err)
	content := []byte("hi")
	n, err := pb.Write(content)
	require.Nil(testSuite.T(), err)
	require.Equal(testSuite.T(), len(content), n)

	pb.Reuse()

	assert.Equal(testSuite.T(), int64(0), pb.Size())
	output, err := io.ReadAll(pb.Reader())
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), []byte{}, output)
	assert.Equal(testSuite.T(), int64(0), pb.GetId())
	assert.NotNil(testSuite.T(), pb.NotificationChannel())
}

func (testSuite *prefetchBlockTest) TestPrefetchBlockGetIdForEmptyBlock() {
	pb, err := createPrefetchBlock(12)
	require.Nil(testSuite.T(), err)

	// Initially, the ID should be 0
	assert.Equal(testSuite.T(), int64(0), pb.GetId())
}

// TestPrefetchBlockSetId sets the ID of the prefetch block and verifies it.
func (testSuite *prefetchBlockTest) TestPrefetchBlockSetId() {
	pb, err := createPrefetchBlock(12)
	require.Nil(testSuite.T(), err)

	// Set a new ID
	newId := int64(5)
	pb.SetId(newId)

	// Verify that the ID is set correctly
	assert.Equal(testSuite.T(), newId, pb.GetId())
}
