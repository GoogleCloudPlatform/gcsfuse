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

"""This program takes in a json test-config file, finds out valid FIO

workloads from it and generates and deploys a helm chart for each FIO
workload.
"""

import argparse
import subprocess
import fio_workload


def run_command(command: str):
  """Runs the given string command as a subprocess."""
  result = subprocess.run(command.split(' '), capture_output=True, text=True)
  print(result.stdout)
  print(result.stderr)


def createHelmInstallCommands(fioWorkloads):
  """Create helm install commands for the given set of fioWorkload objects."""
  helm_commands = []
  for fioWorkload in fioWorkloads:
    for readType in fioWorkload.readTypes:
      commands = [
          (
              'helm install'
              f' fio-loading-test-{fioWorkload.fileSize.lower()}-{readType}-{fioWorkload.scenario} loading-test'
          ),
          f'--set bucketName={fioWorkload.bucket}',
          f'--set scenario={fioWorkload.scenario}',
          f'--set fio.readType={readType}',
          f'--set fio.fileSize={fioWorkload.fileSize}',
          f'--set fio.blockSize={fioWorkload.blockSize}',
          f'--set fio.filesPerThread={fioWorkload.filesPerThread}',
          f'--set fio.numThreads={fioWorkload.numThreads}',
      ]

      helm_command = ' '.join(commands)
      helm_commands.append(helm_command)
  return helm_commands


def main(args) -> None:
  fioWorkloads = fio_workload.ParseTestConfigForFioWorkloads(
      args.workload_config
  )
  helmInstallCommands = createHelmInstallCommands(fioWorkloads)
  for helmInstallCommand in helmInstallCommands:
    print(f'{helmInstallCommand}')
    if not args.dry_run:
      run_command(helmInstallCommand)


if __name__ == '__main__':
  parser = argparse.ArgumentParser(
      prog='FIO test runner',
      description=(
          'This program takes in a json test-config file, finds out valid FIO'
          ' workloads from it and generates and deploys a helm chart for each'
          ' FIO workload.'
      ),
  )
  parser.add_argument(
      '--workload-config',
      help='Runs FIO tests using this JSON workload configuration',
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
