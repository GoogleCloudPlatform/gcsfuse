// Copyright 2023 Google LLC
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

package config

import (
	"fmt"
	"math"
	"time"
)

const (
	IgnoreInterruptsFlagName   = "ignore-interrupts"
	AnonymousAccess            = "anonymous-access"
	KernelListCacheTtlFlagName = "kernel-list-cache-ttl-secs"
	PrometheusPortFlagName     = "prometheus-port"
	TtlInSecsInvalidValueError = "the value of ttl-secs can't be less than -1"
	TtlInSecsTooHighError      = "the value of ttl-secs is too high to be supported. Max is 9223372036"

	// MaxSupportedTtlInSeconds represents maximum multiple of seconds representable by time.Duration.
	MaxSupportedTtlInSeconds = math.MaxInt64 / int64(time.Second)
	MaxSupportedTtl          = time.Duration(MaxSupportedTtlInSeconds * int64(time.Second))
)

// IsTtlInSecsValid return nil error if ttlInSecs is valid.
func IsTtlInSecsValid(ttlInSecs int64) error {
	if ttlInSecs < -1 {
		return fmt.Errorf(TtlInSecsInvalidValueError)
	}

	if ttlInSecs > MaxSupportedTtlInSeconds {
		return fmt.Errorf(TtlInSecsTooHighError)
	}

	return nil
}

func ListCacheTtlSecsToDuration(secs int64) time.Duration {
	err := IsTtlInSecsValid(secs)
	if err != nil {
		panic(fmt.Sprintf("invalid argument: %d, %v", secs, err))
	}

	if secs == -1 {
		return MaxSupportedTtl
	}

	return time.Duration(secs * int64(time.Second))
}
