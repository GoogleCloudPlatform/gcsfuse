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
# To run the script, run in terminal :
# python3 generate_folders_and_files.py <config-file.json>

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
LOG_ERROR = "error"
LOG_INFO = "info"

logging.basicConfig(
    level=logging.ERROR,
    format='%(asctime)s [%(levelname)s] %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)],
)
logger = logging.getLogger()


def _logmessage(message,type) -> None:
  with open(OUTPUT_FILE, 'a') as out:
    out.write(message)
  if type == LOG_ERROR:
    logger.error(message)
  elif type == LOG_INFO:
    logger.info(message)


def _check_for_config_file_inconsistency(config) -> (int):
  """
  Checks for inconsistencies in the provided configuration.

  Args:
      config: The configuration dictionary to be checked.

  Returns:
      0 if no inconsistencies are found, 1 otherwise.
  """
  if "name" not in config:
    _logmessage("Bucket name not specified",LOG_ERROR)
    return 1

  if "folders" in config:
    if not ("num_folders" in config["folders"] or "folder_structure" in config[
      "folders"]):
      _logmessage("Key missing for nested folder",LOG_ERROR)
      return 1

    if config["folders"]["num_folders"] != len(
        config["folders"]["folder_structure"]):
      _logmessage("Inconsistency in the folder structure",LOG_ERROR)
      return 1

  if "nested_folders" in config:
    if not ("folder_name" in config["nested_folders"] or
            "num_folders" in config["nested_folders"] or
            "folder_structure" in config["nested_folders"]):
      _logmessage("Key missing for nested folder",LOG_ERROR)
      return 1

    if config["nested_folders"]["num_folders"] != len(
        config["nested_folders"]["folder_structure"]):
      _logmessage("Inconsistency in the nested folder",LOG_ERROR)
      return 1

  return 0


def _list_directory(path) -> list:
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
    _logmessage(e.output.decode('utf-8'),LOG_ERROR)


def _compare_folder_structure(folder, folder_url) -> bool:
  """Checks if the number of files inside folder in GCS bucket matches the
  num_files parameter for folder.

  Example folder structure
  {
  "name": "example-folder" ,
  "folders" : {
    "num_folders": 1,
    "folder_structure" : [
      {
        "name": "1k_files_folder" ,
        "num_files": 1000 ,
        "file_name_prefix": "file" ,
        "file_size": "1kb"
      }
    ]
    },
   "nested_folders": {
    "folder_name": "nested_folder",
    "num_folders": 1,
    "folder_structure" :  [
      {
        "name": "1k_files_nested_folder" ,
        "num_files": 1000 ,
        "file_name_prefix": "file" ,
        "file_size": "1kb"
      }
      ]
    }
  }

  Args:
    folder: Json Object representing the folder.
    folder_url: Corresponding folder url in the GCS bucket.

  Returns:
    true if the number of files in the folder in GCS bucket matches the num_files
    parameter of the folder JSON object.
    false otherwise
  """
  try:
    files_in_folder = _list_directory(folder_url)
    if len(files_in_folder) != folder["num_files"]:
      return False
  except:
    # If the list directory fails wth url did not match object, folder
    # specified in JSON folder object does not exist in bucket.
    return False

  return True


def _compare_folders(folder_structure, parent_url) -> bool:
  """ Checks that the folder structure matches for each folder under parent_url.

  Args:
    folder_structure: JSON object representing folders
    parent_url: The GCS URL of the parent directory

  Returns:
    true if the structure of the parent folder represented by parent_url in GCS
    bucket matches the folder_structure JSON object.
    false otherwise
  """
  for folder in folder_structure:
    folder_url = '{}/{}'.format(parent_url, folder["name"])
    match = _compare_folder_structure(folder, folder_url)
    if not match:
      return False
  return True


