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

package auth

import (
	"fmt"
	"testing"

	googleAuth "cloud.google.com/go/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"cloud.google.com/go/auth/credentials"
)

////////////////////////////////////////////////////////////////////////
// Mocks
////////////////////////////////////////////////////////////////////////

type MockDetectCredentials struct {
	mock.Mock
}

func (m *MockDetectCredentials) DetectDefault(opts *credentials.DetectOptions) (*googleAuth.Credentials, error) {
	args := m.Called(opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*googleAuth.Credentials), args.Error(1)
}

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////
type credentialsTestSuite struct {
	suite.Suite
	mockDetector *MockDetectCredentials
}

func TestCredentialsTestSuite(t *testing.T) {
	suite.Run(t, new(credentialsTestSuite))
}

func (suite *credentialsTestSuite) SetupTest() {
	suite.mockDetector = new(MockDetectCredentials)
	detectCredentials = suite.mockDetector.DetectDefault
}

func (suite *credentialsTestSuite) TearDownTest() {
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////
func (suite *credentialsTestSuite) Test_GetCredentials_Success() {
	tests := []struct {
		name    string
		keyFile string
	}{
		{
			name:    "valid key file",
			keyFile: "/path/to/key.json",
		},
		{
			name:    "empty key file",
			keyFile: "",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			suite.mockDetector.On("DetectDefault", mock.AnythingOfType("*credentials.DetectOptions")).Return(&googleAuth.Credentials{}, nil).Once()

			creds, err := GetCredentials(tc.keyFile)

			assert.NoError(suite.T(), err)
			assert.NotNil(suite.T(), creds)
			suite.mockDetector.AssertExpectations(suite.T())
		})
	}
}

func (suite *credentialsTestSuite) Test_GetCredentials_Error() {
	expectedErr := fmt.Errorf("simulated detection error")
	suite.mockDetector.On("DetectDefault", mock.AnythingOfType("*credentials.DetectOptions")).Return(nil, expectedErr).Once()
	keyFile := "/path/to/key.json"

	creds, err := GetCredentials(keyFile)

	require.Error(suite.T(), err)
	assert.Nil(suite.T(), creds)
	assert.ErrorIs(suite.T(), err, expectedErr)
	suite.mockDetector.AssertExpectations(suite.T())
}
