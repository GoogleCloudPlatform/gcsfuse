# Copyright 2023 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# Create bucket for testing.

BUCKET_NAME=$1
PROJECT_ID=$2
gcloud storage buckets create gs://$BUCKET_NAME --project=$PROJECT_ID  --location=us-west1 --uniform-bucket-level-access 2> ~/output.txt
if [ $? -eq 1 ]; then
  if grep "HTTPError 409" ~/output.txt; then
    echo "Bucket already exist."
    rm ~/output.txt
  else
    rm ~/output.txt
    exit 1
  fi
fi
rm ~/output.txt
