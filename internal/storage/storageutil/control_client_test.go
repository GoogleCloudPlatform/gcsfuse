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

package storageutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/api/option"
)

type ControlClientTest struct {
	suite.Suite
}

func TestControlClientTestSuite(t *testing.T) {
	suite.Run(t, new(ControlClientTest))
}

func (testSuite *ControlClientTest) SetupTest() {
}

func (testSuite *ControlClientTest) TearDownTest() {
}

func (testSuite *ControlClientTest) TestStorageControlClientWithGaxRetries() {
	var clientOpts []option.ClientOption
	clientOpts = append(clientOpts, option.WithoutAuthentication())

	controlClient, err := CreateGRPCControlClient(context.Background(), clientOpts, false)

	require.Nil(testSuite.T(), err)
	require.NotNil(testSuite.T(), controlClient)
	require.NotNil(testSuite.T(), controlClient.CallOptions)
	assert.Greater(testSuite.T(), len(controlClient.CallOptions.CreateFolder), 0)
	assert.Greater(testSuite.T(), len(controlClient.CallOptions.GetFolder), 0)
	assert.Greater(testSuite.T(), len(controlClient.CallOptions.DeleteFolder), 0)
	assert.Greater(testSuite.T(), len(controlClient.CallOptions.RenameFolder), 0)
}

func (testSuite *ControlClientTest) TestStorageControlClientWithoutGaxRetries() {
	var clientOpts []option.ClientOption
	clientOpts = append(clientOpts, option.WithoutAuthentication())

	controlClient, err := CreateGRPCControlClient(context.Background(), clientOpts, true)

	require.Nil(testSuite.T(), err)
	require.NotNil(testSuite.T(), controlClient)
	if controlClient.CallOptions != nil {
		assert.Empty(testSuite.T(), controlClient.CallOptions.CreateFolder)
		assert.Empty(testSuite.T(), controlClient.CallOptions.GetFolder)
		assert.Empty(testSuite.T(), controlClient.CallOptions.DeleteFolder)
		assert.Empty(testSuite.T(), controlClient.CallOptions.RenameFolder)
	}
}