def _check_if_dir_structure_exists(directory_structure) -> bool:
  """Checks if the directory structure mentioned in the config file already
  exists in the GCS bucket.

  Args:
    directory_structure: Json Object representing the directory structure.

  Returns:
    true if the existing structure in GCS bucket exactly matches with the config
     file.
    false otherwise.
  """
  bucket_name = directory_structure["name"]
  bucket_url = 'gs://{}'.format(bucket_name)

  # Check for top level folders.
  folders = _list_directory(bucket_url)
  nested_folder_count = "nested_folders" in directory_structure
  if "folders" in directory_structure:
    # Note: It is already validated during input file consistency check that the
    # keys num_folder,folder_structure is specified whenever folder section is
    # included.
    if len(folders) != directory_structure["folders"][
      "num_folders"] + nested_folder_count:
      return False

    # For each non-nested folder , check the count of files.
    match = _compare_folders(directory_structure["folders"]["folder_structure"],
                             bucket_url)
    if not match:
      return False

  # Check the number of second level folders in nested folders.
  if nested_folder_count:
    # Note: It is already validated during input file consistency check that the
    # keys folder_name,num_folder,folder_structure is specified whenever folder
    # section is included.
    nested_folder = directory_structure["nested_folders"]["folder_name"]
    nested_folder_url = '{}/{}'.format(bucket_url, nested_folder)
    try:

      second_level_folders = _list_directory(nested_folder_url)
      if len(second_level_folders) != directory_structure["nested_folders"][
        "num_folders"]:
        return False

      # For each second level folder in "nested" folders, check the count of files.
      match = _compare_folders(
          directory_structure["nested_folders"]["folder_structure"],
          nested_folder_url)
      if not match:
        return False
    except:
      # Folder specified in JSON config file under the nested folder structrue does
      # not exist in bucket.
      return False

  return True


def _delete_existing_data_in_gcs_bucket(gcs_bucket)->(int):
  # TODO: Handle case where delete operation called on empty bucket results in
  #  exiting due to "gcs object not found at url" when using gcloud command
  try:
    subprocess.check_output(
        'gcloud alpha storage rm -r gs://{}/*'.format(gcs_bucket), shell=True)
    return 0
  except subprocess.CalledProcessError as e:
    _logmessage(e.output.decode('utf-8'),LOG_ERROR)
    return 1


def _generate_files_and_upload_to_gcs_bucket(destination_blob_name, num_of_files,
    file_size_unit, file_size,
    filename_prefix) -> int:
  # Creating folders locally in temp directory and copying to gcs bucket.
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
      _logmessage("Files were not created locally",LOG_ERROR)
      return 1

    # Starting upload to the gcs bucket.
    try:
        subprocess.Popen(
        'gcloud storage cp --recursive {}/* {}'.format(TEMPORARY_DIRECTORY,
                                         destination_blob_name),
        shell=True).communicate()
    except subprocess.CalledProcessError as e:
      _logmessage("Issue while uploading files to GCS bucket.Aborting...",LOG_ERROR)
      return 1

    # Delete local files from temporary directory.
    subprocess.call('rm -rf {}/*'.format(TEMPORARY_DIRECTORY), shell=True)

    # Writing number of files uploaded to output file after every batch uploads.
    _logmessage('{}/{} files uploaded to {}\n'.format(len(num_files), num_of_files,
                                                     destination_blob_name),LOG_INFO)
  return 0


def _create_folders(folder,destination_url_prefix) -> (int):
  """
   Creates folder at destination url using folder JSON object.

   Args:
       folder: JSON object representing folder to be created.
       destination_url_prefix: Destination at which folder is to be created.

   Returns:
       0 if error is encountered, 1 otherwise.
   """
  folder_name = folder["name"]
  num_files = folder["num_files"]
  filename_prefix = folder["file_name_prefix"]
  file_size = folder["file_size"][:-2]
  file_size_unit = folder["file_size"][-2:]
  destination_blob_name = '{}/{}/'.format(destination_url_prefix, folder_name)
  exit_code= _generate_files_and_upload_to_gcs_bucket(destination_blob_name,
                                           int(num_files),
                                           file_size_unit,
                                           int(file_size),
                                           filename_prefix)
  return exit_code


