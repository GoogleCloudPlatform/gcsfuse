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

package bytes

import "fmt"

// Present the supplied number of bytes in a human-readable format.
func Bytes(v float64) string {
	switch {
	case v >= 1<<30:
		return fmt.Sprintf("%.2f GiB", v/(1<<30))

	case v >= 1<<20:
		return fmt.Sprintf("%.2f MiB", v/(1<<20))

	case v >= 1<<10:
		return fmt.Sprintf("%.2f KiB", v/(1<<10))

	default:
		return fmt.Sprintf("%.2f bytes", v)
	}
}
