#!/usr/bin/env python

# Copyright 2018 The Kubernetes Authors.
# Copyright 2022 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import subprocess


def run_command(command: str):
  result = subprocess.run(command.split(" "), capture_output=True, text=True)
  print(result.stdout)
  print(result.stderr)


bucketName_fileSize_blockSize = [
    ("gke-fio-64k-1m", "64K", "64K"),
    ("gke-fio-128k-1m", "128K", "128K"),
    ("gke-fio-1mb-1m", "1M", "256K"),
    ("gke-fio-100mb-50k", "100M", "1M"),
    ("gke-fio-200gb-1", "200G", "1M"),
]

scenarios = ["gcsfuse-file-cache", "gcsfuse-no-file-cache", "local-ssd"]

for bucketName, fileSize, blockSize in bucketName_fileSize_blockSize:
  for readType in ["read", "randread"]:
    for scenario in scenarios:
      if readType == "randread" and fileSize in ["64K", "128K"]:
        continue

      commands = [
          (
              "helm install"
              f" fio-loading-test-{fileSize.lower()}-{readType}-{scenario} loading-test"
          ),
          f"--set bucketName={bucketName}",
          f"--set scenario={scenario}",
          f"--set fio.readType={readType}",
          f"--set fio.fileSize={fileSize}",
          f"--set fio.blockSize={blockSize}",
      ]

      if fileSize == "100M":
        commands.append("--set fio.filesPerThread=1000")

      helm_command = " ".join(commands)

      run_command(helm_command)
