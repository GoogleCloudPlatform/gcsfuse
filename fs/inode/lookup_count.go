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

package inode

import (
	"fmt"
	"log"
)

// A helper struct for implementing lookup counts. destroy will be called when
// the count hits zero, with errors logged but otherwise ignored. External
// synchronization is required.
type lookupCount struct {
	count   uint64
	destroy func() error
}

func (lc *lookupCount) Inc() {
	lc.count++
}

func (lc *lookupCount) Dec(n uint64) (destroyed bool) {
	// Make sure n is in range.
	if n > lc.count {
		panic(fmt.Sprintf(
			"n is greater than lookup count: %v vs. %v",
			n,
			lc.count))
	}

	// Decrement and destroy if necessary.
	lc.count -= n

	if lc.count == 0 {
		err := lc.destroy()
		if err != nil {
			log.Printf("Error destroying: %v", err)
		}

		destroyed = true
	}

	return
}
