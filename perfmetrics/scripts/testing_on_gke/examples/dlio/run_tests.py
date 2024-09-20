#!/usr/bin/env python

# Copyright 2018 The Kubernetes Authors.
# Copyright 2024 Google LLC
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

"""Generates and deploys helm charts for DLIO workloads.

This program takes in a json test-config file, finds out valid
DLIO workloads from it and generates and deploys a helm chart for
each valid DLIO workload.
"""

# system imports
import argparse
import os
import subprocess
import sys

# local imports from other directories
sys.path.append(os.path.join(os.path.dirname(__file__), '..', 'utils'))
from run_tests_common import escape_commas_in_string, parse_args, run_command

# local imports from same directory
import dlio_workload


def createHelmInstallCommands(
    dlioWorkloads: set,
    instanceId: str,
    machineType: str,
) -> list:
  """Creates helm install commands for the given dlioWorkload objects."""
  helm_commands = []
  for dlioWorkload in dlioWorkloads:
    for batchSize in dlioWorkload.batchSizes:
      chartName, podName, outputDirPrefix = dlio_workload.DlioChartNamePodName(
          dlioWorkload, instanceId, batchSize
      )
      commands = [
          f'helm install {chartName} unet3d-loading-test',
          f'--set bucketName={dlioWorkload.bucket}',
          f'--set scenario={dlioWorkload.scenario}',
          f'--set dlio.numFilesTrain={dlioWorkload.numFilesTrain}',
          f'--set dlio.recordLength={dlioWorkload.recordLength}',
          f'--set dlio.batchSize={batchSize}',
          f'--set instanceId={instanceId}',
          (
              '--set'
              f' gcsfuse.mountOptions={escape_commas_in_string(dlioWorkload.gcsfuseMountOptions)}'
          ),
          f'--set nodeType={machineType}',
          f'--set podName={podName}',
          f'--set outputDirPrefix={outputDirPrefix}',
      ]

      helm_command = ' '.join(commands)
      helm_commands.append(helm_command)
  return helm_commands


def main(args) -> None:
  dlioWorkloads = dlio_workload.ParseTestConfigForDlioWorkloads(
      args.workload_config
  )
  helmInstallCommands = createHelmInstallCommands(
      dlioWorkloads,
      args.instance_id,
      args.machine_type,
  )
  for helmInstallCommand in helmInstallCommands:
    print(f'{helmInstallCommand}')
    if not args.dry_run:
      run_command(helmInstallCommand)


if __name__ == '__main__':
  args = parse_args()
  main(args)
