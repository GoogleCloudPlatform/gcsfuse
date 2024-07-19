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

# limitations under the License.# This script takes json file as input which
# contains the number of folders and their respective structure and creates the
# specified structure  ( folders and subfolders only )and
# uploads on the gcs bucket
# Progress in output.out files
# To run the script, run in terminal :
# python3 generate_folder_and_files.py <config-file.json>

import argparse
import json
from datetime import datetime as dt
import logging
import os
import sys
import subprocess
from subprocess import Popen

OUTPUT_FILE = str(dt.now().isoformat()) + '.out'
TEMPORARY_DIRECTORY = './tmp/data_gen'
BATCH_SIZE = 100

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)],
)
logger = logging.getLogger()


def logmessage(message) -> None:
  with open(OUTPUT_FILE, 'a') as out:
    out.write(message)
  logger.info(message)


def generate_files_and_upload_to_gcs_bucket(destination_blob_name, num_of_files,
    file_size_unit, file_size,
    filename_prefix) -> int:
  for batch_start in range(1, num_of_files + 1, BATCH_SIZE):
    for file_num in range(batch_start, batch_start + BATCH_SIZE):
      if file_num > num_of_files:
        break

      file_name = '{}_{}'.format(filename_prefix, file_num)
      temp_file = '{}/{}.txt'.format(TEMPORARY_DIRECTORY, file_name)

      # Creating files in temporary folder:
      with open(temp_file, 'wb') as out:
        if (file_size_unit.lower() == 'gb'):
          out.truncate(1024 * 1024 * 1024 * int(file_size))
        if (file_size_unit.lower() == 'mb'):
          out.truncate(1024 * 1024 * int(file_size))
        if (file_size_unit.lower() == 'kb'):
          out.truncate(1024 * int(file_size))
        if (file_size_unit.lower() == 'b'):
          out.truncate(int(file_size))

    num_files = os.listdir(TEMPORARY_DIRECTORY)

    if not num_files:
      logmessage("Files were not created locally")
      return -1

    # starting upload to the gcs bucket
    process = Popen(
        'gsutil -m cp -r {}/* {}'.format(TEMPORARY_DIRECTORY,
                                         destination_blob_name),
        shell=True)
    process.communicate()
    exit_code = process.wait()
    if exit_code != 0:
      return exit_code

    # Delete local files from temporary directory
    subprocess.call('rm -rf {}/*'.format(TEMPORARY_DIRECTORY), shell=True)

    # Writing number of files uploaded to output file after every batch uploads:
    logmessage('{}/{} files uploaded to {}\n'.format(file_num, num_of_files,
                                                     destination_blob_name))
  return 0


def parse_and_generate_directory_structure(dir_str) -> int:
  if not dir_str:
    logmessage("Directory structure not specified via config file.")
    return -1
  else:
    bucket_name = dir_str["name"]
    # Making temporary folder and local bucket directory:
    logmessage('Making a temporary directory.\n')
    subprocess.call(['mkdir', '-p', TEMPORARY_DIRECTORY])

    # creating a folder structure in gcs bucket
    if "folders" not in dir_str:
      logmessage("No folders specified in the config file")
    else:
      if dir_str["folders"]["num_folders"] != len(
          dir_str["folders"]["folder_structure"]):
        logmessage("Inconsistency in the directory structure config file")
        return -1
      else:
        for folder in dir_str["folders"]["folder_structure"]:
          # create the folder
          folder_name = folder["name"]
          num_files = folder["num_files"]
          filename_prefix = folder["file_name_prefix"]
          file_size = folder["file_size"][:-2]
          file_size_unit = folder["file_size"][-2:]
          # Creating folders locally in temp directory and copying to gcs bucket:
          destination_blob_name = 'gs://{}/{}/'.format(bucket_name, folder_name)
          generate_files_and_upload_to_gcs_bucket(destination_blob_name,
                                                  int(num_files),
                                                  file_size_unit,
                                                  int(file_size),
                                                  filename_prefix)

    # creating a nested folder structure in gcs bucket
    if "nested_folders" not in dir_str:
      logmessage("No nested folders specified in the config file")
    else:
      if dir_str["nested_folders"]["num_folders"] != len(
          dir_str["nested_folders"]["folder_structure"]):
        logmessage("Inconsistency in the directory structure config file")
        return -1
      else:
        sub_folder_name = dir_str["nested_folders"]["folder_name"]
        for folder in dir_str["nested_folders"]["folder_structure"]:
          # create the folder
          folder_name = folder["name"]
          num_files = folder["num_files"]
          filename_prefix = folder["file_name_prefix"]
          file_size = folder["file_size"][:-2]
          file_size_unit = folder["file_size"][-2:]
          file_size_unit = folder["file_size"][-2:]

          # # Creating folders locally in temp directory and copying to gcs bucket:
          destination_blob_name = 'gs://{}/{}/{}/'.format(bucket_name,
                                                          sub_folder_name,
                                                          folder_name)
          generate_files_and_upload_to_gcs_bucket(destination_blob_name,
                                                  int(num_files),
                                                  file_size_unit,
                                                  int(file_size),
                                                  filename_prefix)

    # Deleting temporary folder:
    logmessage('Deleting the temporary directory.\n')
    subprocess.call(['rm', '-r', TEMPORARY_DIRECTORY])

    return 0


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

  # Checking that gsutil is installed:
  logmessage('Checking whether gsutil is installed.\n')
  process = Popen('gsutil version', shell=True)
  process.communicate()
  exit_code = process.wait()
  if exit_code != 0:
    print('Gsutil not installed.')
    subprocess.call('bash', shell=True)

  directory_structure = json.load(open(args.config_file))
  exit_code = parse_and_generate_directory_structure(directory_structure)

  if exit_code != 0:
    print('Exited with code {}'.format(exit_code))
    subprocess.call('bash', shell=True)
