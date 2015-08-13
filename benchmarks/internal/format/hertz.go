// Copyright 2015 Google Inc. All Rights Reserved.
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

package format

import "fmt"

// Present the supplied rate in Hz in a human-readable format.
func Hertz(v float64) string {
	switch {
	case v >= 1e9:
		return fmt.Sprintf("%.2f GHz", v/1e9)

	case v >= 1e6:
		return fmt.Sprintf("%.2f MHz", v/1e6)

	case v >= 1e3:
		return fmt.Sprintf("%.2f KHz", v/1e3)

	default:
		return fmt.Sprintf("%.2f Hz", v)
	}
}
