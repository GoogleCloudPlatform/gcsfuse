// Copyright 2022 Google LLC
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

// Package tags provides the tags to annotate the monitoring metrics.
package tags

import (
	"go.opencensus.io/tag"
)

var (
	// IOMethod annotates the event that opens or closes a connection or file.
	IOMethod = tag.MustNewKey("io_method")

	// GCSMethod annotates the method called in the GCS client library.
	GCSMethod = tag.MustNewKey("gcs_method")

	// FSOp annotates the file system op processed.
	FSOp = tag.MustNewKey("fs_op")

	// FSError annotates the file system failed operations with the error type
	FSError = tag.MustNewKey("fs_error")

	// FSErrCategory reduces the cardinality of FSError by grouping errors together.
	FSErrCategory = tag.MustNewKey("fs_error_category")

	// ReadType annotates the read operation with the type - Sequential/Random
	ReadType = tag.MustNewKey("read_type")

	// CacheHit annotates the read operation from file cache with true or false.
	CacheHit = tag.MustNewKey("cache_hit")
)
