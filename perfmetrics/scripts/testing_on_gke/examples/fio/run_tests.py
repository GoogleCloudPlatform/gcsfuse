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

"""Generates and deploys helm charts for FIO workloads.

This program takes in a json test-config file, finds out valid FIO workloads in
it and generates and deploys a helm chart for each valid FIO workload.
"""

# system imports
import argparse
import os
import subprocess
import sys

# local imports from other directories
sys.path.append(os.path.join(os.path.dirname(__file__), '..', 'utils'))
from run_tests_common import escape_commas_in_string, parse_args, add_iam_role_for_buckets
from utils import UnknownMachineTypeError, resource_limits, run_command

# local imports from same directory
import fio_workload


def createHelmInstallCommands(
    fioWorkloads: set,
    instanceId: str,
    machineType: str,
) -> list:
  """Creates helm install commands for the given fioWorkload objects."""
  helm_commands = []
  try:
    resourceLimits, resourceRequests = resource_limits(machineType)
  except UnknownMachineTypeError:
    print(
        f'Found unknown machine-type: {machineType}, defaulting resource limits'
        ' to cpu=0,memory=0'
    )
    resourceLimits = {'cpu': 0, 'memory': '0'}
    resourceRequests = resourceLimits

  for fioWorkload in fioWorkloads:
    for readType in fioWorkload.readTypes:
      chartName, podName, outputDirPrefix = fio_workload.FioChartNamePodName(
          fioWorkload, instanceId, readType
      )
      commands = [
          f'helm install {chartName} loading-test',
          f'--set bucketName={fioWorkload.bucket}',
          f'--set scenario={fioWorkload.scenario}',
          f'--set fio.readType={readType}',
          f'--set fio.fileSize={fioWorkload.fileSize}',
          f'--set fio.blockSize={fioWorkload.blockSize}',
          f'--set fio.filesPerThread={fioWorkload.filesPerThread}',
          f'--set fio.numThreads={fioWorkload.numThreads}',
          f'--set instanceId={instanceId}',
          (
              '--set'
              f' gcsfuse.mountOptions={escape_commas_in_string(fioWorkload.gcsfuseMountOptions)}'
          ),
          f'--set nodeType={machineType}',
          f'--set podName={podName}',
          f'--set outputDirPrefix={outputDirPrefix}',
          f"--set resourceLimits.cpu={resourceLimits['cpu']}",
          f"--set resourceLimits.memory={resourceLimits['memory']}",
          f"--set resourceRequests.cpu={resourceRequests['cpu']}",
          f"--set resourceRequests.memory={resourceRequests['memory']}",
      ]

      helm_command = ' '.join(commands)
      helm_commands.append(helm_command)
  return helm_commands


def main(args) -> None:
  fioWorkloads = fio_workload.ParseTestConfigForFioWorkloads(
      args.workload_config
  )
  helmInstallCommands = createHelmInstallCommands(
      fioWorkloads,
      args.instance_id,
      args.machine_type,
  )
  buckets = (fioWorkload.bucket for fioWorkload in fioWorkloads)
  role = 'roles/storage.objectUser'
  add_iam_role_for_buckets(
      buckets,
      role,
      args.project_id,
      args.project_number,
      args.namespace,
      args.ksa,
  )
  for helmInstallCommand in helmInstallCommands:
    print(f'{helmInstallCommand}')
    if not args.dry_run:
      run_command(helmInstallCommand)


if __name__ == '__main__':
  args = parse_args()
  main(args)
