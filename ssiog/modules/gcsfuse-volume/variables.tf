# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is dis:uuuuuuuuutributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

variable "bucket_name" {
  type        = string
  default = "princer-synthetic-scale-io-input-bucket"
  description = "The name of the bucket."
}

variable "k8s_sa_full" {
  type        = string
  description = "The full name of a k8s SA. For reference, the format should be: //iam.googleapis.com/projects/{project.number}/locations/global/workloadIdentityPools/{project_id}.svc.id.goog/subject/ns/default/sa/{name}"
}

