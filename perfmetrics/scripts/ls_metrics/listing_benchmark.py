"""Python script for benchmarking listing operation.

This python script benchmarks and compares the latency of listing operation in
persistent disk vs GCS bucket. It creates the necessary directory structure,
containing files and folders, needed to test the listing operation. Furthermore it can
optionally upload the results of the test to a Google Sheet. It takes input a
config file through which multiple tests of different configurations can be
performed in a single run.

Typical usage example:
  $ python3 listing_benchmark.py [-h] [--keep_files] [--upload] [--message MESSAGE] config_file

  Flag -h: Typical help interface of the script.
  Flag --keep_files: Do not delete the generated directory structure from the persistent disk after running the tests.
  Flag --upload: Uploads the results of the test to the Google Sheet.
  Flag --message MESSAGE: Takes input a message string, which describes/titles the test.
  config_file (required): Path to the config file which contains the details of the tests.

Note: This python script is dependent on generate_files.py. So keep both the files at the same location.
"""

import argparse
import ast
import configparser
import logging
import os
import statistics as stat
import subprocess
import sys
import time

import generate_files
import numpy as np
import texttable


logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)],
)
log = logging.getLogger()


def OutputResults(config, gcs_bucket_results, persistent_disk_results, message) -> None:
  """Outputs the results on the console.

  This function takes in dictionary containing the list of results (for all samples) for each
  testing folder, for both the gcs bucket and persistent disk. Then it generates various metrics
  out of these lists and outputs them into the console. It further uses the texttable library to
  output the results in forms of tables. A single row of the table represents a single test(testing folder).
  For every test a side by side comparision is made for the GCS bucket and persistent disk.
  Also a quantile comparision is shown in the table.

  The metrics present in the output are (in msec):
  Mean, Median, Standard Dev, 0th %ile, 20th %ile, 40th %ile, 60th %ile, 80th %ile, 90th %ile,
  95th %ile, 98th %ile, 99th %ile, 100th %ile.

  Args:
    config: Dictionary containing the parsed config file.
    gcs_bucket_results: Dictionary containing the list of results (for all samples)
                        for each testing folder in the GCS bucket.
    persistent_disk_results: Dictionary containing the list of results (for all samples)
                             for each testing folder in the persistent disk.
    message: String which describes/titles the test.
  """
  log.info('Showing Results of %s samples:\n', config['numSamples'])

  print('Test Description:  {}\n'.format(message))

  main_table_contents = []
  main_table_contents.append(['Test Desc.', 'GCS Bucket Results', 'Persistent Disk Results', 'Quantile Comparision'])

  for testing_folder in config['folderList']:
    gcs_bucket_results[testing_folder] = sorted(gcs_bucket_results[testing_folder])
    persistent_disk_results[testing_folder] = sorted(persistent_disk_results[testing_folder])

    gcs_bucket_subtable = texttable.Texttable()
    gcs_bucket_subtable.set_deco(texttable.Texttable.HEADER)
    gcs_bucket_subtable.set_cols_dtype(['t', 'f'])
    gcs_bucket_subtable.set_cols_align(['l', 'r'])
    gcs_bucket_subtable.add_rows([
        ['Statistic', 'Value (msec)'],
        ['Mean', stat.mean(gcs_bucket_results[testing_folder])],
        ['Median', stat.median(gcs_bucket_results[testing_folder])],
        ['Standard Dev', stat.stdev(gcs_bucket_results[testing_folder])],
    ])

    persistent_disk_subtable = texttable.Texttable()
    persistent_disk_subtable.set_deco(texttable.Texttable.HEADER)
    persistent_disk_subtable.set_cols_dtype(['t', 'f'])
    persistent_disk_subtable.set_cols_align(['l', 'r'])
    persistent_disk_subtable.add_rows([
        ['Statistic', 'Value (msec)'],
        ['Mean', stat.mean(persistent_disk_results[testing_folder])],
        ['Median', stat.median(persistent_disk_results[testing_folder])],
        ['Standard Dev', stat.stdev(persistent_disk_results[testing_folder])],

    ])

    quantiles_gcs = []
    for percentile in range(0, 100, 20):
      quantiles_gcs.append(np.percentile(gcs_bucket_results[testing_folder], percentile))

    quantiles_gcs.append(np.percentile(gcs_bucket_results[testing_folder], 90))
    quantiles_gcs.append(np.percentile(gcs_bucket_results[testing_folder], 95))
    quantiles_gcs.append(np.percentile(gcs_bucket_results[testing_folder], 98))
    quantiles_gcs.append(np.percentile(gcs_bucket_results[testing_folder], 99))
    quantiles_gcs.append(np.percentile(gcs_bucket_results[testing_folder], 100))

    quantiles_pd = []
    for percentile in range(0, 100, 20):
      quantiles_pd.append(np.percentile(persistent_disk_results[testing_folder], percentile))

    quantiles_pd.append(np.percentile(persistent_disk_results[testing_folder], 90))
    quantiles_pd.append(np.percentile(persistent_disk_results[testing_folder], 95))
    quantiles_pd.append(np.percentile(persistent_disk_results[testing_folder], 98))
    quantiles_pd.append(np.percentile(persistent_disk_results[testing_folder], 99))
    quantiles_pd.append(np.percentile(persistent_disk_results[testing_folder], 100))

    quantile_table = texttable.Texttable()
    quantile_table.set_deco(texttable.Texttable.HEADER)
    quantile_table.set_cols_dtype(['t', 'f', 'f'])
    quantile_table.set_cols_align(['l', 'r', 'r'])
    quantile_table.add_rows([
        ['Quantile', 'GCS Bucket', 'Persistent Disk'],
        ['0th %ile', quantiles_gcs[0], quantiles_pd[0]],
        ['20th %ile', quantiles_gcs[1], quantiles_pd[1]],
        ['40th %ile', quantiles_gcs[2], quantiles_pd[2]],
        ['60th %ile', quantiles_gcs[3], quantiles_pd[3]],
        ['80th %ile', quantiles_gcs[4], quantiles_pd[4]],
        ['90th %ile', quantiles_gcs[5], quantiles_pd[5]],
        ['95th %ile', quantiles_gcs[6], quantiles_pd[6]],
        ['98th %ile', quantiles_gcs[7], quantiles_pd[7]],
        ['99th %ile', quantiles_gcs[8], quantiles_pd[8]],
        ['100th %ile', quantiles_gcs[9], quantiles_pd[9]]
    ])

    main_table_contents.append(['{}subdir_{}files_{}'.format(config[testing_folder]['numSubDirectories'],
                                                             config[testing_folder]['numFiles'],
                                                             config[testing_folder]['fileSizeString']),
                                gcs_bucket_subtable.draw(),
                                persistent_disk_subtable.draw(),
                                quantile_table.draw()])

  main_table = texttable.Texttable()
  main_table.set_cols_align(['l', 'c', 'c', 'c'])
  main_table.set_cols_valign(['m', 'm', 'm', 'm'])
  main_table.set_cols_width([30, 35, 35, 60])
  main_table.add_rows(main_table_contents)

  print(main_table.draw())
  print('')


