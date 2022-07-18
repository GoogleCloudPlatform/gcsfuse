"""Executes fio_metrics.py and vm_metrics.py by passing appropriate arguments.
"""
import socket
import sys
import time
from fio import fio_metrics
from vm_metrics import vm_metrics
from gsheet import gsheet

INSTANCE = socket.gethostname()
PERIOD_SEC = 120

# Google sheet worksheets
FIO_WORKSHEET_NAME = 'fio_metrics'
VM_WORKSHEET_NAME = 'vm_metrics'

if __name__ == '__main__':
  argv = sys.argv
  if len(argv) != 2:
    raise TypeError('Incorrect number of arguments.\n'
                    'Usage: '
                    'python3 fetch_metrics.py <fio output json filepath>')

  fio_metrics_obj = fio_metrics.FioMetrics()
  print('Getting fio metrics...')
  temp = fio_metrics_obj.get_metrics(argv[1], FIO_WORKSHEET_NAME)

  print('Waiting for 250 seconds for metrics to be updated on VM...')
  # It takes up to 240 seconds for sampled data to be visible on the VM metrics graph
  # So, waiting for 250 seconds to ensure the returned metrics are not empty
  time.sleep(250)

  vm_metrics_obj = vm_metrics.VmMetrics()
  vm_metrics_data = []
  # Getting VM metrics for every job
  for ind, job in enumerate(temp):
    start_time_sec = job[fio_metrics.consts.START_TIME]
    end_time_sec = job[fio_metrics.consts.END_TIME]
    rw = job[fio_metrics.consts.PARAMS][fio_metrics.consts.RW]
    print(f'Getting VM metrics for job at index {ind+1}...')
    metrics_data = vm_metrics_obj.fetch_metrics(start_time_sec, end_time_sec, INSTANCE, PERIOD_SEC, rw)
    for row in metrics_data:
      vm_metrics_data.append(row)

  gsheet.write_to_google_sheet(VM_WORKSHEET_NAME, vm_metrics_data)

