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

# Here $1 refers to the testBucket/testdir argument
echo "This is from directory .. file fileInUnsupportedImplicitDir1" > fileInUnsupportedImplicitDir1
# bucket/testdir/../fileInImplicitDir1
gcloud storage cp fileInUnsupportedImplicitDir1 gs://$1/../
echo "This is from directory . file fileInUnsupportedImplicitDir2" > fileInUnsupportedImplicitDir2
# bucket/testdir/./fileInImplicitDir2
gcloud storage cp fileInUnsupportedImplicitDir2 gs://$1/./
echo "This is from directory \"\" file fileInUnsupportedImplicitDir3" > fileInUnsupportedImplicitDir3
# bucket/testdir//fileInImplicitDir3
gcloud storage cp fileInUnsupportedImplicitDir3 gs://$1//fileInUnsupportedImplicitDir3 
