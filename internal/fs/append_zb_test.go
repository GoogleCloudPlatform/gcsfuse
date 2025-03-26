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

// This file contains appends specific tests for zonal buckets.

package fs_test

import (
	"context"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type AppendsZBTest struct {
	fsTest
	suite.Suite
}

func (t *AppendsZBTest) SetupSuite() {
	t.serverCfg.NewConfig = &cfg.Config{
		Write: cfg.WriteConfig{
			BlockSizeMb:           1,
			CreateEmptyFile:       false,
			EnableStreamingWrites: true,
			GlobalMaxBlocks:       5,
			MaxBlocksPerFile:      1,
		},
	}
	t.mountCfg.DisableWritebackCaching = true
	bucketType = gcs.BucketType{Zonal: true, Hierarchical: true}
	t.fsTest.SetUpTestSuite()
}

func (t *AppendsZBTest) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}
func (t *AppendsZBTest) SetupTest() {
}

func (t *AppendsZBTest) TearDownTest() {
	t.fsTest.TearDown()
}

func TestAppendsZBTest(t *testing.T) {
	suite.Run(t, new(AppendsZBTest))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *AppendsZBTest) TestUnFinalizedObjectsCanBeLookedUp() gcs.Writer {
	ctx := context.Background()
	req := &gcs.CreateObjectRequest{Name: fileName, GenerationPrecondition: nil}
	writer, err := bucket.CreateObjectChunkWriter(ctx, req, util.MiB, nil)
	require.NoError(t.T(), err)
	offset, err := bucket.FlushPendingWrites(ctx, writer)
	require.NoError(t.T(), err)
	require.Equal(t.T(), int64(0), offset)

	statRes, err := operations.StatFile(path.Join(mntDir, fileName))

	require.NoError(t.T(), err)
	assert.Equal(t.T(), fileName, (*statRes).Name())
	assert.Equal(t.T(), int64(0), (*statRes).Size())
	return writer
}

func (t *AppendsZBTest) TestUnFinalizedObjectsSizeChangeIsReflected() {
	writer := t.TestUnFinalizedObjectsCanBeLookedUp()
	var dataLength int64 = util.MiB * 3.5
	content, err := operations.GenerateRandomData(dataLength)
	require.NoError(t.T(), err)
	offset, err := writer.Write(content)
	require.NoError(t.T(), err)
	assert.EqualValues(t.T(), dataLength, offset)
	flushOffset, err := bucket.FlushPendingWrites(ctx, writer)
	require.NoError(t.T(), err)
	require.Equal(t.T(), dataLength, flushOffset)

	statRes, err := operations.StatFile(path.Join(mntDir, fileName))

	require.NoError(t.T(), err)
	assert.Equal(t.T(), fileName, (*statRes).Name())
	assert.Equal(t.T(), dataLength, (*statRes).Size())
}
