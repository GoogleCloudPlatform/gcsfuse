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

# Here $1 refers to the testBucket argument
echo "This is from directory fileInImplicitDir1 file implicitDirectory" > fileInImplicitDir1
# bucket/implicitDirectory/fileInImplicitDir1
gcloud storage cp fileInImplicitDir1 gs://$1/implicitDirectory/
echo "This is from directory implicitDirectory/implicitSubDirectory file fileInImplicitDir2" > fileInImplicitDir2
# bucket/implicitDirectory/implicitSubDirectory/fileInImplicitDir2
gcloud storage cp fileInImplicitDir2 gs://$1/implicitDirectory/implicitSubDirectory/
