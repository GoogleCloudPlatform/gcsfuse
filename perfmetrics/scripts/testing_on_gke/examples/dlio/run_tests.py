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

metadataCacheTtlSecs = 6048000
bucketName_numFilesTrain_recordLength_batchSize = [
    ("gke-dlio-unet3d-100kb-500k", 500000, 102400, 800), 
    ("gke-dlio-unet3d-100kb-500k", 500000, 102400, 128),
    ("gke-dlio-unet3d-500kb-1m", 1000000, 512000, 800),
    ("gke-dlio-unet3d-500kb-1m", 1000000, 512000, 128),
    ("gke-dlio-unet3d-3mb-100k", 100000, 3145728, 200),
    ("gke-dlio-unet3d-150mb-5k", 5000, 157286400, 4)
    ]

scenarios = ["gcsfuse-file-cache", "gcsfuse-no-file-cache", "local-ssd"]

for bucketName, numFilesTrain, recordLength, batchSize in bucketName_numFilesTrain_recordLength_batchSize:    
    for scenario in scenarios:
        commands = [f"helm install {bucketName}-{batchSize}-{scenario} unet3d-loading-test",
                    f"--set bucketName={bucketName}",
                    f"--set scenario={scenario}",
                    f"--set dlio.numFilesTrain={numFilesTrain}",
                    f"--set dlio.recordLength={recordLength}",
                    f"--set dlio.batchSize={batchSize}"]
        
        helm_command = " ".join(commands)

        run_command(helm_command)
