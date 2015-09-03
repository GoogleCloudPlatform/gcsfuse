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

package wiring

// MakeBucket will return a special fake bucket when it is given this name,
// which is not a legal name. The bucket contains the following canned objects:
//
//     Name       Contents
//     ----       --------
//     foo        "taco"
//     bar/baz    "burrito"
//
// Cf. https://cloud.google.com/storage/docs/bucket-naming?hl=en
const FakeBucket = "fake@bucket"
