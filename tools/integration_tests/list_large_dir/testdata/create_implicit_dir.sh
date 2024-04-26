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

# $1 testbucket
# $2 PrefixImplicitDirInLargeDirListTest
# $3 NumberOfImplicitDirsInDirectoryWithTwelveThousandFiles - 100

TEST_BUCKET=$1
IMPLICIT_DIR=$2
NUMBER_OF_FILES=$3

a=1
#Iterate the loop until a greater than 100
touch testFile.txt
while [ $a -le $NUMBER_OF_FILES ]
do
   dir=$IMPLICIT_DIR$a
   a=`expr $a + 1`
   gcloud storage cp testFile.txt gs://$TEST_BUCKET/$dir/
done
