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
	"fmt"
)

// FileClobberedError represents a file clobbering scenario where a file was
// modified or deleted while it was being accessed.
type FileClobberedError struct {
	Err error
}

func (fce *FileClobberedError) Error() string {
	return fmt.Sprintf("The file was modified or deleted by another process, possibly due to concurrent access: %v", fce.Err)
}

func (fce *FileClobberedError) Unwrap() error {
	return fce.Err
}
