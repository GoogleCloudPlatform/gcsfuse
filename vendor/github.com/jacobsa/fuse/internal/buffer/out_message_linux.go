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

package buffer

// The maximum read size that we expect to ever see from the kernel, used for
// calculating the size of out messages.
//
// For 4 KiB pages, this is 1024 KiB (cf. https://github.com/torvalds/linux/blob/15db16837a35d8007cb8563358787412213db25e/fs/fuse/fuse_i.h#L38-L40)
const MaxReadSize = 1 << 20
