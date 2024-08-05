import os
import json
import sys
import logging
import argparse
import subprocess
import time
import statistics as stat
import re
import numpy as np

sys.path.insert(0, '..')
from utils.mount_unmount_util import mount_gcs_bucket,unmount_gcs_bucket
from utils.checks_util import check_dependencies


# The script requires the num of samples to be even in order to restore test
# data to original state.
# Common flags for both flat and hns bucket mounting.
GCSFUSE_MOUNT_FLAGS= "--implicit-dirs --rename-dir-limit=1000000"

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)],
)
log = logging.getLogger()

def _get_values_to_export(dir,metrics,test_type):
  metrics_data=[]
  # Getting values corrresponding to non nested folders.
  for folder in dir["folders"]["folder_structure"]:
    num_files = folder["num_files"]
    num_folders = 1

    row = [
        'Renaming Operation',
        test_type,
        num_files,
        num_folders,
        metrics[test_type][folder["name"]]['Number of samples'],
        metrics[test_type][folder["name"]]['Mean'],
        metrics[test_type][folder["name"]]['Median'],
        metrics[test_type][folder["name"]]['Standard Dev'],
        metrics[test_type][folder["name"]]['Quantiles']['0 %ile'],
        metrics[test_type][folder["name"]]['Quantiles']['20 %ile'],
        metrics[test_type][folder["name"]]['Quantiles']['50 %ile'],
        metrics[test_type][folder["name"]]['Quantiles']['90 %ile'],
        metrics[test_type][folder["name"]]['Quantiles']['95 %ile'],
        metrics[test_type][folder["name"]]['Quantiles']['98 %ile'],
        metrics[test_type][folder["name"]]['Quantiles']['99 %ile'],
        metrics[test_type][folder["name"]]['Quantiles']['99.5 %ile'],
        metrics[test_type][folder["name"]]['Quantiles']['99.9 %ile'],
        metrics[test_type][folder["name"]]['Quantiles']['100 %ile']

    ]

    metrics_data.append(row)

  return metrics_data


def _parse_results(dir,results,num_samples ,test_type):
  test_metrics = dict()
  metrics = dict()
  # Parsing metrics for non-nested folders
  for folder in dir["folders"]["folder_structure"]:
    folder_name= folder["name"]
    metrics[folder_name]=dict()
    metrics[folder_name]['Number of samples'] = num_samples

    # Sorting based on time to get the mean,median,etc.
    results[test_type][folder_name]=sorted(results[test_type][folder_name])

    metrics[folder_name]['Mean']=round(
        stat.mean(results[test_type][folder_name]),3)
    metrics[folder_name]['Median'] = round(
        stat.median(results[test_type][folder_name]), 3)
    metrics[folder_name]['Standard Dev'] = round(
        stat.stdev(results[test_type][folder_name]), 3)
    metrics[folder_name]['Quantiles'] = dict()
    sample_set = [0, 20, 50, 90, 95, 98, 99, 99.5, 99.9, 100]
    for percentile in sample_set:
      metrics[folder_name]['Quantiles'][
        '{} %ile'.format(percentile)] = round(
          np.percentile(results[test_type][folder_name], percentile), 3)
  test_metrics[test_type]=metrics
  return test_metrics


def _extract_folder_name_prefix(folder_name) ->(str):
  """
  Extract the prefix from folder_name_for_ease of rename
  """
  try:
    folder_prefix = re.search("(?s:.*)\_", folder_name).group()
    return folder_prefix
  except:
    log.error("Folder name format is incorrect. Must be in the format prefix_0 \
          to begin.Exiting...")
    subprocess.call('bash',shell=True)


def _record_time_of_operation(mount_point,dir,num_samples):
  results= dict()
  # Collecting metrics for non-nested folders
  for folder in dir["folders"]["folder_structure"]:
    folder_prefix = _extract_folder_name_prefix(folder["name"])
    folder_path_prefix='{}/{}'.format(mount_point,folder_prefix)
    time_op=[]
    for iter in range(num_samples):
      if iter < num_samples/2:
        rename_from='{}{}'.format(folder_path_prefix,iter)
        rename_to= '{}{}'.format(folder_path_prefix,iter+1)
      else:
        rename_from='{}{}'.format(folder_path_prefix,num_samples - iter)
        rename_to= '{}{}'.format(folder_path_prefix,num_samples - iter-1)
      start_time_sec = time.time()
      subprocess.call('mv ./{} ./{}'.format(rename_from,rename_to),shell=True)
      end_time_sec = time.time()
      time_op.append(end_time_sec - start_time_sec)


    results[folder["name"]]=time_op

  return results


def _perform_testing(dir,test_type,num_samples,results):

  if test_type== "flat":

    # Mounting the flat bucket .
    flat_mount_flags=GCSFUSE_MOUNT_FLAGS
    flat_bucket= mount_gcs_bucket( dir["name"],flat_mount_flags,log)

    # Record time of operation
    flat_results=_record_time_of_operation(flat_bucket,dir,num_samples)
    results["flat"]=flat_results

    unmount_gcs_bucket(dir["name"],log)

  elif test_type == "hns":
    #TODO add mount function for test type hns
    pass
  else:
      log.error('Incorrect test type passed.Must be either \"flat\" or \"hns\"\n')
      return 1



def _parse_arguments(argv):

  argv = sys.argv
  parser = argparse.ArgumentParser()

  parser.add_argument(
      'dir_config_file',
      help='Provide path of the config file.',
      action='store'
  )
  parser.add_argument(
      '--upload_gs',
      help='Upload the results to the Google Sheet.',
      action='store_true',
      default=False,
      required=False,
  )
  parser.add_argument(
      '--num_samples',
      help='Number of samples to collect of each test.',
      action='store',
      default=10,
      required=False,
      type=int,
  )

  return parser.parse_args(argv[1:])


if __name__ == '__main__':
  argv = sys.argv
  if len(argv) < 4:
    raise TypeError('Incorrect number of arguments.\n'
                    'Usage: '
                    'python3 listing_benchmark.py  [--upload_gs] [--num_samples NUM_SAMPLES]    config_file ')

  args = _parse_arguments(argv)
  check_dependencies(['gsutil', 'gcsfuse'],log)

  with open(os.path.abspath(args.dir_config_file)) as file:
    dir_str = json.load(file)

  if args.num_samples %2 !=0:
    log.error("Only even number of samples allowed to restore the test data to\
                original state at the end of test.")
    subprocess.call('bash',shell=True)

  results = dict()
  _perform_testing(dir_str,"flat",args.num_samples,results)
  flat_parsed_metrics=_parse_results(dir_str,results,args.num_samples,"flat")
  upload_values_flat = _get_values_to_export(dir_str,flat_parsed_metrics,'flat')






