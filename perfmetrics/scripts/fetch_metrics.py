"""Executes fio_metrics.py and vm_metrics.py by passing appropriate arguments.
"""
import socket
import sys
import time
import argparse
from fio import fio_metrics
from vm_metrics import vm_metrics
from gsheet import gsheet

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
      '--upload',
      help='Upload the results to the Google Sheet.',
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

  if args.upload:
    temp = fio_metrics_obj.get_metrics(args.fio_json_output_path, FIO_WORKSHEET_NAME)
  else:
    temp = fio_metrics_obj.get_metrics(args.fio_json_output_path)

  print('Waiting for 360 seconds for metrics to be updated on VM...')
  # It takes up to 240 seconds for sampled data to be visible on the VM metrics graph
  # So, waiting for 360 seconds to ensure the returned metrics are not empty.
  # Intermittenly custom metrics are not available after 240 seconds, hence
  # waiting for 360 secs instead of 240 secs
  time.sleep(360)

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

  if args.upload:
    gsheet.write_to_google_sheet(VM_WORKSHEET_NAME, vm_metrics_data)
