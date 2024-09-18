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

"""Common code for fio/run_tests.py and dlio/run_tests.py"""

import argparse
import subprocess
import sys


def run_command(command: str) -> int:
  """Runs the given string command as a subprocess.

  Returns exit-code which would be non-zero for error.
  """
  result = subprocess.run(
      [word for word in command.split(' ') if (word and not str.isspace(word))],
      capture_output=True,
      text=True,
  )
  print(result.stdout)
  print(result.stderr)
  return result.returncode


def escape_commas_in_string(unescapedStr: str) -> str:
  """Returns equivalent string with ',' replaced with '\,' ."""
  return unescapedStr.replace(',', '\,')


def parse_args():
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
      metavar='JSON workload configuration file path',
      help='Runs FIO tests from this JSON workload configuration file.',
      required=True,
  )
  parser.add_argument(
      '--instance-id',
      metavar='A unique string ID to represent the test-run',
      help=(
          'Set to a unique string ID for current test-run. Do not put spaces'
          ' in it.'
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
  for argument in [
      'instance_id',
      'machine_type',
  ]:
    value = getattr(args, argument)
    if not value.strip():
      raise Exception(
          f'Argument {argument} (value="{value}") is empty or contains only'
          ' spaces.'
      )
    if ' ' in value:
      raise Exception(
          f'Argument {argument} (value="{value}") contains space in it, which'
          ' is not supported.'
      )

  return args
