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

package gcs

import (
	storagev1 "google.golang.org/api/storage/v1"
)

// OAuth scopes for GCS. For use with e.g. google.DefaultTokenSource.
const (
	Scope_FullControl = storagev1.DevstorageFullControlScope
	Scope_ReadOnly    = storagev1.DevstorageReadOnlyScope
	Scope_ReadWrite   = storagev1.DevstorageReadWriteScope
)