def Testing(config, gcs_bucket):
  """This function tests the listing operation on the testing folders.

  Going through all the testing folders one by one for both GCS bucket and
  peristent disk, we calculate the latency (in msec) of listing operation
  and store the results in a list of that particular testing folder. Reading
  are taken multiple times as specified in config['numSamples'].

  Args:
    config: Dictionary containing the parsed config file.
    gcs_bucket: Name of the directory to which GCS bucket is mounted to.

  Returns:
    gcs_bucket_results: A dictionary containing the list of results (all samples)
                        for each testing folder.
    persistent_disk_results: A dictionary containing the list of results (all samples)
                             for each testing folder.
  """
  def RecordTimeOfOperation(command, path):
    st = time.time()
    subprocess.call('{} {}'.format(command, path), shell=True,
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.STDOUT)
    et = time.time()
    return (et-st)*1000

  gcs_bucket_results = {}
  persistent_disk_results = {}

  command = config['command']
  for flag in config['commandFlags']:
    command += ' '
    command += flag

  for idx in range(config['numSamples']):
    log.info('Started testing for sample number: %s.\n', idx+1)

    for testing_folder in config['folderList']:
      local_dir_path = './{}/{}/'.format(config['rootFolder'], testing_folder)
      gcs_bucket_path = './{}/{}/{}/'.format(gcs_bucket, config['rootFolder'], testing_folder)

      if testing_folder not in persistent_disk_results.keys():
        persistent_disk_results[testing_folder] = []

      if testing_folder not in gcs_bucket_results.keys():
        gcs_bucket_results[testing_folder] = []

      interval = RecordTimeOfOperation(command, local_dir_path)
      persistent_disk_results[testing_folder].append(interval)

      interval = RecordTimeOfOperation(command, gcs_bucket_path)
      gcs_bucket_results[testing_folder].append(interval)

  log.info('Testing completed. Generating output.\n')
  return gcs_bucket_results, persistent_disk_results


