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

package storageutil

import (
	"errors"
	"io"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/googleapi"
)

func TestShouldRetryReturnsTrueWithGoogleApiError(t *testing.T) {
	// 401
	var err401 = googleapi.Error{
		Code: 401,
		Body: "Invalid Credential",
	}
	// 50x error
	var err502 = googleapi.Error{
		Code: 502,
	}
	// 429 - rate limiting error
	var err429 = googleapi.Error{
		Code: 429,
		Body: "API rate limit exceeded",
	}

	assert.Equal(t, true, ShouldRetry(&err401))
	assert.Equal(t, true, ShouldRetry(&err502))
	assert.Equal(t, true, ShouldRetry(&err429))
}

func TestShouldRetryReturnsFalseWithGoogleApiError400(t *testing.T) {
	// 400 - bad request
	var err400 = googleapi.Error{
		Code: 400,
	}

	assert.Equal(t, false, ShouldRetry(&err400))
}

func TestShouldRetryReturnsTrueWithUnexpectedEOFError(t *testing.T) {
	assert.Equal(t, true, ShouldRetry(io.ErrUnexpectedEOF))
}

func TestShouldRetryReturnsTrueWithNetworkError(t *testing.T) {
	assert.Equal(t, true, ShouldRetry(net.ErrClosed))
}

func TestShouldRetryReturnsTrueForConnectionRefusedAndResetErrors(t *testing.T) {
	testCases := []struct {
		name           string
		err            error
		expectedResult bool
	}{
		{
			name:           "URL Error - Connection Refused",
			err:            &url.Error{Err: errors.New("connection refused")},
			expectedResult: true,
		},
		{
			name:           "URL Error - Connection Reset",
			err:            &url.Error{Err: errors.New("connection reset")},
			expectedResult: true,
		},
		{
			name:           "URL Error - connection reset by peer",
			err:            &url.Error{Err: errors.New("connection reset by peer")},
			expectedResult: true,
		},
		{
			name:           "URL Error - connection refused by peer",
			err:            &url.Error{Err: errors.New("connection refused by peer")},
			expectedResult: true,
		},
		{
			name:           "Op Error - Connection Refused",
			err:            &net.OpError{Err: errors.New("connection refused")},
			expectedResult: true,
		},
		{
			name:           "Op Error - Connection Reset",
			err:            &net.OpError{Err: errors.New("connection reset")},
			expectedResult: true,
		},
		{
			name:           "Op Error - connection reset by peer",
			err:            &net.OpError{Err: errors.New("connection reset by peer")},
			expectedResult: true,
		},
		{
			name:           "Op Error - connection refused by peer",
			err:            &net.OpError{Err: errors.New("connection refused by peer")},
			expectedResult: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualResult := ShouldRetry(tc.err)
			assert.Equal(t, tc.expectedResult, actualResult)
		})
	}
}
