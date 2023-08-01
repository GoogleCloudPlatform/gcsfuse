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

# $1 - local file path
# $2 - object path without `gs://`
# $3 - Optional. If passed, upload is with gzip-encode enabled.
if [[ $# -lt 2 ]] ; then 
	echo "Min 2 arguments expected. Args: <file-to-upload> <gcs-path-with-prefix> [<pass-anything-to-enable-gzip-encoding>]"
	exit 1
elif [[ $# -lt 3 ]] ; then
    gsutil cp $1 gs://$2
else
    gsutil cp -Z $1 gs://$2
fi
