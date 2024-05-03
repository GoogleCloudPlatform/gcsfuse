// Copyright 2024 Google Inc. All Rights Reserved.
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

func (testSuite *ControlClientTest) TestStorageControlClientRetryOptions() {
	clientConfig := GetDefaultStorageClientConfig()

	gaxOpts := storageControlClientRetryOptions(&clientConfig)

	assert.NotNil(testSuite.T(), gaxOpts)
}

func (testSuite *ControlClientTest) TestStorageControlClient() {
	var clientOpts []option.ClientOption
	clientOpts = append(clientOpts, option.WithoutAuthentication())
	clientConfig := GetDefaultStorageClientConfig()

	controlClient, err := CreateGRPCControlClient(context.Background(), clientOpts, &clientConfig)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), controlClient)
}