def _delete_temp_directory():
  _logmessage('Deleting the temporary directory.\n',LOG_INFO)
  try:
    subprocess.call(['rm', '-r', TEMPORARY_DIRECTORY])
  except subprocess.CalledProcessError as e:
    _logmessage("Could not delete directory.\n",LOG_ERROR)
    return 1
  return 0


def _parse_and_generate_directory_structure(dir_str) -> int:
  if not dir_str:
    _logmessage("Directory structure not specified via config file.",LOG_ERROR)
    return 1
  else:
    bucket_name = dir_str["name"]
    # Making temporary folder and local bucket directory:
    _logmessage('Making a temporary directory.\n',LOG_INFO)
    subprocess.call(['mkdir', '-p', TEMPORARY_DIRECTORY])

    # Creating a folder structure in gcs bucket.
    if "folders" not in dir_str:
      _logmessage("No folders specified in the config file",LOG_INFO)
    else:
      destination_url="gs://{}".format(bucket_name)
      for folder in dir_str["folders"]["folder_structure"]:
        # Create the folder at destination url.
        exit_code = _create_folders(folder,destination_url)
        if exit_code != 0:
          _logmessage("Could not create files for folder {}".format(folder["name"]),LOG_ERROR)
          _delete_temp_directory()
          return exit_code


    # Creating a nested folder structure in gcs bucket.
    if "nested_folders" not in dir_str:
      _logmessage("No nested folders specified in the config file",LOG_INFO)
    else:
      sub_folder_name = dir_str["nested_folders"]["folder_name"]
      destination_url = 'gs://{}/{}'.format(bucket_name,sub_folder_name)
      for folder in dir_str["nested_folders"]["folder_structure"]:
        # Create the folder at destination url.
        exit_code = _create_folders(folder,destination_url)
        if exit_code != 0:
          _logmessage("Could not create files for folder {}/{}".format(
              sub_folder_name,folder["name"]),LOG_ERROR)
          _delete_temp_directory()
          return exit_code

    # Deleting temporary folder if all file creation is successful:
    exit_code = _delete_temp_directory()
    return exit_code


if __name__ == '__main__':
  argv = sys.argv
  if len(argv) < 2:
    raise TypeError('Incorrect number of arguments.\n'
                    'Usage: '
                    'python3 generate_folders_and_files.py <config_file> [--keep_files]')

  parser = argparse.ArgumentParser()
  parser.add_argument(
      'config_file',
      help='Provide path of the JSON config file', )
  parser.add_argument(
      '--keep_files',
      help='Please specify whether to keep local files/folders or not',
      action='store_true',
      default=False,
      required=False)

  args = parser.parse_args(argv[1:])

  # Checking that gcloud is installed:
  _logmessage('Checking whether gcloud is installed.\n',LOG_INFO)
  process = subprocess.Popen('gcloud version', shell=True)
  process.communicate()
  exit_code = process.wait()
  if exit_code != 0:
    _logmessage('gcloud not installed.',LOG_ERROR)
    subprocess.call('bash', shell=True)

  directory_structure = json.load(open(args.config_file))

  exit_code = _check_for_config_file_inconsistency(directory_structure)
  if exit_code != 0:
    _logmessage('Exited with code {}'.format(exit_code),LOG_ERROR)
    subprocess.call('bash', shell=True)

  # Compare the directory structure with the JSON config file to avoid recreation of
  # same test data.
  dir_structure_present = _check_if_dir_structure_exists(directory_structure)

  # If directory structure does not exist/match the structure in the config file
  # delete any existing files in bucket.
  if not dir_structure_present:
    exit_code = _delete_existing_data_in_gcs_bucket(directory_structure["name"])
    if exit_code != 0:
      _logmessage('Error while deleting content in bucket.Exiting...!', LOG_ERROR)
      subprocess.call('bash', shell=True)

    exit_code = _parse_and_generate_directory_structure(directory_structure)
    if exit_code != 0:
      _logmessage('Error while trying to generate files...', LOG_ERROR)