def CreateDirectoryStructure(config, gcs_bucket, create_files_in_gcs) -> None:
  """Creates new directory structure using generate_files.py as a library.

  This function creates new directory structure in persistent disk. If create_files_in_gcs
  is True, then it also creates the same structure in GCS bucket.
  For more info regarding how the generation of files is happening, please read the generate_files.py.

  Args:
    config: Dictionary containing the parsed config file.
    gcs_bucket: Name of the directory to which the GCS bucket is mounted to.
    create_files_in_gcs: Bool value which is True if we have to create files in GCS bucket (same directory
                         strucutre not present). Otherwise it is False, means that we will not
                         create files in GCS bucket from scratch.

  Raises:
    An error into the logs if an error is returned by the generate_files.py script. Error means that
    files cannot be uploaded sucessfully to the GCS bucket.
  """
  if os.path.exists('./{}'.format(config['rootFolder'])):
    subprocess.call('rm -rf {}'.format(config['rootFolder']), shell=True)
  subprocess.call('mkdir ./{}'.format(config['rootFolder']), shell=True)

  if create_files_in_gcs and os.path.exists('./{}/{}'.format(gcs_bucket, config['rootFolder'])):
    log.info('Deleting previously present directory in the GCS bucket.\n')
    subprocess.call('rm -rf ./{}/{}'.format(gcs_bucket, config['rootFolder']), shell=True)

  if create_files_in_gcs:
    subprocess.call('mkdir ./{}/{}'.format(gcs_bucket, config['rootFolder']), shell=True)

  temp_dir = generate_files.TEMPORARY_DIRECTORY
  if os.path.exists(os.path.dirname(temp_dir)):
    subprocess.call('rm -rf {}'.format(os.path.dirname(temp_dir)), shell=True)
  subprocess.call('mkdir -p {}'.format(temp_dir), shell=True)

  for testing_folder in config['folderList']:
    subprocess.call('mkdir ./{}/{}'.format(config['rootFolder'], testing_folder), shell=True)
    if create_files_in_gcs:
      subprocess.call('mkdir ./{}/{}/{}'.format(gcs_bucket, config['rootFolder'], testing_folder), shell=True)

    if config[testing_folder]['numSubDirectories'] == 0:
      local_destination_path = './{}/{}/'.format(config['rootFolder'], testing_folder)
      destination_blob_name = 'gs://{}/{}/{}/'.format(config['bucketName'], config['rootFolder'], testing_folder)

      exit_code = generate_files.generate_files_and_upload_to_gcs_bucket(destination_blob_name,
                                                                         config[testing_folder]['numFiles'],
                                                                         'b', config[testing_folder]['fileSize'],
                                                                         config[testing_folder]['fileNamePrefix'],
                                                                         local_destination_path, create_files_in_gcs,
                                                                         'return', log)

      if exit_code != 0:
        log.error('Cannot upload files to the GCS bucket. Exited with exit code %s.\n', exit_code)
        subprocess.call('rm -rf {}'.format(os.path.dirname(temp_dir)), shell=True)
        UnmountGCSBucket(gcs_bucket)
        subprocess.call('bash', shell=True)

    else:
      remainder = config[testing_folder]['numFiles'] % config[testing_folder]['numSubDirectories']
      files_in_subdir = config[testing_folder]['numFiles'] // config[testing_folder]['numSubDirectories']

      for subdir_index in range(config[testing_folder]['numSubDirectories']):
        subdir_name = '{}_{}'.format(config[testing_folder]['subDirectoryNamePrefix'], subdir_index)

        subprocess.call('mkdir ./{}/{}/{}'.format(config['rootFolder'], testing_folder, subdir_name), shell=True)
        if create_files_in_gcs:
          subprocess.call('mkdir -p ./{}/{}/{}/{}'.format(gcs_bucket, config['rootFolder'],
                                                          testing_folder, subdir_name), shell=True)

        if remainder > 0:
          local_destination_path = './{}/{}/{}/'.format(config['rootFolder'], testing_folder, subdir_name)
          destination_blob_name = 'gs://{}/{}/{}/{}/'.format(config['bucketName'],
                                                             config['rootFolder'], testing_folder, subdir_name)

          exit_code = generate_files.generate_files_and_upload_to_gcs_bucket(destination_blob_name,
                                                                             files_in_subdir + 1,
                                                                             'b', config[testing_folder]['fileSize'],
                                                                             config[testing_folder]['fileNamePrefix'],
                                                                             local_destination_path, create_files_in_gcs,
                                                                             'return', log)

          remainder -= 1

        else:
          local_destination_path = './{}/{}/{}/'.format(config['rootFolder'], testing_folder, subdir_name)
          destination_blob_name = 'gs://{}/{}/{}/{}/'.format(config['bucketName'],
                                                             config['rootFolder'], testing_folder, subdir_name)

          exit_code = generate_files.generate_files_and_upload_to_gcs_bucket(destination_blob_name,
                                                                             files_in_subdir, 'b',
                                                                             config[testing_folder]['fileSize'],
                                                                             config[testing_folder]['fileNamePrefix'],
                                                                             local_destination_path, create_files_in_gcs,
                                                                             'return', log)

        if exit_code != 0:
          log.error('Cannot upload files to the GCS bucket. Exited with exit code %s.\n', exit_code)
          subprocess.call('rm -rf {}'.format(os.path.dirname(temp_dir)), shell=True)
          UnmountGCSBucket(gcs_bucket)
          subprocess.call('bash', shell=True)

  subprocess.call('rm -rf {}'.format(os.path.dirname(temp_dir)), shell=True)
  log.info('Files created successfully.\n')
  return


