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

"""This program takes in a json dlio test-config file and generates and deploys helm charts."""

import argparse
from collections.abc import Sequence
import os
import subprocess

from absl import app
import dlio_workload


def run_command(command: str):
  result = subprocess.run(command.split(' '), capture_output=True, text=True)
  print(result.stdout)
  print(result.stderr)


def createHelmInstallCommands(dlioWorkloads):
  helm_commands = []
  for dlioWorkload in dlioWorkloads:
    for batchSize in dlioWorkload.batchSizes:
      commands = [
          (
              'helm install'
              f' {dlioWorkload.bucket}-{batchSize}-{dlioWorkload.scenario} unet3d-loading-test'
          ),
          f'--set bucketName={dlioWorkload.bucket}',
          f'--set scenario={dlioWorkload.scenario}',
          f'--set dlio.numFilesTrain={dlioWorkload.numFilesTrain}',
          f'--set dlio.recordLength={dlioWorkload.recordLength}',
          f'--set dlio.batchSize={batchSize}',
      ]

      helm_command = ' '.join(commands)
      helm_commands.append(helm_command)
  return helm_commands


def main(args) -> None:
  dlioWorkloads = dlio_workload.ParseTestConfigForDlioWorkloads(
      args.workload_config
  )
  helmInstallCommands = createHelmInstallCommands(dlioWorkloads)
  for helmInstallCommand in helmInstallCommands:
    print(f'{helmInstallCommand}')
    run_command(helmInstallCommand)


if __name__ == '__main__':
  parser = argparse.ArgumentParser(
      prog='DLIO test runner',
      description=(
          'This program takes in a json dlio test-config file and generates'
          ' helm install commands.'
      ),
      # epilog='Text at the bottom of help',
  )
  parser.add_argument('--workload-config')  # positional argument
  args = parser.parse_args()
  main(args)
