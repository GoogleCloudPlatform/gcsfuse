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

package lease

// Create a read proxy consisting of the contents defined by the supplied
// refreshers concatenated. See NewReadProxy for more.
//
// If rl is non-nil, it will be used as the first temporary copy of the
// contents, and must match the concatenation of the content returned by the
// refreshers.
func NewMultiReadProxy(
	fl FileLeaser,
	refreshers []Refresher,
	rl ReadLease) (rp ReadProxy) {
	panic("TODO")
}
