#!/bin/bash
# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

sudo apt-get update
# Upgrade gcloud version.
# Kokoro machine's outdated gcloud version prevents the use of the "gcloud storage" feature.
gcloud version
wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
sudo tar xzf gcloud.tar.gz && sudo cp -r google-cloud-sdk /usr/local && sudo rm -r google-cloud-sdk
sudo /usr/local/google-cloud-sdk/install.sh
export PATH=/usr/local/google-cloud-sdk/bin:$PATH
echo 'export PATH=/usr/local/google-cloud-sdk/bin:$PATH' >> ~/.bashrc
gcloud version && rm gcloud.tar.gz
sudo /usr/local/google-cloud-sdk/bin/gcloud components update
sudo /usr/local/google-cloud-sdk/bin/gcloud components install alpha
