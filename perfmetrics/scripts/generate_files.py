# Copyright 2024 Google LLC
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

"""This script takes input the number of files and the file size and creates
required files and simultaneously uploads them on the gcs bucket.
The progress is updated in 'output.out' file.
Usage: nohup python3 generate_files.py <config file> [--keep_files]
The progress is updated in output.out file.
"""
import argparse
import configparser
from datetime import datetime as dt
import logging
import os
import subprocess
from subprocess import Popen
import sys

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
                                            filename_prefix,
                                            local_destination_folder,
                                            upload_to_gcs_bucket):

  for batch_start in range(1, num_of_files + 1, BATCH_SIZE):
    for file_num in range(batch_start, batch_start + BATCH_SIZE):
      if file_num > num_of_files:
        break

      file_name = '{}_{}'.format(filename_prefix, file_num)
      temp_file = '{}/{}.txt'.format(TEMPORARY_DIRECTORY, file_name)

      # Creating files in temporary folder:
      with open(temp_file, 'wb') as out:
        if(file_size_unit.lower() == 'gb'):
          out.truncate(1024 * 1024 * 1024 * int(file_size))
        if(file_size_unit.lower() == 'mb'):
          out.truncate(1024 * 1024 * int(file_size))
        if(file_size_unit.lower() == 'kb'):
          out.truncate(1024 * int(file_size))
        if(file_size_unit.lower() == 'b'):
          out.truncate(int(file_size))

    num_files = os.listdir(TEMPORARY_DIRECTORY)
    if not num_files:
      return 0

    # Uploading batch files to GCS
    if upload_to_gcs_bucket:
      process = Popen(
          'gcloud storage cp --recursive {}/* {}'.format(TEMPORARY_DIRECTORY,
                                          destination_blob_name),
          shell=True)
      process.communicate()
      exit_code = process.wait()
      if exit_code != 0:
        return exit_code

    # Copying batch files from temporary to local destination folder:
    subprocess.call(
        'cp -r {}/* {}'.format(TEMPORARY_DIRECTORY, local_destination_folder),
        shell=True)

    # Deleting batch files from temporary folder:
    subprocess.call('rm -rf {}/*'.format(TEMPORARY_DIRECTORY), shell=True)

    # Writing number of files uploaded to output file after every batch uploads:
    logmessage('{}/{} files uploaded to {}\n'.format(file_num, num_of_files,
                                                     destination_blob_name))

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
      help='Provide path of the config file',)
  parser.add_argument(
      '--keep_files',
      help='Please specify whether to keep local files or not',
      action='store_true',
      default=False,
      required=False)

  args = parser.parse_args(argv[1:])

  # Checking that gcloud is installed:
  logmessage('Checking whether gcloud is installed.\n')
  process = Popen('gcloud -v', shell=True)
  process.communicate()
  exit_code = process.wait()
  if(exit_code != 0):
    print('gcloud not installed.')
    subprocess.call('bash', shell=True)

  config = configparser.ConfigParser()
  config.read(os.path.abspath(args.config_file))

  bucket_name = config['DEFAULT']['bucket_name']

  # Making temporary folder and local bucket directory:
  logmessage('Making a temporary directory.\n')
  subprocess.call(['mkdir', '-p', TEMPORARY_DIRECTORY])
  subprocess.call(['mkdir', bucket_name])

  for section in config.sections():
    destination_folder = config[section]['destination_folder']

    # Checking whether subfolder exists:
    if('destination_sub_folder' in config[section]):
      destination_sub_folder = config[section]['destination_sub_folder']
    else:
      destination_sub_folder = ''

    num_of_files = config[section]['num_of_files']
    file_size_unit = config[section]['file_size'][-2:]
    file_size = config[section]['file_size'][:-2]
    file_name_prefix = config[section]['file_name_prefix']

    # Creating folders locally:
    local_destination_folder = '{}/{}/{}/'.format(bucket_name,
                                                  destination_folder,
                                                  destination_sub_folder)
    subprocess.call(['mkdir', '-p', local_destination_folder])

    destination_blob_name = 'gs://{}/{}/{}/'.format(bucket_name,
                                                    destination_folder,
                                                    destination_sub_folder)

    exit_code = generate_files_and_upload_to_gcs_bucket(destination_blob_name,
                                                        int(num_of_files), file_size_unit,
                                                        int(file_size), file_name_prefix,
                                                        local_destination_folder,
                                                        True)
    if exit_code != 0:
      print('Exited with code {}'.format(exit_code))
      subprocess.call('bash', shell=True)

    keep_files = args.keep_files
    if(keep_files == False):
    # Deleting bucket directory:
      logmessage('Deleting the local directories.\n')
      subprocess.call(['rm', '-r', bucket_name])

  # Deleting temporary folder:
  logmessage('Deleting the temporary directory.\n')
  subprocess.call(['rm', '-r', TEMPORARY_DIRECTORY])

  logmessage('Process complete.\n')
