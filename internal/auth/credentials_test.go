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

	"cloud.google.com/go/auth"
	"cloud.google.com/go/auth/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

////////////////////////////////////////////////////////////////////////
// Mock
////////////////////////////////////////////////////////////////////////

type MockDetectCredentials struct {
	mock.Mock
}

func (m *MockDetectCredentials) DetectDefault(opts *credentials.DetectOptions) (*auth.Credentials, error) {
	args := m.Called(opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.Credentials), args.Error(1)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func Test_getCredentials_Success(t *testing.T) {
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
		t.Run(tc.name, func(t *testing.T) {
			mockDetector := new(MockDetectCredentials)
			opts := &credentials.DetectOptions{
				CredentialsFile: tc.keyFile,
				Scopes:          []string{scope},
			}
			mockDetector.On("DetectDefault", opts).Return(&auth.Credentials{}, nil).Once()

			creds, err := getCredentials(tc.keyFile, mockDetector.DetectDefault)

			assert.NoError(t, err)
			assert.NotNil(t, creds)
			mockDetector.AssertExpectations(t)
		})
	}
}

func Test_getCredentials_Error(t *testing.T) {
	mockDetector := new(MockDetectCredentials)
	keyFile := "/path/to/key.json"
	expectedErr := fmt.Errorf("simulated detection error")
	opts := &credentials.DetectOptions{
		CredentialsFile: keyFile,
		Scopes:          []string{scope},
	}
	mockDetector.On("DetectDefault", opts).Return(nil, expectedErr).Once()

	creds, err := getCredentials(keyFile, mockDetector.DetectDefault)

	assert.Error(t, err)
	assert.Nil(t, creds)
	assert.ErrorIs(t, err, expectedErr)
	mockDetector.AssertExpectations(t)
}
