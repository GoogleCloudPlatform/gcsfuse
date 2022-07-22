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


def logmessage(message) -> None:
  with open(OUTPUT_FILE, 'a') as out:
    out.write(message)
  print(message)


def generate_files_and_upload_to_gcs_bucket(destination_blob_name, num_of_files,
                                            file_size_unit, file_size,
                                            filename_prefix,
                                            local_destination_folder,
                                            upload_to_gcs_bucket,
                                            logging_method, log):

  for batch_start in range(1, num_of_files + BATCH_SIZE + 1, BATCH_SIZE):
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
          'gsutil -m cp -r {}/* {}'.format(TEMPORARY_DIRECTORY,
                                          destination_blob_name),
          shell=True)
      process.communicate()

      exit_code = process.wait()

      if(exit_code != 0):
        if logging_method == 'output_file':
          print('Exited with code {}'.format(exit_code))
          subprocess.call('bash', shell=True)
        elif logging_method == 'return':
          return exit_code

    # Copying batch files from temporary to local destination folder:
    subprocess.call(
        'cp -r {}/* {}'.format(TEMPORARY_DIRECTORY, local_destination_folder),
        shell=True)

    # Deleting batch files from temporary folder:
    subprocess.call('rm -rf {}/*'.format(TEMPORARY_DIRECTORY), shell=True)

    # Writing number of files uploaded to output file after every batch uploads:
    if logging_method == 'output_file':
      logmessage('{}/{} files uploaded to {}\n'.format(file_num, num_of_files,
                                                      destination_blob_name))
    elif logging_method == 'return':
      log.info('%s/%s files created.\n', min(file_num, num_of_files), num_of_files)

    if file_num > num_of_files:
      return 0

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

  # Checking that gsutil is installed:
  logmessage('Checking whether gsutil is installed.\n')
  process = Popen('gsutil version', shell=True)
  process.communicate()
  exit_code = process.wait()
  if(exit_code != 0):
    print('Gsutil not installed.')
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

    generate_files_and_upload_to_gcs_bucket(destination_blob_name,
                                            int(num_of_files), file_size_unit,
                                            int(file_size), file_name_prefix,
                                            local_destination_folder,
                                            True, 'output_file', None)
    keep_files = args.keep_files
    if(keep_files == False):
    # Deleting bucket directory:
      logmessage('Deleting the local directories.\n')
      subprocess.call(['rm', '-r', bucket_name])

  # Deleting temporary folder:
  logmessage('Deleting the temporary directory.\n')
  subprocess.call(['rm', '-r', TEMPORARY_DIRECTORY])

  logmessage('Process complete.\n')

