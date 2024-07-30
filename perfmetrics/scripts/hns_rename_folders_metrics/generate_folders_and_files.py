# Copyright 2024 Google Inc. All Rights Reserved.
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
# To run the script, run in terminal :
# python3 generate_folder_and_files.py <config-file.json>

import argparse
import json
from datetime import datetime as dt
import logging
import sys
import subprocess
from subprocess import Popen

OUTPUT_FILE = str(dt.now().isoformat()) + '.out'

logging.basicConfig(
    level=logging.ERROR,
    format='%(asctime)s [%(levelname)s] %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)],
)
logger = logging.getLogger()


def logmessage(message) -> None:
  with open(OUTPUT_FILE, 'a') as out:
    out.write(message)
  logger.error(message)


def check_for_config_file_inconsistency(config) -> (int):
  """
  Checks for inconsistencies in the provided configuration.

  Args:
      config: The configuration dictionary to be checked.

  Returns:
      0 if no inconsistencies are found, 1 otherwise.
  """
  if "name" not in config:
    logmessage("Bucket name not specified")
    return 1

  if "folders" in config:
    if not ("num_folders" in config["folders"] or "folder_structure" in config[
      "folders"]):
      logmessage("Key missing for nested folder")
      return 1

    if config["folders"]["num_folders"] != len(
        config["folders"]["folder_structure"]):
      logmessage("Inconsistency in the folder structure")
      return 1

  if "nested_folders" in config:
    if not ("folder_name" in config["nested_folders"] or
            "num_folders" in config["nested_folders"] or
            "folder_structure" in config["nested_folders"]):
      logmessage("Key missing for nested folder")
      return 1

    if config["nested_folders"]["num_folders"] != len(
        config["nested_folders"]["folder_structure"]):
      logmessage("Inconsistency in the nested folder")
      return 1

  return 0


def list_directory(path) -> list:
  """Returns the list containing path of all the contents present in the current directory.

  Args:
    path: Path of the directory.

  Returns:
    A list containing path of all contents present in the input path.
  """
  try:
    contents = subprocess.check_output(
        'gcloud storage ls {}'.format(path), shell=True)
    contents_url = contents.decode('utf-8').split('\n')[:-1]
    return contents_url
  except subprocess.CalledProcessError as e:
    logmessage(e.output.decode('utf-8'))
    subprocess.call('bash', shell=True)


if __name__ == '__main__':
  argv = sys.argv
  if len(argv) < 2:
    raise TypeError('Incorrect number of arguments.\n'
                    'Usage: '
                    'python3 generate_files.py <config_file> [--keep_files]')

  parser = argparse.ArgumentParser()
  parser.add_argument(
      'config_file',
      help='Provide path of the config file', )
  parser.add_argument(
      '--keep_files',
      help='Please specify whether to keep local files/folders or not',
      action='store_true',
      default=False,
      required=False)

  args = parser.parse_args(argv[1:])

  # Checking that gcloud is installed:
  logmessage('Checking whether gcloud is installed.\n')
  process = Popen('gcloud version', shell=True)
  process.communicate()
  exit_code = process.wait()
  if exit_code != 0:
    print('gcloud not installed.')
    subprocess.call('bash', shell=True)

  directory_structure = json.load(open(args.config_file))

  exit_code = check_for_config_file_inconsistency(directory_structure)
  if exit_code != 0:
    print('Exited with code {}'.format(exit_code))
    subprocess.call('bash', shell=True)
