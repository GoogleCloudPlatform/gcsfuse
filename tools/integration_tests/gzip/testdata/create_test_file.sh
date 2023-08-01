# Copyright 2023 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#	http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# $1 - output file path.
# $2 - size in bytes
# $3 - Optional. If passed, file is gzip-compressed.
if [ $# -lt 2 ] ; then 
  echo "Expected min 2 arguments. Received: " $# ". Expected args: <output-file-path> <file-size> [<pass-anything-to-create-gzip>]"
  exit 1
elif [ $# -lt 3 ] ; then 
  yes This is a test file | head -c $2 > $1
else
  yes This is a test file | head -c $2 | gzip -c > $1
fi
