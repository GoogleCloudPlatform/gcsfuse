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

package cfg

import (
	"math"
	"time"
)

const (
	// Logging-level constants

	TRACE   string = "TRACE"
	DEBUG   string = "DEBUG"
	INFO    string = "INFO"
	WARNING string = "WARNING"
	ERROR   string = "ERROR"
	OFF     string = "OFF"
)

const (
	// ExperimentalMetadataPrefetchOnMountDisabled is the mode without metadata-prefetch.
	ExperimentalMetadataPrefetchOnMountDisabled = "disabled"
	// ExperimentalMetadataPrefetchOnMountSynchronous is the prefetch-mode where mounting is not marked complete until prefetch is complete.
	ExperimentalMetadataPrefetchOnMountSynchronous = "sync"
	// ExperimentalMetadataPrefetchOnMountAsynchronous is the prefetch-mode where mounting is marked complete once prefetch has started.
	ExperimentalMetadataPrefetchOnMountAsynchronous = "async"
)

const (
	// maxSequentialReadSizeMb is the max value supported by sequential-read-size-mb flag.
	maxSequentialReadSizeMB = 1024
)

const (
	// MaxSupportedTTLInSeconds represents maximum multiple of seconds representable by time.Duration.
	MaxSupportedTTLInSeconds = math.MaxInt64 / int64(time.Second)
	maxSupportedTTL          = time.Duration(MaxSupportedTTLInSeconds * int64(time.Second))
)
