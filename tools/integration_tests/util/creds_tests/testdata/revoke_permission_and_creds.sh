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

# Delete key file after testing
SERVICE_ACCOUNT=$1
KEY_FILE=$2

gcloud auth revoke $SERVICE_ACCOUNT
# Crete key file output
# e.g. created key [key_id] of type [json] as [key_file_path] for [service_account]
# capturing third word from the file to get key-id
# e.g. capture [KEY_ID]
KEY_ID=$(cat ~/key_id.txt | cut -d " " -f 3)
# removing braces
# e.g. capture KEY_ID
KEY_ID=${KEY_ID:1:40}

gcloud iam service-accounts keys delete $KEY_ID --iam-account=$SERVICE_ACCOUNT
rm ~/key_id.txt
rm $KEY_FILE
