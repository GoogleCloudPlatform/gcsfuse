# Copyright 2023 Google LLC
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
set -e
# $1 testbucket
# $2 DirectoryWithTwelveThousandFiles
# $3 PrefixFileInDirectoryWithTwelveThousandFiles
TEST_BUCKET=$1
DIR_WITH_TWELVE_THOUSAND_FILES=$2
FILES=$3

cd $DIR_WITH_TWELVE_THOUSAND_FILES
gcloud storage mv $FILES* gs://$TEST_BUCKET/
cd ../
rm -r $DIR_WITH_TWELVE_THOUSAND_FILES
