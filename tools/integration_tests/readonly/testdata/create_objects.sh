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
echo "This is from directory Test file a" >> a.txt
gsutil mv a.txt gs://$1/Test/
echo "This is from file Test1" >> Test1.txt
gsutil mv Test1.txt gs://$1/
echo "This is from directory Test/b file b" >> b.txt
gsutil mv b.txt gs://$1/Test/b/
