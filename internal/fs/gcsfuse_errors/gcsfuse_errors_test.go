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

package gcsfuse_errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileClobberedError(t *testing.T) {
	testCases := []struct {
		name       string
		err        error
		wantErrMsg string
	}{
		{
			name:       "with_underlying_error",
			err:        fmt.Errorf("some error"),
			wantErrMsg: "The file was modified or deleted by another process, possibly due to concurrent access: some error",
		},
		{
			name:       "without_underlying_error",
			err:        nil,
			wantErrMsg: "The file was modified or deleted by another process, possibly due to concurrent access: <nil>",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clobberedErr := &FileClobberedError{Err: tc.err}

			gotErrMsg := clobberedErr.Error()

			assert.Equal(t, tc.wantErrMsg, gotErrMsg)
			if tc.err != nil {
				assert.True(t, errors.Is(clobberedErr, tc.err))
			}
		})
	}
}
