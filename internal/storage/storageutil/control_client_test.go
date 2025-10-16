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
	"time"

	control "cloud.google.com/go/storage/control/apiv2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/api/option"
)

type ControlClientTest struct {
	suite.Suite
}

func TestControlClientTestSuite(t *testing.T) {
	suite.Run(t, new(ControlClientTest))
}

func (testSuite *ControlClientTest) TestStorageControlClientWithGaxRetries() {
	// In this test, we are not actually creating a storageControlClient, but just checking if the
	// gax retries are set correctly in the CallOptions.
	// The test is not creating a real GRPC client because it requires authentication.
	// The test is not using a fake GRPC client because it is not possible to check the
	// CallOptions from the fake client.
	// The test is not using a mock GRPC client because it is not possible to check the
	// CallOptions from the mock client.
	clientConfig := &StorageClientConfig{MaxRetrySleep: 100 * time.Microsecond, MaxRetryAttempts: 5}
	gaxRetryOptions := storageControlClientGaxRetryOptions(clientConfig)
	var clientOpts []option.ClientOption
	clientOpts = append(clientOpts, option.WithGRPCDialOption(nil)) // a dummy GRPC dial option to avoid authentication.

	scc, err := CreateGRPCControlClient(context.Background(), clientOpts, false)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), scc)
	assert.Equal(testSuite.T(), gaxRetryOptions, scc.CallOptions.CreateFolder)
	assert.Equal(testSuite.T(), gaxRetryOptions, scc.CallOptions.DeleteFolder)
	assert.Equal(testSuite.T(), gaxRetryOptions, scc.CallOptions.GetFolder)
	assert.Equal(testSuite.T(), gaxRetryOptions, scc.CallOptions.ListFolders)
	assert.Equal(testSuite.T(), gaxRetryOptions, scc.CallOptions.RenameFolder)
	assert.Equal(testSuite.T(), gaxRetryOptions, scc.CallOptions.GetStorageLayout)
}

func (testSuite *ControlClientTest) TestStorageControlClientWithoutGaxRetries() {
	var clientOpts []option.ClientOption
	clientOpts = append(clientOpts, option.WithGRPCDialOption(nil)) // a dummy GRPC dial option to avoid authentication.
	scc, err := CreateGRPCControlClient(context.Background(), clientOpts, true)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), scc)
	assert.Equal(testSuite.T(), new(control.StorageControlCallOptions), scc.CallOptions)
}
