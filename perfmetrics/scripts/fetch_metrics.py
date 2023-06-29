"""Executes fio_metrics.py and vm_metrics.py by passing appropriate arguments.
"""
import socket
import sys
import time
import argparse
from fio import fio_metrics
from vm_metrics import vm_metrics
from gsheet import gsheet
from bigquery import bigquery
from bigquery import constants

INSTANCE = socket.gethostname()
PERIOD_SEC = 120

# Google sheet worksheets
FIO_WORKSHEET_NAME = 'fio_metrics'
VM_WORKSHEET_NAME = 'vm_metrics'


def _parse_arguments(argv):
  """Parses the arguments provided to the script via command line.

  Args:
    argv: List of arguments recevied by the script.

  Returns:
    A class containing the parsed arguments.
  """
  argv = sys.argv
  parser = argparse.ArgumentParser()
  parser.add_argument(
      'fio_json_output_path',
      help='Provide path of the output json file.',
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
      '--upload_bq',
      help='Upload the results to the BigQuery.',
      action='store_true',
      default=False,
      required=False,
  )
  parser.add_argument(
      '--config_id',
      help='Configuration ID of the experiment',
      action='store_true',
      default=False,
      required=False,
  )
  parser.add_argument(
      '--start_time_build',
      help='Start time of the build.',
      action='store_true',
      default=False,
      required=False,
  )
  return parser.parse_args(argv[1:])


if __name__ == '__main__':
  argv = sys.argv

  fio_metrics_obj = fio_metrics.FioMetrics()
  print('Getting fio metrics...')

  args = _parse_arguments(argv)

  temp = fio_metrics_obj.get_metrics(args.fio_json_output_path)
  metrics_data = fio_metrics_obj.get_values_to_upload(temp)

  if args.upload_gs:
    fio_metrics_obj.upload_metrics_to_gsheet(metrics_data, FIO_WORKSHEET_NAME)

  if args.upload_bq:
    if not args.config_id or not args.start_time_build:
      raise Exception("Pass required arguments experiments configuration ID and start time of build for uploading to BigQuery")
    bigquery_obj = bigquery.ExperimentsGCSFuseBQ(constants.PROJECT_ID, constants.DATASET_ID)
    fio_metrics_obj.upload_metrics_to_bigquery(metrics_data, args.config_id[0], args.start_time_build[0], constants.FIO_TABLE_ID)

  print('Waiting for 360 seconds for metrics to be updated on VM...')
  # It takes up to 240 seconds for sampled data to be visible on the VM metrics graph
  # So, waiting for 360 seconds to ensure the returned metrics are not empty.
  # Intermittently custom metrics are not available after 240 seconds, hence
  # waiting for 360 secs instead of 240 secs
  time.sleep(360)

  # Print the start and end time of all the fio-jobs
  # This is just make sure, the total run time of fio jobs are constant over
  # different execution.
  for ind, job in enumerate(temp):
    start_time_sec = job[fio_metrics.consts.START_TIME]
    end_time_sec = job[fio_metrics.consts.END_TIME]

    print(f'Start and end time for job {ind + 1}...')

    # Print start and end time of jobs
    print("Start time: ", start_time_sec)
    print("End time: ", end_time_sec)

  vm_metrics_obj = vm_metrics.VmMetrics()
  vm_metrics_data = []
  # Getting VM metrics for every job
  for ind, job in enumerate(temp):
    start_time_sec = job[fio_metrics.consts.START_TIME]
    end_time_sec = job[fio_metrics.consts.END_TIME]

    # Print start and end time of jobs
    print("Start time: ", start_time_sec)
    print("End time: ", end_time_sec)

    rw = job[fio_metrics.consts.PARAMS][fio_metrics.consts.RW]
    print(f'Getting VM metrics for job at index {ind + 1}...')
    metrics_data = vm_metrics_obj.fetch_metrics(start_time_sec, end_time_sec,
                                                INSTANCE, PERIOD_SEC, rw)
    for row in metrics_data:
      vm_metrics_data.append(row)

  # Only some extracted metrics will be uploaded to Google Spreadsheets and BigQuery
  vm_metrics_data_upload = [row[1:] + [None]*8 for row in vm_metrics_data]

  if args.upload_gs:
    gsheet.write_to_google_sheet(VM_WORKSHEET_NAME, vm_metrics_data_upload)

  if args.upload_bq:
    if not args.config_id or not args.start_time_build:
      raise Exception("Pass required arguments experiments configuration ID and start time of build for uploading to BigQuery")
    bigquery_obj = bigquery.ExperimentsGCSFuseBQ(constants.PROJECT_ID, constants.DATASET_ID)
    bigquery_obj.upload_metrics_to_table(constants.VM_TABLE_ID, args.config_id[0], args.start_time_build[0], vm_metrics_data_upload)