def CompareDirectoryStructure(config, gcs_bucket) -> bool:
  """Compares the directory structure present in the GCS bucket with the structure present in the config file.

  This function checks whether a same directory structure is present in the GCS bucket or not. If a same structure
  is already present then we dont have to make the whole structure from scratch everytime. This saves a lot of time
  as writing to the GCS bucket is a time expensive operation.
  The structure should be exactly same including the files and folders names to be same with the config.

  Steps to check the directory structure:
    1. Firstly check whether there is a folder with name config['rootFolder'] in the GCS Bucket. If not then
       return false. If yes then this is the root folder.

    2. Get a recursive list of the root folder which consist of all the files and folders present
       inside the root folder.

    3. Iterate in the recursive list and perform the following checks:
        -> If the current directory is root folder then just check that there should be no files present at the
           current level and and the number of subdirectories (testing folders) should be equal to
           config['numFolders'].

        -> If current directory is a testing folder then perform the following checks:
              = Name of the current directory should be present in the keys config dictionary.

              = Number of subdirectories should be equal to config[curr_directory]['numSubDirectories'].

              = If number of subdirectories are 0 then the number of files present in the testing folder should
                be equal to the config[curr_directory]['numFiles']. Also the size of each files should be equal to
                config[curr_directory]['fileSize']. At last the name of the files should match the prefix name
                mentioned in the config.

              = If number of subdirectories is not equal to 0 then there should be no files present in the
                current level. Also the names of the subdirectories should match prefix name mentioned in
                the config.

        -> If the current directory is a subdirectory inside the testing folder then perform the following checks:
              = There should be no subdirectories in this level.

              = The size of each file should be equal to config[testing_folder]['fileSize'] and the name of each file
                should match the file prefix name present in the config.

              = Files in testing folder should be equally distributed among the subdirectories.

  Args:
    config: Dictionary containing the parsed config file.
    gcs_bucket: Name of the directory to which the GCS bucket is mounted.

  Returns:
    A boolean value which is true if the exact same directory structure is already present in the GCS bucket.
    Otherwise returns a false and we have to create the directory structure from scratch in the GCS bucket.
  """
  def PrintNotFound():
    os.chdir('..')
    log.info(
        'Similar directory structure not found in the bucket. Creating a new one.\n')
    return False

  os.chdir('./{}'.format(gcs_bucket))

  if not os.path.exists('./{}'.format(config['rootFolder'])):
    return PrintNotFound()

  recursive_list = os.walk('./{}'.format(config['rootFolder']))

  curr_folder = ''
  rem_folders = 0
  count_files = 0

  for curr_level in recursive_list:
    curr_directory_path, sub_directories, files = curr_level
    curr_directory = os.path.basename(curr_directory_path)

    if os.path.join('./', config['rootFolder']) == curr_directory_path:
      if files or len(sub_directories) != config['numFolders']:
        return PrintNotFound()

    elif os.path.join('./', config['rootFolder'], curr_directory) == curr_directory_path:
      if curr_directory not in config.keys():
        return PrintNotFound()

      if len(sub_directories) != config[curr_directory]['numSubDirectories']:
        return PrintNotFound()

      if not sub_directories and len(files) == config[curr_directory]['numFiles']:
        for file in files:
          if config[curr_directory]['fileSize'] != os.stat(os.path.join(curr_directory_path, file)).st_size:
            return PrintNotFound()

      if not sub_directories and len(files) != config[curr_directory]['numFiles']:
        return PrintNotFound()

      if not sub_directories:
        files = sorted(files)
        idx = 1
        for file in files:
          if file != '{}_{}.txt'.format(config[curr_directory]['fileNamePrefix'], idx):
            return PrintNotFound()
          idx += 1
        continue

      curr_folder = curr_directory
      rem_folders = len(sub_directories)
      count_files = 0
      remainder = config[curr_folder]['numFiles'] % config[curr_folder]['numSubDirectories']

      sub_directories = sorted(sub_directories)
      idx = 0
      for sub_directory in sub_directories:
        if sub_directory != '{}_{}'.format(config[curr_directory]['subDirectoryNamePrefix'], idx):
          return PrintNotFound()
        idx += 1

      if sub_directories and files:
        return PrintNotFound()

    elif os.path.join('./', config['rootFolder'], curr_folder, curr_directory) == curr_directory_path:
      count_files += len(files)
      rem_folders -= 1

      if sub_directories:
        return PrintNotFound()

      curr_directory_num = int(curr_directory.split('_')[-1])
      if (curr_directory_num < remainder
          and len(files) != config[curr_folder]['numFiles'] // config[curr_folder]['numSubDirectories'] + 1):
        return PrintNotFound()
      elif (curr_directory_num >= remainder
            and len(files) != config[curr_folder]['numFiles'] // config[curr_folder]['numSubDirectories']):
        return PrintNotFound()

      files = sorted(files)
      idx = 1
      for file in files:
        if file != '{}_{}.txt'.format(config[curr_folder]['fileNamePrefix'], idx):
          return PrintNotFound()
        idx += 1

      for file in files:
        if config[curr_folder]['fileSize'] != os.stat(os.path.join(curr_directory_path, file)).st_size:
          return PrintNotFound()

      if rem_folders == 0 and count_files != config[curr_folder]['numFiles']:
        return PrintNotFound()

  log.info("""Similar directory structure already present in GCS bucket.
           Will not create a new directory structure for the bucket.\n""")
  os.chdir('..')
  return True


def UnmountGCSBucket(gcs_bucket) -> None:
  """Unmounts the GCS bucket.

  Args:
    gcs_bucket: Name of the directory to which GCS bucket is mounted to.

  Raises:
    An error if the GCS bucket cound not be umnounted.
  """
  log.info('Unmounting the GCS Bucket.\n')

  exit_code = subprocess.call('umount -l {}'.format(gcs_bucket), shell=True)

  if exit_code != 0:
    log.error('Error encountered in umounting the bucket. Aborting!\n')
    subprocess.call('bash', shell=True)

  else:
    subprocess.call('rm -rf {}'.format(gcs_bucket), shell=True)
    log.info('Successfully unmounted the bucket and deleted %s directory.\n', gcs_bucket)


def MountGCSBucket(bucket_name) -> str:
  """Mounts the GCS bucket into the gcs_bucket directory.

  Args:
    bucket_name: Name of the bucket to be mounted.

  Returns:
    A string which contains the name of the directory to which the bucket is mounted.

  Raises:
    An error into the logs if cannot mount the bucket due to some error. Also shut offs the program.
  """
  log.info('Started mounting the GCS Bucket using GCSFuse.\n')
  gcs_bucket = 'gcsBucket'

  subprocess.call('mkdir {}'.format(gcs_bucket), shell=True)

  exit_code = subprocess.call('gcsfuse --implicit-dirs --disable-http2 --max-conns-per-host 100 {} {}'.format(
      bucket_name, gcs_bucket), shell=True)
  print('')

  if exit_code != 0:
    log.error('Cannot mount the GCS bucket due to exit code %s.\n', exit_code)
    subprocess.call('bash', shell=True)
  else:
    return gcs_bucket


def ParseArguments(argv):
  """Parses the arguments provided to the script via command line.

  Args:
    argv: List of arguments recevied by the script.

  Returns:
    A class containing the parsed arguments.
  """
  argv = sys.argv

  parser = argparse.ArgumentParser()

  parser.add_argument(
      'config_file',
      help='Provide path of the config file.',
      action='store'
  )

  parser.add_argument(
      '--keep_files',
      help='Does not delete the directory structure in persistent disk.',
      action='store_true',
      default=False,
      required=False,
  )

  parser.add_argument(
      '--upload',
      help='Upload the results to the Google Sheet.',
      action='store_true',
      default=False,
      required=False,
  )

  parser.add_argument(
      '--message',
      help='Puts a message/title describing the test.',
      action='store',
      nargs=1,
      default='Performance Listing Benchmark',
      required=False,
  )

  args = parser.parse_args(argv[1:])

  return args


def ParseConfig(config_file_path) -> dict:
  """Parses the config file provived to the script via command line.

  Notation:
    config_dict['bucketName']: Name of the GCS bucket.
    config_dict['command']: Command to do the testing on.
    config_dict['commandFlags']: Flags to run with the command.
    config_dict['numSamples']: Number of tests to run for each testing folder.
    config_dict['numFolders']: Number of testing folders present.
    config_dict['listFolder']: List containing the names of the testing folders.
    config_dict['rootFolder']: Name of the root folder. Root folder is nothing but an environment to keep all the
                               tests / testing folder at one place.

    config_dict[folder_name]: Dictionary containing the configuration of testing folder (with name same as
                              string folder_name). Testing folder represents a single test or single
                              section(config) of the config file. Multiple testing folder / configs can be
                              created to run the benchmarking for different scenarios of number of files
                              and folders.

    config_dict[folder_name]['folder']: Name of the testing folder.(folder_name == config[folder_name]['folder])
    config_dict[folder_name]['numFiles']: Number of files in the testing folder.
    config_dict[folder_name]['numSubDirectories']: Number of subdirectories in the testing folder.
    config_dict[folder_name]['fileSize']: Size in bytes of the files present in the testing folder.
    config[folder_name]['fileSizeString']: String containing the size of files in native format. i.e.
                                           if the sizes in config file were in kb then it will hold
                                           the sizes in kb.

    config[folder_name]['fileNamePrefix']: Prefix of the file names in the testing folder.
    config[folder_name]['subDirectoryNamePrefix']: Prefix of the subdirectory names in the testing folder.

  Args:
    config_file_path: Path of the config file.

  Returns:
    A dictionary containing the parsed config file.
  """
  config = configparser.ConfigParser()
  config.read(os.path.abspath(config_file_path))

  config_dict = {}

  config_dict['bucketName'] = config['DEFAULT']['bucket_name']
  config_dict['command'] = config['DEFAULT']['command']

  if 'command_flags' in config['DEFAULT']:
    config_dict['commandFlags'] = ast.literal_eval(
        config.get('DEFAULT', 'command_flags')
    )

  config_dict['numSamples'] = int(config['DEFAULT']['num_samples'])
  config_dict['rootFolder'] = config['DEFAULT']['root_folder']

  count = 1

  folder_list = []
  for section in config.sections():
    testing_folder = config[section]['folder']
    num_files = int(config[section]['num_of_files'])
    num_sub_directories = int(config[section]['num_of_subdir'])

    file_size_string = ''
    file_size = 0
    if int(config[section]['num_of_files']) > 0:
      file_size_string = config[section]['file_size']
      file_size_unit = config[section]['file_size'][-2:]
      file_size = config[section]['file_size'][:-2]

      if file_size_unit.lower() == 'gb':
        file_size = (1024 * 1024 * 1024 * int(file_size))

      if file_size_unit.lower() == 'mb':
        file_size = (1024 * 1024 * int(file_size))

      if file_size_unit.lower() == 'kb':
        file_size = (1024 * int(file_size))

    file_name_prefix = ''
    if int(config[section]['num_of_files']) > 0:
      file_name_prefix = config[section]['file_name_prefix']

    sub_directory_name_prefix = ''
    if int(config[section]['num_of_subdir']) > 0:
      sub_directory_name_prefix = config[section]['subdir_name_prefix']

    sub_directory_dict = {
        'folderName': testing_folder,
        'numFiles': num_files,
        'numSubDirectories': num_sub_directories,
        'fileSize': file_size,
        'fileNamePrefix': file_name_prefix,
        'subDirectoryNamePrefix': sub_directory_name_prefix,
        'fileSizeString': file_size_string
    }

    folder_list.append(testing_folder)
    config_dict[testing_folder] = sub_directory_dict
    count += 1

  config_dict['numFolders'] = count - 1
  config_dict['folderList'] = folder_list

  return config_dict


def CheckDependencies(packages) -> None:
  """Check whether the dependencies are installed or not.

  Args:
    packages: List containing the names of the dependencies to be checked.

  Raises:
    An error into the logs if a particular dependency is not installed and shuts of the program.
  """
  for curr_package in packages:
    log.info('Checking whether %s is installed.\n', curr_package)

    exit_code = subprocess.call(
        '{} --version'.format(curr_package), shell=True)
    print('')

    if exit_code != 0:
      log.error('%s not installed. Please install. Aborting!\n', curr_package)
      subprocess.call('bash', shell=True)

  return


if __name__ == '__main__':
  argv = sys.argv
  if len(argv) < 2:
    raise TypeError('Incorrect number of arguments.\n'
                    'Usage: '
                    'python3 listing_benchmark.py [--keep_files] [--upload] [--message MESSAGE] config_file')

  args = ParseArguments(argv)

  CheckDependencies(['gsutil', 'gcsfuse'])

  config = ParseConfig(args.config_file)

  gcs_bucket = MountGCSBucket(config['bucketName'])

  directory_structure_present = CompareDirectoryStructure(config, gcs_bucket)

  CreateDirectoryStructure(config, gcs_bucket, not directory_structure_present)

  gcs_bucket_results, persistent_disk_results = Testing(config, gcs_bucket)

  OutputResults(config, gcs_bucket_results, persistent_disk_results, args.message)

  if not args.keep_files:
    log.info('Deleting files from persistent disk.\n')
    subprocess.call('rm -rf {}'.format(config['rootFolder']), shell=True)

  UnmountGCSBucket(gcs_bucket)
