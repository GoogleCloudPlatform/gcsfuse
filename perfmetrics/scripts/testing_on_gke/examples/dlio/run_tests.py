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

"""This program takes in a json test-config file, finds out valid

DLIO workloads from it and generates and deploys a helm chart for
each DLIO workload.
"""

import argparse
import subprocess
import dlio_workload


def run_command(command: str):
  """Runs the given string command as a subprocess."""
  result = subprocess.run(command.split(' '), capture_output=True, text=True)
  print(result.stdout)
  print(result.stderr)


DEFAULT_GCSFUSE_MOUNT_OPTIONS = 'implicit-dirs'


def escapeCommasInString(gcsfuseMountOptions: str) -> str:
  return gcsfuseMountOptions.replace(',', '\,')


def createHelmInstallCommands(
    dlioWorkloads: set,
    instanceId: str,
    gcsfuseMountOptions: str,
    machineType: str,
):
  """Create helm install commands for the given set of dlioWorkload objects."""
  helm_commands = []
  if not gcsfuseMountOptions:
    gcsfuseMountOptions = DEFAULT_GCSFUSE_MOUNT_OPTIONS
  for dlioWorkload in dlioWorkloads:
    for batchSize in dlioWorkload.batchSizes:
      commands = [
          (
              'helm install'
              f' dlio-unet3d-{dlioWorkload.scenario}-{dlioWorkload.numFilesTrain}-{dlioWorkload.recordLength}-{batchSize} unet3d-loading-test'
          ),
          f'--set bucketName={dlioWorkload.bucket}',
          f'--set scenario={dlioWorkload.scenario}',
          f'--set dlio.numFilesTrain={dlioWorkload.numFilesTrain}',
          f'--set dlio.recordLength={dlioWorkload.recordLength}',
          f'--set dlio.batchSize={batchSize}',
          f'--set instanceId={instanceId}',
          (
              '--set'
              f' gcsfuse.mountOptions={escapeCommasInString(gcsfuseMountOptions)}'
          ),
          f'--set nodeType={machineType}',
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
      args.gcsfuse_mount_options,
      args.machine_type,
  )
  for helmInstallCommand in helmInstallCommands:
    print(f'{helmInstallCommand}')
    if not args.dry_run:
      run_command(helmInstallCommand)


if __name__ == '__main__':
  parser = argparse.ArgumentParser(
      prog='DLIO Unet3d test runner',
      description=(
          'This program takes in a json test-config file, finds out valid DLIO'
          ' workloads from it and generates and deploys a helm chart for each'
          ' DLIO workload.'
      ),
  )
  parser.add_argument(
      '--workload-config',
      metavar='JSON workload configuration file path',
      help='Runs DLIO Unet3d tests using this JSON workload configuration.',
      required=True,
  )
  parser.add_argument(
      '--instance-id',
      help='unique string ID for current test-run',
      required=True,
  )
  parser.add_argument(
      '--gcsfuse-mount-options',
      metavar='GCSFuse mount options',
      help=(
          'GCSFuse mount-options, in JSON stringified format, to be set for the'
          ' scenario gcsfuse-generic.'
      ),
      required=True,
  )
  parser.add_argument(
      '--machine-type',
      metavar='Machine-type of the GCE VM or GKE cluster node',
      help='Machine-type of the GCE VM or GKE cluster node e.g. n2-standard-32',
      required=True,
  )
  parser.add_argument(
      '-n',
      '--dry-run',
      action='store_true',
      help=(
          'Only print out the test configurations that will run,'
          ' not actually run them.'
      ),
  )
  args = parser.parse_args()
  main(args)
