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

package storage

import (
	"time"
)

// MinObject is a record representing subset of properties of a particular
// generation object in GCS.
//
// See here for more information about its fields:
//
//	https://cloud.google.com/storage/docs/json_api/v1/objects#resource
type MinObject struct {
	Name           string
	Size           uint64
	Generation     int64
	MetaGeneration int64
	Updated        time.Time
	Metadata       map[string]string
}
