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

# Deleting key files and creds after testing.
rm  ~/admin_creds.json
rm  ~/creds.json

arr=$(gcloud iam service-accounts keys list --iam-account=multi-project-service-account@gcs-fuse-test-ml.iam.gserviceaccount.com)

# e.g. output of arr
# KEY_ID                                    CREATED_AT            EXPIRES_AT            DISABLED
# 97698dcc14a5db0b8d08eb104c33507fb97eb66a  2023-07-01T11:36:54Z  2023-09-29T11:36:54Z
# KEYID will be array[4]

eval key_id=($arr)
gcloud iam service-accounts keys delete ${key_id[4]} \
    --iam-account=multi-project-service-account@gcs-fuse-test-ml.iam.gserviceaccount.com
